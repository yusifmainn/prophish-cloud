package importer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParseSampleEmails parses every shipped samples/*.eml through ParseEmail
// (the same path the admin "Import Email" feature uses) and asserts each yields
// a usable template: a subject, an HTML body, and the per-target {{.URL}} click
// placeholder preserved intact. This guards the Outlook-tuned .eml files against
// a malformed MIME structure or an encoding that would mangle the placeholders.
func TestParseSampleEmails(t *testing.T) {
	dir := filepath.Join("..", "..", "samples")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("samples dir not available: %v", err)
	}

	var seen int
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".eml") {
			continue
		}
		seen++
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("%s: read: %v", e.Name(), err)
		}
		em, err := ParseEmail(string(raw))
		if err != nil {
			t.Errorf("%s: ParseEmail: %v", e.Name(), err)
			continue
		}
		if strings.TrimSpace(em.Subject) == "" {
			t.Errorf("%s: empty subject", e.Name())
		}
		if strings.TrimSpace(em.HTML) == "" {
			t.Errorf("%s: empty HTML part", e.Name())
		}
		// The click link is what turns a lure into a tracked campaign; a broken
		// transfer-encoding would corrupt the "{{.URL}}" placeholder.
		if !strings.Contains(em.HTML, "{{.URL}}") && !strings.Contains(em.Text, "{{.URL}}") {
			t.Errorf("%s: {{.URL}} placeholder missing after parse", e.Name())
		}
	}
	if seen == 0 {
		t.Skip("no .eml samples found")
	}
}
