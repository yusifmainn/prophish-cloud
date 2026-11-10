package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"secdrill/internal/auth"
)

// ---- Email templates ----

func (s *Server) handleTemplatesList(w http.ResponseWriter, r *http.Request) {
	tmpls, err := s.store.ListTemplates(r.Context())
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.render(w, r, http.StatusOK, "templates.html", map[string]any{
		"Templates": tmpls, "Import": r.URL.Query().Get("import"),
		"Err": r.URL.Query().Get("err")})
}

// handleTemplateUpdate edits a template's content in place.
func (s *Server) handleTemplateUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	name := strings.TrimSpace(r.FormValue("name"))
	subject := strings.TrimSpace(r.FormValue("subject"))
	html := r.FormValue("html_body")
	text := r.FormValue("text_body")
	if name == "" || subject == "" || strings.TrimSpace(html) == "" {
		s.flashErr(r, "Ad, mövzu və HTML məzmun boş ola bilməz.")
		http.Redirect(w, r, "/templates/"+id, http.StatusSeeOther)
		return
	}
	headers := parseHeaderLines(r.FormValue("headers"))
	if err := s.store.UpdateTemplate(r.Context(), id, name, subject, html, text, headers); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.audit(r, "template.update", "template", id, name)
	s.flashOK(r, "Şablon yeniləndi.")
	http.Redirect(w, r, "/templates/"+id, http.StatusSeeOther)
}

// handleTemplateDelete removes a template unless a campaign still uses it.
func (s *Server) handleTemplateDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	n, err := s.store.CountCampaignsUsingTemplate(r.Context(), id)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if n > 0 {
		s.flashErr(r, "Şablon silinmədi — onu bir və ya bir neçə kampaniya istifadə edir.")
		http.Redirect(w, r, "/templates", http.StatusSeeOther)
		return
	}
	if err := s.store.DeleteTemplate(r.Context(), id); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.audit(r, "template.delete", "template", id, "")
	s.flashOK(r, "Şablon silindi.")
	http.Redirect(w, r, "/templates", http.StatusSeeOther)
}

func (s *Server) handleTemplateCreate(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	subject := strings.TrimSpace(r.FormValue("subject"))
	html := r.FormValue("html_body")
	text := r.FormValue("text_body")
	if name == "" || subject == "" || strings.TrimSpace(html) == "" {
		s.flashErr(r, "Ad, mövzu və HTML məzmun boş ola bilməz.")
		http.Redirect(w, r, "/templates", http.StatusSeeOther)
		return
	}
	headers := parseHeaderLines(r.FormValue("headers"))
	tid, err := s.store.CreateTemplate(r.Context(), nil, name, subject, html, text, headers)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.audit(r, "template.create", "template", tid, name)
	s.flashOK(r, "Şablon yaradıldı.")
	http.Redirect(w, r, "/templates", http.StatusSeeOther)
}

// ---- Landing pages ----

func (s *Server) handlePagesList(w http.ResponseWriter, r *http.Request) {
	pages, err := s.store.ListLandingPages(r.Context())
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.render(w, r, http.StatusOK, "pages.html", map[string]any{
		"Pages": pages, "Import": r.URL.Query().Get("import")})
}

func (s *Server) handlePageCreate(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	html := r.FormValue("html_body")
	redirect := strings.TrimSpace(r.FormValue("redirect_url"))
	if name == "" || strings.TrimSpace(html) == "" {
		s.flashErr(r, "Ad və HTML məzmun boş ola bilməz.")
		http.Redirect(w, r, "/pages", http.StatusSeeOther)
		return
	}
	if err := validLandingHTML(html); err != nil {
		s.flashErr(r, "Səhifə saxlanmadı — şablon xətası (ehtimal ki, mətndə artıq {{ }} var): "+err.Error())
		http.Redirect(w, r, "/pages", http.StatusSeeOther)
		return
	}
	pid, err := s.store.CreateLandingPage(r.Context(), nil, name, html, redirect)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.audit(r, "page.create", "landing_page", pid, name)
	s.flashOK(r, "Açılış səhifəsi yaradıldı.")
	http.Redirect(w, r, "/pages", http.StatusSeeOther)
}

// ---- Sending profiles ----

func (s *Server) handleProfilesList(w http.ResponseWriter, r *http.Request) {
	profiles, err := s.store.ListSendingProfiles(r.Context())
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.render(w, r, http.StatusOK, "profiles.html", map[string]any{
		"Profiles": profiles,
		"Test":     r.URL.Query().Get("test"),
		// Flash carries the specific reason a test send failed (set by
		// handleProfileTest), so the operator sees the real cause.
		"Flash": s.sessions.PopString(r.Context(), "flash_profile")})
}

func (s *Server) handleProfileCreate(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	host := strings.TrimSpace(r.FormValue("smtp_host"))
	from := strings.TrimSpace(r.FormValue("from_address"))
	user := strings.TrimSpace(r.FormValue("smtp_username"))
	password := r.FormValue("smtp_password")
	if name == "" || host == "" || from == "" {
		s.flashErr(r, "Ad, SMTP host və göndərən ünvan boş ola bilməz.")
		http.Redirect(w, r, "/profiles", http.StatusSeeOther)
		return
	}
	port, ok := parsePort(r.FormValue("smtp_port"))
	if !ok {
		s.flashErr(r, "Port 1–65535 aralığında rəqəm olmalıdır.")
		http.Redirect(w, r, "/profiles", http.StatusSeeOther)
		return
	}

	var enc []byte
	if password != "" {
		var err error
		enc, err = auth.Encrypt(s.cfg.EncryptionKey, []byte(password))
		if err != nil {
			http.Error(w, "encrypt error", http.StatusInternalServerError)
			return
		}
	}
	headers := parseHeaderLines(r.FormValue("headers"))
	pid, err := s.store.CreateSendingProfile(r.Context(), name, host, port, user, from, enc, headers)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.audit(r, "profile.create", "profile", pid, name+" ("+host+")")
	s.flashOK(r, "SMTP profili yaradıldı.")
	http.Redirect(w, r, "/profiles", http.StatusSeeOther)
}

// handleProfileUpdate edits a sending profile. The password field left blank
// keeps the stored one; a "clear_password" checkbox removes it; otherwise the
// typed value replaces it (re-encrypted).
func (s *Server) handleProfileUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	name := strings.TrimSpace(r.FormValue("name"))
	host := strings.TrimSpace(r.FormValue("smtp_host"))
	from := strings.TrimSpace(r.FormValue("from_address"))
	user := strings.TrimSpace(r.FormValue("smtp_username"))
	password := r.FormValue("smtp_password")
	clearPw := r.FormValue("clear_password") != ""
	if name == "" || host == "" || from == "" {
		s.flashErr(r, "Ad, SMTP host və göndərən ünvan boş ola bilməz.")
		http.Redirect(w, r, "/profiles", http.StatusSeeOther)
		return
	}
	port, ok := parsePort(r.FormValue("smtp_port"))
	if !ok {
		s.flashErr(r, "Port 1–65535 aralığında rəqəm olmalıdır.")
		http.Redirect(w, r, "/profiles", http.StatusSeeOther)
		return
	}

	var enc []byte
	replace := false
	switch {
	// A typed password always wins — even if the "clear" box is also ticked — so
	// a replacement entered while swapping credentials is never silently dropped.
	case password != "":
		e, err := auth.Encrypt(s.cfg.EncryptionKey, []byte(password))
		if err != nil {
			http.Error(w, "encrypt error", http.StatusInternalServerError)
			return
		}
		enc, replace = e, true
	case clearPw:
		replace = true // enc stays nil → stored password is cleared
	}
	headers := parseHeaderLines(r.FormValue("headers"))
	if err := s.store.UpdateSendingProfile(r.Context(), id, name, host, port, user, from, enc, replace, headers); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.audit(r, "profile.update", "profile", id, name+" ("+host+")")
	s.flashOK(r, "SMTP profili yeniləndi.")
	http.Redirect(w, r, "/profiles", http.StatusSeeOther)
}

// handleProfileDelete removes a sending profile unless a campaign still uses it.
func (s *Server) handleProfileDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	n, err := s.store.CountCampaignsUsingProfile(r.Context(), id)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if n > 0 {
		http.Redirect(w, r, "/profiles?test=inuse", http.StatusSeeOther)
		return
	}
	if err := s.store.DeleteSendingProfile(r.Context(), id); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.audit(r, "profile.delete", "profile", id, "")
	s.flashOK(r, "SMTP profili silindi.")
	http.Redirect(w, r, "/profiles", http.StatusSeeOther)
}

// parsePort parses an SMTP port. Empty → the 587 default; a non-numeric or
// out-of-range value is rejected (ok=false) instead of being silently coerced
// to 587 (which would then fail opaquely at send time).
func parsePort(raw string) (int, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 587, true
	}
	p, err := strconv.Atoi(raw)
	if err != nil || p < 1 || p > 65535 {
		return 0, false
	}
	return p, true
}
