package engine

import (
	"strings"
	"testing"
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
