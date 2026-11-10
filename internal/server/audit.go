package server

import (
	"net"
	"net/http"
)

// auditIP returns the true client IP for an audit record. The admin router does
// NOT trust X-Forwarded-For (so the login lockout cannot be spoofed), so the
// real transport peer in RemoteAddr is authoritative here too.
func auditIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// audit records a security-relevant action by the currently-authenticated
// operator. It is best-effort: a logging failure never blocks the action, but
// the action itself is what the audit trail is for, so callers invoke this on
// the success path of each sensitive operation.
func (s *Server) audit(r *http.Request, action, targetType, targetID, detail string) {
	uid := s.currentUserID(r)
	email := ""
	if uid != "" {
		if u, err := s.store.GetUser(r.Context(), uid); err == nil && u != nil {
			email = u.Email
		}
	}
	_ = s.store.RecordAudit(r.Context(), uid, email, action, targetType, targetID, detail, auditIP(r))
}

// handleAuditList renders the audit trail (admin-only).
func (s *Server) handleAuditList(w http.ResponseWriter, r *http.Request) {
	rows, err := s.store.ListAudit(r.Context(), 300)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.render(w, r, http.StatusOK, "audit.html", map[string]any{"Rows": rows})
}
