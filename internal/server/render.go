package server

import (
	"bytes"
	"embed"
	"html/template"
	"net/http"
	"strings"
	"time"

	"prophish/internal/models"
)

//go:embed templates/*.html
var templateFS embed.FS

// templateFuncs are helpers available inside every template.
var templateFuncs = template.FuncMap{
	// lt ("local time") formats a timestamp in the process time zone
	// (PROPHISH_TIMEZONE, set on time.Local at startup). Timestamps come back
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
	// i18n translation helpers. These are placeholders registered at parse time
	// so templates that reference {{t}}/{{tf}}/{{curlang}}/{{eventlabel}}/…
	// parse; render() overrides each per request with a language-bound
	// implementation (see langTemplateFuncs). The defaults resolve in the
	// default language so the base template remains renderable on its own.
	"t":           func(key string) template.HTML { return template.HTML(tr(defaultLang, key)) },
	"tf":          func(key string, args ...any) template.HTML { return template.HTML(trf(defaultLang, key, args...)) },
	"curlang":     func() string { return defaultLang },
	"langs":       func() []string { return supportedLangs },
	"langName":    func(code string) string { return langNames[code] },
	"langFullName": func(code string) string { return langNamesFull[code] },
	// eventlabel localizes a tracking-event type for the timeline (language-aware
	// at render time; raw code fallback for unknown/future types).
	"eventlabel": func(code string) string { return label(defaultLang, "event", code) },
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
	// actionlabel localizes an audit-log action code for the audit screen
	// (language-aware at render time; raw code fallback for unknown/future codes).
	"actionlabel": func(code string) string { return label(defaultLang, "action", code) },
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
	lang := langFromRequest(r)
	if m, ok := data.(map[string]any); ok {
		if _, exists := m["Toast"]; !exists {
			if f := s.popFlash(r); f != nil {
				m["Toast"] = f
			}
		}
		// Localize handler-set messages, whose source text is Azerbaijani, into
		// the request language: the one-off flash toast and any inline form error.
		if f, ok := m["Toast"].(*FlashMessage); ok && f != nil {
			f.Msg = trFlash(lang, f.Msg)
		}
		if e, ok := m["Error"].(string); ok {
			m["Error"] = trFlash(lang, e)
		}
	}
	// Bind the request's UI language into the template helpers. Cloning the
	// parsed set per request and layering language-bound funcs over it lets every
	// template (and scalar-data sub-templates like {{template "statuspill"
	// .Status}}) call {{t "key"}} without threading the language through any
	// handler's data.
	tmpl, err := s.templates.Clone()
	if err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	tmpl = tmpl.Funcs(langTemplateFuncs(lang))

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = buf.WriteTo(w)
}
