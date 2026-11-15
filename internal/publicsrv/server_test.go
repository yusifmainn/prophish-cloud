package publicsrv

import (
	"testing"
	"time"
)

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
