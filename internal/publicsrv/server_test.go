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

	justSent := time.Now().Add(-2 * time.Second)
	if !isAutomated(humanUA, &justSent) {
		t.Error("a human-looking hit moments after send should be treated as automated (scanner prefetch)")
	}

	longAfter := time.Now().Add(-10 * time.Minute)
	if isAutomated(humanUA, &longAfter) {
		t.Error("a human-looking hit well after send should count as a real interaction")
	}

	// A scanner UA is automated regardless of timing.
	if !isAutomated("curl/8.4.0", &longAfter) {
		t.Error("a scanner UA should be automated even long after send")
	}

	// No sent timestamp: fall back to UA only.
	if isAutomated(humanUA, nil) {
		t.Error("with no sent time, a human UA should not be treated as automated")
	}
}

func TestRecentlyProbed(t *testing.T) {
	if recentlyProbed(nil, scannerPairWindow) {
		t.Error("no probe must mean not probed")
	}
	stale := time.Now().Add(-scannerPairWindow - time.Minute)
	if recentlyProbed(&stale, scannerPairWindow) {
		t.Error("a probe older than the pair window must not suppress hits")
	}
	fresh := time.Now().Add(-time.Second)
	if !recentlyProbed(&fresh, scannerPairWindow) {
		t.Error("a probe inside the pair window must suppress hits as automated")
	}
}

func TestIsOpenAutomated(t *testing.T) {
	// Image proxies send sparse/empty UAs on a genuine open — opens must NOT be
	// UA-filtered, only timing-filtered.
	if isOpenAutomated(nil, nil) {
		t.Error("a proxy fetch long after send with no probe is a real open (must not be automated)")
	}
	sent := time.Now().Add(-time.Hour)
	if isOpenAutomated(&sent, nil) {
		t.Error("an open an hour after send must count (must not be automated)")
	}
	justSent := time.Now().Add(-2 * time.Second)
	if !isOpenAutomated(&justSent, nil) {
		t.Error("an image fetch 2s after send is a scanner prefetch")
	}
	probe := time.Now().Add(-time.Second)
	if !isOpenAutomated(&sent, &probe) {
		t.Error("an image fetch inside the scanner burst is automated")
	}
	staleProbe := time.Now().Add(-time.Minute)
	if isOpenAutomated(&sent, &staleProbe) {
		t.Error("an open a minute after the burst is human")
	}
}
