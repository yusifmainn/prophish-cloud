// Package report renders a client-facing campaign report as a PDF, entirely in
// pure Go (no headless browser, no external binary), so it works inside the
// minimal distroless container. Fonts are embedded so Azerbaijani characters
// (ə, ğ, ş, ç, ı, ö, ü) render correctly.
package report

import (
	"bytes"
	_ "embed"
	"fmt"
	"time"

	"github.com/go-pdf/fpdf"

	"prophish/internal/models"
)

// DejaVu Sans is embedded for full Azerbaijani/Unicode coverage. It is free and
// redistributable (see fonts/LICENSE).
//
//go:embed fonts/DejaVuSans.ttf
var fontRegular []byte

//go:embed fonts/DejaVuSans-Bold.ttf
var fontBold []byte

// GroupStat mirrors store.GroupStat (kept local so this package has no store dep).
type GroupStat struct {
	Name      string
	Total     int
	Sent      int
	Clicked   int
	Submitted int
	Reported  int
}

// Data is everything the report needs. Counts are absolute; percentages are
// computed here against Total.
type Data struct {
	CampaignName string
	ClientName   string
	AuthScope    string
	AuthValidTo  time.Time
	GeneratedAt  time.Time
	GeneratedBy  string

	Total     int
	Sent      int
	Opened    int
	Clicked   int
	Submitted int
	Reported  int

	Groups []GroupStat
}

// palette — the app's design-system tokens (layout.html :root, light theme),
// kept in exact sync so the client-facing PDF matches the on-screen report.
var (
	colInk     = [3]int{10, 37, 64}
	colMuted   = [3]int{105, 115, 134}
	colLine    = [3]int{230, 235, 241}
	colTrack   = [3]int{238, 243, 248}
	colAccent  = [3]int{99, 91, 255}
	colInfo    = [3]int{10, 111, 214}
	colOK      = [3]int{14, 138, 95}
	colWarn    = [3]int{183, 121, 31}
	colCrit    = [3]int{223, 27, 65}
	colNeutral = [3]int{105, 115, 134}
)

func pct(n, total int) int { return models.Pct(n, total) }

// riskLabel uses the shared models.RiskLevel so the PDF and the on-screen report
// can never drift apart, then maps the status class to a PDF colour.
func riskLabel(d Data) (string, [3]int) {
	label, class := models.RiskLevel(pct(d.Submitted, d.Total), pct(d.Clicked, d.Total))
	switch class {
	case "crit":
		return label, colCrit
	case "warn":
		return label, colWarn
	default:
		return label, colOK
	}
}

// Generate renders the report and returns the PDF bytes.
func Generate(d Data) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 18)
	pdf.AddUTF8FontFromBytes("DejaVu", "", fontRegular)
	pdf.AddUTF8FontFromBytes("DejaVu", "B", fontBold)

	const left = 15.0
	const contentW = 180.0

	// Footer drawn via SetFooterFunc (not inline): fpdf marks it as the footer so
	// it does NOT trigger the auto page break. An inline footer near the bottom
	// margin would spill onto a blank second page on every report.
	footerBy := d.GeneratedBy
	if footerBy == "" {
		footerBy = "—"
	}
	footerText := "PROPhish · Hazırladı: " + footerBy + " · " + d.GeneratedAt.In(time.Local).Format("2006-01-02 15:04")
	pdf.SetFooterFunc(func() {
		pdf.SetY(-13)
		pdf.SetDrawColor(colLine[0], colLine[1], colLine[2])
		pdf.Line(left, pdf.GetY(), left+contentW, pdf.GetY())
		pdf.Ln(1.5)
		pdf.SetFont("DejaVu", "", 8)
		pdf.SetTextColor(colMuted[0], colMuted[1], colMuted[2])
		pdf.CellFormat(contentW, 5, footerText, "", 0, "L", false, 0, "")
	})

	pdf.AddPage()

	setText := func(c [3]int) { pdf.SetTextColor(c[0], c[1], c[2]) }
	setFill := func(c [3]int) { pdf.SetFillColor(c[0], c[1], c[2]) }

	// ---- Brand header ----
	setFill(colAccent)
	pdf.Rect(left, 15, contentW, 14, "F")
	pdf.SetXY(left+4, 18)
	pdf.SetFont("DejaVu", "B", 15)
	pdf.SetTextColor(255, 255, 255)
	pdf.CellFormat(contentW-8, 8, "PROPhish — Phishing Məlumatlandırma Hesabatı", "", 0, "L", false, 0, "")
	pdf.Ln(16)

	// ---- Meta ----
	pdf.SetY(34)
	metaRow := func(label, val string) {
		pdf.SetX(left)
		pdf.SetFont("DejaVu", "B", 10)
		setText(colMuted)
		pdf.CellFormat(34, 6, label, "", 0, "L", false, 0, "")
		pdf.SetFont("DejaVu", "", 10)
		setText(colInk)
		pdf.CellFormat(contentW-34, 6, val, "", 1, "L", false, 0, "")
	}
	metaRow("Kampaniya:", nz(d.CampaignName))
	metaRow("Müştəri:", nz(d.ClientName))
	validTo := "—"
	if !d.AuthValidTo.IsZero() {
		validTo = d.AuthValidTo.In(time.Local).Format("2006-01-02")
	}
	metaRow("İcazə (RoE):", trim(nz(d.AuthScope), 70)+"  (bitmə: "+validTo+")")
	metaRow("Hesabat tarixi:", d.GeneratedAt.In(time.Local).Format("2006-01-02 15:04"))

	pdf.Ln(2)
	setDraw(pdf, colLine)
	y := pdf.GetY()
	pdf.Line(left, y, left+contentW, y)
	pdf.Ln(4)

	// ---- Funnel ----
	sectionTitle(pdf, left, "Davranış icmalı — "+fmt.Sprintf("%d hədəf", d.Total))
	bar := func(label string, n int, c [3]int) {
		p := pct(n, d.Total)
		rowY := pdf.GetY()
		// label
		pdf.SetFont("DejaVu", "", 10)
		setText(colInk)
		pdf.SetXY(left, rowY)
		pdf.CellFormat(40, 7, label, "", 0, "L", false, 0, "")
		// track
		trackX := left + 42
		trackW := contentW - 42 - 26
		setFill(colTrack)
		pdf.Rect(trackX, rowY+1.3, trackW, 4.6, "F")
		// fill
		fw := trackW * float64(p) / 100.0
		if fw > 0 {
			setFill(c)
			pdf.Rect(trackX, rowY+1.3, fw, 4.6, "F")
		}
		// value
		pdf.SetFont("DejaVu", "B", 10)
		setText(colInk)
		pdf.SetXY(trackX+trackW+2, rowY)
		pdf.CellFormat(24, 7, fmt.Sprintf("%d  (%d%%)", n, p), "", 1, "L", false, 0, "")
	}
	bar("Göndərildi", d.Sent, colNeutral)
	bar("Açıldı", d.Opened, colInfo)
	bar("Kliklədi", d.Clicked, colWarn)
	bar("Məlumat verdi", d.Submitted, colCrit)
	pdf.Ln(3)

	// ---- Risk + Reported cards ----
	cardY := pdf.GetY()
	cardW := (contentW - 6) / 2
	// risk card
	risk, riskCol := riskLabel(d)
	setDraw(pdf, colLine)
	pdf.Rect(left, cardY, cardW, 24, "D")
	pdf.SetXY(left+4, cardY+3)
	pdf.SetFont("DejaVu", "", 9)
	setText(colMuted)
	pdf.CellFormat(cardW-8, 5, "Risk balı", "", 2, "L", false, 0, "")
	pdf.SetFont("DejaVu", "B", 16)
	setText(riskCol)
	pdf.CellFormat(cardW-8, 9, risk, "", 2, "L", false, 0, "")
	pdf.SetFont("DejaVu", "", 8)
	setText(colMuted)
	pdf.CellFormat(cardW-8, 5, fmt.Sprintf("Klik %d%% · Məlumat vermə %d%%", pct(d.Clicked, d.Total), pct(d.Submitted, d.Total)), "", 2, "L", false, 0, "")
	// reported card
	rx := left + cardW + 6
	pdf.Rect(rx, cardY, cardW, 24, "D")
	pdf.SetXY(rx+4, cardY+3)
	pdf.SetFont("DejaVu", "", 9)
	setText(colMuted)
	pdf.CellFormat(cardW-8, 5, "Phishing kimi bildirdi (müsbət)", "", 2, "L", false, 0, "")
	pdf.SetFont("DejaVu", "B", 16)
	setText(colOK)
	pdf.SetX(rx + 4)
	pdf.CellFormat(cardW-8, 9, fmt.Sprintf("%d  (%d%%)", d.Reported, pct(d.Reported, d.Total)), "", 2, "L", false, 0, "")
	pdf.SetX(rx + 4)
	pdf.SetFont("DejaVu", "", 8)
	setText(colMuted)
	pdf.CellFormat(cardW-8, 5, "Şübhəli e-poçtu bildirən işçilər", "", 2, "L", false, 0, "")
	pdf.SetY(cardY + 24)
	pdf.Ln(5)

	// ---- Per-group breakdown ----
	if len(d.Groups) > 0 {
		sectionTitle(pdf, left, "Şöbələr üzrə bölgü")
		// header
		cols := []float64{64, 24, 28, 28, 36}
		headers := []string{"Şöbə", "Hədəf", "Kliklədi", "Məlumat verdi", "Bildirdi"}
		pdf.SetFont("DejaVu", "B", 9)
		setFill(colTrack)
		setText(colInk)
		for i, h := range headers {
			pdf.CellFormat(cols[i], 7, h, "", 0, alignFor(i), true, 0, "")
		}
		pdf.Ln(7)
		pdf.SetFont("DejaVu", "", 9)
		for _, g := range d.Groups {
			setText(colInk)
			pdf.CellFormat(cols[0], 6.5, trim(g.Name, 40), "B", 0, "L", false, 0, "")
			pdf.CellFormat(cols[1], 6.5, fmt.Sprintf("%d", g.Total), "B", 0, "C", false, 0, "")
			pdf.CellFormat(cols[2], 6.5, fmt.Sprintf("%d%%", pct(g.Clicked, g.Total)), "B", 0, "C", false, 0, "")
			cell(pdf, cols[3], fmt.Sprintf("%d%%", pct(g.Submitted, g.Total)), pct(g.Submitted, g.Total) >= 15)
			pdf.CellFormat(cols[4], 6.5, fmt.Sprintf("%d (%d%%)", g.Reported, pct(g.Reported, g.Total)), "B", 1, "C", false, 0, "")
		}
		pdf.Ln(4)
	}

	// ---- Privacy note ----
	setFill([3]int{244, 247, 252})
	noteY := pdf.GetY()
	pdf.Rect(left, noteY, contentW, 14, "F")
	pdf.SetXY(left+4, noteY+2.5)
	pdf.SetFont("DejaVu", "B", 9)
	setText(colOK)
	pdf.CellFormat(contentW-8, 5, "Məxfilik", "", 2, "L", false, 0, "")
	pdf.SetX(left + 4)
	pdf.SetFont("DejaVu", "", 8.5)
	setText(colInk)
	pdf.MultiCell(contentW-8, 4.2, "Bu simulyasiyada işçilərin daxil etdiyi heç bir parol və ya məlumat saxlanılmır — yalnız formanın göndərildiyi faktı qeyd olunur.", "", "L", false)

	// (the footer is rendered by SetFooterFunc on every page)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func sectionTitle(pdf *fpdf.Fpdf, x float64, s string) {
	pdf.SetX(x)
	pdf.SetFont("DejaVu", "B", 12)
	pdf.SetTextColor(colInk[0], colInk[1], colInk[2])
	pdf.CellFormat(180, 8, s, "", 1, "L", false, 0, "")
	pdf.Ln(1)
}

func setDraw(pdf *fpdf.Fpdf, c [3]int) { pdf.SetDrawColor(c[0], c[1], c[2]) }

func cell(pdf *fpdf.Fpdf, w float64, txt string, danger bool) {
	if danger {
		pdf.SetTextColor(colCrit[0], colCrit[1], colCrit[2])
	} else {
		pdf.SetTextColor(colInk[0], colInk[1], colInk[2])
	}
	pdf.CellFormat(w, 6.5, txt, "B", 0, "C", false, 0, "")
	pdf.SetTextColor(colInk[0], colInk[1], colInk[2])
}

func alignFor(i int) string {
	if i == 0 {
		return "L"
	}
	return "C"
}

func nz(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func trim(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
