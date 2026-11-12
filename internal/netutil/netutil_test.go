package netutil

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func lookupServer(t *testing.T, body string, status int, calls *int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls != nil {
			*calls++
		}
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
}

func testDetector(lookupURL string, ifaces []net.IP, probe func(context.Context, string) bool) *Detector {
	return &Detector{
		LookupURL: lookupURL,
		TTL:       time.Minute,
		Client:    &http.Client{Timeout: time.Second},
		IfaceIPs:  func() ([]net.IP, error) { return ifaces, nil },
		ProbeFn:   probe,
	}
}

func TestDetectPublicIPOnLocalInterface(t *testing.T) {
	srv := lookupServer(t, "203.0.113.7", http.StatusOK, nil)
	defer srv.Close()
	d := testDetector(srv.URL, []net.IP{net.ParseIP("192.168.1.5"), net.ParseIP("203.0.113.7")}, nil)
	res := d.Detect(context.Background())
	if !res.Viable || res.PublicIP != "203.0.113.7" {
		t.Fatalf("got %+v, want viable 203.0.113.7", res)
	}
	if got := res.BaseURL(); got != "http://203.0.113.7" {
		t.Errorf("BaseURL = %q", got)
	}
}

func TestDetectPublicIPWithHairpinProbe(t *testing.T) {
	srv := lookupServer(t, "169.58.27.10", http.StatusOK, nil)
	defer srv.Close()
	// Container case: host NIC invisible, but the inbound path answers.
	d := testDetector(srv.URL, []net.IP{net.ParseIP("172.18.0.3")}, func(_ context.Context, ip string) bool { return ip == "169.58.27.10" })
	res := d.Detect(context.Background())
	if !res.Viable {
		t.Fatalf("hairpin-probe success should be viable, got %+v", res)
	}
}

func TestDetectPublicIPProbeFails(t *testing.T) {
	srv := lookupServer(t, "169.58.27.10", http.StatusOK, nil)
	defer srv.Close()
	d := testDetector(srv.URL, []net.IP{net.ParseIP("172.18.0.3")}, func(context.Context, string) bool { return false })
	if res := d.Detect(context.Background()); res.Viable {
		t.Fatalf("failed probe should be non-viable, got %+v", res)
	}
}

func TestDetectPrivateOutboundIP(t *testing.T) {
	for _, ip := range []string{"10.3.4.5", "192.168.0.9", "172.16.1.2", "100.64.3.3"} {
		srv := lookupServer(t, ip, http.StatusOK, nil)
		d := testDetector(srv.URL, nil, func(context.Context, string) bool { return true })
		if res := d.Detect(context.Background()); res.Viable {
			t.Errorf("outbound %s (private/CGNAT) must not be viable", ip)
		}
		srv.Close()
	}
}

func TestDetectLookupError(t *testing.T) {
	srv := lookupServer(t, "", http.StatusBadGateway, nil)
	defer srv.Close()
	d := testDetector(srv.URL, nil, func(context.Context, string) bool { return true })
	res := d.Detect(context.Background())
	if res.Viable || res.PublicIP != "" || res.BaseURL() != "" {
		t.Fatalf("lookup failure must yield empty non-viable result, got %+v", res)
	}
}

func TestDetectGarbageBody(t *testing.T) {
	srv := lookupServer(t, "<html>not an ip</html>", http.StatusOK, nil)
	defer srv.Close()
	d := testDetector(srv.URL, nil, nil)
	if res := d.Detect(context.Background()); res.PublicIP != "" {
		t.Fatalf("garbage body must be rejected, got %+v", res)
	}
}

func TestDetectCaching(t *testing.T) {
	calls := 0
	srv := lookupServer(t, "203.0.113.7", http.StatusOK, &calls)
	defer srv.Close()
	d := testDetector(srv.URL, []net.IP{net.ParseIP("203.0.113.7")}, nil)
	d.Detect(context.Background())
	d.Detect(context.Background())
	d.Detect(context.Background())
	if calls != 1 {
		t.Fatalf("expected 1 lookup with warm cache, got %d", calls)
	}
	// Expired cache refetches.
	d.TTL = -time.Second
	d.Detect(context.Background())
	if calls != 2 {
		t.Fatalf("expired cache should refetch, got %d lookups", calls)
	}
}

func TestResultBaseURLEmptyWhenNotViable(t *testing.T) {
	if got := (Result{PublicIP: "203.0.113.7", Viable: false}).BaseURL(); got != "" {
		t.Errorf("non-viable result must not produce a base URL, got %q", got)
	}
	if got := (Result{}).BaseURL(); got != "" {
		t.Errorf("empty result must not produce a base URL, got %q", got)
	}
}
