package server

import "testing"

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
