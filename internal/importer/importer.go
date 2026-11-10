// Package importer clones a web page into a landing page and parses raw emails
// into templates — the equivalents of Gophish's "Import Site" and "Import Email".
package importer

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"syscall"
	"time"

	"golang.org/x/net/html"
)

const maxFetch = 6 << 20 // 6 MB

// errBlockedIP is returned when a clone target resolves to a non-public address.
var errBlockedIP = errors.New("blocked: target resolves to a private/loopback/link-local/metadata address")

// isBlockedIP reports whether an IP must never be fetched server-side (SSRF guard):
// loopback, link-local (incl. 169.254.0.0/16 cloud metadata), private/ULA,
// multicast, or the unspecified address.
func isBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified() || ip.IsPrivate() {
		return true
	}
	// Explicit cloud metadata address (also covered by link-local, kept for clarity).
	if ip.Equal(net.ParseIP("169.254.169.254")) {
		return true
	}
	return false
}

// ssrfSafeClient builds an HTTP client whose dialer re-checks the ACTUAL dialed
// IP (defeating DNS rebinding) and which refuses redirects to blocked hosts.
func ssrfSafeClient() *http.Client {
	control := func(network, address string, _ syscall.RawConn) error {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return err
		}
		ip := net.ParseIP(host)
		if isBlockedIP(ip) {
			return errBlockedIP
		}
		return nil
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second, Control: control}
	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
	}
	return &http.Client{
		Timeout:   20 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("too many redirects")
			}
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return fmt.Errorf("blocked redirect scheme: %s", req.URL.Scheme)
			}
			return nil // the dialer re-validates the resolved IP on every hop
		},
	}
}

// CloneSite fetches a URL and returns HTML suitable for a landing page: form
// actions are rewritten to {{.SubmitURL}} and relative resource links are made
// absolute so the page renders standalone. No credentials are ever captured —
// the form only triggers a submission event.
func CloneSite(rawURL string) (string, error) {
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}
	base, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid url: %w", err)
	}

	client := ssrfSafeClient()
	req, _ := http.NewRequest("GET", rawURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; SecDrill/1.0)")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	doc, err := html.Parse(io.LimitReader(resp.Body, maxFetch))
	if err != nil {
		return "", fmt.Errorf("parse html: %w", err)
	}

	rewrite(doc, base)

	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func rewrite(n *html.Node, base *url.URL) {
	if n.Type == html.ElementNode {
		switch n.Data {
		case "form":
			setAttr(n, "method", "post")
			setAttr(n, "action", "{{.SubmitURL}}")
		case "a", "link":
			absolutize(n, "href", base)
		case "img", "script":
			absolutize(n, "src", base)
		case "base":
			// Drop <base> so our absolutized URLs win.
			removeAttr(n, "href")
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		rewrite(c, base)
	}
}

func absolutize(n *html.Node, attr string, base *url.URL) {
	for i, a := range n.Attr {
		if a.Key != attr {
			continue
		}
		v := strings.TrimSpace(a.Val)
		if v == "" || strings.HasPrefix(v, "data:") || strings.HasPrefix(v, "#") ||
			strings.HasPrefix(v, "mailto:") || strings.HasPrefix(v, "javascript:") {
			return
		}
		if ref, err := base.Parse(v); err == nil {
			n.Attr[i].Val = ref.String()
		}
		return
	}
}

func setAttr(n *html.Node, key, val string) {
	for i, a := range n.Attr {
		if a.Key == key {
			n.Attr[i].Val = val
			return
		}
	}
	n.Attr = append(n.Attr, html.Attribute{Key: key, Val: val})
}

func removeAttr(n *html.Node, key string) {
	out := n.Attr[:0]
	for _, a := range n.Attr {
		if a.Key != key {
			out = append(out, a)
		}
	}
	n.Attr = out
}

// Email is the parsed result of a raw .eml message.
type Email struct {
	Subject string
	HTML    string
	Text    string
}

// ParseEmail parses a raw RFC 822 message (as exported from a mail client) into
// a subject and HTML/text bodies for use as a template.
func ParseEmail(raw string) (*Email, error) {
	msg, err := mail.ReadMessage(strings.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("parse email: %w", err)
	}
	out := &Email{Subject: decodeWord(msg.Header.Get("Subject"))}

	mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil {
		// Treat as plain text.
		b, _ := io.ReadAll(msg.Body)
		out.Text = string(b)
		return out, nil
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		mr := multipart.NewReader(msg.Body, params["boundary"])
		for {
			part, perr := mr.NextPart()
			if perr == io.EOF {
				break
			}
			if perr != nil {
				break
			}
			ct, _, _ := mime.ParseMediaType(part.Header.Get("Content-Type"))
			body := decodeBody(part, part.Header.Get("Content-Transfer-Encoding"))
			switch {
			case strings.HasPrefix(ct, "text/html"):
				out.HTML = body
			case strings.HasPrefix(ct, "text/plain"):
				out.Text = body
			}
		}
	} else {
		b, _ := io.ReadAll(msg.Body)
		body := string(b)
		if strings.HasPrefix(mediaType, "text/html") {
			out.HTML = body
		} else {
			out.Text = body
		}
	}
	return out, nil
}

func decodeBody(r io.Reader, encoding string) string {
	b, _ := io.ReadAll(r)
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "base64":
		if dec, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(string(b), "\n", "")); err == nil {
			return string(dec)
		}
	case "quoted-printable":
		if dec, err := io.ReadAll(quotedprintable.NewReader(bytes.NewReader(b))); err == nil {
			return string(dec)
		}
	}
	return string(b)
}

func decodeWord(s string) string {
	dec := new(mime.WordDecoder)
	if out, err := dec.DecodeHeader(s); err == nil {
		return out
	}
	return s
}
