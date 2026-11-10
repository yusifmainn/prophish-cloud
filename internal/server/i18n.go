package server

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
)

// UI languages. Azerbaijani is the source/default language; English and Russian
// are full translations. To add a language: add its code here, add it to
// supportedLangs + langNames, and add a column to each entry in messages (any
// missing string falls back to Azerbaijani).
const (
	langAZ = "az"
	langEN = "en"
	langRU = "ru"
)

// defaultLang is used when no valid language cookie is present. Keeping this at
// Azerbaijani preserves the tool's original behaviour for anyone who never
// touches the switcher.
const defaultLang = langAZ

// langCookie stores the chosen UI language. It mirrors the session cookie's
// flags (see cmd/prophish/main.go) so it behaves consistently behind TLS.
const langCookie = "prophish_lang"

// supportedLangs is the switcher order (left → right).
var supportedLangs = []string{langAZ, langEN, langRU}

// langNames are the short code labels shown on the switcher buttons.
var langNames = map[string]string{langAZ: "AZ", langEN: "EN", langRU: "RU"}

// langNamesFull are the endonyms (each language's own name), used by the
// full-width language picker on the login screen. They are language-independent.
var langNamesFull = map[string]string{langAZ: "Azərbaycanca", langEN: "English", langRU: "Русский"}

func isSupportedLang(code string) bool {
	switch code {
	case langAZ, langEN, langRU:
		return true
	}
	return false
}

// langFromRequest resolves the active UI language: the cookie value if valid,
// otherwise the default. It is dependency-free so it works identically on the
// authenticated panel and the unauthenticated login/setup screens.
func langFromRequest(r *http.Request) string {
	if c, err := r.Cookie(langCookie); err == nil && isSupportedLang(c.Value) {
		return c.Value
	}
	return defaultLang
}

// tr returns the message for key in lang, falling back to Azerbaijani and then
// to the raw key, so a missing translation degrades visibly but never panics.
func tr(lang, key string) string {
	if m, ok := messages[key]; ok {
		if s, ok := m[lang]; ok && s != "" {
			return s
		}
		if s, ok := m[defaultLang]; ok && s != "" {
			return s
		}
	}
	return key
}

// trf is tr + fmt.Sprintf, for messages that interpolate values whose position
// differs between languages (e.g. "%d total campaigns" vs "всего %d кампаний").
func trf(lang, key string, args ...any) string {
	return fmt.Sprintf(tr(lang, key), args...)
}

// label looks up an enum-style code (event/action type) under a namespace and
// returns the raw code when there is no translation, so unknown or future codes
// never render as an empty cell.
func label(lang, namespace, code string) string {
	if m, ok := messages[namespace+"."+code]; ok {
		if s, ok := m[lang]; ok && s != "" {
			return s
		}
		if s, ok := m[defaultLang]; ok && s != "" {
			return s
		}
	}
	return code
}

// t is a request-scoped translator used by handler code (e.g. flash messages).
func (s *Server) t(r *http.Request, key string) string { return tr(langFromRequest(r), key) }

// tf is the request-scoped formatting translator used by handler code.
func (s *Server) tf(r *http.Request, key string, args ...any) string {
	return trf(langFromRequest(r), key, args...)
}

// handleSetLang persists the chosen language in a cookie and returns the user to
// the page they came from. It accepts only known codes and only same-origin
// relative redirect targets, so it can never be used as an open redirect.
func (s *Server) handleSetLang(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if !isSupportedLang(code) {
		code = defaultLang
	}
	http.SetCookie(w, &http.Cookie{
		Name:     langCookie,
		Value:    code,
		Path:     "/",
		MaxAge:   365 * 24 * 3600,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.SecureCookies,
	})
	http.Redirect(w, r, safeReturnTo(r), http.StatusSeeOther)
}

// safeReturnTo derives a safe local redirect target from the Referer header,
// falling back to "/" when it is missing, cross-host, or not a plain absolute
// path. Only the path and query are ever used.
func safeReturnTo(r *http.Request) string {
	ref := r.Header.Get("Referer")
	if ref == "" {
		return "/"
	}
	u, err := url.Parse(ref)
	if err != nil {
		return "/"
	}
	if u.Host != "" && r.Host != "" && u.Host != r.Host {
		return "/"
	}
	p := u.EscapedPath()
	if p == "" || !strings.HasPrefix(p, "/") || strings.HasPrefix(p, "//") {
		return "/"
	}
	if u.RawQuery != "" {
		return p + "?" + u.RawQuery
	}
	return p
}

// langTemplateFuncs returns the per-request template helpers bound to lang.
// render() layers these over the parsed templates via Clone so every template —
// including sub-templates invoked with scalar data like {{template "statuspill"
// .Status}} — can call {{t "key"}} with no per-handler data threading.
func langTemplateFuncs(lang string) template.FuncMap {
	return template.FuncMap{
		// t translates a key. Messages are trusted, developer-authored static
		// strings, so returning template.HTML lets a few of them embed inline
		// markup (e.g. <code>…</code>); it stays auto-escaped in attribute
		// contexts.
		"t": func(key string) template.HTML { return template.HTML(tr(lang, key)) },
		// tf translates a format string and interpolates args.
		"tf": func(key string, args ...any) template.HTML {
			return template.HTML(trf(lang, key, args...))
		},
		// curlang / langs / langName drive the language switcher and <html lang>.
		"curlang":  func() string { return lang },
		"langs":    func() []string { return supportedLangs },
		"langName": func(code string) string { return langNames[code] },
		// eventlabel / actionlabel become language-aware (they previously
		// returned hard-coded Azerbaijani). Unknown codes fall through to the raw
		// code via label().
		"eventlabel":  func(code string) string { return label(lang, "event", code) },
		"actionlabel": func(code string) string { return label(lang, "action", code) },
	}
}
