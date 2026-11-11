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
