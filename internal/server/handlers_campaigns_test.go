package server

import (
	"net/http/httptest"
	"strings"
	"testing"

	"prophish/internal/config"
)

// A loopback PROPHISH_PUBLIC_BASE_URL (the .env.example default) must NOT become
// the "host" listener URL — that would point every tracking link at the target's
// own loopback, so opens/clicks never register. listenerData must skip it and
// derive this host's real address instead.
func TestListenerDataIgnoresLoopback(t *testing.T) {
	s := &Server{cfg: &config.Config{PublicBaseURL: "http://127.0.0.1:8080", PublicAddr: "0.0.0.0:8080"}}
	_, hostURL := s.listenerData(httptest.NewRequest("GET", "/campaigns", nil))
	if strings.Contains(hostURL, "127.0.0.1") || strings.Contains(strings.ToLower(hostURL), "localhost") {
		t.Errorf("host listener URL must not be loopback, got %q", hostURL)
	}
}

func TestSanitizePublicBaseURL(t *testing.T) {
	cases := []struct {
		in     string
		want   string
		wantOK bool
	}{
		{"", "", true},
		{"   ", "", true},
		{"https://abc.trycloudflare.com", "https://abc.trycloudflare.com", true},
		{" https://abc.trycloudflare.com/ ", "https://abc.trycloudflare.com", true},
		{"http://127.0.0.1:8080/", "http://127.0.0.1:8080", true},
		{"abc.trycloudflare.com", "", false},
		{"javascript:alert(1)", "", false},
		{"ftp://x.az", "", false},
	}
	for _, c := range cases {
		got, ok := sanitizePublicBaseURL(c.in)
		if ok != c.wantOK || got != c.want {
			t.Errorf("sanitizePublicBaseURL(%q) = (%q, %v), want (%q, %v)", c.in, got, ok, c.want, c.wantOK)
		}
	}
}

func TestIsLoopbackBaseURL(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"http://127.0.0.1:8080", true},
		{"http://127.0.0.1:8080/", true},
		{"http://localhost:8080", true},
		{"https://LocalHost", true},
		{"http://[::1]:8080", true},
		{"http://127.5.5.5", true},
		{"http://10.20.51.58:8080", false},
		{"http://192.168.1.10:8080", false},
		{"https://abc.trycloudflare.com", false},
		{"https://mail-check.example.com", false},
	}
	for _, c := range cases {
		if got := isLoopbackBaseURL(c.in); got != c.want {
			t.Errorf("isLoopbackBaseURL(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestSanitizeRedirectURL(t *testing.T) {
	cases := []struct {
		in     string
		want   string
		wantOK bool
	}{
		// Required: blank is invalid for a public campaign.
		{"", "", false},
		{"   ", "", false},
		// Full path/query is preserved verbatim (no trailing-slash trimming).
		{"https://portal.example.com/giris", "https://portal.example.com/giris", true},
		{" https://portal.example.com/a/ ", "https://portal.example.com/a/", true},
		{"http://10.0.0.5/login?ref=1", "http://10.0.0.5/login?ref=1", true},
		// Must be an http(s) URL.
		{"example.com", "", false},
		{"javascript:alert(1)", "", false},
		{"ftp://x.az", "", false},
	}
	for _, c := range cases {
		got, ok := sanitizeRedirectURL(c.in)
		if ok != c.wantOK || got != c.want {
			t.Errorf("sanitizeRedirectURL(%q) = (%q, %v), want (%q, %v)", c.in, got, ok, c.want, c.wantOK)
		}
	}
}
