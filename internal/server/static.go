package server

import (
	"embed"
	"net/http"
	"strings"
)

//go:embed static/app.js
var staticAppJS []byte

//go:embed static/editor.js
var staticEditorJS []byte

// Self-hosted UI fonts (Hanken Grotesk + IBM Plex Mono). Served same-origin so
// the strict CSP allows them and no admin/target data ever leaks to a font CDN.
//
//go:embed static/fonts/*.woff2
var fontFS embed.FS

// handleStaticJS serves the admin's progressive-enhancement script from the same
// origin, so it is allowed by the CSP (script-src falls back to default-src 'self';
// inline scripts are blocked). It is an unauthenticated GET — the file is not secret.
func (s *Server) handleStaticJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	_, _ = w.Write(staticAppJS)
}

// handleStaticEditorJS serves the same-origin WYSIWYG editor script.
func (s *Server) handleStaticEditorJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	_, _ = w.Write(staticEditorJS)
}

// handleStaticFont serves a self-hosted woff2 by name (no path traversal).
func (s *Server) handleStaticFont(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/static/fonts/")
	if name == "" || strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		http.NotFound(w, r)
		return
	}
	b, err := fontFS.ReadFile("static/fonts/" + name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "font/woff2")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	_, _ = w.Write(b)
}
