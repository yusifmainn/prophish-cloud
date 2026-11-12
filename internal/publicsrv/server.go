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

func (s *Server) handleOpen(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if t, err := s.store.ResultByToken(r.Context(), token); err == nil {
		rid := t.ResultID
		// Record the event only on the first transition into this stage, so a
		// target (or an attacker who knows the token) cannot flood the events table.
		if changed, _ := s.store.AdvanceStatus(r.Context(), rid, "opened"); changed {
			_ = s.store.RecordEvent(r.Context(), t.CampaignID, &rid, "email_opened", s.clientIP(r), r.UserAgent())
		}
	}
	w.Header().Set("Content-Type", "image/gif")
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	_, _ = w.Write(pixelGIF)
}

func (s *Server) handleClick(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	t, err := s.store.TrackedByToken(r.Context(), token)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	rid := t.ResultID
	if changed, _ := s.store.AdvanceStatus(r.Context(), rid, "clicked"); changed {
		_ = s.store.RecordEvent(r.Context(), t.CampaignID, &rid, "link_clicked", s.clientIP(r), r.UserAgent())
	}

	// Submit URL: the configured public base when set; otherwise same-origin
	// relative — correct however the target reached us (host IP or tunnel).
	submitURL := "/s/" + token
	if s.baseURL != "" {
		submitURL = fmt.Sprintf("%s/s/%s", s.baseURL, token)
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
