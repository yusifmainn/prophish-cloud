package engine

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeSMTP is a minimal SMTP server for tests. It records the AUTH line it
// receives and accepts a single message. When tlsConfig is non-nil and
// implicit is true, it performs a TLS handshake immediately on accept (port
// 465 behaviour); otherwise it starts in plaintext and advertises STARTTLS
// (port 587 behaviour).
type fakeSMTP struct {
	ln        net.Listener
	tlsConfig *tls.Config
	implicit  bool

	mu       sync.Mutex
	authSeen string
	gotData  bool
	tlsUsed  bool
	heloArg  string
	dataBody string
}

func newFakeSMTP(t *testing.T, implicit bool) *fakeSMTP {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	s := &fakeSMTP{ln: ln, tlsConfig: testTLSConfig(t), implicit: implicit}
	go s.serve()
	t.Cleanup(func() { ln.Close() })
	return s
}

func (s *fakeSMTP) hostPort() (string, int) {
	host, portStr, _ := net.SplitHostPort(s.ln.Addr().String())
	port, _ := strconv.Atoi(portStr)
	return host, port
}

func (s *fakeSMTP) serve() {
	conn, err := s.ln.Accept()
	if err != nil {
		return
	}
	defer conn.Close()

	if s.implicit {
		tconn := tls.Server(conn, s.tlsConfig)
		if err := tconn.Handshake(); err != nil {
			return
		}
		s.mu.Lock()
		s.tlsUsed = true
		s.mu.Unlock()
		conn = tconn
	}

	s.session(conn, !s.implicit)
}

// session runs the SMTP dialogue. offerStartTLS controls whether STARTTLS is
// advertised and handled (plaintext/587 path).
func (s *fakeSMTP) session(conn net.Conn, offerStartTLS bool) {
	br := bufio.NewReader(conn)
	w := func(format string, a ...any) { fmt.Fprintf(conn, format, a...) }

	w("220 fake ESMTP\r\n")
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return
		}
		cmd := strings.ToUpper(strings.TrimSpace(line))
		switch {
		case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
			// Record the greeting argument with original case.
			if fields := strings.Fields(strings.TrimSpace(line)); len(fields) > 1 {
				s.mu.Lock()
				if s.heloArg == "" {
					s.heloArg = fields[1]
				}
				s.mu.Unlock()
			}
			w("250-fake greets you\r\n")
			if offerStartTLS {
				w("250-STARTTLS\r\n")
			}
			w("250 AUTH PLAIN LOGIN\r\n")
		case cmd == "STARTTLS":
			w("220 go ahead\r\n")
			tconn := tls.Server(conn, s.tlsConfig)
			if err := tconn.Handshake(); err != nil {
				return
			}
			s.mu.Lock()
			s.tlsUsed = true
			s.mu.Unlock()
			conn = tconn
			br = bufio.NewReader(conn)
			w = func(format string, a ...any) { fmt.Fprintf(conn, format, a...) }
		case strings.HasPrefix(cmd, "AUTH"):
			s.mu.Lock()
			s.authSeen = cmd
			s.mu.Unlock()
			// For both PLAIN (creds inline) and LOGIN (challenge/response) just
			// consume any continuation lines and accept.
			if strings.HasPrefix(cmd, "AUTH LOGIN") {
				w("334 VXNlcm5hbWU6\r\n") // "Username:"
				br.ReadString('\n')
				w("334 UGFzc3dvcmQ6\r\n") // "Password:"
				br.ReadString('\n')
			}
			w("235 2.7.0 Authentication successful\r\n")
		case strings.HasPrefix(cmd, "MAIL FROM"):
			w("250 ok\r\n")
		case strings.HasPrefix(cmd, "RCPT TO"):
			w("250 ok\r\n")
		case cmd == "DATA":
			w("354 end with .\r\n")
			var body strings.Builder
			for {
				dl, err := br.ReadString('\n')
				if err != nil {
					return
				}
				if strings.TrimRight(dl, "\r\n") == "." {
					break
				}
				body.WriteString(dl)
			}
			s.mu.Lock()
			s.dataBody = body.String()
			s.gotData = true
			s.mu.Unlock()
			w("250 queued\r\n")
		case cmd == "QUIT":
			w("221 bye\r\n")
			return
		default:
			w("250 ok\r\n")
		}
	}
}

func testTLSConfig(t *testing.T) *tls.Config {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("cert: %v", err)
	}
	cert := tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
	return &tls.Config{Certificates: []tls.Certificate{cert}}
}

func sendTo(t *testing.T, s *fakeSMTP) error {
	t.Helper()
	host, port := s.hostPort()
	cfg := SMTPConfig{
		Host:     host,
		Port:     port,
		Username: "user@example.com",
		Password: "secret",
		From:     "ProPhish <noreply@example.com>",
	}
	// The test cert is self-signed, so skip verification. The implicit-TLS flag
	// is driven directly (the listener is on an ephemeral port, not 465).
	return sendImpl(cfg, Message{
		To:       "target@example.com",
		Subject:  "Hello",
		HTMLBody: "<p>hi</p>",
		TextBody: "hi",
	}, &tls.Config{InsecureSkipVerify: true}, s.implicit)
}

func TestSendImplicitTLS465(t *testing.T) {
	s := newFakeSMTP(t, true)
	if err := sendTo(t, s); err != nil {
		t.Fatalf("send over implicit TLS: %v", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.tlsUsed {
		t.Error("expected the connection to be TLS-wrapped from the start")
	}
	if !s.gotData {
		t.Error("message body was not delivered")
	}
	if s.authSeen == "" {
		t.Error("expected AUTH to be attempted over implicit TLS")
	}
}

func TestSendStartTLS587(t *testing.T) {
	s := newFakeSMTP(t, false)
	if err := sendTo(t, s); err != nil {
		t.Fatalf("send over STARTTLS: %v", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.tlsUsed {
		t.Error("expected STARTTLS upgrade")
	}
	if !s.gotData {
		t.Error("message body was not delivered")
	}
	if s.authSeen == "" {
		t.Error("expected AUTH after STARTTLS")
	}
}

// TestSendDeliverabilityHeaders verifies the anti-spam improvements: the client
// greets with the sender's domain (never "localhost") and the message carries a
// Message-ID header.
func TestSendDeliverabilityHeaders(t *testing.T) {
	s := newFakeSMTP(t, false)
	if err := sendTo(t, s); err != nil {
		t.Fatalf("send: %v", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.heloArg == "" || strings.EqualFold(s.heloArg, "localhost") {
		t.Errorf("expected EHLO to use the sender domain, got %q", s.heloArg)
	}
	// From is "noreply@example.com", so the greeting should be example.com.
	if !strings.EqualFold(s.heloArg, "example.com") {
		t.Errorf("expected EHLO %q, got %q", "example.com", s.heloArg)
	}
	if !strings.Contains(s.dataBody, "Message-ID:") {
		t.Error("expected a Message-ID header in the message")
	}
	if !strings.Contains(s.dataBody, "@example.com>") {
		t.Errorf("expected Message-ID to use the sender domain; body headers:\n%s", headerBlock(s.dataBody))
	}
}

// headerBlock returns just the header portion of a raw message for error output.
func headerBlock(raw string) string {
	if i := strings.Index(raw, "\r\n\r\n"); i != -1 {
		return raw[:i]
	}
	return raw
}
