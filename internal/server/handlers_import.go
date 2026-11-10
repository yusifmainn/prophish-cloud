package server

import (
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"secdrill/internal/importer"
)

// handlePageImport clones a live URL into a new landing page.
func (s *Server) handlePageImport(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	target := strings.TrimSpace(r.FormValue("url"))
	if name == "" || target == "" {
		http.Redirect(w, r, "/pages?import=bad", http.StatusSeeOther)
		return
	}
	htmlBody, err := importer.CloneSite(target)
	if err != nil {
		http.Redirect(w, r, "/pages?import=err", http.StatusSeeOther)
		return
	}
	pid, err := s.store.CreateLandingPage(r.Context(), nil, name, htmlBody, strings.TrimSpace(r.FormValue("redirect_url")))
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.audit(r, "page.import", "landing_page", pid, name)
	http.Redirect(w, r, "/pages?import=ok", http.StatusSeeOther)
}

// handleTemplateImport parses a raw email into a new template.
func (s *Server) handleTemplateImport(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	raw := r.FormValue("raw")
	if name == "" || strings.TrimSpace(raw) == "" {
		http.Redirect(w, r, "/templates?import=bad", http.StatusSeeOther)
		return
	}
	em, err := importer.ParseEmail(raw)
	if err != nil {
		http.Redirect(w, r, "/templates?import=err", http.StatusSeeOther)
		return
	}
	subject := em.Subject
	if subject == "" {
		subject = name
	}
	html := em.HTML
	if html == "" {
		html = "<pre>" + em.Text + "</pre>"
	}
	tid, err := s.store.CreateTemplate(r.Context(), nil, name, subject, html, em.Text, nil)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.audit(r, "template.import", "template", tid, name)
	http.Redirect(w, r, "/templates?import=ok", http.StatusSeeOther)
}

// handleTemplateDetail shows a template and its attachments.
func (s *Server) handleTemplateDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tmpl, err := s.store.GetTemplate(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	s.render(w, r, http.StatusOK, "template_detail.html", map[string]any{"T": tmpl})
}

// handleAttachmentUpload stores an uploaded file as a template attachment.
func (s *Server) handleAttachmentUpload(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	r.Body = http.MaxBytesReader(w, r.Body, 20<<20)
	if err := r.ParseMultipartForm(16 << 20); err != nil {
		http.Redirect(w, r, "/templates/"+id, http.StatusSeeOther)
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()
	file, hdr, err := r.FormFile("file")
	if err != nil {
		http.Redirect(w, r, "/templates/"+id, http.StatusSeeOther)
		return
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, 16<<20))
	if err != nil {
		http.Error(w, "read error", http.StatusInternalServerError)
		return
	}
	ct := hdr.Header.Get("Content-Type")
	if ct == "" {
		ct = "application/octet-stream"
	}
	_ = s.store.AddAttachment(r.Context(), id, hdr.Filename, ct, content)
	http.Redirect(w, r, "/templates/"+id, http.StatusSeeOther)
}

func (s *Server) handleAttachmentDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	aid := chi.URLParam(r, "aid")
	_ = s.store.DeleteAttachment(r.Context(), aid)
	http.Redirect(w, r, "/templates/"+id, http.StatusSeeOther)
}
