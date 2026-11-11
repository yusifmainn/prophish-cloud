package server

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/go-chi/chi/v5"

	"prophish/internal/library"
)

// libKitVM is the per-request, language-resolved view of a catalog kit that
// library.html ranges over. It is a package-level type (not a closure-local
// one) so the template test can construct it directly.
type libKitVM struct {
	ID       string
	Brand    string
	Category string
	Title    string
	Summary  string
	HasEmail bool
}

// handleLibraryList renders the gallery of built-in simulation kits.
func (s *Server) handleLibraryList(w http.ResponseWriter, r *http.Request) {
	lang := langFromRequest(r)
	var kits []libKitVM
	for _, k := range library.Catalog() {
		kits = append(kits, libKitVM{
			ID:       k.ID,
			Brand:    k.Brand,
			Category: k.Category,
			Title:    library.Pick(k.Title, lang),
			Summary:  library.Pick(k.Summary, lang),
			HasEmail: k.HasEmail(),
		})
	}
	s.render(w, r, http.StatusOK, "library.html", map[string]any{"Kits": kits})
}

// writeInlinePreview renders a stored template body against sample data for the
// admin preview iframe/new-tab. It mirrors handlePagePreview's hardening: a
// permissive CSP so brand styles/images load but inline scripts stay blocked,
// and SAMEORIGIN so the gallery can embed it. The body is developer-authored
// catalog content, never target input.
func writeInlinePreview(w http.ResponseWriter, body string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src * data:; style-src * 'unsafe-inline'; font-src * data:; script-src 'none'")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	t, err := template.New("pv").Option("missingkey=zero").Parse(body)
	if err == nil {
		var buf bytes.Buffer
		if err = t.Execute(&buf, data); err == nil {
			_, _ = buf.WriteTo(w)
			return
		}
	}
	_, _ = w.Write([]byte("<!doctype html><meta charset=utf-8><body style=\"font-family:sans-serif;padding:20px;color:#b00\">Preview error: " +
		template.HTMLEscapeString(err.Error()) + "</body>"))
}

// handleLibraryPagePreview renders a kit's landing page with sample target data.
func (s *Server) handleLibraryPagePreview(w http.ResponseWriter, r *http.Request) {
	k, ok := library.Get(chi.URLParam(r, "id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeInlinePreview(w, k.LandingHTML(), previewTarget{
		FirstName: "Aysel", LastName: "Məmmədova", Position: "Mühasib", SubmitURL: "#",
	})
}

// handleLibraryEmailPreview renders a kit's email lure with sample recipient data.
func (s *Server) handleLibraryEmailPreview(w http.ResponseWriter, r *http.Request) {
	k, ok := library.Get(chi.URLParam(r, "id"))
	if !ok || !k.HasEmail() {
		http.NotFound(w, r)
		return
	}
	writeInlinePreview(w, k.EmailHTML(), emailPreviewData{
		FirstName: "Aysel", LastName: "Məmmədova",
		Email: "aysel.mammadova@numune.az", Position: "Mühasib", URL: "#",
	})
}

// handleLibraryUse instantiates a kit: it copies the landing page (and matching
// email template, if any) into ordinary, fully editable records so the operator
// can tweak them before a campaign. Created content is global (client_id NULL),
// exactly like the manual "New page"/"New template" forms.
func (s *Server) handleLibraryUse(w http.ResponseWriter, r *http.Request) {
	k, ok := library.Get(chi.URLParam(r, "id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	lang := langFromRequest(r)
	name := library.Pick(k.Title, lang)

	html := k.LandingHTML()
	if err := validLandingHTML(html); err != nil {
		s.flashErr(r, s.tf(r, "flash.page_save_tmpl_err", err.Error()))
		http.Redirect(w, r, "/library", http.StatusSeeOther)
		return
	}
	pid, err := s.store.CreateLandingPage(r.Context(), nil, name, html, k.RedirectURL)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.audit(r, "library.use", "landing_page", pid, k.Brand)

	if k.HasEmail() {
		tid, terr := s.store.CreateTemplate(r.Context(), nil, name,
			library.Pick(k.EmailSubject, lang), k.EmailHTML(), "", nil)
		if terr == nil {
			s.audit(r, "library.use", "template", tid, k.Brand)
			s.flashOK(r, s.tf(r, "lib.flash_used_both", name))
		} else {
			s.flashOK(r, s.tf(r, "lib.flash_used_page", name))
		}
	} else {
		s.flashOK(r, s.tf(r, "lib.flash_used_page", name))
	}
	http.Redirect(w, r, "/pages/"+pid, http.StatusSeeOther)
}
