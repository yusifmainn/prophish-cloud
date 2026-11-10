package server

import (
	"net/http"
	"net/url"
)

// secureHeaders applies conservative security headers to every response.
func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		// same-origin (not no-referrer): same-origin requests still send a Referer,
		// which the CSRF origin check uses, while cross-origin navigations leak none.
		h.Set("Referrer-Policy", "same-origin")
		h.Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:")
		h.Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

const sessionUserKey = "user_id"

// requireAuth blocks unauthenticated access AND revalidates the operator against
// the database on every request, so a deactivated or deleted account loses access
// immediately rather than keeping a live session.
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid := s.sessions.GetString(r.Context(), sessionUserKey)
		if uid == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		u, err := s.store.GetUser(r.Context(), uid)
		if err != nil || u == nil || !u.IsActive {
			_ = s.sessions.Destroy(r.Context())
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requireAdmin restricts a route to operators with the admin role. It must be
// used inside a group that already applies requireAuth.
func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, err := s.store.GetUser(r.Context(), s.currentUserID(r))
		if err != nil || u == nil || u.Role != "admin" {
			http.Error(w, "Bu əməliyyat yalnız admin üçündür.", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// checkOrigin is a CSRF defense-in-depth layer: for state-changing requests it
// requires the Origin/Referer host to match the request host. Combined with the
// SameSite=Lax session cookie this blocks cross-site form submissions.
func (s *Server) checkOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			if !originMatches(r) {
				http.Error(w, "CSRF: origin uyğunsuzluğu.", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func originMatches(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = r.Header.Get("Referer")
	}
	if origin == "" {
		// Neither header present. A real cross-site CSRF request made by a
		// victim's browser ALWAYS carries an Origin (cross-origin POST) or a
		// Referer, so a request with neither is not a browser CSRF vector (it is
		// a non-browser client). Allow it; authenticated actions are still also
		// protected by the SameSite=Lax session cookie.
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return u.Host == r.Host
}

// currentUserID returns the logged-in operator id, or "".
func (s *Server) currentUserID(r *http.Request) string {
	return s.sessions.GetString(r.Context(), sessionUserKey)
}
