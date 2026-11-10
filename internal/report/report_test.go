package report

import (
	"os"
	"testing"
	"time"
)

// TestGeneratePDF renders a report with Azerbaijani data and checks it is a
// valid, non-trivial PDF. Set REPORT_OUT=<path> to also write the PDF to disk
// for visual inspection.
func TestGeneratePDF(t *testing.T) {
	data := Data{
		CampaignName: "Q4 Təhlükəsizlik Məlumatlandırma Testi",
		ClientName:   "Nümunə Bank ASC",
		AuthScope:    "example.com — bütün şöbələr üzrə işçilər",
		AuthValidTo:  time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		GeneratedAt:  time.Date(2026, 11, 9, 17, 0, 0, 0, time.UTC),
		GeneratedBy:  "operator@nümunə.az",
		Total:        120, Sent: 118, Opened: 95, Clicked: 44, Submitted: 19, Reported: 27,
		Groups: []GroupStat{
			{Name: "Mühasibatlıq", Total: 30, Sent: 30, Clicked: 18, Submitted: 9, Reported: 4},
			{Name: "İnformasiya Texnologiyaları", Total: 25, Sent: 25, Clicked: 3, Submitted: 0, Reported: 11},
			{Name: "Satış və Marketinq", Total: 40, Sent: 39, Clicked: 15, Submitted: 7, Reported: 6},
			{Name: "İnsan Resursları", Total: 25, Sent: 24, Clicked: 8, Submitted: 3, Reported: 6},
		},
	}
	b, err := Generate(data)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(b) < 1000 || string(b[:5]) != "%PDF-" {
		t.Fatalf("not a valid PDF: len=%d head=%q", len(b), b[:min(8, len(b))])
	}
	if out := os.Getenv("REPORT_OUT"); out != "" {
		if err := os.WriteFile(out, b, 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
}
