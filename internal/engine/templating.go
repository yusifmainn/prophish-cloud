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
	return out, nil
}

func injectBeforeBodyClose(html, snippet string) string {
	idx := strings.LastIndex(strings.ToLower(html), "</body>")
	if idx == -1 {
		return html + snippet
	}
	return html[:idx] + snippet + html[idx:]
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
