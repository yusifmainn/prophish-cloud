package server

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"prophish/internal/config"
)

func mustCIDR(t *testing.T, s string) *net.IPNet {
	t.Helper()
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return n
}

func TestAdminIPAllow(t *testing.T) {
	allow := []*net.IPNet{
		mustCIDR(t, "10.30.22.0/24"),
		mustCIDR(t, "10.20.51.0/24"),
	}
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	cases := []struct {
		name   string
		cidrs  []*net.IPNet
		remote string
		want   int
	}{
		{"empty list allows any", nil, "203.0.113.9:5000", http.StatusOK},
		{"in first subnet", allow, "10.30.22.7:4000", http.StatusOK},
		{"in second subnet", allow, "10.20.51.200:4000", http.StatusOK},
		{"loopback always allowed", allow, "127.0.0.1:4000", http.StatusOK},
		{"ipv6 loopback allowed", allow, "[::1]:4000", http.StatusOK},
		{"outside subnets blocked", allow, "10.20.52.5:4000", http.StatusForbidden},
		{"public ip blocked", allow, "8.8.8.8:4000", http.StatusForbidden},
		{"docker gateway blocked", allow, "172.18.0.1:4000", http.StatusForbidden},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := &Server{cfg: &config.Config{AdminAllowCIDRs: c.cidrs}}
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = c.remote
			rec := httptest.NewRecorder()
			s.adminIPAllow(ok).ServeHTTP(rec, req)
			if rec.Code != c.want {
				t.Errorf("remote %s: got %d, want %d", c.remote, rec.Code, c.want)
			}
		})
	}
}
