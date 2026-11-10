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

// pixelTag builds the tracking pixel img tag for a given base URL and token.
func pixelTag(baseURL, token string) template.HTML {
	return template.HTML(fmt.Sprintf(`<img src="%s/p/%s.png" width="1" height="1" alt="" style="display:none">`, baseURL, token))
}

// clickURL builds the per-target landing-page link.
func clickURL(baseURL, token string) string {
	return fmt.Sprintf("%s/t/%s", baseURL, token)
}
