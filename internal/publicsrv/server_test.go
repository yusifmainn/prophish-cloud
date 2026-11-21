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
	if !isAutomated(false, 15*time.Second, humanUA, &justSent) {
		t.Error("a human-looking hit moments after send should be treated as automated (scanner prefetch)")
	}

	longAfter := time.Now().Add(-10 * time.Minute)
	if isAutomated(false, 15*time.Second, humanUA, &longAfter) {
		t.Error("a human-looking hit well after send should count as a real interaction")
	}

	// A scanner UA is automated regardless of timing.
	if !isAutomated(false, 15*time.Second, "curl/8.4.0", &longAfter) {
		t.Error("a scanner UA should be automated even long after send")
	}

	// No sent timestamp: fall back to UA only.
	if isAutomated(false, 15*time.Second, humanUA, nil) {
		t.Error("with no sent time, a human UA should not be treated as automated")
	}
}

func TestRecentlyProbed(t *testing.T) {
	if recentlyProbed(nil, 10 * time.Second) {
		t.Error("no probe must mean not probed")
	}
	stale := time.Now().Add(-10 * time.Second - time.Minute)
	if recentlyProbed(&stale, 10 * time.Second) {
		t.Error("a probe older than the pair window must not suppress hits")
	}
	fresh := time.Now().Add(-time.Second)
	if !recentlyProbed(&fresh, 10 * time.Second) {
		t.Error("a probe inside the pair window must suppress hits as automated")
	}
}

func TestIsOpenAutomated(t *testing.T) {
	// Image proxies send sparse/empty UAs on a genuine open — opens must NOT be
	// UA-filtered, only timing-filtered.
	if isOpenAutomated(false, 15*time.Second, 10*time.Second, nil, nil) {
		t.Error("a proxy fetch long after send with no probe is a real open (must not be automated)")
	}
	sent := time.Now().Add(-time.Hour)
	if isOpenAutomated(false, 15*time.Second, 10*time.Second, &sent, nil) {
		t.Error("an open an hour after send must count (must not be automated)")
	}
	justSent := time.Now().Add(-2 * time.Second)
	if !isOpenAutomated(false, 15*time.Second, 10*time.Second, &justSent, nil) {
		t.Error("an image fetch 2s after send is a scanner prefetch")
	}
	probe := time.Now().Add(-time.Second)
	if !isOpenAutomated(false, 15*time.Second, 10*time.Second, &sent, &probe) {
		t.Error("an image fetch inside the scanner burst is automated")
	}
	staleProbe := time.Now().Add(-time.Minute)
	if isOpenAutomated(false, 15*time.Second, 10*time.Second, &sent, &staleProbe) {
		t.Error("an open a minute after the burst is human")
	}
}

func TestPreWindowDecision(t *testing.T) {
	// The reason tokens land verbatim in tracking_hits.reason, so they are part
	// of the contract: operators filter/diagnose by them in the raw-hits table.
	longAfter := time.Now().Add(-10 * time.Minute)
	if why := preWindowDecision(false, 15*time.Second, "curl/8.4.0", &longAfter); why != "ua" {
		t.Errorf("scanner UA long after send: got reason %q, want ua", why)
	}
	if why := preWindowDecision(false, 15*time.Second, "curl/8.4.0", nil); why != "ua" {
		t.Errorf("scanner UA with no sent time: got reason %q, want ua", why)
	}
	justSent := time.Now().Add(-2 * time.Second)
	humanUA := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36"
	if why := preWindowDecision(false, 15*time.Second, humanUA, &justSent); why != "grace_window" {
		t.Errorf("human UA moments after send: got reason %q, want grace_window", why)
	}
	if why := preWindowDecision(false, 15*time.Second, humanUA, &longAfter); why != "" {
		t.Errorf("human UA long after send: got reason %q, want empty (not discarded)", why)
	}
	if why := preWindowDecision(false, 15*time.Second, humanUA, nil); why != "" {
		t.Errorf("human UA with no sent time: got reason %q, want empty", why)
	}
}

func TestOpenIgnoreReason(t *testing.T) {
	sent := time.Now().Add(-time.Hour)
	if why := openIgnoreReason(false, 15*time.Second, 10*time.Second, &sent, nil); why != "" {
		t.Errorf("open an hour after send: got reason %q, want empty (counts as a real open)", why)
	}
	justSent := time.Now().Add(-2 * time.Second)
	if why := openIgnoreReason(false, 15*time.Second, 10*time.Second, &justSent, nil); why != "grace_window" {
		t.Errorf("prefetch right after send: got reason %q, want grace_window", why)
	}
	probe := time.Now().Add(-time.Second)
	if why := openIgnoreReason(false, 15*time.Second, 10*time.Second, &sent, &probe); why != "pair_window" {
		t.Errorf("open inside the scanner burst: got reason %q, want pair_window", why)
	}
	if why := openIgnoreReason(false, 15*time.Second, 10*time.Second, nil, nil); why != "" {
		t.Errorf("open with no timing context: got reason %q, want empty", why)
	}
}

func TestTrustedDeliveryCountsFirstHits(t *testing.T) {
	// Trusted mode = the operator asserts the sender is allowlisted in the
	// target tenant: the FIRST open and the FIRST click are the user, no matter
	// what the timing or User-Agent says. All filtering must be bypassed.
	justSent := time.Now().Add(-2 * time.Second)
	humanUA := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36"

	if why := preWindowDecision(true, 15*time.Second, "curl/8.4.0", &justSent); why != "" {
		t.Errorf("trusted mode: scanner UA must not be discarded, got reason %q", why)
	}
	if why := preWindowDecision(true, 15*time.Second, "", &justSent); why != "" {
		t.Errorf("trusted mode: even an empty UA counts, got reason %q", why)
	}
	if why := preWindowDecision(true, 15*time.Second, humanUA, &justSent); why != "" {
		t.Errorf("trusted mode: a click moments after send must count, got reason %q", why)
	}
	probe := time.Now().Add(-time.Second)
	if why := openIgnoreReason(true, 15*time.Second, 10*time.Second, &justSent, &probe); why != "" {
		t.Errorf("trusted mode: an open inside a probe burst must still count, got reason %q", why)
	}
	if isAutomated(true, 15*time.Second, "curl/8.4.0", &justSent) {
		t.Error("trusted mode: isAutomated must always be false")
	}
	if isOpenAutomated(true, 15*time.Second, 10*time.Second, &justSent, &probe) {
		t.Error("trusted mode: isOpenAutomated must always be false")
	}
	// Untrusted mode is unaffected by the flag.
	if why := preWindowDecision(false, 15*time.Second, "curl/8.4.0", nil); why != "ua" {
		t.Errorf("untrusted mode: scanner UA must still be discarded, got reason %q", why)
	}
}

func TestSetScannerWindows(t *testing.T) {
	// The presets are deployment-tunable via env at startup; the setter must
	// keep the current value for any non-positive field (0 = "use default").
	defer SetScannerWindows(ScannerWindows{Grace: 5 * time.Second, Pair: 5 * time.Second},
		ScannerWindows{Grace: 15 * time.Second, Pair: 10 * time.Second})
	baseGrace := defInternal.Grace

	SetScannerWindows(ScannerWindows{}, ScannerWindows{Pair: 3 * time.Second})
	if defInternal.Grace != baseGrace {
		t.Errorf("internal grace must be untouched by a zero-field override, got %s", defInternal.Grace)
	}
	if defExternal.Pair != 3*time.Second {
		t.Errorf("external pair must be applied, got %s", defExternal.Pair)
	}

	SetScannerWindows(ScannerWindows{Grace: 2 * time.Second}, ScannerWindows{})
	if defInternal.Grace != 2*time.Second {
		t.Errorf("internal grace must be applied, got %s", defInternal.Grace)
	}
	if defExternal.Pair != 3*time.Second {
		t.Errorf("external pair must be untouched by a zero-field override, got %s", defExternal.Pair)
	}
}

func TestRouteWindows(t *testing.T) {
	// The 4-mode matrix in one scenario: a human hit 8 seconds after send.
	// On the INTERNAL route (fast delivery) it is a plausible real open and
	// the 5s grace window must NOT discard it; on the EXTERNAL route (slow
	// delivery) an 8s-old hit can only be delivery-time prefetch, so the 15s
	// window must discard it. Trusted mode ignores windows on both routes.
	humanUA := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36"
	sent8sAgo := time.Now().Add(-8 * time.Second)

	if why := preWindowDecision(false, defInternal.Grace, humanUA, &sent8sAgo); why != "" {
		t.Errorf("internal route: an 8s-old human hit must count, got reason %q", why)
	}
	if why := preWindowDecision(false, defExternal.Grace, humanUA, &sent8sAgo); why != "grace_window" {
		t.Errorf("external route: an 8s-old hit must be treated as delivery-time prefetch, got reason %q", why)
	}
	if why := preWindowDecision(true, defExternal.Grace, humanUA, &sent8sAgo); why != "" {
		t.Errorf("trusted mode: windows are off on every route, got reason %q", why)
	}

	if windowsFor("internal") != defInternal {
		t.Error("windowsFor(\"internal\") must return the internal preset")
	}
	if windowsFor("external") != defExternal || windowsFor("") != defExternal || windowsFor("bogus") != defExternal {
		t.Error("anything not explicitly internal must get the conservative external preset")
	}
}

func TestIsInfraIP(t *testing.T) {
	// The live phantom events arrived from 48.209.223.88 (AS8075 Microsoft,
	// RIPE-verified covering prefix 48.208.0.0/13) — the built-in list must
	// catch it, and must leave corporate/user space alone.
	for _, ip := range []string{"48.209.223.88", "13.64.0.1", "40.76.0.1", "104.40.1.1", "65.52.0.1"} {
		if !isInfraIP(ip) {
			t.Errorf("expected %s to be recognized as infrastructure", ip)
		}
	}
	for _, ip := range []string{"203.0.113.7", "10.20.51.5", "85.132.12.9", "", "not-an-ip"} {
		if isInfraIP(ip) {
			t.Errorf("expected %q to NOT be infrastructure", ip)
		}
	}
}

func TestTrustedHitDrop(t *testing.T) {
	browser := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36"
	// Infrastructure IP loses regardless of a browser-looking UA.
	if why := trustedHitDrop(browser, "48.209.223.88"); why != "infra" {
		t.Errorf("infra IP with browser UA: got %q, want infra", why)
	}
	// Bot/empty UA loses on a user network.
	if why := trustedHitDrop("curl/8.4.0", "203.0.113.7"); why != "ua" {
		t.Errorf("bot UA on user IP: got %q, want ua", why)
	}
	if why := trustedHitDrop("", "203.0.113.7"); why != "ua" {
		t.Errorf("empty UA on user IP: got %q, want ua", why)
	}
	// The human passes.
	if why := trustedHitDrop(browser, "203.0.113.7"); why != "" {
		t.Errorf("human hit: got %q, want empty (counted)", why)
	}
}

func TestDynamicInfraCIDRs(t *testing.T) {
	// The ASN-fetched list replaces wholesale on each refresh and is consulted
	// alongside the static built-ins.
	defer SetDynamicInfraCIDRs(nil)
	if isInfraIP("90.100.1.2") {
		t.Fatal("90.100.1.2 must not be infrastructure before the refresh")
	}
	SetDynamicInfraCIDRs(mustCIDRs("90.100.0.0/16"))
	if !isInfraIP("90.100.1.2") {
		t.Error("dynamic list must be consulted by isInfraIP")
	}
	SetDynamicInfraCIDRs(nil)
	if isInfraIP("90.100.1.2") {
		t.Error("replacing with nil must clear the dynamic list")
	}
}
