package server

import "net/http"

// FlashMessage is a one-off banner shown on the next rendered page. It is set
// before a redirect (the POST→redirect→GET pattern) and cleared when read, so a
// success/error message survives exactly one navigation.
type FlashMessage struct {
	Kind string // "ok" | "err" | "info"
	Msg  string
}

// flash stores a one-off message to display after the next redirect.
func (s *Server) flash(r *http.Request, kind, msg string) {
	s.sessions.Put(r.Context(), "flash_kind", kind)
	s.sessions.Put(r.Context(), "flash_msg", msg)
}

// flashOK / flashErr are convenience wrappers for the two common kinds.
func (s *Server) flashOK(r *http.Request, msg string)  { s.flash(r, "ok", msg) }
func (s *Server) flashErr(r *http.Request, msg string) { s.flash(r, "err", msg) }

// popFlash reads and clears the pending flash, returning nil when none is set.
// Called by render() so the message is injected into template data automatically.
func (s *Server) popFlash(r *http.Request) *FlashMessage {
	msg := s.sessions.PopString(r.Context(), "flash_msg")
	if msg == "" {
		return nil
	}
	kind := s.sessions.PopString(r.Context(), "flash_kind")
	if kind == "" {
		kind = "info"
	}
	return &FlashMessage{Kind: kind, Msg: msg}
}
