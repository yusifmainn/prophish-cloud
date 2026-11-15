package engine

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"os"
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

// Send delivers one message. The transport security depends on the port:
//
//   - Port 465 (SMTPS): the connection is wrapped in TLS from the very first
//     byte (implicit TLS). No STARTTLS handshake takes place.
//   - Port 587 / 25 (and anything else): the connection starts as plaintext and
//     is upgraded with STARTTLS when the server advertises it.
//
// This is a simulation mailer for authorized awareness campaigns; rate limiting
// is handled by the engine.
func Send(cfg SMTPConfig, msg Message) error {
	return sendWithTLS(cfg, msg, nil)
}

// sendWithTLS is the implementation behind Send. tlsCfg lets callers (chiefly
// tests, which use self-signed certificates) override the TLS settings; when
// nil a secure default is used. ServerName and MinVersion are always enforced
// on the config actually used.
func sendWithTLS(cfg SMTPConfig, msg Message, tlsCfg *tls.Config) error {
	// Port 465 speaks implicit TLS: dial with a TLS handshake up front rather
	// than connecting in the clear and upgrading via STARTTLS.
	return sendImpl(cfg, msg, tlsCfg, cfg.Port == 465)
}

// sendImpl is the core send routine. implicitTLS selects implicit TLS (465) vs.
// plaintext+STARTTLS (587/25). It is a separate parameter so tests can exercise
// the implicit-TLS path on an ephemeral port that is not literally 465.
func sendImpl(cfg SMTPConfig, msg Message, tlsCfg *tls.Config, implicitTLS bool) error {
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	if tlsCfg == nil {
		tlsCfg = &tls.Config{}
	} else {
		tlsCfg = tlsCfg.Clone()
	}
	if tlsCfg.ServerName == "" {
		tlsCfg.ServerName = cfg.Host
	}
	if tlsCfg.MinVersion == 0 {
		tlsCfg.MinVersion = tls.VersionTLS12
	}

	var conn net.Conn
	var err error
	if implicitTLS {
		dialer := &net.Dialer{Timeout: 15 * time.Second}
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, tlsCfg)
	} else {
		conn, err = net.DialTimeout("tcp", addr, 15*time.Second)
	}
	if err != nil {
		return fmt.Errorf("dial smtp: %w", err)
	}
	c, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("smtp client: %w", err)
	}
	defer c.Close()

	if err := c.Hello(heloName(cfg.From)); err != nil {
		return err
	}

	// secure tracks whether the channel is encrypted: either implicit TLS (465)
	// or a successful STARTTLS upgrade. Credentials are only sent once secure.
	secure := implicitTLS

	// On a plaintext connection, upgrade with STARTTLS when offered. On an
	// implicit-TLS connection (465) the channel is already encrypted, so this
	// step is skipped.
	if !implicitTLS {
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
// secure reports whether the underlying connection is encrypted. We do not rely
// on net/smtp's own ServerInfo.TLS flag because it is only set after a STARTTLS
// upgrade and stays false on an implicit-TLS (465) connection — so both auth
// mechanisms below consult secure instead, and refuse to send credentials in the
// clear.
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

// plainAuth implements the SMTP AUTH PLAIN mechanism. It mirrors net/smtp's
// PlainAuth but gates on our own secure flag instead of ServerInfo.TLS, so it
// works over an implicit-TLS (465) connection while still refusing to transmit
// credentials over an unencrypted channel.
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
// PlainAuth — this is permitted only over a TLS connection.
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
	altBoundary := randomBoundary("alt_")
	mixBoundary := randomBoundary("mix_")
	hasAttach := len(m.Attachments) > 0

	var b strings.Builder
	// All header values are stripped of CR/LF to prevent header/body injection,
	// and the Subject is RFC 2047-encoded (also forces a single, safe header line).
	fmt.Fprintf(&b, "From: %s\r\n", stripCRLF(from))
	fmt.Fprintf(&b, "To: %s\r\n", stripCRLF(m.To))
	fmt.Fprintf(&b, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", m.Subject))
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	// A Message-ID is set unless the caller supplied one via custom headers;
	// every normal client includes one, and its absence looks like spam.
	if !hasHeader(m.Headers, "Message-ID") {
		fmt.Fprintf(&b, "Message-ID: %s\r\n", newMessageID(from))
	}
	// A List-Unsubscribe header is a recognized "legitimate bulk sender" signal
	// that lowers spam scoring; point it at the sender's own address unless the
	// caller supplied one.
	if !hasHeader(m.Headers, "List-Unsubscribe") {
		if ua := addrOnly(from); strings.Contains(ua, "@") {
			fmt.Fprintf(&b, "List-Unsubscribe: <mailto:%s?subject=unsubscribe>\r\n", stripCRLF(ua))
		}
	}
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

	// Alternative part (text + html). Always include a plain-text part: an
	// HTML-only message is a notable spam signal, so derive text from the HTML
	// when the caller supplied none.
	textBody := m.TextBody
	if strings.TrimSpace(textBody) == "" {
		textBody = htmlToText(m.HTMLBody)
	}
	fmt.Fprintf(&b, "Content-Type: multipart/alternative; boundary=%q\r\n\r\n", altBoundary)
	if strings.TrimSpace(textBody) != "" {
		fmt.Fprintf(&b, "--%s\r\n", altBoundary)
		b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		b.WriteString(textBody)
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

// htmlToText produces a readable plain-text approximation of an HTML body, used
// as the text/plain alternative when the caller provides none. It is intentionally
// simple: drop script/style, turn block-level tags into line breaks, strip the
// remaining tags, unescape entities, and collapse excess whitespace.
func htmlToText(html string) string {
	s := html
	// Remove <script>…</script> and <style>…</style> blocks wholesale.
	for _, tag := range []string{"script", "style", "head", "title"} {
		s = removeElement(s, tag)
	}
	// Block-level boundaries become newlines so words don't run together.
	repl := strings.NewReplacer(
		"</p>", "\n", "<br>", "\n", "<br/>", "\n", "<br />", "\n",
		"</div>", "\n", "</tr>", "\n", "</li>", "\n", "</h1>", "\n",
		"</h2>", "\n", "</h3>", "\n", "</h4>", "\n",
	)
	s = repl.Replace(s)
	// Strip any remaining tags.
	var out strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			out.WriteRune(r)
		}
	}
	text := html5Unescape(out.String())
	// Collapse runs of blank space/lines.
	lines := strings.Split(text, "\n")
	var kept []string
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln != "" {
			kept = append(kept, ln)
		}
	}
	return strings.Join(kept, "\n")
}

// removeElement strips <tag …>…</tag> (case-insensitive) from s.
func removeElement(s, tag string) string {
	lower := strings.ToLower(s)
	open := "<" + tag
	close := "</" + tag + ">"
	for {
		i := strings.Index(lower, open)
		if i < 0 {
			return s
		}
		j := strings.Index(lower[i:], close)
		if j < 0 {
			// No closing tag: drop from the opening tag to the end.
			return s[:i]
		}
		end := i + j + len(close)
		s = s[:i] + s[end:]
		lower = strings.ToLower(s)
	}
}

// html5Unescape decodes the handful of entities common in email HTML.
func html5Unescape(s string) string {
	return strings.NewReplacer(
		"&nbsp;", " ", "&amp;", "&", "&lt;", "<", "&gt;", ">",
		"&quot;", "\"", "&#39;", "'", "&apos;", "'", "&mdash;", "—", "&ndash;", "–",
	).Replace(s)
}

// stripCRLF removes carriage returns and newlines so a value written into an
// email header cannot smuggle additional headers or split the message body.
func stripCRLF(s string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(s)
}

// hasHeader reports whether the caller already supplied a header with the given
// name (case-insensitive), so we don't emit a duplicate.
func hasHeader(headers map[string]string, name string) bool {
	want := strings.ToLower(strings.TrimSpace(name))
	for k := range headers {
		if strings.ToLower(strings.TrimSpace(k)) == want {
			return true
		}
	}
	return false
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

// heloName returns the hostname to present in the EHLO/HELO greeting. A real
// mail client greets with a fully-qualified domain name; greeting as
// "localhost" is a strong spam signal that many servers (Exchange/Outlook
// included) penalize or reject. We use the sender's own domain, which matches
// what a legitimate client for that address would send, falling back to the OS
// hostname and finally "localhost" only if nothing better is available.
func heloName(from string) string {
	if d := addrDomain(from); d != "" {
		return d
	}
	if h, err := os.Hostname(); err == nil && h != "" && h != "localhost" {
		return h
	}
	return "localhost"
}

// addrDomain extracts the domain part of an email address, tolerating a
// "Name <user@host>" display form.
func addrDomain(addr string) string {
	a := addrOnly(addr)
	if i := strings.LastIndex(a, "@"); i != -1 && i+1 < len(a) {
		return strings.TrimSpace(a[i+1:])
	}
	return ""
}

// newMessageID builds a unique RFC 5322 Message-ID. Every well-behaved mail
// client sets one; its absence is a common spam-filter trigger.
func newMessageID(from string) string {
	domain := addrDomain(from)
	if domain == "" {
		domain = "localhost"
	}
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		// Fall back to a time-based token; uniqueness is best-effort.
		return fmt.Sprintf("<%d@%s>", time.Now().UnixNano(), domain)
	}
	return fmt.Sprintf("<%s@%s>", hex.EncodeToString(buf[:]), domain)
}

// randomBoundary returns an unpredictable MIME boundary. Static, tool-specific
// boundary strings fingerprint the sender and can be matched by spam rules.
func randomBoundary(prefix string) string {
	var buf [12]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return prefix + strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return prefix + hex.EncodeToString(buf[:])
}
