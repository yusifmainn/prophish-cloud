package library

import (
	"html/template"
	"strings"
	"testing"
	ttemplate "text/template"
)

// sample mirrors the variable set the engine/public server inject at render time.
type sample struct {
	FirstName, LastName, Email, Position, URL, SubmitURL string
	TrackingPixel                                        template.HTML
}

var langs = []string{"az", "en", "ru"}

func sampleData() sample {
	return sample{
		FirstName: "Aysel", LastName: "Məmmədova", Email: "a@numune.az",
		Position: "Mühasib", URL: "https://example.test/t/x", SubmitURL: "https://example.test/s/x",
		TrackingPixel: template.HTML(`<img src="x">`),
	}
}

// TestCatalogIntegrity asserts every kit is complete, localized, and renders.
func TestCatalogIntegrity(t *testing.T) {
	cat := Catalog()
	if len(cat) == 0 {
		t.Fatal("catalog is empty")
	}
	seen := map[string]bool{}
	for _, k := range cat {
		if k.ID == "" || seen[k.ID] {
			t.Fatalf("missing or duplicate kit ID %q", k.ID)
		}
		seen[k.ID] = true

		for _, lang := range langs {
			if strings.TrimSpace(k.Title[lang]) == "" {
				t.Errorf("kit %s: Title missing %q", k.ID, lang)
			}
			if strings.TrimSpace(k.Summary[lang]) == "" {
				t.Errorf("kit %s: Summary missing %q", k.ID, lang)
			}
		}

		// Landing page: present, privacy-preserving, and a valid html/template.
		lp := k.LandingHTML()
		if lp == "" {
			t.Fatalf("kit %s: empty landing HTML", k.ID)
		}
		if !strings.Contains(lp, "{{.SubmitURL}}") {
			t.Errorf("kit %s: landing form does not post to {{.SubmitURL}}", k.ID)
		}
		if !strings.Contains(strings.ToLower(lp), `method="post"`) {
			t.Errorf("kit %s: landing form is not method=post", k.ID)
		}
		if lt, err := template.New("lp").Option("missingkey=zero").Parse(lp); err != nil {
			t.Errorf("kit %s: landing parse: %v", k.ID, err)
		} else if err := lt.Execute(discard{}, sampleData()); err != nil {
			t.Errorf("kit %s: landing execute: %v", k.ID, err)
		}

		if !k.HasEmail() {
			continue
		}
		for _, lang := range langs {
			if strings.TrimSpace(k.EmailSubject[lang]) == "" {
				t.Errorf("kit %s: EmailSubject missing %q", k.ID, lang)
			}
			// Subjects are rendered with text/template at send time.
			if st, err := ttemplate.New("s").Option("missingkey=zero").Parse(k.EmailSubject[lang]); err != nil {
				t.Errorf("kit %s: subject parse (%s): %v", k.ID, lang, err)
			} else if err := st.Execute(discard{}, sampleData()); err != nil {
				t.Errorf("kit %s: subject execute (%s): %v", k.ID, lang, err)
			}
		}
		eb := k.EmailHTML()
		if !strings.Contains(eb, "{{.URL}}") {
			t.Errorf("kit %s: email has no {{.URL}} click link", k.ID)
		}
		if et, err := template.New("eb").Option("missingkey=zero").Parse(eb); err != nil {
			t.Errorf("kit %s: email parse: %v", k.ID, err)
		} else if err := et.Execute(discard{}, sampleData()); err != nil {
			t.Errorf("kit %s: email execute: %v", k.ID, err)
		}
	}
}

func TestGet(t *testing.T) {
	if _, ok := Get("m365"); !ok {
		t.Error("expected m365 kit")
	}
	if _, ok := Get("does-not-exist"); ok {
		t.Error("unexpected kit for unknown id")
	}
}

func TestPickFallback(t *testing.T) {
	m := map[string]string{"az": "A", "en": "E"}
	if got := Pick(m, "en"); got != "E" {
		t.Errorf("Pick en = %q, want E", got)
	}
	if got := Pick(m, "ru"); got != "A" { // falls back to az
		t.Errorf("Pick ru fallback = %q, want A", got)
	}
	if got := Pick(nil, "az"); got != "" {
		t.Errorf("Pick nil = %q, want empty", got)
	}
}

// discard is an io.Writer that throws template output away.
type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }
