package publicsrv

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"prophish/internal/store"
)

// Server is the public-facing listener that targets interact with. It serves
// landing pages and records tracking events. It deliberately captures NO
// submitted form values — only the fact that an interaction happened.
type Server struct {
	store   *store.Store
	baseURL string
	trusted []*net.IPNet // proxies whose X-Forwarded-For we honor
}

func NewServer(st *store.Store, baseURL string, trusted []*net.IPNet) *Server {
	return &Server{store: st, baseURL: strings.TrimRight(baseURL, "/"), trusted: trusted}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(publicHeaders)
	// Hardening against unauthenticated abuse of this internet-facing listener:
	// a global in-flight cap, plus a per-IP rate limit, so flooding cannot
	// amplify into the shared database.
	r.Use(middleware.Throttle(128))
	r.Use(s.rateLimit)

	r.Get("/p/{token}.png", s.handleOpen) // open-tracking pixel
	r.Get("/t/{token}", s.handleClick)    // landing page (click)
	r.Get("/c/{token}", s.handleCanary)   // hidden scanner-canary link
	r.Post("/s/{token}", s.handleSubmit)  // form submission (fact only)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	return r
}

// publicHeaders sets minimal safe headers without a strict CSP (landing pages
// intentionally contain cloned third-party markup).
func publicHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

// bannerB64 is a real, visible banner image (600x100 teal gradient) served for
// the open-tracking URL. It is deliberately NOT a 1x1 transparent pixel: some
// clients and gateways skip or strip tiny/hidden images, and an invisible image
// gives the recipient no reason to download images at all. A normal-looking
// banner renders as part of the email design, so downloading images (the
// moment we want to record) actually happens.
//
//go:embed banner.b64
var bannerB64 string

var bannerPNG, _ = base64.StdEncoding.DecodeString(strings.TrimSpace(bannerB64))
// ScannerWindows is one route's classification-window preset: how long after
// send a hit is treated as automated (Grace), and how long after a canary
// probe a hit is attributed to the same scanner burst (Pair).
type ScannerWindows struct {
	Grace time.Duration
	Pair  time.Duration
}

// The two route presets. Delivery speed is what separates them: with fast
// INTERNAL delivery a REAL user opens the mail within seconds of sending (a
// live incident had every genuine open discarded by the old 15s grace window),
// so the internal preset is short; with EXTERNAL delivery across a relay the
// message lands later, and the delivery-time prefetch that HAS no canary probe
// arrives well after send — only the long external preset catches it.
var (
	defInternal = ScannerWindows{Grace: 5 * time.Second, Pair: 5 * time.Second}
	defExternal = ScannerWindows{Grace: 15 * time.Second, Pair: 10 * time.Second}
)

// SetScannerWindows overrides the per-route window presets (any zero field
// keeps its current value). Called once at startup from the
// PROPHISH_SCANNER_*_SECONDS env vars; the optimal values depend on how fast
// delivery is on each route in the target environment.
func SetScannerWindows(internal, external ScannerWindows) {
	if internal.Grace > 0 {
		defInternal.Grace = internal.Grace
	}
	if internal.Pair > 0 {
		defInternal.Pair = internal.Pair
	}
	if external.Grace > 0 {
		defExternal.Grace = external.Grace
	}
	if external.Pair > 0 {
		defExternal.Pair = external.Pair
	}
}

// windowsFor returns the classification preset for a campaign's delivery
// route. Anything that is not explicitly "internal" gets the conservative
// external preset (matching the DB default).
func windowsFor(route string) ScannerWindows {
	if route == "internal" {
		return defInternal
	}
	return defExternal
}

// infraCIDRs are IP ranges that belong to scanner/security-proxy
// infrastructure (cloud datacenters) rather than end users. In TRUSTED
// delivery mode, hits from these ranges are ignored: the operator asserts no
// scanner touches the mail, so whatever still arrives from cloud datacenter
// space is the scanner rendering/verifying the message — not the human
// opening it. This is what makes "opened = human opened, clicked = human
// clicked" hold even when the tenant's ATP/SafeLinks still scans a sender.
//
// Built-ins are Microsoft's long-stable Azure/MS supernets — the egress of
// M365 ATP, SafeLinks and the image proxy (verified live: phantom
// opens/clicks arrived from 48.209.223.88 = 48.208.0.0/13, AS8075). Extend
// via PROPHISH_INFRA_CIDRS. Dropped hits are still written to tracking_hits
// (outcome 'ignored', reason 'infra') so every decision stays auditable.
var infraCIDRs = mustCIDRs(
	"13.64.0.0/11", "13.96.0.0/13", "13.104.0.0/14",
	"40.64.0.0/10", "48.208.0.0/13", "65.52.0.0/14",
	"72.144.0.0/14", "104.40.0.0/13", "157.54.0.0/15",
	"157.56.0.0/14", "168.61.0.0/16", "168.62.0.0/15", "191.232.0.0/13",
	"199.60.28.0/22", "207.46.0.0/16",
)

func mustCIDRs(ss ...string) []*net.IPNet {
	out := make([]*net.IPNet, 0, len(ss))
	for _, s := range ss {
		_, n, err := net.ParseCIDR(s)
		if err != nil {
			panic("publicsrv: bad built-in infra CIDR " + s + ": " + err.Error())
		}
		out = append(out, n)
	}
	return out
}

// infraMu guards dynamicInfraCIDRs, the live ASN-fetched list (see
// SetDynamicInfraCIDRs). The static built-ins never change, so they are read
// lock-free; only the dynamic slice needs synchronisation.
var (
	infraMu           sync.RWMutex
	dynamicInfraCIDRs []*net.IPNet
)

// SetDynamicInfraCIDRs REPLACES the ASN-fetched infrastructure list (the
// built-in static ranges stay). Called by the periodic refresher in
// cmd/prophish with the prefixes currently announced by the configured
// scanner ASNs (default AS8075 Microsoft): cloud ASNs add and move ranges
// constantly, so any fixed list goes stale within weeks.
func SetDynamicInfraCIDRs(nets []*net.IPNet) {
	infraMu.Lock()
	dynamicInfraCIDRs = nets
	infraMu.Unlock()
}

// AddInfraCIDRs appends operator-configured infrastructure ranges
// (PROPHISH_INFRA_CIDRS) to the built-in list.
func AddInfraCIDRs(nets []*net.IPNet) {
	infraCIDRs = append(infraCIDRs, nets...)
}

// isInfraIP reports whether the client IP belongs to known scanner/proxy
// infrastructure: the static built-ins (plus env extras) and the live
// ASN-fetched dynamic list.
func isInfraIP(ipStr string) bool {
	ip := net.ParseIP(strings.TrimSpace(ipStr))
	if ip == nil {
		return false
	}
	for _, n := range infraCIDRs {
		if n.Contains(ip) {
			return true
		}
	}
	infraMu.RLock()
	defer infraMu.RUnlock()
	for _, n := range dynamicInfraCIDRs {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// trustedHitDrop reports why a trusted-mode click/submit hit is NOT the
// human: it comes from known scanner infrastructure, or its User-Agent is a
// bot/scanner signature (an empty UA is treated as automated too — real
// browsers always send one on clicks).
func trustedHitDrop(ua, ip string) string {
	if isInfraIP(ip) {
		return "infra"
	}
	if isScannerUA(ua) {
		return "ua"
	}
	return ""
}

// scannerRaceWait is how long countInteraction waits and re-reads the canary
// probe before counting a hit. Scanners fetch the canary and the real link
// CONCURRENTLY (observed gap between the two hits: 3ms — both requests are in
// flight at once), so a hit's initial row read predates the probe's commit and
// the naive probe check misses it. Re-reading after a couple of seconds lets
// the probe land before the decision; a human never notices a 2s delay.
const scannerRaceWait = 2 * time.Second

// recentlyProbed reports whether a canary probe fired within `window` before
// now — i.e. this hit is part of the scanner's paired burst, not the recipient.
func recentlyProbed(probe *time.Time, window time.Duration) bool {
	return probe != nil && time.Since(*probe) < window
}

// scannerUASignatures are case-insensitive substrings found in the User-Agent
// of automated clients: security gateways, link/image proxies, link-preview
// bots, and generic HTTP libraries. Deliberately excludes real mail-client and
// webmail-proxy agents (e.g. Gmail's image proxy fetches when a human actually
// opens the mail), so genuine opens are not dropped.
var scannerUASignatures = []string{
	"bot", "crawl", "spider", "slurp", "preview", "scan", "fetch",
	"proofpoint", "barracuda", "mimecast", "forcepoint", "fireeye",
	"trendmicro", "symantec", "mcafee", "sophos", "cisco", "ironport",
	"safelinks", "urldefense", "microsoftpreview", "bingpreview",
	"curl", "wget", "python-requests", "go-http-client", "java/",
	"libwww", "okhttp", "httpclient", "headlesschrome", "phantomjs",
	"facebookexternalhit", "whatsapp", "telegrambot", "slackbot",
	"discordbot", "skypeuripreview", "linkedinbot", "googlebot",
}

// isScannerUA reports whether a User-Agent looks like an automated client. An
// empty User-Agent is also treated as automated: normal browsers always send one.
func isScannerUA(ua string) bool {
	ua = strings.ToLower(strings.TrimSpace(ua))
	if ua == "" {
		return true
	}
	for _, sig := range scannerUASignatures {
		if strings.Contains(ua, sig) {
			return true
		}
	}
	return false
}

// preWindowDecision classifies a click/submit hit by identity and send-time,
// BEFORE the canary pair window is consulted. grace is the route's post-send
// window: a hit inside it is a delivery-time prefetch (no human can receive
// and act that fast on THIS route). Returns the machine-readable ignore reason
// ("" when the hit survives this layer): a scanner User-Agent signature, or
// arrival inside the grace window. In trusted delivery mode nothing is
// discarded at this layer — the first clicker is the user by definition.
func preWindowDecision(trusted bool, grace time.Duration, ua string, sentAt *time.Time) string {
	if trusted {
		return ""
	}
	if isScannerUA(ua) {
		return "ua"
	}
	if sentAt != nil && time.Since(*sentAt) < grace {
		return "grace_window"
	}
	return ""
}

// isAutomated reports whether a tracking hit should be ignored as automated,
// either because the User-Agent is a known scanner/bot or because it arrived
// inside the route's post-send grace window.
func isAutomated(trusted bool, grace time.Duration, ua string, sentAt *time.Time) bool {
	return preWindowDecision(trusted, grace, ua, sentAt) != ""
}

// openIgnoreReason attributes an image fetch (open). Unlike clicks, opens are
// deliberately NOT User-Agent filtered: mail image proxies (Gmail, Yahoo,
// Apple, Microsoft) legitimately send sparse or even EMPTY User-Agents when a
// HUMAN downloads images, so UA filtering drops genuine opens. Automation is
// proven by timing instead: inside the route's grace window, or inside the
// scanner's paired burst (pair) after a canary probe. Everything else —
// including a proxy fetch minutes after delivery — is a real open. In trusted
// delivery mode every fetch counts: the first opener is the user. Returns the
// ignore reason ("" when the fetch counts as a real open).
func openIgnoreReason(trusted bool, grace, pair time.Duration, sentAt, probe *time.Time) string {
	if trusted {
		return ""
	}
	if sentAt != nil && time.Since(*sentAt) < grace {
		return "grace_window"
	}
	if recentlyProbed(probe, pair) {
		return "pair_window"
	}
	return ""
}

// isOpenAutomated reports whether an image fetch is treated as automated.
func isOpenAutomated(trusted bool, grace, pair time.Duration, sentAt, probe *time.Time) bool {
	return openIgnoreReason(trusted, grace, pair, sentAt, probe) != ""
}

func (s *Server) handleOpen(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	hitAt := time.Now()
	// TrackedByToken (not ResultByToken): the latter scanned landing_page_id
	// into a plain string and failed on NULL for public campaigns, silently
	// killing ALL open tracking for them.
	if t, err := s.store.TrackedByToken(r.Context(), token); err == nil {
		rid := t.ResultID
		outcome, reason := store.HitCounted, "counted"
		ignored := ""
		if t.TrustedDelivery {
			// Human-only: the instant "open" fired at delivery by the M365
			// image proxy / ATP render comes from Microsoft datacenter space.
			// A real user's image download comes from their own network, so
			// the IP is the reliable discriminator (UA is deliberately NOT
			// used here — desktop clients send sparse UAs on image fetches).
			if isInfraIP(s.clientIP(r)) {
				ignored = "infra"
			}
		} else {
			win := windowsFor(t.DeliveryRoute)
			ignored = openIgnoreReason(false, win.Grace, win.Pair, t.SentAt, t.ScannerProbeAt)
		}
		if ignored != "" {
			outcome, reason = store.HitIgnored, ignored
		} else if changed, _ := s.store.AdvanceStatus(r.Context(), rid, "opened"); !changed {
			outcome, reason = store.HitDuplicate, "duplicate"
		} else {
			// Record the event only on the first transition into this stage, so a
			// target (or an attacker who knows the token) cannot flood the events table.
			_ = s.store.RecordEvent(r.Context(), t.CampaignID, &rid, "email_opened", s.clientIP(r), r.UserAgent())
		}
		s.recordHit(r, t, "open", hitAt, outcome, reason)
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	_, _ = w.Write(bannerPNG)
}

func (s *Server) handleClick(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	t, err := s.store.TrackedByToken(r.Context(), token)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	s.countInteraction(r, t, "click", time.Now(), "clicked", "link_clicked", "suspected_click")
	s.serveTracked(w, r, t, token)
}

// recordHit writes the raw-request row for a tracking hit. A failure is logged,
// never blocks the response: forensic logging must not become an outage path.
// hitAt is the request's arrival time, captured before any classification wait,
// so the row reflects when the request actually landed.
func (s *Server) recordHit(r *http.Request, t *store.TrackedFull, route string, hitAt time.Time, outcome, reason string) {
	if err := s.store.RecordHit(r.Context(), t.ResultID, t.CampaignID, route, s.clientIP(r), r.UserAgent(), outcome, reason, hitAt); err != nil {
		log.Printf("publicsrv: record %s hit (result %s): %v", route, t.ResultID, err)
	}
}

// countInteraction advances a result's status for a tracked hit (click or
// submit), attributing it to the recipient only when it does NOT look
// automated. Every request — counted, suppressed or ignored — lands in
// tracking_hits with its outcome and machine-readable reason.
//
// Trusted delivery mode: "the first clicker is the user", with one exception
// — hits from known scanner infrastructure (cloud datacenter IPs, bot/empty
// User-Agents) are still dropped, so trusted campaigns report human events
// only. No windows, no canary, no race wait.
//
// Untrusted mode: the scanner filter applies — UA signatures, the route's
// grace window, and the canary pair window (with the race wait).
func (s *Server) countInteraction(r *http.Request, t *store.TrackedFull, route string, hitAt time.Time, status, eventType, suspectedType string) {
	rid := t.ResultID
	if t.TrustedDelivery {
		if why := trustedHitDrop(r.UserAgent(), s.clientIP(r)); why != "" {
			s.recordHit(r, t, route, hitAt, store.HitIgnored, why)
			return
		}
		s.countHumanHit(r, t, route, hitAt, status, eventType)
		return
	}
	win := windowsFor(t.DeliveryRoute)
	if why := preWindowDecision(false, win.Grace, r.UserAgent(), t.SentAt); why != "" {
		s.recordHit(r, t, route, hitAt, store.HitIgnored, why)
		return
	}
	// Race-safe probe check: the canary fetch that marks this crawl may be in
	// flight right now (scanners open all links concurrently), so re-read the
	// probe after a short wait before deciding the hit is human.
	probe := t.ScannerProbeAt
	if !recentlyProbed(probe, win.Pair) {
		time.Sleep(scannerRaceWait)
		probe, _ = s.store.ProbeTime(r.Context(), rid)
	}
	if recentlyProbed(probe, win.Pair) {
		s.recordHit(r, t, route, hitAt, store.HitSuspected, "pair_window")
		s.recordSuspected(r, t.CampaignID, rid, suspectedType, win.Pair)
		return
	}
	s.countHumanHit(r, t, route, hitAt, status, eventType)
}

// countHumanHit records a hit as a genuine user interaction: the result
// advances once (later hits of the same kind are duplicates) and the
// transition event is logged with the hit's fingerprint.
func (s *Server) countHumanHit(r *http.Request, t *store.TrackedFull, route string, hitAt time.Time, status, eventType string) {
	rid := t.ResultID
	outcome, reason := store.HitCounted, "counted"
	if changed, _ := s.store.AdvanceStatus(r.Context(), rid, status); !changed {
		outcome, reason = store.HitDuplicate, "duplicate"
	} else {
		_ = s.store.RecordEvent(r.Context(), t.CampaignID, &rid, eventType, s.clientIP(r), r.UserAgent())
	}
	s.recordHit(r, t, route, hitAt, outcome, reason)
}

// recordSuspected logs a human-looking hit that the canary correlation
// attributed to a scanner crawl. Deduplicated inside the cluster window so a
// scanner replaying the link cannot flood the events table.
func (s *Server) recordSuspected(r *http.Request, campaignID, rid, suspectedType string, window time.Duration) {
	if recent, _ := s.store.HasRecentEvent(r.Context(), rid, suspectedType, window); !recent {
		_ = s.store.RecordEvent(r.Context(), campaignID, &rid, suspectedType, s.clientIP(r), r.UserAgent())
	}
}

// handleCanary serves the hidden canary link the engine injects into every HTML
// email (a display:none anchor no human sees). Mail-security scanners fetch
// every link in a message, so a hit here marks the recipient as scanned. The
// response mirrors a real click — same redirect or landing page — so the probe
// is indistinguishable from genuine traffic and cannot be filtered out by the
// scanner itself. The hit is never counted as a click.
func (s *Server) handleCanary(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	hitAt := time.Now()
	t, err := s.store.TrackedByToken(r.Context(), token)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	s.recordHit(r, t, "canary", hitAt, store.HitProbe, "canary")
	_ = s.store.MarkScannerProbe(r.Context(), t.ResultID)
	win := windowsFor(t.DeliveryRoute)
	// Log every probe as an event. scanner_probe_at keeps only the LATEST probe;
	// the event log preserves each hit with its UA/IP, which is what lets us
	// measure the interval between the canary fetch and the scanner's real-link
	// fetch — the number the cluster window must cover. Not deduplicated on
	// purpose: repeat probes are exactly the data we want to see.
	rid := t.ResultID
	_ = s.store.RecordEvent(r.Context(), t.CampaignID, &rid, "scanner_probe", s.clientIP(r), r.UserAgent())
	// Retrospective retraction, two layers:
	//
	//  1. PRE-DELIVERY (unbounded lookback to the FIRST probe): the first canary
	//     probe fires when the gateway scans the message — at or before delivery
	//     to the inbox. A human cannot click a link in a message they have not
	//     received, so ANY counted link/submit event stamped BEFORE the first
	//     probe is the scanner's own fetch (a slow burst that raced ahead of its
	//     canary), not a person. Retract it no matter how far back it is.
	//  2. RECENT BURST (pair window before THIS probe): a slow sequential crawl
	//     of a later rescan may have had its real-link hit counted moments before
	//     this probe landed. Pull that back too.
	//
	// Hits between the first probe and a later rescan are NOT touched: by then
	// the human HAS the message, so those clicks can be genuine.
	if first, err := s.store.FirstProbeTime(r.Context(), rid); err == nil {
		if first == nil {
			first = t.ScannerProbeAt // probes predate probe-event logging (0011)
		}
		if first != nil {
			if hits, herr := s.store.CountedHitsAround(r.Context(), rid, *first, win.Pair); herr == nil && len(hits) > 0 {
				openRetracted := false
				for _, h := range hits {
					if h.Type == "email_opened" {
						openRetracted = true
						continue // no suspected event for opens; the probe event tells the story
					}
					suspected := "suspected_click"
					if h.Type == "data_submitted" {
						suspected = "suspected_submit"
					}
					s.recordSuspected(r, t.CampaignID, rid, suspected, win.Pair)
				}
				// Demote: back to 'opened' only if a genuine open survives the
				// retraction; if the open itself is retracted, back to 'sent'.
				base := "sent"
				if !openRetracted {
					if opened, _ := s.store.HasRecentEvent(r.Context(), rid, "email_opened", 8760*time.Hour); opened {
						base = "opened"
					}
				}
				_ = s.store.DemoteStatus(r.Context(), rid, base)
			}
		}
	}
	s.serveTracked(w, r, t, token)
}

// serveTracked renders the post-click experience for both real clicks and
// canary probes: public campaigns redirect to the configured real site, local
// campaigns render the landing page. Identical output for both keeps the
// canary indistinguishable.
func (s *Server) serveTracked(w http.ResponseWriter, r *http.Request, t *store.TrackedFull, token string) {
	// Public campaign: the click is recorded (by the caller), so there is nothing
	// hosted to show — send the target straight on to the configured real site.
	// No landing page, no form, nothing to submit. A 302 keeps the method GET and
	// our no-referrer policy (set by publicHeaders) prevents the tracking token
	// leaking to the destination in a Referer header. If the destination is
	// somehow empty (should be impossible — enforced at create time and by a DB
	// CHECK), fall back to the neutral awareness notice rather than a broken 302.
	if t.CampaignKind == "public" {
		if dest := strings.TrimSpace(t.CampaignRedirectURL); dest != "" {
			http.Redirect(w, r, dest, http.StatusFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(awarenessPage))
		return
	}

	// Submit URL: prefer the campaign's OWN public base (the HTTPS tunnel URL for
	// a gateway campaign, the host URL otherwise) so the form posts back to the
	// SAME origin the target loaded the page from. Otherwise a page served over
	// the HTTPS tunnel would post to the http:// host base, which the browser
	// blocks as an insecure/mixed-content submission ("not secure" interstitial).
	// Fall back to the global base, then to a same-origin relative path.
	base := s.baseURL
	if camp, err := s.store.GetCampaign(r.Context(), t.CampaignID); err == nil && camp.PublicBaseURL != "" {
		base = strings.TrimRight(camp.PublicBaseURL, "/")
	}
	submitURL := "/s/" + token
	if base != "" {
		submitURL = fmt.Sprintf("%s/s/%s", base, token)
	}

	page := renderLanding(t.PageHTML, landingData{
		FirstName: t.FirstName, LastName: t.LastName, Position: t.Position,
		SubmitURL: submitURL,
	})
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(page))
}

func (s *Server) handleSubmit(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	t, err := s.store.TrackedByToken(r.Context(), token)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	// IMPORTANT: we intentionally do NOT read or store r.PostForm values.
	// Only the fact of submission is recorded, for awareness reporting.
	s.countInteraction(r, t, "submit", time.Now(), "submitted", "data_submitted", "suspected_submit")

	redirect := t.RedirectURL
	if redirect == "" {
		// Default: show an inline awareness notice instead of redirecting anywhere.
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(awarenessPage))
		return
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

type landingData struct {
	FirstName string
	LastName  string
	Position  string
	SubmitURL string
}

func renderLanding(tmpl string, d landingData) string {
	t, err := template.New("lp").Option("missingkey=zero").Parse(tmpl)
	if err != nil {
		log.Printf("landing template parse error: %v", err)
		return "<!doctype html><title>Page</title>"
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, d); err != nil {
		log.Printf("landing template exec error: %v", err)
		return "<!doctype html><title>Page</title>"
	}
	return buf.String()
}

// peerIP returns the direct TCP peer address (never client-controlled).
func peerIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// clientIP returns the real client IP. X-Forwarded-For is honored ONLY when the
// direct peer is a configured trusted proxy; otherwise the spoofable header is
// ignored and the peer address is used, so event evidence cannot be forged.
func (s *Server) clientIP(r *http.Request) string {
	peer := peerIP(r)
	if len(s.trusted) == 0 || !s.ipTrusted(peer) {
		return peer
	}
	xff := r.Header.Get("X-Forwarded-For")
	if xff == "" {
		return peer
	}
	parts := strings.Split(xff, ",")
	// Walk right-to-left, returning the first hop that is NOT itself a trusted proxy.
	for i := len(parts) - 1; i >= 0; i-- {
		ip := strings.TrimSpace(parts[i])
		if ip != "" && !s.ipTrusted(ip) {
			return ip
		}
	}
	return peer
}

func (s *Server) ipTrusted(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, n := range s.trusted {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

const awarenessPage = `<!doctype html><html lang="az"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1"><title>Təhlükəsizlik xəbərdarlığı</title>
<style>body{font-family:system-ui,sans-serif;background:#0f1420;color:#e7ecf3;display:flex;min-height:100vh;align-items:center;justify-content:center;margin:0}
.box{max-width:520px;padding:32px;background:#1a2132;border:1px solid #2a3350;border-radius:14px;text-align:center}
h1{color:#ffcf5c}</style></head><body><div class="box">
<h1>⚠️ Bu bir phishing simulyasiyası idi</h1>
<p>Siz məlumatlandırma təlimi çərçivəsində simulyasiya edilmiş bir e-poçtla qarşılaşdınız.
Daxil etdiyiniz məlumatlar <strong>saxlanılmadı</strong>. Real hücumda bu, hesabınızın ələ keçirilməsinə səbəb ola bilərdi.</p>
<p>Şübhəli linklərə klikləməzdən əvvəl göndəricini və ünvanı yoxlayın.</p>
</div></body></html>`
