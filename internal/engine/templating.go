package engine

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	ttemplate "text/template"
)

// TargetData is the variable set available to email templates and landing pages.
type TargetData struct {
	FirstName string
	LastName  string
	Email     string
	Position  string
	// URL is the per-target phishing link (landing page with the tracking token).
	URL string
	// CanaryURL is the hidden scanner-canary link. renderHTML injects it in a
	// display:none block at the very start of the body when a template omits
	// {{.CanaryURL}}; humans never see it, mail-security scanners fetch it.
	CanaryURL string
	// TrackingPixel is an <img> that registers an "opened" event; inject it in HTML emails.
	TrackingPixel template.HTML
}

// renderText applies a text/template (used for the subject line and plaintext body).
func renderText(tmpl string, data TargetData) (string, error) {
	t, err := ttemplate.New("t").Option("missingkey=zero").Parse(tmpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// renderHTML applies an html/template (auto-escapes variables for safety).
func renderHTML(tmpl string, data TargetData) (string, error) {
	t, err := template.New("h").Option("missingkey=zero").Parse(tmpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	out := buf.String()
	// Ensure the open-tracking pixel is present even if the author forgot {{.TrackingPixel}}.
	if data.TrackingPixel != "" && !strings.Contains(out, string(data.TrackingPixel)) {
		out = injectBeforeBodyClose(out, string(data.TrackingPixel))
	}
	// Ensure the hidden scanner-canary link is present even if the author forgot
	// {{.CanaryURL}}. It goes at the very START of the body so crawlers that
	// fetch links in document order hit it BEFORE the real tracking link and the
	// scan is marked before its "click" arrives.
	if data.CanaryURL != "" && !strings.Contains(out, data.CanaryURL) {
		out = injectAfterBodyOpen(out, canarySnippet(data.CanaryURL))
	}
	return out, nil
}

// canarySnippet builds the hidden canary anchor. Deliberately not wrapped in
// display:none on the anchor itself (some link checkers skip those): the outer
// div hides it visually while the href stays in the raw HTML for every
// extractor, rewriter, and DOM crawler to find.
func canarySnippet(url string) string {
	return fmt.Sprintf(`<div style="display:none;font-size:0;line-height:0;max-height:0;overflow:hidden;mso-hide:all"><a href="%s">&#160;</a></div>`, url)
}

// injectAfterBodyOpen inserts snippet right after the opening <body...> tag.
// The search is case-insensitive and done on the ORIGINAL string byte-by-byte
// (not via strings.ToLower, which shifts byte lengths for some runes and would
// corrupt the splice point — see lastIndexBodyClose).
func injectAfterBodyOpen(html, snippet string) string {
	const open = "<body"
	for i := 0; i+len(open) <= len(html); i++ {
		if strings.EqualFold(html[i:i+len(open)], open) {
			// Find the end of the tag (the closing '>' of <body ...>).
			for j := i + len(open); j < len(html); j++ {
				if html[j] == '>' {
					return html[:j+1] + snippet + html[j+1:]
				}
			}
			return html + snippet // unterminated tag: append instead
		}
	}
	return snippet + html // no <body>: prepend
}

func injectBeforeBodyClose(html, snippet string) string {
	idx := lastIndexBodyClose(html)
	if idx == -1 {
		return html + snippet
	}
	return html[:idx] + snippet + html[idx:]
}

// lastIndexBodyClose returns the index of the last "</body>" (matched
// case-insensitively) in s.
//
// The index must be computed against the ORIGINAL string. The previous
// implementation searched strings.ToLower(html) and then sliced html at that
// offset; ToLower changes byte lengths (e.g. the Azerbaijani/Turkish İ shrinks
// from 2 bytes to 1), so every such rune before </body> shifted the splice
// point and the tracking pixel landed inside an unrelated tag — leaving stray
// text like "able>" visible in the received email.
func lastIndexBodyClose(s string) int {
	const needle = "</body>"
	for i := len(s) - len(needle); i >= 0; i-- {
		if strings.EqualFold(s[i:i+len(needle)], needle) {
			return i
		}
	}
	return -1
}

// pixelTag builds the open-tracking pixel img tag for a given base URL and token.
//
// The pixel is a visible 1x1 image, deliberately NOT hidden with
// display:none / visibility:hidden / opacity:0: several mail clients (notably
// Outlook) skip downloading images that are hidden that way, which would stop
// the "opened" event from ever firing. A 1x1 transparent image placed at the
// end of the body is effectively invisible to the recipient while still being
// requested by every client that loads remote images. (Clients that block
// remote images entirely — Outlook's default, many corporate policies, Apple
// Mail Privacy Protection — will still never report an open; that is inherent
// to pixel-based open tracking, not something markup can fix.)
func pixelTag(baseURL, token string) template.HTML {
	return template.HTML(fmt.Sprintf(`<img src="%s/p/%s.png" alt="" width="1" height="1" border="0" style="width:1px;height:1px;border:0;margin:0;padding:0;overflow:hidden;line-height:1px;">`, baseURL, token))
}

// clickURL builds the per-target landing-page link.
func clickURL(baseURL, token string) string {
	return fmt.Sprintf("%s/t/%s", baseURL, token)
}

// canaryURL builds the per-target hidden scanner-canary link. A hit on it never
// counts as a click — it marks the recipient as scanner-probed so the real
// link's hits inside the cluster window can be attributed to the scanner.
func canaryURL(baseURL, token string) string {
	return fmt.Sprintf("%s/c/%s", baseURL, token)
}
