package server

import (
	"io/fs"
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestMessagesComplete asserts every catalog entry defines all supported
// languages with a non-empty value, so no screen can silently fall back.
func TestMessagesComplete(t *testing.T) {
	for key, m := range messages {
		for _, lang := range supportedLangs {
			if strings.TrimSpace(m[lang]) == "" {
				t.Errorf("message %q missing or empty for language %q", key, lang)
			}
		}
	}
}

var (
	reTmplT  = regexp.MustCompile(`\{\{-?\s*t\s+"([^"]+)"`)
	reTmplTf = regexp.MustCompile(`\{\{-?\s*tf\s+"([^"]+)"`)
	reGoT    = regexp.MustCompile(`\.t\(r,\s*"([^"]+)"`)
	reGoTf   = regexp.MustCompile(`\.tf\(r,\s*"([^"]+)"`)
)

// TestTemplateKeysDefined asserts every key referenced by a {{t}}/{{tf}} call in
// a template is present in the catalog. This is the guard that lets templates and
// the catalog be edited independently without a key drifting out of sync.
func TestTemplateKeysDefined(t *testing.T) {
	err := fs.WalkDir(templateFS, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".html") {
			return err
		}
		b, err := templateFS.ReadFile(path)
		if err != nil {
			return err
		}
		for _, re := range []*regexp.Regexp{reTmplT, reTmplTf} {
			for _, mch := range re.FindAllStringSubmatch(string(b), -1) {
				if _, ok := messages[mch[1]]; !ok {
					t.Errorf("%s references undefined message key %q", path, mch[1])
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestHandlerKeysDefined asserts every key referenced by a request-scoped
// s.t(r, …) / s.tf(r, …) call in the package's Go sources is defined.
func TestHandlerKeysDefined(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		b, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, re := range []*regexp.Regexp{reGoT, reGoTf} {
			for _, mch := range re.FindAllStringSubmatch(string(b), -1) {
				if _, ok := messages[mch[1]]; !ok {
					t.Errorf("%s references undefined message key %q", name, mch[1])
				}
			}
		}
	}
}

// TestRenderLangs is a smoke test: the login screen renders in every language,
// shows that language's strings, and carries the language switcher with the
// active language marked.
func TestRenderLangs(t *testing.T) {
	tmpl, err := parseTemplates()
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		langAZ: "Giriş",
		langEN: "Sign in",
		langRU: "Вход",
	}
	for lang, phrase := range want {
		cl, err := tmpl.Clone()
		if err != nil {
			t.Fatal(err)
		}
		cl = cl.Funcs(langTemplateFuncs(lang))
		var sb strings.Builder
		if err := cl.ExecuteTemplate(&sb, "login.html", map[string]any{}); err != nil {
			t.Fatalf("render login (%s): %v", lang, err)
		}
		out := sb.String()
		if !strings.Contains(out, phrase) {
			t.Errorf("login (%s) missing expected phrase %q", lang, phrase)
		}
		if !strings.Contains(out, `class="langpick"`) {
			t.Errorf("login (%s) missing language picker", lang)
		}
		if !strings.Contains(out, `href="/lang/`+lang+`"`) {
			t.Errorf("login (%s) missing switcher link for %s", lang, lang)
		}
	}
}
