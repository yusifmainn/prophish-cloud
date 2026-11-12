package server

import "net/http"

// handleDocumentation renders the in-app user guide (documentation.html): an
// illustrated, step-by-step walkthrough of every screen, from first-run setup
// through campaign results. It is a static page — no per-request data.
func (s *Server) handleDocumentation(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, http.StatusOK, "documentation.html", map[string]any{})
}
