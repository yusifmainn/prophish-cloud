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

	conn, err := net.DialTimeout("tcp", addr, 15*time.Second)
	if err != nil {
		return fmt.Errorf("dial smtp: %w", err)
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

	if ok, _ := c.Extension("STARTTLS"); ok {
		tlsCfg := &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12}
		if err := c.StartTLS(tlsCfg); err != nil {
			return fmt.Errorf("starttls: %w", err)
		}
	}

	if cfg.Username != "" {
		if ok, mechs := c.Extension("AUTH"); ok {
			auth, err := pickAuth(mechs, cfg)
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
// It prefers PLAIN (the net/smtp default, which Gmail and most servers accept)
// and falls back to LOGIN for servers that advertise only LOGIN — notably
// Microsoft 365 / Office 365, which rejects PLAIN with "504 5.7.4 Unrecognized
// authentication type".
func pickAuth(mechs string, cfg SMTPConfig) (smtp.Auth, error) {
	up := " " + strings.ToUpper(mechs) + " "
	switch {
	case strings.Contains(up, " PLAIN "):
		return smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host), nil
	case strings.Contains(up, " LOGIN "):
		return &loginAuth{username: cfg.Username, password: cfg.Password, host: cfg.Host}, nil
	default:
		return nil, fmt.Errorf("smtp auth: server offers no supported mechanism (has %q, need PLAIN or LOGIN)", mechs)
	}
}

// loginAuth implements the SMTP AUTH LOGIN mechanism. net/smtp ships only
// PlainAuth, but Microsoft 365 / Office 365 advertises LOGIN (not PLAIN), so we
// add it here. Credentials are base64-encoded (not hashed), so — exactly like
// PlainAuth — this is permitted only over a TLS connection.
type loginAuth struct {
	username, password, host string
	step                     int
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	if !server.TLS {
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
