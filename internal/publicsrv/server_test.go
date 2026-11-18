package publicsrv

import (
	"strings"
	"testing"
	"time"
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

func TestIsScannerUA(t *testing.T) {
	scanners := []string{
		"",
		"curl/8.4.0",
		"python-requests/2.31.0",
		"Mozilla/5.0 (compatible; Barracuda/1.0)",
		"Mozilla/5.0 SafeLinks",
		"Mozilla/5.0 (compatible; proofpoint)",
		"Go-http-client/1.1",
		"facebookexternalhit/1.1",
		"Mozilla/5.0 (Windows NT 10.0) HeadlessChrome/120.0",
		"SlackBot-LinkExpanding 1.0",
	}
	for _, ua := range scanners {
		if !isScannerUA(ua) {
			t.Errorf("expected %q to be detected as a scanner", ua)
		}
	}

	humans := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_1 like Mac OS X) AppleWebKit/605.1.15 Mobile/15E148 Safari/604.1",
		"Microsoft Outlook 16.0", // a real desktop client fetching images
	}
	for _, ua := range humans {
		if isScannerUA(ua) {
			t.Errorf("expected %q to be treated as a human browser", ua)
		}
	}
}

func TestIsAutomatedTimingWindow(t *testing.T) {
	humanUA := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36"
	grace := scannerGraceWindowDefault

	justSent := time.Now().Add(-2 * time.Second)
	if !isAutomated(humanUA, &justSent, grace) {
		t.Error("a human-looking hit moments after send should be treated as automated (scanner prefetch)")
	}

	longAfter := time.Now().Add(-10 * time.Minute)
	if isAutomated(humanUA, &longAfter, grace) {
		t.Error("a human-looking hit well after send should count as a real interaction")
	}

	// A scanner UA is automated regardless of timing.
	if !isAutomated("curl/8.4.0", &longAfter, grace) {
		t.Error("a scanner UA should be automated even long after send")
	}

	// No sent timestamp: fall back to UA only.
	if isAutomated(humanUA, nil, grace) {
		t.Error("with no sent time, a human UA should not be treated as automated")
	}

	// CLICK path: an empty User-Agent is a bot (real browsers always send one),
	// so it stays automated even long after send.
	if !isAutomated("", &longAfter, grace) {
		t.Error("an empty UA on the click path should be treated as automated even long after send")
	}
}

// TestOpenAutomatedReason pins the OPEN-pixel behaviour: empty / mail-proxy
// fetches must count as genuine opens once they arrive after the grace window,
// while delivery-time prefetches and known scanners are still dropped. This is
// the fix for opens never registering behind Microsoft 365 / Outlook image
// proxies (which fetch the pixel server-side with no User-Agent).
func TestOpenAutomatedReason(t *testing.T) {
	grace := scannerGraceWindowDefault
	justSent := time.Now().Add(-2 * time.Second)
	longAfter := time.Now().Add(-10 * time.Minute)
	humanUA := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36"

	cases := []struct {
		name   string
		ua     string
		sentAt *time.Time
		want   string // "" means it counts as a real open
	}{
		{"empty UA after grace counts (M365/Outlook proxy open)", "", &longAfter, ""},
		{"empty UA within grace is a delivery-time prefetch", "", &justSent, "grace-window"},
		{"human UA after grace counts", humanUA, &longAfter, ""},
		{"human UA within grace is a prefetch", humanUA, &justSent, "grace-window"},
		{"known scanner UA is always dropped", "curl/8.4.0", &longAfter, "scanner-ua"},
		{"safelinks UA is always dropped", "Mozilla/5.0 SafeLinks", &longAfter, "scanner-ua"},
		{"empty UA with no sent time counts (cannot be a prefetch)", "", nil, ""},
	}
	for _, c := range cases {
		if got := openAutomatedReason(c.ua, c.sentAt, grace); got != c.want {
			t.Errorf("%s: openAutomatedReason(%q)=%q, want %q", c.name, c.ua, got, c.want)
		}
	}
}
