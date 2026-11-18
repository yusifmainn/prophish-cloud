package engine

import (
	"net/smtp"
	"strings"
	"testing"

	"prophish/internal/models"
)

func TestRenderTextMergesVars(t *testing.T) {
	out, err := renderText("Salam {{.FirstName}}, {{.Email}}", TargetData{FirstName: "Eli", Email: "eli@acme.az"})
	if err != nil {
		t.Fatal(err)
	}
	if out != "Salam Eli, eli@acme.az" {
		t.Fatalf("unexpected: %q", out)
	}
}

func TestRenderHTMLInjectsPixel(t *testing.T) {
	data := TargetData{
		FirstName:     "Eli",
		URL:           clickURL("https://p.example.com", "tok123"),
		TrackingPixel: pixelTag("https://p.example.com", "tok123"),
	}
	out, err := renderHTML(`<html><body><a href="{{.URL}}">link</a></body></html>`, data)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "https://p.example.com/t/tok123") {
		t.Errorf("click URL missing: %s", out)
	}
	if !strings.Contains(out, "/p/tok123.png") {
		t.Errorf("tracking pixel missing: %s", out)
	}
	// Pixel must be injected before </body>.
	if strings.Index(out, "/p/tok123.png") > strings.Index(out, "</body>") {
		t.Errorf("pixel should be before </body>")
	}
}

func TestRenderHTMLNoDuplicatePixel(t *testing.T) {
	data := TargetData{TrackingPixel: pixelTag("https://p.example.com", "tok")}
	out, err := renderHTML(`<html><body>{{.TrackingPixel}}</body></html>`, data)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(out, "/p/tok.png"); n != 1 {
		t.Errorf("expected pixel once, got %d", n)
	}
}

// TestRenderHTMLNoPixelWhenOptedOut covers the campaign-level switch: when the
// engine passes an empty TrackingPixel (campaign with open tracking disabled),
// renderHTML must not inject any pixel.
func TestRenderHTMLNoPixelWhenOptedOut(t *testing.T) {
	out, err := renderHTML(`<html><body><p>Hello {{.FirstName}}</p></body></html>`,
		TargetData{FirstName: "Eli", URL: clickURL("https://p.example.com", "tok")})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "/p/tok.png") {
		t.Errorf("no pixel must be injected when open tracking is disabled:\n%s", out)
	}
}

// TestRenderHTMLCanaryInjection covers the scanner-canary: a footer copyright
// link the engine injects before </body> unless the template already carries
// {{.CanaryURL}}.
func TestRenderHTMLCanaryInjection(t *testing.T) {
	data := TargetData{URL: clickURL("https://p.example.com", "tok"),
		CanaryURL: canaryURL("https://p.example.com", "tok")}

	// Template without an explicit canary: one is injected, before </body>.
	out, err := renderHTML(`<html><body><a href="{{.URL}}">click</a></body></html>`, data)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(out, "/c/tok"); n != 1 {
		t.Fatalf("expected canary once, got %d:\n%s", n, out)
	}
	if !strings.Contains(out, canarySnippet(canaryURL("https://p.example.com", "tok"))+"</body>") {
		t.Errorf("canary must be spliced immediately before </body>:\n%s", out)
	}

	// Template with an explicit {{.CanaryURL}}: no duplicate injection.
	out2, err := renderHTML(`<html><body>{{.CanaryURL}}<a href="{{.URL}}">click</a></body></html>`, data)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(out2, "/c/tok"); n != 1 {
		t.Errorf("expected canary exactly once with explicit placeholder, got %d:\n%s", n, out2)
	}
}

// TestRenderHTMLCanaryInjectionWithNonASCIIContent mirrors the pixel splice
// regression: the canary injection point must be computed on original bytes so
// multi-byte runes before </body> cannot shift it mid-tag.
func TestRenderHTMLCanaryInjectionWithNonASCIIContent(t *testing.T) {
	tmpl := "<!doctype html>\n<html lang=\"az\">\n<head><title>BİTMƏ İ</title></head>\n" +
		"<body>\n<p>Salam</p>\n</body>\n</html>\n"
	out, err := renderHTML(tmpl, TargetData{
		CanaryURL: canaryURL("https://p.example.com", "tok"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(out, "/c/tok"); n != 1 {
		t.Fatalf("expected canary exactly once, got %d:\n%s", n, out)
	}
	if !strings.Contains(out, canarySnippet(canaryURL("https://p.example.com", "tok"))+"</body>") {
		t.Errorf("canary must be spliced immediately before </body>:\n%s", out)
	}
}

// TestRenderHTMLPixelInjectionWithNonASCIIContent is a regression test for the
// splice bug where the pixel was injected mid-tag (leaving stray "able>" text
// visible in the mail) whenever the template contained a rune whose lowercase
// form has a different byte length, such as the Azerbaijani/Turkish İ.
func TestRenderHTMLPixelInjectionWithNonASCIIContent(t *testing.T) {
	tmpl := "<!doctype html>\n<html lang=\"az\">\n<head><meta charset=\"utf-8\"></head>\n" +
		"<body>\n<table role=\"presentation\" width=\"100%\"><tr><td>BİTMƏK ÜZRƏ İT</td></tr></table>\n" +
		"</body>\n</html>\n"
	pixel := string(pixelTag("https://p.example.com", "tok"))
	out, err := renderHTML(tmpl, TargetData{TrackingPixel: pixelTag("https://p.example.com", "tok")})
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(out, "/p/tok.png"); n != 1 {
		t.Fatalf("expected pixel exactly once, got %d:\n%s", n, out)
	}
	if !strings.Contains(out, pixel+"</body>") {
		t.Errorf("pixel must be spliced immediately before </body>:\n%s", out)
	}
	if !strings.Contains(out, "</table>\n"+pixel+"</body>") {
		t.Errorf("pixel split the surrounding markup instead of landing between </table> and </body>:\n%s", out)
	}
}

func TestAddrOnly(t *testing.T) {
	if got := addrOnly("IT Support <it@example.com>"); got != "it@example.com" {
		t.Errorf("got %q", got)
	}
	if got := addrOnly("plain@example.com"); got != "plain@example.com" {
		t.Errorf("got %q", got)
	}
}

func TestPickAuthPrefersPlainThenLogin(t *testing.T) {
	cfg := SMTPConfig{Username: "u", Password: "p", Host: "mail.example.com"}
	if a, err := pickAuth("PLAIN LOGIN", cfg, true); err != nil {
		t.Fatal(err)
	} else if _, ok := a.(*plainAuth); !ok {
		t.Errorf("PLAIN offered → want plainAuth, got %T", a)
	}
	if a, err := pickAuth("LOGIN", cfg, true); err != nil {
		t.Fatal(err)
	} else if _, ok := a.(*loginAuth); !ok {
		t.Errorf("only LOGIN offered → want loginAuth, got %T", a)
	}
	if _, err := pickAuth("CRAM-MD5", cfg, true); err == nil {
		t.Error("unsupported mechanism must error")
	}
}

// TestAuthRequiresEncryptedChannel locks in the implicit-TLS fix: smtp.Client
// does NOT set ServerInfo.TLS when it is handed an already-TLS connection (port
// 465), so our auth mechanisms must trust the caller's `secure` flag — allowing
// AUTH over implicit TLS while still refusing it on a genuinely plaintext link.
func TestAuthRequiresEncryptedChannel(t *testing.T) {
	host := "mail.example.com"
	plaintext := &smtp.ServerInfo{Name: host, TLS: false}

	// secure=true (implicit TLS): AUTH is allowed even though ServerInfo.TLS=false.
	if _, _, err := (&plainAuth{username: "u", password: "p", host: host, secure: true}).Start(plaintext); err != nil {
		t.Errorf("PLAIN over implicit TLS should be allowed: %v", err)
	}
	if _, _, err := (&loginAuth{username: "u", password: "p", host: host, secure: true}).Start(plaintext); err != nil {
		t.Errorf("LOGIN over implicit TLS should be allowed: %v", err)
	}

	// secure=false and no server TLS: credentials must never be sent.
	if _, _, err := (&plainAuth{username: "u", password: "p", host: host, secure: false}).Start(plaintext); err == nil {
		t.Error("PLAIN on an unencrypted connection must be refused")
	}
	if _, _, err := (&loginAuth{username: "u", password: "p", host: host, secure: false}).Start(plaintext); err == nil {
		t.Error("LOGIN on an unencrypted connection must be refused")
	}
}

func TestPlainAuthResponseAndHostBinding(t *testing.T) {
	host := "mail.example.com"
	mech, resp, err := (&plainAuth{username: "user@x", password: "secret", host: host, secure: true}).
		Start(&smtp.ServerInfo{Name: host, TLS: true})
	if err != nil {
		t.Fatal(err)
	}
	if mech != "PLAIN" || string(resp) != "\x00user@x\x00secret" {
		t.Errorf("mech=%q resp=%q", mech, resp)
	}
	// A mismatched server name must be rejected so credentials are never sent to
	// an unexpected host (e.g. after a redirect/MITM).
	if _, _, err := (&plainAuth{username: "u", password: "p", host: host, secure: true}).
		Start(&smtp.ServerInfo{Name: "evil.example.com", TLS: true}); err == nil {
		t.Error("mismatched server name must be refused")
	}
}

func TestResolveBaseURLFallbackChain(t *testing.T) {
	host := func() string { return "http://203.0.113.9" }
	l := NewLauncher(nil, nil, "https://global.example.com", host)
	if got := l.resolveBaseURL(&models.Campaign{PublicBaseURL: "https://camp.example.com"}); got != "https://camp.example.com" {
		t.Errorf("campaign override must win, got %q", got)
	}
	if got := l.resolveBaseURL(&models.Campaign{}); got != "https://global.example.com" {
		t.Errorf("global default second, got %q", got)
	}
	l2 := NewLauncher(nil, nil, "", host)
	if got := l2.resolveBaseURL(&models.Campaign{}); got != "http://203.0.113.9" {
		t.Errorf("detected host URL third, got %q", got)
	}
	l3 := NewLauncher(nil, nil, "", func() string { return "" })
	if got := l3.resolveBaseURL(&models.Campaign{}); got != "http://127.0.0.1:8080" {
		t.Errorf("loopback last resort, got %q", got)
	}
}
