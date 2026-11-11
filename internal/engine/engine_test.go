package engine

import (
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

func TestAddrOnly(t *testing.T) {
	if got := addrOnly("IT Support <it@example.com>"); got != "it@example.com" {
		t.Errorf("got %q", got)
	}
	if got := addrOnly("plain@example.com"); got != "plain@example.com" {
		t.Errorf("got %q", got)
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
