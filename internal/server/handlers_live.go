package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// campaignSnapshot is the live-reload fingerprint of the campaign detail page:
// everything that screen shows which can change WITHOUT operator action —
// status, result counters, and the event / raw-hit counts. Deliberately
// EXCLUDES the sent/total counters: while a campaign is running they change
// every message and are updated in place by the running-banner poll; including
// them here would force a full page reload every tick of a drip send.
type campaignSnapshot struct {
	Status    string `json:"s"`
	Opened    int    `json:"o"`
	Clicked   int    `json:"c"`
	Submitted int    `json:"u"`
	Reported  int    `json:"r"`
	Events    int64  `json:"e"`
	Hits      int64  `json:"h"`
}

// buildCampaignSnapshot marshals the page's current fingerprint. Shared by the
// detail handler (embedded into data-snapshot for the first comparison) and the
// snapshot endpoint (polled by the browser), so both produce byte-identical
// JSON as long as nothing changed.
func (s *Server) buildCampaignSnapshot(r *http.Request, id string) ([]byte, error) {
	camp, err := s.store.GetCampaign(r.Context(), id)
	if err != nil {
		return nil, err
	}
	stats, err := s.store.CampaignStats(r.Context(), id)
	if err != nil {
		return nil, err
	}
	ev, err := s.store.CountEvents(r.Context(), id)
	if err != nil {
		return nil, err
	}
	hits, err := s.store.CountHits(r.Context(), id)
	if err != nil {
		return nil, err
	}
	return json.Marshal(campaignSnapshot{
		Status: camp.Status, Opened: stats.Opened, Clicked: stats.Clicked,
		Submitted: stats.Submitted, Reported: stats.Reported, Events: ev, Hits: hits,
	})
}

// handleCampaignSnapshot serves the fingerprint the detail page's live reload
// polls; the page reloads only when the text differs from what it rendered with.
func (s *Server) handleCampaignSnapshot(w http.ResponseWriter, r *http.Request) {
	snap, err := s.buildCampaignSnapshot(r, chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(snap)
}
