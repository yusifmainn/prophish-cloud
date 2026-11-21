package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"prophish/internal/store"
)

// handleHitPromote lets an operator reclassify a suppressed tracking hit
// (outcome suspected/ignored by the scanner filter) as a real interaction:
// the result advances and the corresponding event is recorded with the hit's
// ORIGINAL IP/User-Agent fingerprint. This is the manual escape hatch for the
// rare case where the canary correlation misattributes a genuine user
// click/open — the automatic time windows stay untouched.
func (s *Server) handleHitPromote(w http.ResponseWriter, r *http.Request) {
	campID := chi.URLParam(r, "id")
	hitID := chi.URLParam(r, "hitID")
	hit, err := s.store.GetHit(r.Context(), hitID)
	if err != nil || hit.CampaignID != campID {
		s.flashErr(r, "Hit tapılmadı.")
		http.Redirect(w, r, "/campaigns/"+campID, http.StatusSeeOther)
		return
	}
	// Map the hit's route onto the result lifecycle stage it represents. A
	// canary probe is a scanner fact, not a user interaction — nothing to promote.
	var status, eventType string
	switch hit.Route {
	case "open":
		status, eventType = "opened", "email_opened"
	case "click":
		status, eventType = "clicked", "link_clicked"
	case "submit":
		status, eventType = "submitted", "data_submitted"
	default:
		s.flashErr(r, "Canary probe real istifadəçi hadisəsi sayıla bilməz.")
		http.Redirect(w, r, "/campaigns/"+campID, http.StatusSeeOther)
		return
	}
	if hit.Outcome == store.HitCounted || hit.Outcome == store.HitDuplicate {
		s.flashOK(r, "Bu hit artıq sayılıb — əlavə addım lazım deyil.")
		http.Redirect(w, r, "/campaigns/"+campID, http.StatusSeeOther)
		return
	}
	rid := hit.ResultID
	if changed, _ := s.store.AdvanceStatus(r.Context(), rid, status); changed {
		_ = s.store.RecordEvent(r.Context(), campID, &rid, eventType, hit.IP, hit.UserAgent)
		s.flashOK(r, "Hit real hadisə kimi sayıldı.")
	} else {
		s.flashOK(r, "Nəticənin statusu artıq bu mərhələdə və ya daha irəlidədir.")
	}
	_ = s.store.MarkHitPromoted(r.Context(), hitID)
	s.audit(r, "hit.promote", "campaign", campID, hit.Route+" hit "+hitID[:8])
	http.Redirect(w, r, "/campaigns/"+campID, http.StatusSeeOther)
}
