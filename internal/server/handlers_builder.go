package server

import (
	"bytes"
	"html/template"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// builderCSP relaxes img-src (external image-block URLs in the live canvas) and
// allows data: fonts (the embedded Roboto @font-face) for the builder responses
// only. The global admin CSP (secureHeaders) stays unchanged everywhere else.
const builderCSP = "default-src 'self'; style-src 'self' 'unsafe-inline'; img-src * data:; font-src 'self' data:; media-src 'self' data:"

// builderData assembles the template data for the builder page. JSStrings hands
// the client-side builder script its translated strings (builder.js.* keys) as
// JSON in a hidden element, since a static JS file cannot call {{t}} itself.
func (s *Server) builderData(r *http.Request, edit bool, page any) map[string]any {
	d := map[string]any{
		"Edit":      edit,
		"JSStrings": messagesForPrefix(langFromRequest(r), "builder.js."),
	}
	if page != nil {
		d["P"] = page
	}
	return d
}

func (s *Server) handlePageBuilderNew(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Security-Policy", builderCSP)
	s.render(w, r, http.StatusOK, "page_builder.html", s.builderData(r, false, nil))
}

func (s *Server) handlePageBuilderEdit(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	page, err := s.store.GetLandingPage(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Security-Policy", builderCSP)
	s.render(w, r, http.StatusOK, "page_builder.html", s.builderData(r, true, page))
}

// handlePageBuilderPreview renders the builder's CURRENT (unsaved) canvas in a
// new tab. This replaces the old blob: preview — Chromium propagates the
// opener's CSP into blob: documents, which killed the router/button JS. A
// server-rendered response carries its own bounded CSP instead. Same trust
// model as handlePagePreview: operator-authored content, authenticated, form
// is still template-validated by parse+execute before serving.
func (s *Server) handlePageBuilderPreview(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 8<<20) // embedded media make pages multi-MB
	htmlBody := r.FormValue("html_body")
	if strings.TrimSpace(htmlBody) == "" {
		http.Error(w, "empty html_body", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src * data:; style-src * 'unsafe-inline'; font-src * data:; media-src * data:; script-src 'unsafe-inline'")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")

	sample := previewTarget{FirstName: "Aysel", LastName: "Məmmədova", Position: "Mühasib", SubmitURL: "#"}
	t, err := template.New("lp").Option("missingkey=zero").Parse(htmlBody)
	if err == nil {
		var buf bytes.Buffer
		if err = t.Execute(&buf, sample); err == nil {
			_, _ = buf.WriteTo(w)
			return
		}
	}
	_, _ = w.Write([]byte("<!doctype html><meta charset=utf-8><body style=\"font-family:sans-serif;padding:20px;color:#b00\">Şablon render xətası: " +
		template.HTMLEscapeString(err.Error()) + "</body>"))
}
