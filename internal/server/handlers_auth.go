package server

import (
	"errors"
	"net/http"
	"strings"

	"prophish/internal/auth"
	"prophish/internal/store"
)

func (s *Server) handleSetupForm(w http.ResponseWriter, r *http.Request) {
	n, err := s.store.CountUsers(r.Context())
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if n > 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	s.render(w, r, http.StatusOK, "setup.html", map[string]any{})
}

func (s *Server) handleSetupSubmit(w http.ResponseWriter, r *http.Request) {
	n, err := s.store.CountUsers(r.Context())
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if n > 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")
	if email == "" {
		s.render(w, r, http.StatusBadRequest, "setup.html", map[string]any{"Error": s.t(r, "err.email_required")})
		return
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		s.render(w, r, http.StatusBadRequest, "setup.html", map[string]any{"Error": err.Error()})
		return
	}
	uid, err := s.store.CreateUser(r.Context(), email, hash, "admin")
	if err != nil {
		s.render(w, r, http.StatusInternalServerError, "setup.html", map[string]any{"Error": s.t(r, "err.create_user")})
		return
	}
	_ = s.sessions.RenewToken(r.Context())
	s.sessions.Put(r.Context(), sessionUserKey, uid)
	_ = s.store.RecordAudit(r.Context(), uid, email, "user.create", "user", uid, "first admin (setup)", auditIP(r))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleLoginForm(w http.ResponseWriter, r *http.Request) {
	n, err := s.store.CountUsers(r.Context())
	if err == nil && n == 0 {
		http.Redirect(w, r, "/setup", http.StatusSeeOther)
		return
	}
	s.render(w, r, http.StatusOK, "login.html", map[string]any{})
}

func (s *Server) handleLoginSubmit(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")

	// Two throttle keys: (email, source IP) locks quickly; (account) tolerates
	// more but bounds distributed guessing across many IPs. loginKey derives the
	// IP from the real transport peer (the admin router does not trust XFF).
	key := loginKey(r, email)
	acctKey := "acct:" + email
	if loginLockRemaining(key, loginMaxFails) > 0 || loginLockRemaining(acctKey, loginAccountMaxFails) > 0 {
		s.render(w, r, http.StatusTooManyRequests, "login.html", map[string]any{
			"Error": s.t(r, "err.too_many_attempts")})
		return
	}

	u, hash, err := s.store.GetUserByEmail(r.Context(), email)
	// Always run a bcrypt comparison so response time does not reveal whether the
	// account exists or is active (constant one-bcrypt path on every branch).
	if errors.Is(err, store.ErrNotFound) || u == nil {
		_ = auth.CheckPassword("$2a$10$invalidinvalidinvalidinvalidinvalidinvalidinvalidinv", password)
		loginFail(key)
		loginFail(acctKey)
		_ = s.store.RecordAudit(r.Context(), "", email, "auth.login_failed", "user", "", "no such account", auditIP(r))
		s.render(w, r, http.StatusUnauthorized, "login.html", map[string]any{"Error": s.t(r, "err.invalid_credentials")})
		return
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	ok := auth.CheckPassword(hash, password)
	if !ok || !u.IsActive {
		loginFail(key)
		loginFail(acctKey)
		reason := "wrong password"
		if !u.IsActive {
			reason = "account disabled"
		}
		_ = s.store.RecordAudit(r.Context(), "", email, "auth.login_failed", "user", u.ID, reason, auditIP(r))
		s.render(w, r, http.StatusUnauthorized, "login.html", map[string]any{"Error": s.t(r, "err.invalid_credentials")})
		return
	}

	loginReset(key)
	loginReset(acctKey)
	_ = s.sessions.RenewToken(r.Context())
	s.sessions.Put(r.Context(), sessionUserKey, u.ID)
	_ = s.store.TouchLogin(r.Context(), u.ID)
	_ = s.store.RecordAudit(r.Context(), u.ID, u.Email, "auth.login", "user", u.ID, "", auditIP(r))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.audit(r, "auth.logout", "", "", "")
	_ = s.sessions.Destroy(r.Context())
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
