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
	// Gender is the raw stored value ('f'/'m'/'').
	Gender string
	// Salutation is the Azerbaijani address form: "bəy" (m), "xanım" (f) or ""
	// when unknown. Greeting is the full opening line ("Hörmətli Aziz bəy,"),
	// correct for every gender including unknown.
	Salutation string
	Greeting   string
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
	// Ensure the scanner-canary footer link is present even if the author forgot
	// {{.CanaryURL}}. It lands just before </body> as a copyright line: every
	// link extractor (raw href, DOM, visible-text) picks it up, humans don't.
	if data.CanaryURL != "" && !strings.Contains(out, data.CanaryURL) {
		out = injectBeforeBodyClose(out, canarySnippet(data.CanaryURL))
	}
	return out, nil
}

// canarySnippet builds the scanner-canary link as a VISIBLE footer line — a
// copyright notice nobody clicks. This is deliberate: the earlier display:none
// anchor was extracted by the delivery-time crawler (raw href scan) but SKIPPED
// by the open-time scanner this deployment faces, which extracts only visible
// links — so its real-link fetches arrived with NO probe and were counted. A
// footer copyright link is picked up by every extractor (raw href, DOM, visible
// text) and is virtually never clicked by a human, so the probe still means
// "a scanner saw this message" while staying harmless to genuine metrics.
func canarySnippet(url string) string {
	return fmt.Sprintf(`<div style="text-align:center;font-family:'Segoe UI',Arial,sans-serif;font-size:11px;line-height:18px;color:#8a9a95;padding:10px 0 4px;"><a href="%s" style="color:#8a9a95;text-decoration:none;">&#169; 2026 &#183; B&#252;t&#252;n h&#252;quqlar qorunur</a></div>`, url)
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

// pixelTag builds the open-tracking image tag for a given base URL and token.
//
// It is a REAL, VISIBLE banner (600x100), not a 1x1 transparent pixel: the
// classic 1px GIF is invisible, gives the recipient no reason to download
// images, and is the first thing clients and gateways skip or strip. A
// normal-looking banner renders as part of the email design, so the "download
// images" action — the moment we record as an open — actually happens.
// (Clients that block remote images entirely — Outlook's default, many
// corporate policies, Apple Mail Privacy Protection — still never report an
// open; that is inherent to image-based open tracking.)
func pixelTag(baseURL, token string) template.HTML {
	return template.HTML(fmt.Sprintf(`<img src="%s/p/%s.png" alt="" width="600" height="100" border="0" style="display:block;width:600px;max-width:100%%;height:auto;border:0;margin:0 auto;padding:0;">`, baseURL, token))
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

// salutationFor maps a stored gender to the Azerbaijani address form and the
// full greeting line. Accepts the aliases people naturally type in CSVs
// (f/m, xanım/bəy, qadın/kişi); anything else is treated as unknown and gets
// the gender-neutral form.
func salutationFor(gender, firstName string) (salutation, greeting string) {
	switch strings.ToLower(strings.TrimSpace(gender)) {
	case "f", "x", "q", "xanım", "qadın":
		return "xanım", "Hörmətli " + firstName + " xanım,"
	case "m", "b", "bəy", "kişi":
		return "bəy", "Hörmətli " + firstName + " bəy,"
	default:
		return "", "Hörmətli " + firstName + ","
	}
}
