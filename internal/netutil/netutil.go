// Package netutil detects whether THIS host is directly reachable from the
// public internet, so the campaign form can recommend where the tracking
// listener should live (host listener vs Cloudflare gateway).
//
// Algorithm:
//  1. Fetch the outbound public IP (api.ipify.org, 3s timeout).
//  2. A private/CGNAT outbound address means double NAT — not viable.
//  3. If the public IP is assigned to a LOCAL interface, the host is directly
//     reachable — viable.
//  4. Otherwise we are likely inside a container/VM that cannot see the host
//     NIC: hairpin-probe http://<public-ip>/ (port 80 is published to the
//     public server). ANY HTTP response proves the inbound path — viable.
//  5. Any failure → not viable (Cloudflare gateway recommended).
//
// Results are cached (10min TTL); the first call takes at most ~5s, later
// calls are instant. Callers should prewarm asynchronously at startup.
package netutil

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Result is the detection outcome.
type Result struct {
	PublicIP string // detected outbound public IP ("" when detection failed)
	Viable   bool   // true when hosting the listener directly on this host is viable
}

// BaseURL returns "http://<public-ip>" when a viable public IP was detected,
// else "".
func (r Result) BaseURL() string {
	if r.PublicIP == "" || !r.Viable {
		return ""
	}
	return "http://" + r.PublicIP
}

// Detector performs and caches the viability check.
type Detector struct {
	LookupURL string        // outbound-IP lookup endpoint (default api.ipify.org)
	TTL       time.Duration // cache lifetime (default 10min)
	Client    *http.Client  // lookup client (default 3s timeout)

	// Test hooks (nil = real implementation).
	IfaceIPs func() ([]net.IP, error)                  // local interface addresses
	ProbeFn  func(ctx context.Context, ip string) bool // hairpin inbound probe

	mu   sync.Mutex
	res  Result
	at   time.Time
	have bool
}

// NewDetector builds a Detector with production defaults.
func NewDetector() *Detector {
	return &Detector{
		LookupURL: "https://api.ipify.org",
		TTL:       10 * time.Minute,
		Client:    &http.Client{Timeout: 3 * time.Second},
	}
}

// Detect returns the (cached) viability result.
func (d *Detector) Detect(ctx context.Context) Result {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.have && time.Since(d.at) < d.ttl() {
		return d.res
	}
	d.res = d.detect(ctx)
	d.at = time.Now()
	d.have = true
	return d.res
}

func (d *Detector) ttl() time.Duration {
	if d.TTL != 0 {
		return d.TTL
	}
	return 10 * time.Minute
}

func (d *Detector) lookupURL() string {
	if d.LookupURL != "" {
		return d.LookupURL
	}
	return "https://api.ipify.org"
}

func (d *Detector) detect(ctx context.Context) Result {
	ip := d.fetchPublicIP(ctx)
	if ip == "" {
		return Result{}
	}
	parsed := net.ParseIP(ip)
	if parsed == nil || !isPublicIP(parsed) {
		// Private/CGNAT outbound address — behind NAT, inbound impossible.
		return Result{PublicIP: ip, Viable: false}
	}
	if d.onLocalInterface(parsed) {
		return Result{PublicIP: ip, Viable: true}
	}
	// Container/VM: the host NIC is invisible to us. Hairpin probe: if an HTTP
	// request to our own public IP gets ANY HTTP answer, something (this app's
	// published public server) answers on that address — inbound path works.
	probe := d.ProbeFn
	if probe == nil {
		probe = d.hairpinProbe
	}
	return Result{PublicIP: ip, Viable: probe(ctx, ip)}
}

func (d *Detector) httpClient() *http.Client {
	if d.Client != nil {
		return d.Client
	}
	return &http.Client{Timeout: 3 * time.Second}
}

func (d *Detector) fetchPublicIP(ctx context.Context) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.lookupURL(), nil)
	if err != nil {
		return ""
	}
	resp, err := d.httpClient().Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		return ""
	}
	ip := strings.TrimSpace(string(b))
	if net.ParseIP(ip) == nil {
		return ""
	}
	return ip
}

// isPublicIP reports whether ip is a globally routable unicast address
// (rejects private, loopback, link-local, multicast and CGNAT 100.64/10).
func isPublicIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() || ip.IsPrivate() {
		return false
	}
	if v4 := ip.To4(); v4 != nil && v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127 {
		return false // CGNAT (RFC 6598)
	}
	return true
}

func (d *Detector) onLocalInterface(want net.IP) bool {
	if d.IfaceIPs != nil {
		ips, err := d.IfaceIPs()
		if err != nil {
			return false
		}
		for _, ip := range ips {
			if ip.Equal(want) {
				return true
			}
		}
		return false
	}
	ifs, err := net.Interfaces()
	if err != nil {
		return false
	}
	for _, iface := range ifs {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil && ip.Equal(want) {
				return true
			}
		}
	}
	return false
}

// hairpinProbe issues an HTTP request to http://<ip>/ and reports whether ANY
// HTTP response came back (status code irrelevant — a 404 from our own server
// still proves the inbound path).
func (d *Detector) hairpinProbe(ctx context.Context, ip string) bool {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+ip+"/", nil)
	if err != nil {
		return false
	}
	resp, err := d.httpClient().Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 512))
	return true
}
