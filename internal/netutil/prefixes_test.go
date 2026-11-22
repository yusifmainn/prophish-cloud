package netutil

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func TestParseIPv4Prefixes(t *testing.T) {
	nets, err := parseIPv4Prefixes([]string{"72.144.0.0/14", "2001:db8::/32", "junk", " 10.0.0.0/8 "})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nets) != 2 {
		t.Fatalf("want 2 v4 prefixes (v6 and junk skipped), got %d", len(nets))
	}
	if !nets[0].Contains(net.ParseIP("72.145.93.148")) {
		t.Error("72.145.93.148 must be inside the first prefix")
	}
	if _, err := parseIPv4Prefixes([]string{"junk", "2001:db8::/32"}); err == nil {
		t.Error("an input with no usable v4 prefixes must fail")
	}
}

func TestPrefixCacheRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")
	nets, err := parseIPv4Prefixes([]string{"72.144.0.0/14", "13.64.0.0/11"})
	if err != nil {
		t.Fatal(err)
	}
	if err := SavePrefixCache(path, nets); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, ok := LoadPrefixCache(path, 24*time.Hour)
	if !ok || len(got) != 2 {
		t.Fatalf("fresh cache must load 2 prefixes, ok=%v n=%d", ok, len(got))
	}
	if _, ok := LoadPrefixCache(path, time.Nanosecond); ok {
		t.Error("a cache older than maxAge must be rejected")
	}
	if _, ok := LoadPrefixCache(filepath.Join(dir, "missing.json"), 24*time.Hour); ok {
		t.Error("a missing cache file must be rejected")
	}
}

func TestFetchDatacenterRanges(t *testing.T) {
	// The live list is plain text, one CIDR per line, '#' comments, v6 and
	// junk lines silently skipped — mirror the shape of the real X4BNet file.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/list.txt" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte("# comment line\n\n94.46.32.0/19\n 203.0.113.0/24 \n2001:db8::/32\njunk\n"))
	}))
	defer srv.Close()

	nets, err := FetchDatacenterRanges(context.Background(), srv.URL+"/list.txt")
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(nets) != 2 {
		t.Fatalf("want 2 usable v4 ranges, got %d", len(nets))
	}
	if !nets[0].Contains(net.ParseIP("94.46.34.70")) {
		t.Error("94.46.34.70 (live incident IP) must be inside the first range")
	}

	if _, err := FetchDatacenterRanges(context.Background(), srv.URL+"/missing"); err == nil {
		t.Error("a non-200 response must fail")
	}
	if DefaultDatacenterListURL == "" {
		t.Error("the default list URL must be set")
	}
}
