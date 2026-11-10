package server

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"secdrill/internal/auth"
)

func (s *Server) handleUsersList(w http.ResponseWriter, r *http.Request) {
	me, _ := s.store.GetUser(r.Context(), s.currentUserID(r))
	users, err := s.store.ListUsers(r.Context())
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	isAdmin := me != nil && me.Role == "admin"
	hasKey, _ := s.store.HasAPIKey(r.Context(), s.currentUserID(r))
	// A freshly generated key is shown exactly once via a session flash.
	newKey := s.sessions.PopString(r.Context(), "flash_apikey")
	s.render(w, r, http.StatusOK, "users.html", map[string]any{
		"Users":   users,
		"IsAdmin": isAdmin,
		"Me":      me,
		"HasKey":  hasKey,
		"NewKey":  newKey,
	})
}

func (s *Server) handleUserCreate(w http.ResponseWriter, r *http.Request) {
	if !s.isAdmin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	role := r.FormValue("role")
	if role != "admin" {
		role = "operator"
	}
	pw := r.FormValue("password")
	if email == "" {
		s.flashErr(r, "Email boş ola bilməz.")
		http.Redirect(w, r, "/users", http.StatusSeeOther)
		return
	}
	if len(pw) < 12 {
		s.flashErr(r, "Parol ən azı 12 simvol olmalıdır.")
		http.Redirect(w, r, "/users", http.StatusSeeOther)
		return
	}
	hash, err := auth.HashPassword(pw)
	if err != nil {
		s.flashErr(r, "Operator yaradılmadı.")
		http.Redirect(w, r, "/users", http.StatusSeeOther)
		return
	}
	newID, err := s.store.CreateUser(r.Context(), email, hash, role)
	if err != nil {
		s.flashErr(r, "Bu email artıq mövcuddur.")
		http.Redirect(w, r, "/users", http.StatusSeeOther)
		return
	}
	s.audit(r, "user.create", "user", newID, email+" ("+role+")")
	s.flashOK(r, "Operator yaradıldı.")
	http.Redirect(w, r, "/users", http.StatusSeeOther)
}

func (s *Server) handleUserDelete(w http.ResponseWriter, r *http.Request) {
	if !s.isAdmin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	id := chi.URLParam(r, "id")
	if id == s.currentUserID(r) {
		s.flashErr(r, "Öz hesabınızı silə bilməzsiniz.")
		http.Redirect(w, r, "/users", http.StatusSeeOther)
		return
	}
	// Capture the target's email before deletion so the audit entry is meaningful.
	target, _ := s.store.GetUser(r.Context(), id)
	_ = s.store.DeleteUser(r.Context(), id)
	targetEmail := id
	if target != nil {
		targetEmail = target.Email
	}
	s.audit(r, "user.delete", "user", id, targetEmail)
	s.flashOK(r, "Operator silindi.")
	http.Redirect(w, r, "/users", http.StatusSeeOther)
}

func (s *Server) handleAccountPassword(w http.ResponseWriter, r *http.Request) {
	uid := s.currentUserID(r)
	// Require the current password before changing it.
	curHash, err := s.store.GetPasswordHash(r.Context(), uid)
	if err != nil || !auth.CheckPassword(curHash, r.FormValue("current_password")) {
		s.flashErr(r, "Cari parol yanlışdır.")
		http.Redirect(w, r, "/users", http.StatusSeeOther)
		return
	}
	pw := r.FormValue("password")
	if len(pw) < 12 {
		s.flashErr(r, "Parol ən azı 12 simvol olmalıdır.")
		http.Redirect(w, r, "/users", http.StatusSeeOther)
		return
	}
	hash, err := auth.HashPassword(pw)
	if err != nil {
		s.flashErr(r, "Parol dəyişdirilmədi.")
		http.Redirect(w, r, "/users", http.StatusSeeOther)
		return
	}
	if err := s.store.SetPassword(r.Context(), uid, hash); err != nil {
		s.flashErr(r, "Parol dəyişdirilmədi.")
		http.Redirect(w, r, "/users", http.StatusSeeOther)
		return
	}
	// Rotate the session token so a previously-captured session id is invalidated.
	_ = s.sessions.RenewToken(r.Context())
	s.audit(r, "user.password_change", "user", uid, "own password")
	s.flashOK(r, "Parol dəyişdirildi.")
	http.Redirect(w, r, "/users", http.StatusSeeOther)
}

func (s *Server) handleAccountAPIKey(w http.ResponseWriter, r *http.Request) {
	uid := s.currentUserID(r)
	if r.FormValue("action") == "revoke" {
		_ = s.store.SetAPIKey(r.Context(), uid, "")
		s.audit(r, "user.apikey_revoke", "user", uid, "own API key")
	} else {
		key := auth.NewToken()
		if err := s.store.SetAPIKey(r.Context(), uid, key); err == nil {
			// Show the plaintext once; only its hash is stored.
			s.sessions.Put(r.Context(), "flash_apikey", key)
			s.audit(r, "user.apikey_create", "user", uid, "own API key")
		}
	}
	http.Redirect(w, r, "/users", http.StatusSeeOther)
}

func (s *Server) isAdmin(r *http.Request) bool {
	me, _ := s.store.GetUser(r.Context(), s.currentUserID(r))
	return me != nil && me.Role == "admin"
}
