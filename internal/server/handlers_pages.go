package server

import (
	"bytes"
	"html/template"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// previewTarget is the sample data a landing page is rendered against in the
// admin preview — mirrors the fields the public server injects at click time.
type previewTarget struct {
	FirstName string
	LastName  string
	Position  string
	SubmitURL string
}

// validLandingHTML reports whether stored landing markup parses AND executes as a
// Go template. Cloned/pasted pages sometimes contain stray "{{ }}", which would
// make the public server render a blank page to live targets; we reject on save
// instead, with the exact error.
func validLandingHTML(htmlBody string) error {
	t, err := template.New("lp").Option("missingkey=zero").Parse(htmlBody)
	if err != nil {
		return err
	}
	return t.Execute(io.Discard, previewTarget{})
}

func (s *Server) handlePageDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	page, err := s.store.GetLandingPage(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	s.render(w, r, http.StatusOK, "page_detail.html", map[string]any{"P": page})
}

func (s *Server) handlePageUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	name := strings.TrimSpace(r.FormValue("name"))
	htmlBody := r.FormValue("html_body")
	if name == "" || strings.TrimSpace(htmlBody) == "" {
		s.flashErr(r, "Ad və HTML məzmun boş ola bilməz.")
		http.Redirect(w, r, "/pages/"+id, http.StatusSeeOther)
		return
	}
	if err := validLandingHTML(htmlBody); err != nil {
		s.flashErr(r, "HTML saxlanmadı — şablon xətası (ehtimal ki, mətndə artıq {{ }} var): "+err.Error())
		http.Redirect(w, r, "/pages/"+id, http.StatusSeeOther)
		return
	}
	if err := s.store.UpdateLandingPage(r.Context(), id, name, htmlBody, strings.TrimSpace(r.FormValue("redirect_url"))); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.audit(r, "page.update", "landing_page", id, name)
	s.flashOK(r, "Landing səhifəsi yeniləndi.")
	http.Redirect(w, r, "/pages/"+id, http.StatusSeeOther)
}

func (s *Server) handlePageDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	n, err := s.store.CountCampaignsUsingLandingPage(r.Context(), id)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if n > 0 {
		s.flashErr(r, "Səhifə silinmədi — onu bir və ya bir neçə kampaniya istifadə edir.")
		http.Redirect(w, r, "/pages", http.StatusSeeOther)
		return
	}
	if err := s.store.DeleteLandingPage(r.Context(), id); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.audit(r, "page.delete", "landing_page", id, "")
	s.flashOK(r, "Landing səhifəsi silindi.")
	http.Redirect(w, r, "/pages", http.StatusSeeOther)
}

// handlePagePreview renders the stored landing HTML with sample target data so the
// operator can see it before a live campaign. It overrides the admin's strict
// headers for THIS response only: a permissive CSP (so cloned external images and
// styles load, scripts still blocked) and X-Frame-Options SAMEORIGIN (so the
// page_detail preview iframe can embed it).
func (s *Server) handlePagePreview(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	page, err := s.store.GetLandingPage(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Pages authored in the visual builder can carry an inline <script> for
	// operator-written button actions; the builder marker gates a relaxed
	// script-src for those pages only. Cloned/pasted pages (which may contain
	// third-party markup) keep script-src 'none'.
	csp := "default-src 'self'; img-src * data:; style-src * 'unsafe-inline'; font-src * data:; media-src * data:; script-src 'none'"
	if strings.Contains(page.HTMLBody, "prophish-builder:") {
		csp = "default-src 'self'; img-src * data:; style-src * 'unsafe-inline'; font-src * data:; media-src * data:; script-src 'unsafe-inline'"
	}
	w.Header().Set("Content-Security-Policy", csp)
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")

	sample := previewTarget{FirstName: "Aysel", LastName: "Məmmədova", Position: "Mühasib", SubmitURL: "#"}
	t, err := template.New("lp").Option("missingkey=zero").Parse(page.HTMLBody)
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

// emailPreviewData is the sample recipient an email template is rendered against
// in the admin preview — mirrors the fields the engine injects at send time.
type emailPreviewData struct {
	FirstName     string
	LastName      string
	Email         string
	Position      string
	URL           string
	TrackingPixel template.HTML
}

// handleTemplatePreview renders the stored email HTML with sample recipient data
// so the operator sees the resolved lure (variables filled in). Same per-response
// permissive CSP + SAMEORIGIN as the landing preview; scripts stay blocked.
func (s *Server) handleTemplatePreview(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tmpl, err := s.store.GetTemplate(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src * data:; style-src * 'unsafe-inline'; font-src * data:; script-src 'none'")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")

	sample := emailPreviewData{
		FirstName: "Aysel", LastName: "Məmmədova",
		Email: "aysel.mammadova@numune.az", Position: "Mühasib", URL: "#",
	}
	t, err := template.New("tpl").Option("missingkey=zero").Parse(tmpl.HTMLBody)
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
