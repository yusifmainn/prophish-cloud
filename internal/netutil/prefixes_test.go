package netutil

import (
	"net"
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
