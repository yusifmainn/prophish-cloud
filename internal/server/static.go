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

//go:embed static/auth.js
var staticAuthJS []byte

// Login hero artwork (the company brand panel) — the exact supplied asset, one
// per theme. Served same-origin so the CSP (img-src 'self') allows it.
//
//go:embed static/hero-dark.webp
var heroDark []byte

//go:embed static/hero-light.webp
var heroLight []byte

// Brand logo marks (square) + browser icons, served same-origin (CSP img-src
// 'self'). The two logo marks are theme-specific; the favicons are theme-neutral.
//
//go:embed static/logo-mark.svg
var logoLightSVG []byte

//go:embed static/logo-mark-dark-bg.svg
var logoDarkSVG []byte

//go:embed static/favicon.svg
var faviconSVG []byte

//go:embed static/favicon.ico
var faviconICO []byte

//go:embed static/apple-touch-icon.png
var appleTouchPNG []byte

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
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(staticAppJS)
}

// handleStaticEditorJS serves the same-origin WYSIWYG editor script.
func (s *Server) handleStaticEditorJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(staticEditorJS)
}

// handleStaticAuthJS serves the same-origin auth-page script (login/setup theme
// toggle + password reveal). Unauthenticated GET; the file is not secret.
func (s *Server) handleStaticAuthJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(staticAuthJS)
}

// handleStaticHero serves the login hero image for the requested theme.
func (s *Server) handleStaticHero(w http.ResponseWriter, r *http.Request) {
	var b []byte
	switch strings.TrimPrefix(r.URL.Path, "/static/hero/") {
	case "dark.webp":
		b = heroDark
	case "light.webp":
		b = heroLight
	default:
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/webp")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	_, _ = w.Write(b)
}

// handleStaticLogo serves a theme-specific brand logo mark.
func (s *Server) handleStaticLogo(w http.ResponseWriter, r *http.Request) {
	var b []byte
	switch strings.TrimPrefix(r.URL.Path, "/static/") {
	case "logo-mark.svg":
		b = logoLightSVG
	case "logo-mark-dark-bg.svg":
		b = logoDarkSVG
	default:
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=604800")
	_, _ = w.Write(b)
}

// handleBrandIcon serves the browser tab / home-screen icons from the site root.
func (s *Server) handleBrandIcon(w http.ResponseWriter, r *http.Request) {
	var b []byte
	var ct string
	switch r.URL.Path {
	case "/favicon.ico":
		b, ct = faviconICO, "image/x-icon"
	case "/favicon.svg":
		b, ct = faviconSVG, "image/svg+xml"
	case "/apple-touch-icon.png":
		b, ct = appleTouchPNG, "image/png"
	default:
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "public, max-age=604800")
	_, _ = w.Write(b)
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
