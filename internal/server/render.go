package server

import (
	"bytes"
	"embed"
	"html/template"
	"net/http"
	"strings"
	"time"

	"secdrill/internal/models"
)

//go:embed templates/*.html
var templateFS embed.FS

// templateFuncs are helpers available inside every template.
var templateFuncs = template.FuncMap{
	// lt ("local time") formats a timestamp in the process time zone
	// (SECDRILL_TIMEZONE, set on time.Local at startup). Timestamps come back
	// from Postgres in UTC, so this converts them for display, e.g.
	// {{lt "2006-01-02 15:04:05" .CreatedAt}}.
	"lt": func(layout string, t time.Time) string {
		return t.In(time.Local).Format(layout)
	},
	// auditclass maps an audit action to a status-pill colour class.
	"auditclass": func(a string) string {
		switch {
		case strings.HasPrefix(a, "auth.login_failed"):
			return "crit"
		case strings.Contains(a, "delete"), strings.Contains(a, "cancel"):
			return "warn"
		case a == "campaign.launch", a == "auth.login":
			return "ok"
		default:
			return "neutral"
		}
	},
	// eventlabel localizes a tracking-event type to Azerbaijani for the timeline.
	"eventlabel": func(t string) string {
		switch t {
		case "email_sent":
			return "Email göndərildi"
		case "email_opened":
			return "Email açıldı"
		case "link_clicked":
			return "Link kliklədi"
		case "data_submitted":
			return "Məlumat göndərdi"
		case "email_reported":
			return "Phishing kimi bildirdi"
		case "training_completed":
			return "Təlim tamamladı"
		case "error":
			return "Xəta (göndərilmədi)"
		default:
			return t
		}
	},
	// eventclass maps a tracking-event type to a status-pill colour class.
	"eventclass": func(t string) string {
		switch t {
		case "data_submitted", "error":
			return "crit"
		case "link_clicked":
			return "warn"
		case "email_opened":
			return "info"
		case "email_reported":
			return "ok"
		default:
			return "neutral"
		}
	},
	// actionlabel localizes an audit-log action code to Azerbaijani for the audit
	// screen. Unknown or future codes fall through to the raw code so nothing breaks.
	"actionlabel": func(a string) string {
		switch a {
		case "auth.login":
			return "Daxil oldu"
		case "auth.login_failed":
			return "Uğursuz giriş"
		case "auth.logout":
			return "Çıxış etdi"
		case "user.create":
			return "Operator yaradıldı"
		case "user.delete":
			return "Operator silindi"
		case "user.password_change":
			return "Parol dəyişdirildi"
		case "user.apikey_create":
			return "API açarı yaradıldı"
		case "user.apikey_revoke":
			return "API açarı ləğv edildi"
		case "imap.config_save":
			return "IMAP parametrləri saxlanıldı"
		case "client.create":
			return "Müştəri yaradıldı"
		case "client.update":
			return "Müştəri yeniləndi"
		case "client.delete":
			return "Müştəri silindi"
		case "authorization.create":
			return "İcazə əlavə edildi"
		case "authorization.update":
			return "İcazə yeniləndi"
		case "authorization.delete":
			return "İcazə silindi"
		case "group.create":
			return "Qrup yaradıldı"
		case "group.update":
			return "Qrup yeniləndi"
		case "group.delete":
			return "Qrup silindi"
		case "target.create":
			return "Hədəf əlavə edildi"
		case "target.update":
			return "Hədəf yeniləndi"
		case "target.delete":
			return "Hədəf silindi"
		case "target.import":
			return "Hədəflər import edildi"
		case "template.create":
			return "Şablon yaradıldı"
		case "template.update":
			return "Şablon yeniləndi"
		case "template.delete":
			return "Şablon silindi"
		case "template.import":
			return "Şablon import edildi"
		case "profile.create":
			return "SMTP profili yaradıldı"
		case "profile.update":
			return "SMTP profili yeniləndi"
		case "profile.delete":
			return "SMTP profili silindi"
		case "profile.test":
			return "Test emaili göndərildi"
		case "page.create":
			return "Səhifə yaradıldı"
		case "page.update":
			return "Səhifə yeniləndi"
		case "page.delete":
			return "Səhifə silindi"
		case "page.import":
			return "Səhifə import edildi"
		case "campaign.create":
			return "Kampaniya yaradıldı"
		case "campaign.update":
			return "Kampaniya yeniləndi"
		case "campaign.rename":
			return "Kampaniya adı dəyişdirildi"
		case "campaign.launch":
			return "Kampaniya işə salındı"
		case "campaign.schedule":
			return "Kampaniya planlaşdırıldı"
		case "campaign.cancel":
			return "Kampaniya ləğv edildi"
		case "campaign.archive":
			return "Kampaniya arxivləndi"
		case "campaign.delete":
			return "Kampaniya silindi"
		case "campaign.resend":
			return "Yenidən göndərildi"
		case "campaign.report_pdf":
			return "PDF hesabat yükləndi"
		default:
			return a
		}
	},
	// authstate classifies an authorization window against the current time for
	// the status pill on the client page: active / future / expired. valid_to is
	// stored at end-of-day, so a same-day window reads as active.
	"authstate": func(from, to time.Time) string {
		now := time.Now()
		switch {
		case now.Before(from):
			return "future"
		case now.After(to):
			return "expired"
		default:
			return "active"
		}
	},
	// pct exposes the shared rounding helper to templates: {{pct .Clicked .Total}}.
	"pct": models.Pct,
}

func parseTemplates() (*template.Template, error) {
	return template.New("").Funcs(templateFuncs).ParseFS(templateFS, "templates/*.html")
}

// render executes a named template into a buffer first, so a template error
// never results in a half-written response. It also injects any pending one-off
// flash message (set via s.flash before a redirect) into map-based template data
// under the key "Toast", so every page shows success/error banners with no extra
// per-handler wiring.
func (s *Server) render(w http.ResponseWriter, r *http.Request, status int, name string, data any) {
	if m, ok := data.(map[string]any); ok {
		if _, exists := m["Toast"]; !exists {
			if f := s.popFlash(r); f != nil {
				m["Toast"] = f
			}
		}
	}
	var buf bytes.Buffer
	if err := s.templates.ExecuteTemplate(&buf, name, data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = buf.WriteTo(w)
}
