package engine

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"prophish/internal/models"
)

// SMTPConfig describes how to connect to a sending server.
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	// UseSSL selects implicit TLS (SMTPS, typically port 465): TLS is negotiated
	// as soon as the TCP connection opens, before any SMTP command. When false,
	// Send connects in plaintext and upgrades with STARTTLS when the server offers
	// it (ports 587/25).
	UseSSL bool
}

// Message is a single outbound email.
type Message struct {
	To          string
	Subject     string
	HTMLBody    string
	TextBody    string
	Headers     map[string]string
	Attachments []models.Attachment
}

// Send delivers one message using STARTTLS when offered. This is a simulation
// mailer for authorized awareness campaigns; rate limiting is handled by the engine.
func Send(cfg SMTPConfig, msg Message) error {
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	tlsCfg := &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12}

	// secure tracks whether the connection is actually encrypted, as WE determine
	// it — not via smtp.Client's internal tls flag, which it does NOT set when we
	// hand it an already-TLS connection (implicit TLS). The AUTH mechanisms below
	// consult this so credentials are only ever sent over an encrypted channel.
	var conn net.Conn
	var err error
	secure := false

	if cfg.UseSSL {
		// Implicit TLS (SMTPS, e.g. port 465): the whole session is wrapped in TLS
		// from the first byte, so we dial with tls.Dial and never issue STARTTLS.
		conn, err = tls.DialWithDialer(&net.Dialer{Timeout: 15 * time.Second}, "tcp", addr, tlsCfg)
		if err != nil {
			return fmt.Errorf("dial smtps: %w", err)
		}
		secure = true
	} else {
		conn, err = net.DialTimeout("tcp", addr, 15*time.Second)
		if err != nil {
			return fmt.Errorf("dial smtp: %w", err)
		}
	}

	c, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("smtp client: %w", err)
	}
	defer c.Close()

	if err := c.Hello(clientHostname()); err != nil {
		return err
	}

	// On a plaintext connection, upgrade with STARTTLS when the server offers it.
	// On an implicit-TLS connection the session is already encrypted, so STARTTLS
	// is neither needed nor advertised.
	if !cfg.UseSSL {
		if ok, _ := c.Extension("STARTTLS"); ok {
			if err := c.StartTLS(tlsCfg); err != nil {
				return fmt.Errorf("starttls: %w", err)
			}
			secure = true
		}
	}

	if cfg.Username != "" {
		if ok, mechs := c.Extension("AUTH"); ok {
			auth, err := pickAuth(mechs, cfg, secure)
			if err != nil {
				return err
			}
			if err := c.Auth(auth); err != nil {
				return fmt.Errorf("smtp auth: %w", err)
			}
		}
	}

	if err := c.Mail(addrOnly(cfg.From)); err != nil {
		return err
	}
	if err := c.Rcpt(msg.To); err != nil {
		return err
	}
	wc, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := wc.Write([]byte(buildMIME(cfg.From, msg))); err != nil {
		wc.Close()
		return err
	}
	if err := wc.Close(); err != nil {
		return err
	}
	return c.Quit()
}

// pickAuth selects an SMTP auth mechanism from the server's advertised list.
// It prefers PLAIN (which Gmail and most servers accept) and falls back to LOGIN
// for servers that advertise only LOGIN — notably Microsoft 365 / Office 365,
// which rejects PLAIN with "504 5.7.4 Unrecognized authentication type".
//
// secure is the caller's own determination of whether the connection is
// encrypted (implicit TLS, or STARTTLS completed). We use our own plainAuth
// rather than smtp.PlainAuth because the latter reads smtp.Client's internal tls
// flag, which is NOT set for a connection handed to smtp.NewClient already wrapped
// in TLS (implicit TLS / port 465) — making it wrongly refuse "unencrypted".
func pickAuth(mechs string, cfg SMTPConfig, secure bool) (smtp.Auth, error) {
	up := " " + strings.ToUpper(mechs) + " "
	switch {
	case strings.Contains(up, " PLAIN "):
		return &plainAuth{username: cfg.Username, password: cfg.Password, host: cfg.Host, secure: secure}, nil
	case strings.Contains(up, " LOGIN "):
		return &loginAuth{username: cfg.Username, password: cfg.Password, host: cfg.Host, secure: secure}, nil
	default:
		return nil, fmt.Errorf("smtp auth: server offers no supported mechanism (has %q, need PLAIN or LOGIN)", mechs)
	}
}

// plainAuth implements AUTH PLAIN. It mirrors smtp.PlainAuth but trusts the
// caller's `secure` determination instead of smtp.Client's internal tls flag,
// so it works over implicit TLS (port 465) while still refusing to send
// credentials over a genuinely unencrypted connection.
type plainAuth struct {
	username, password, host string
	secure                   bool
}

func (a *plainAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	if !a.secure && !server.TLS {
		return "", nil, fmt.Errorf("smtp: refusing AUTH PLAIN on an unencrypted connection to %q", a.host)
	}
	if server.Name != a.host {
		return "", nil, fmt.Errorf("smtp: AUTH PLAIN server name %q does not match %q", server.Name, a.host)
	}
	resp := []byte("\x00" + a.username + "\x00" + a.password)
	return "PLAIN", resp, nil
}

func (a *plainAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		return nil, fmt.Errorf("smtp: unexpected server challenge during AUTH PLAIN")
	}
	return nil, nil
}

// loginAuth implements the SMTP AUTH LOGIN mechanism. net/smtp ships only
// PlainAuth, but Microsoft 365 / Office 365 advertises LOGIN (not PLAIN), so we
// add it here. Credentials are base64-encoded (not hashed), so — exactly like
// PLAIN — this is permitted only over an encrypted connection.
type loginAuth struct {
	username, password, host string
	secure                   bool
	step                     int
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	if !a.secure && !server.TLS {
		return "", nil, fmt.Errorf("smtp: refusing AUTH LOGIN on an unencrypted connection to %q", a.host)
	}
	if server.Name != a.host {
		return "", nil, fmt.Errorf("smtp: AUTH LOGIN server name %q does not match %q", server.Name, a.host)
	}
	return "LOGIN", nil, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	// Reply by challenge order (1 = username, 2 = password) rather than matching
	// the prompt text, which differs across servers.
	a.step++
	switch a.step {
	case 1:
		return []byte(a.username), nil
	case 2:
		return []byte(a.password), nil
	default:
		return nil, fmt.Errorf("smtp: unexpected extra AUTH LOGIN challenge %q", string(fromServer))
	}
}

func buildMIME(from string, m Message) string {
	altBoundary := "prophish_alt_7f3a9c"
	mixBoundary := "prophish_mix_4b1e8d"
	hasAttach := len(m.Attachments) > 0

	var b strings.Builder
	// All header values are stripped of CR/LF to prevent header/body injection,
	// and the Subject is RFC 2047-encoded (also forces a single, safe header line).
	fmt.Fprintf(&b, "From: %s\r\n", stripCRLF(from))
	fmt.Fprintf(&b, "To: %s\r\n", stripCRLF(m.To))
	fmt.Fprintf(&b, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", m.Subject))
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	b.WriteString("MIME-Version: 1.0\r\n")
	// Custom headers (reserved ones are skipped so they can't be overridden;
	// key and value are sanitized so a malicious value cannot inject new headers).
	for k, v := range m.Headers {
		if isReservedHeader(k) {
			continue
		}
		k = stripCRLF(k)
		if k == "" {
			continue
		}
		fmt.Fprintf(&b, "%s: %s\r\n", k, stripCRLF(v))
	}

	if hasAttach {
		fmt.Fprintf(&b, "Content-Type: multipart/mixed; boundary=%q\r\n\r\n", mixBoundary)
		fmt.Fprintf(&b, "--%s\r\n", mixBoundary)
	}

	// Alternative part (text + html).
	fmt.Fprintf(&b, "Content-Type: multipart/alternative; boundary=%q\r\n\r\n", altBoundary)
	if m.TextBody != "" {
		fmt.Fprintf(&b, "--%s\r\n", altBoundary)
		b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		b.WriteString(m.TextBody)
		b.WriteString("\r\n")
	}
	fmt.Fprintf(&b, "--%s\r\n", altBoundary)
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
	b.WriteString(m.HTMLBody)
	b.WriteString("\r\n")
	fmt.Fprintf(&b, "--%s--\r\n", altBoundary)

	if hasAttach {
		for _, a := range m.Attachments {
			fmt.Fprintf(&b, "--%s\r\n", mixBoundary)
			ct := a.ContentType
			if ct == "" {
				ct = "application/octet-stream"
			}
			fmt.Fprintf(&b, "Content-Type: %s\r\n", ct)
			b.WriteString("Content-Transfer-Encoding: base64\r\n")
			fmt.Fprintf(&b, "Content-Disposition: attachment; filename=%q\r\n\r\n", stripCRLF(a.Filename))
			b.WriteString(base64Wrap(a.Content))
			b.WriteString("\r\n")
		}
		fmt.Fprintf(&b, "--%s--\r\n", mixBoundary)
	}
	return b.String()
}

// stripCRLF removes carriage returns and newlines so a value written into an
// email header cannot smuggle additional headers or split the message body.
func stripCRLF(s string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(s)
}

// isReservedHeader guards headers PROPhish sets itself.
func isReservedHeader(k string) bool {
	switch strings.ToLower(strings.TrimSpace(k)) {
	case "from", "to", "subject", "date", "mime-version", "content-type", "content-transfer-encoding":
		return true
	}
	return false
}

// base64Wrap encodes to base64 split into 76-char lines (RFC 2045).
func base64Wrap(data []byte) string {
	enc := base64.StdEncoding.EncodeToString(data)
	var b strings.Builder
	for len(enc) > 76 {
		b.WriteString(enc[:76])
		b.WriteString("\r\n")
		enc = enc[76:]
	}
	b.WriteString(enc)
	return b.String()
}

func addrOnly(a string) string {
	if i := strings.LastIndex(a, "<"); i != -1 {
		if j := strings.LastIndex(a, ">"); j > i {
			return a[i+1 : j]
		}
	}
	return a
}

func clientHostname() string {
	if h, err := net.LookupCNAME("localhost"); err == nil && h != "" {
		return strings.TrimRight(h, ".")
	}
	return "localhost"
}
