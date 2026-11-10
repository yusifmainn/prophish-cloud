package publicsrv

import (
	"strings"
	"testing"
)

func TestRenderLandingMergesSubmitURL(t *testing.T) {
	html := `<form method="post" action="{{.SubmitURL}}"><input name="u"></form>Salam {{.FirstName}}`
	out := renderLanding(html, landingData{FirstName: "Eli", SubmitURL: "https://p.example.com/s/tok"})
	if !strings.Contains(out, `action="https://p.example.com/s/tok"`) {
		t.Errorf("submit url missing: %s", out)
	}
	if !strings.Contains(out, "Salam Eli") {
		t.Errorf("name missing: %s", out)
	}
}

func TestRenderLandingBadTemplateIsSafe(t *testing.T) {
	out := renderLanding("{{.Nope.Bad}}", landingData{})
	if out == "" {
		t.Error("expected fallback output")
	}
}
