package publicsrv

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"strings"
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
	// grace is how long after SEND a pixel/link hit is treated as an automated
	// scanner prefetch rather than a human interaction (see scannerGraceWindowDefault).
	grace time.Duration
}

func NewServer(st *store.Store, baseURL string, trusted []*net.IPNet, grace time.Duration) *Server {
	if grace <= 0 {
		grace = scannerGraceWindowDefault
	}
	return &Server{store: st, baseURL: strings.TrimRight(baseURL, "/"), trusted: trusted, grace: grace}
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

// 1x1 transparent GIF, served for the open pixel.
var pixelGIF, _ = base64.StdEncoding.DecodeString("R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7")

// scannerGraceWindowDefault is how long after a message is sent we treat
// pixel/link hits as automated, unless overridden per deployment
// (PROPHISH_OPEN_SCANNER_GRACE). Mail-security scanners and link proxies fetch
// the content within seconds of the message passing through them — long before
// a human recipient could realistically open and act on it. A hit inside this
// window is almost certainly a scanner, so it does not count as an open/click.
//
// The window is measured from SEND (sent_at), not delivery — the server cannot
// observe delivery. Tenants with deferred inbound delivery (greylisting, Safe
// Attachments "detonation", large drip sends) may see a scanner's image
// prefetch arrive AFTER this window and be counted as an open; raise
// PROPHISH_OPEN_SCANNER_GRACE there, or disable open tracking for the campaign.
const scannerGraceWindowDefault = 30 * time.Second

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

// hasScannerSignature reports whether a NON-empty User-Agent contains a known
// automated-client signature. Unlike isScannerUA it does NOT treat an empty
// User-Agent as automated, because an empty UA means different things on the
// two paths: for a page navigation it is a bot (browsers always send a UA),
// but for an image fetch it is routine — Microsoft 365 / Outlook-on-the-web /
// "new Outlook" / Gmail proxy the remote image server-side and that fetch
// commonly carries no User-Agent.
func hasScannerSignature(ua string) bool {
	ua = strings.ToLower(strings.TrimSpace(ua))
	if ua == "" {
		return false
	}
	for _, sig := range scannerUASignatures {
		if strings.Contains(ua, sig) {
			return true
		}
	}
	return false
}

// isScannerUA reports whether a User-Agent looks like an automated client. An
// empty User-Agent is also treated as automated: normal browsers always send
// one. Used by the CLICK/landing path (handleClick), where a hit with no UA is
// a bot. The open-pixel path is deliberately more lenient — see openAutomatedReason.
func isScannerUA(ua string) bool {
	return strings.TrimSpace(ua) == "" || hasScannerSignature(ua)
}

// withinGrace reports whether a hit arrived within the grace window of send.
func withinGrace(sentAt *time.Time, grace time.Duration) bool {
	return sentAt != nil && time.Since(*sentAt) < grace
}

// isAutomated reports whether a CLICK/landing hit should be ignored as
// automated, either because the User-Agent is a known scanner/bot (empty UA
// included) or because it arrived within the grace window of the message being
// sent.
func isAutomated(ua string, sentAt *time.Time, grace time.Duration) bool {
	return isScannerUA(ua) || withinGrace(sentAt, grace)
}

// openAutomatedReason classifies an OPEN-PIXEL hit. It returns a short reason
// string when the hit should NOT count as a human open, or "" when it counts.
//
// Crucially, an EMPTY User-Agent is NOT disqualifying here. The previous
// behaviour treated empty-UA as an automatic scanner, which silently dropped
// EVERY open from Microsoft 365 / Outlook-on-the-web / "new Outlook" / Gmail —
// their image proxies fetch the pixel server-side with no User-Agent — so
// standalone opens never registered while clicks (a real browser, full UA) did.
//
// Instead we rely on two signals that still separate scanners from humans:
//   - a known vendor/bot UA signature  -> always automated ("scanner-ua")
//   - the grace window                 -> a hit moments after send is a
//     delivery-time scanner prefetch ("grace-window"); a hit after the window
//     is the provider's proxy fetching because the recipient opened the mail.
func openAutomatedReason(ua string, sentAt *time.Time, grace time.Duration) string {
	if hasScannerSignature(ua) {
		return "scanner-ua"
	}
	if withinGrace(sentAt, grace) {
		return "grace-window"
	}
	return ""
}

func (s *Server) handleOpen(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	// Always serve the pixel; recording is best-effort and must never block it.
	defer func() {
		w.Header().Set("Content-Type", "image/gif")
		w.Header().Set("Cache-Control", "no-store, max-age=0")
		_, _ = w.Write(pixelGIF)
	}()

	t, err := s.store.ResultByToken(r.Context(), token)
	if err != nil {
		// Unknown/expired token — log it (never the token itself) so stray or
		// forged pixel hits are visible, then just serve the blank pixel.
		log.Printf("open pixel: unknown token ip=%s ua=%q", s.clientIP(r), r.UserAgent())
		return
	}
	rid := t.ResultID
	ua := r.UserAgent()
	ip := s.clientIP(r)

	// Ignore automated fetches (mail-security scanners, link/image proxies) that
	// pre-fetch the pixel before — or instead of — the human recipient; they
	// would otherwise register a false "opened" the moment the message is scanned.
	// Every decision is logged (with the User-Agent and time-since-send, never
	// the token) so an operator can SEE whether a client's fetch arrives and why
	// it is or isn't counted — the only way to observe opens, since a later click
	// jumps straight to "clicked" and leaves no trace of whether an open happened.
	if reason := openAutomatedReason(ua, t.SentAt, s.grace); reason != "" {
		log.Printf("open pixel ignored (%s): result=%s ip=%s since_send=%s ua=%q", reason, rid, ip, sinceSend(t.SentAt), ua)
		return
	}
	// Record the event only on the FIRST transition into this stage, so a target
	// (or an attacker who knows the token) cannot flood the events table.
	changed, aerr := s.store.AdvanceStatus(r.Context(), rid, "opened")
	switch {
	case aerr != nil:
		log.Printf("open pixel error: result=%s ip=%s: %v", rid, ip, aerr)
	case changed:
		_ = s.store.RecordEvent(r.Context(), t.CampaignID, &rid, "email_opened", ip, ua)
		log.Printf("open pixel recorded: result=%s ip=%s since_send=%s ua=%q", rid, ip, sinceSend(t.SentAt), ua)
	default:
		// Already opened/clicked/submitted — a repeat fetch, not a new open.
		log.Printf("open pixel noop (status=%s): result=%s ip=%s ua=%q", t.Status, rid, ip, ua)
	}
}

// sinceSend renders how long ago a message was sent, for log lines (or "n/a"
// when the send time is unknown).
func sinceSend(sentAt *time.Time) string {
	if sentAt == nil {
		return "n/a"
	}
	return time.Since(*sentAt).Round(time.Second).String()
}

func (s *Server) handleClick(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	t, err := s.store.TrackedByToken(r.Context(), token)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	rid := t.ResultID
	ua := r.UserAgent()
	ip := s.clientIP(r)
	// Ignore automated link-following (mail-security scanners, SafeLinks-style
	// URL rewriters) that crawl every link before delivery. We still serve the
	// landing page/redirect below so a later genuine click by the recipient is
	// recorded normally — only the automated hit is skipped. Unlike the open
	// pixel, an empty User-Agent here is itself a bot signal (a real browser
	// navigation always sends one), so isAutomated still drops it.
	if isAutomated(ua, t.SentAt, s.grace) {
		log.Printf("click ignored (automated): result=%s ip=%s since_send=%s ua=%q", rid, ip, sinceSend(t.SentAt), ua)
	} else if changed, aerr := s.store.AdvanceStatus(r.Context(), rid, "clicked"); aerr != nil {
		log.Printf("click error: result=%s ip=%s: %v", rid, ip, aerr)
	} else if changed {
		_ = s.store.RecordEvent(r.Context(), t.CampaignID, &rid, "link_clicked", ip, ua)
		log.Printf("click recorded: result=%s ip=%s since_send=%s ua=%q", rid, ip, sinceSend(t.SentAt), ua)
	} else {
		log.Printf("click noop (status=%s): result=%s ip=%s ua=%q", t.Status, rid, ip, ua)
	}

	// Public campaign: the click is now recorded (above), so there is nothing
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
	rid := t.ResultID
	if changed, _ := s.store.AdvanceStatus(r.Context(), rid, "submitted"); changed {
		_ = s.store.RecordEvent(r.Context(), t.CampaignID, &rid, "data_submitted", s.clientIP(r), r.UserAgent())
	}

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
