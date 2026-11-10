package server

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"prophish/internal/auth"
	"prophish/internal/store"
)

func (s *Server) handleIMAPForm(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.store.GetIMAPConfig(r.Context())
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	reported, _ := s.store.ListReportedEmails(r.Context())
	s.render(w, r, http.StatusOK, "imap.html", map[string]any{
		"Cfg":      cfg,
		"Reported": reported,
	})
}

func (s *Server) handleIMAPSave(w http.ResponseWriter, r *http.Request) {
	host := strings.TrimSpace(r.FormValue("host"))
	user := strings.TrimSpace(r.FormValue("username"))
	folder := strings.TrimSpace(r.FormValue("folder"))
	if folder == "" {
		folder = "INBOX"
	}
	port, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("port")))
	if port == 0 {
		port = 993
	}
	poll, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("poll_seconds")))
	if poll == 0 {
		poll = 60
	}
	useTLS := r.FormValue("use_tls") == "on"
	enabled := r.FormValue("enabled") == "on"

	var enc []byte
	if pw := r.FormValue("password"); pw != "" {
		e, err := auth.Encrypt(s.cfg.EncryptionKey, []byte(pw))
		if err != nil {
			http.Error(w, "encrypt error", http.StatusInternalServerError)
			return
		}
		enc = e
	}

	if err := s.store.UpsertIMAPConfig(r.Context(), host, port, user, enc, useTLS, folder, enabled, poll); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	state := "disabled"
	if enabled {
		state = "enabled"
	}
	s.audit(r, "imap.config_save", "imap", "", host+" ("+state+")")
	s.flashOK(r, "IMAP parametrləri yadda saxlanıldı.")
	http.Redirect(w, r, "/settings/imap", http.StatusSeeOther)
}
