package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"secdrill/internal/report"
)

// handleCampaignReportPDF generates and streams a client-facing PDF report for a
// campaign. It is produced in pure Go (no browser/binary), so it works in the
// distroless container, and it deliberately contains only aggregate results —
// never any submitted values.
func (s *Server) handleCampaignReportPDF(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	view, err := s.store.CampaignView(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	// Do NOT swallow these errors: a failed query would otherwise render a
	// client-facing PDF full of authoritative-looking zeros ("0 hədəf", low risk)
	// instead of failing honestly.
	stats, err := s.store.CampaignStats(r.Context(), id)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	groups, err := s.store.CampaignStatsByGroup(r.Context(), id)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	by := ""
	if me, err := s.store.GetUser(r.Context(), s.currentUserID(r)); err == nil && me != nil {
		by = me.Email
	}

	gs := make([]report.GroupStat, 0, len(groups))
	for _, g := range groups {
		gs = append(gs, report.GroupStat{
			Name: g.Name, Total: g.Total, Sent: g.Sent,
			Clicked: g.Clicked, Submitted: g.Submitted, Reported: g.Reported,
		})
	}

	pdfBytes, err := report.Generate(report.Data{
		CampaignName: view.Name,
		ClientName:   view.ClientName,
		AuthScope:    view.AuthScope,
		AuthValidTo:  view.AuthValidTo,
		GeneratedAt:  time.Now(),
		GeneratedBy:  by,
		Total:        stats.Total,
		Sent:         stats.Sent,
		Opened:       stats.Opened,
		Clicked:      stats.Clicked,
		Submitted:    stats.Submitted,
		Reported:     stats.Reported,
		Groups:       gs,
	})
	if err != nil {
		http.Error(w, "could not generate PDF", http.StatusInternalServerError)
		return
	}

	s.audit(r, "campaign.report_pdf", "campaign", id, view.Name)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="report_`+id+`.pdf"`)
	_, _ = w.Write(pdfBytes)
}
