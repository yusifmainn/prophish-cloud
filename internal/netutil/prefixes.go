package netutil

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// ripeStatAnnouncedPrefixes is the RIPEstat data call that lists every prefix
// currently announced by an AS. Chosen over static downloads: cloud ASNs
// (Microsoft, Google, ...) add ranges constantly, so any embedded list goes
// stale; this is the live routing truth and needs no API key.
const ripeStatAnnouncedPrefixes = "https://stat.ripe.net/data/announced-prefixes/data.json?resource=AS"

// FetchAnnouncedPrefixes returns the IPv4 prefixes currently announced by the
// given autonomous system (number with or without the "AS" prefix). Used to
// keep the trusted-mode infrastructure list complete — scanner/proxy egress
// (M365 ATP, SafeLinks, image proxies) comes from cloud ASNs whose ranges
// change faster than any hardcoded list.
func FetchAnnouncedPrefixes(ctx context.Context, asn string) ([]*net.IPNet, error) {
	url := ripeStatAnnouncedPrefixes + strings.TrimPrefix(strings.TrimSpace(asn), "AS")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ripestat: HTTP %d", resp.StatusCode)
	}
	var payload struct {
		Data struct {
			Prefixes []struct {
				Prefix string `json:"prefix"`
			} `json:"prefixes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	raw := make([]string, 0, len(payload.Data.Prefixes))
	for _, p := range payload.Data.Prefixes {
		raw = append(raw, p.Prefix)
	}
	return parseIPv4Prefixes(raw)
}

// DefaultDatacenterListURL is the X4BNet VPN-and-datacenter IPv4 list: every
// range classified as "not an eyeball network" (hosting providers, VPN exits,
// cloud/scanner space) rebuilt from ASN data by CI. Plain text, one CIDR per
// line, comments with '#'. MIT-licensed generated output.
const DefaultDatacenterListURL = "https://raw.githubusercontent.com/X4BNet/lists_vpn/main/output/datacenter/ipv4.txt"

// FetchDatacenterRanges downloads a plain-text IPv4 CIDR list (one per line,
// '#' comments allowed) and returns the parsed networks. An empty url falls
// back to DefaultDatacenterListURL. Used for the trusted-mode datacenter
// filter: scanners/security sandboxes egress from hosting/VPN space, never
// from a residential ISP, so a hit from these ranges cannot be the human.
func FetchDatacenterRanges(ctx context.Context, url string) ([]*net.IPNet, error) {
	if strings.TrimSpace(url) == "" {
		url = DefaultDatacenterListURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("datacenter list: HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, err
	}
	var lines []string
	for _, ln := range strings.Split(string(body), "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		lines = append(lines, ln)
	}
	return parseIPv4Prefixes(lines)
}

// parseIPv4Prefixes parses CIDR strings, silently skipping v6 and junk.
func parseIPv4Prefixes(ss []string) ([]*net.IPNet, error) {
	out := make([]*net.IPNet, 0, len(ss))
	for _, s := range ss {
		if strings.Contains(s, ":") { // v6 — the infra check is v4-only for now
			continue
		}
		if _, n, err := net.ParseCIDR(strings.TrimSpace(s)); err == nil {
			out = append(out, n)
		}
	}
	if len(out) == 0 {
		return nil, errors.New("no usable IPv4 prefixes in response")
	}
	return out, nil
}

// infraCacheFile is the on-disk cache for the ASN-fetched prefix list.
type infraCacheFile struct {
	FetchedAt time.Time `json:"fetched_at"`
	Prefixes  []string  `json:"prefixes"`
}

// SavePrefixCache writes the prefix list to disk so a restart does not need a
// fresh RIPEstat round trip.
func SavePrefixCache(path string, nets []*net.IPNet) error {
	f := infraCacheFile{FetchedAt: time.Now()}
	for _, n := range nets {
		f.Prefixes = append(f.Prefixes, n.String())
	}
	b, err := json.Marshal(f)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

// LoadPrefixCache reads a cache written by SavePrefixCache. The second return
// is false when the file is missing, unreadable, unparsable, or older than
// maxAge — callers then treat it as absent and refetch.
func LoadPrefixCache(path string, maxAge time.Duration) ([]*net.IPNet, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var f infraCacheFile
	if json.Unmarshal(b, &f) != nil || len(f.Prefixes) == 0 {
		return nil, false
	}
	if maxAge > 0 && time.Since(f.FetchedAt) > maxAge {
		return nil, false
	}
	nets, err := parseIPv4Prefixes(f.Prefixes)
	if err != nil {
		return nil, false
	}
	return nets, true
}
