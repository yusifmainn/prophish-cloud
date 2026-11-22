package config

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/hkdf"
)

// Config holds all runtime configuration, loaded from environment variables.
type Config struct {
	DatabaseURL     string
	AdminAddr       string
	PublicAddr      string // listen address for the public landing-page/tracking server
	PublicBaseURL   string // external base URL targets reach, e.g. https://mail-check.example.com
	SessionKey      []byte // 32 bytes, used ONLY for session-cookie signing (scs)
	EncryptionKey   []byte // 32 bytes, derived from SessionKey via HKDF; used ONLY to encrypt SMTP/IMAP secrets
	SecureCookies   bool
	TrustedProxies  []*net.IPNet   // CIDRs whose X-Forwarded-For is trusted on the public listener
	AdminAllowCIDRs []*net.IPNet   // CIDRs allowed to reach the ADMIN panel (empty = no restriction; loopback always allowed)
	Location        *time.Location // display/scheduling time zone (PROPHISH_TIMEZONE), defaults to UTC
	// Scanner-classification window presets for the PUBLIC tracking listener,
	// per delivery route (zero fields = the server's built-in defaults: internal
	// 5s/5s, external 15s/10s). Untrusted campaigns use grace+pair to separate
	// scanner prefetch from human hits; trusted campaigns apply ONLY the grace
	// window, and only to VPN/hosting IPs (delivery-time sandbox render vs.
	// later VPN user). Deployment-dependent: with fast internal delivery, real
	// human opens land inside long windows and get discarded as prefetch —
	// see the incident notes on defInternal/defExternal.
	ScannerWinInternal ScannerWindows
	ScannerWinExternal ScannerWindows
	// InfraCIDRs are extra IP ranges (beyond the built-in Microsoft set) the
	// PUBLIC listener treats as scanner/proxy infrastructure and ignores in
	// trusted delivery mode (PROPHISH_INFRA_CIDRS).
	InfraCIDRs []*net.IPNet
	// InfraASNs are autonomous systems whose announced prefixes are treated as
	// scanner/proxy infrastructure (PROPHISH_INFRA_ASNS). Default: every
	// Microsoft ASN that announces IPv4 prefixes (~2.1k in total). Fetched from
	// RIPEstat at startup and refreshed daily.
	InfraASNs []string
	// HumanCIDRs are networks exempt from the trusted-mode datacenter drop
	// (PROPHISH_HUMAN_CIDRS): a company whose web-filtering egress lives in
	// hosting/VPN space would otherwise have its in-window hits ignored.
	HumanCIDRs []*net.IPNet
	// DatacenterListURL overrides the source of the VPN/hosting range list
	// (PROPHISH_DC_LIST_URL). Default: the X4BNet datacenter list on GitHub.
	DatacenterListURL string
}

// Load reads configuration from the process environment. If a .env file exists in the
// working directory it is loaded first (without overriding already-set env vars).
func Load() (*Config, error) {
	_ = loadDotEnv(".env")

	c := &Config{
		DatabaseURL:     getenv("PROPHISH_DB_URL", ""),
		AdminAddr:       getenv("PROPHISH_ADMIN_ADDR", "127.0.0.1:8443"),
		PublicAddr:      getenv("PROPHISH_PUBLIC_ADDR", "127.0.0.1:8080"),
		PublicBaseURL:   strings.TrimRight(getenv("PROPHISH_PUBLIC_BASE_URL", ""), "/"), // empty → engine falls back to auto-detected host URL
		SecureCookies:   strings.EqualFold(getenv("PROPHISH_SECURE_COOKIES", "false"), "true"),
		TrustedProxies:  parseCIDRs(getenv("PROPHISH_TRUSTED_PROXIES", "")),
		AdminAllowCIDRs: parseCIDRs(getenv("PROPHISH_ADMIN_ALLOW_CIDRS", "")),
	}

	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("PROPHISH_DB_URL is required")
	}

	rawKey := getenv("PROPHISH_SESSION_KEY", "")
	if rawKey == "" || rawKey == "replace_with_random_32_byte_secret" {
		return nil, fmt.Errorf("PROPHISH_SESSION_KEY is required; generate one with: openssl rand -base64 32")
	}
	key, err := decodeKey(rawKey)
	if err != nil {
		return nil, fmt.Errorf("invalid PROPHISH_SESSION_KEY: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("PROPHISH_SESSION_KEY must decode to exactly 32 bytes, got %d", len(key))
	}
	c.SessionKey = key

	// Derive a SEPARATE 32-byte key for encrypting stored credentials, so the
	// cookie-signing key and the credential-encryption key are cryptographically
	// independent even though they come from one configured secret (key separation).
	c.EncryptionKey = deriveKey(key, "prophish-credential-encryption-v1")

	// Time zone for scheduling input and timestamp display. Defaults to UTC so
	// behaviour is deterministic; set PROPHISH_TIMEZONE=Asia/Baku (for example)
	// so a campaign scheduled for "09:00" sends at 09:00 local time, not 09:00 UTC.
	// The tz database is embedded in the binary (see cmd/prophish), so named zones
	// resolve even in the minimal distroless container.
	tzName := getenv("PROPHISH_TIMEZONE", "UTC")
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return nil, fmt.Errorf("invalid PROPHISH_TIMEZONE %q: %w", tzName, err)
	}
	c.Location = loc

	c.ScannerWinInternal = ScannerWindows{
		Grace: secondsEnv("PROPHISH_SCANNER_GRACE_INTERNAL_SECONDS"),
		Pair:  secondsEnv("PROPHISH_SCANNER_PAIR_INTERNAL_SECONDS"),
	}
	c.ScannerWinExternal = ScannerWindows{
		Grace: secondsEnv("PROPHISH_SCANNER_GRACE_EXTERNAL_SECONDS"),
		Pair:  secondsEnv("PROPHISH_SCANNER_PAIR_EXTERNAL_SECONDS"),
	}
	c.InfraCIDRs = parseCIDRs(getenv("PROPHISH_INFRA_CIDRS", ""))
	c.InfraASNs = parseASNs(getenv("PROPHISH_INFRA_ASNS", "8075"))
	c.HumanCIDRs = parseCIDRs(getenv("PROPHISH_HUMAN_CIDRS", ""))
	c.DatacenterListURL = strings.TrimSpace(getenv("PROPHISH_DC_LIST_URL", ""))

	return c, nil
}

// ScannerWindows mirrors the public server's per-route classification preset.
type ScannerWindows struct {
	Grace time.Duration
	Pair  time.Duration
}

// secondsEnv parses a whole-seconds env var into a Duration; missing or
// unparseable values yield 0, meaning "use the component's built-in default".
func secondsEnv(k string) time.Duration {
	n, err := strconv.Atoi(strings.TrimSpace(os.Getenv(k)))
	if err != nil || n <= 0 {
		return 0
	}
	return time.Duration(n) * time.Second
}

// parseASNs parses a comma-separated ASN list (numbers or "AS8075" forms),
// skipping empties and junk. Empty input yields the built-in default: every
// Microsoft autonomous system that actually announces IPv4 prefixes today
// (verified against RIPEstat — AS8075 is the big one at ~1.2k prefixes, but
// AS8069 alone adds ~800 more; the rest of Microsoft's ~40 registered ASNs
// announce nothing and are excluded so startup does pointless fetches).
func parseASNs(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(strings.TrimPrefix(part, "AS"))
		part = strings.TrimSpace(strings.TrimPrefix(part, "as"))
		if part == "" {
			continue
		}
		if _, err := strconv.Atoi(part); err == nil {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		out = []string{"8075", "8069", "8068", "3598", "12076", "17144", "46182", "63314", "5761", "54396"}
	}
	return out
}

// parseCIDRs parses a comma-separated list of CIDRs (ignoring unparseable ones).
func parseCIDRs(s string) []*net.IPNet {
	var out []*net.IPNet
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, n, err := net.ParseCIDR(part); err == nil {
			out = append(out, n)
		}
	}
	return out
}

// deriveKey derives a labelled 32-byte subkey from a master secret via HKDF-SHA256.
func deriveKey(master []byte, label string) []byte {
	r := hkdf.New(sha256.New, master, nil, []byte(label))
	out := make([]byte, 32)
	_, _ = io.ReadFull(r, out)
	return out
}

// decodeKey accepts the key as base64 or hex only. A raw-string fallback is
// intentionally NOT provided: it would let a low-entropy passphrase become the
// AES key that protects stored SMTP/IMAP credentials.
func decodeKey(s string) ([]byte, error) {
	if b, err := base64.StdEncoding.DecodeString(s); err == nil && len(b) == 32 {
		return b, nil
	}
	if b, err := hex.DecodeString(s); err == nil && len(b) == 32 {
		return b, nil
	}
	return nil, fmt.Errorf("must be base64- or hex-encoded 32 random bytes; generate with: openssl rand -base64 32")
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// loadDotEnv loads KEY=VALUE pairs from a file into the environment if not already set.
func loadDotEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, val)
		}
	}
	return sc.Err()
}
