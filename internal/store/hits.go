package store

import (
	"time"

	"github.com/jackc/pgx/v5"
)

// HitOutcome classifies what the public server did with one raw tracking
// request. These values land in tracking_hits.outcome.
const (
	HitCounted   = "counted"   // advanced the result status (first transition)
	HitDuplicate = "duplicate" // human-looking, but the status was already at/past this stage
	HitSuspected = "suspected" // suppressed by the canary pair window, logged visibly
	HitIgnored   = "ignored"   // dropped by the UA filter or the post-send grace window
	HitProbe     = "probe"     // canary hit: marks the scanner probe for the result
)

// RecordHit appends one raw tracking request with its classification decision.
// hitAt is the request's arrival time, captured BEFORE any classification waits
// (the click handler re-reads the canary probe after a 2s sleep) so the row's
// timestamp reflects when the request actually arrived.
func (s *Store) RecordHit(c ctx, resultID, campaignID, route, ip, ua, outcome, reason string, hitAt time.Time) error {
	_, err := s.db.Exec(c,
		`INSERT INTO tracking_hits (result_id, campaign_id, route, ip, user_agent, outcome, reason, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		resultID, campaignID, route, nullIfEmpty(ip), nullIfEmpty(ua), outcome, reason, hitAt)
	return err
}

// HitDetail is one raw tracking request for the campaign detail screen.
type HitDetail struct {
	ID        string
	ResultID  string
	Email     string
	Route     string
	IP        string
	UserAgent string
	Outcome   string
	Reason    string
	Promoted  bool
	CreatedAt time.Time
}

// ListHitsDetailed returns the campaign's raw tracking requests, newest first.
// Unlike ListEventsDetailed nothing is deduplicated: every request that reached
// the public listener is here, including the ones classification discarded.
func (s *Store) ListHitsDetailed(c ctx, campaignID string) ([]HitDetail, error) {
	rows, err := s.db.Query(c,
		`SELECT h.id::text, h.result_id::text, coalesce(t.email,'(naməlum)'), h.route,
		        coalesce(host(h.ip),''), coalesce(h.user_agent,''), h.outcome, h.reason,
		        h.promoted, h.created_at
		   FROM tracking_hits h
		   LEFT JOIN campaign_results r ON r.id = h.result_id
		   LEFT JOIN targets t ON t.id = r.target_id
		  WHERE h.campaign_id=$1
		  ORDER BY h.created_at DESC
		  LIMIT 1000`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []HitDetail
	for rows.Next() {
		var d HitDetail
		if err := rows.Scan(&d.ID, &d.ResultID, &d.Email, &d.Route, &d.IP, &d.UserAgent,
			&d.Outcome, &d.Reason, &d.Promoted, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// Hit is one raw tracking request row (used by the promote action).
type Hit struct {
	ID         string
	ResultID   string
	CampaignID string
	Route      string
	IP         string
	UserAgent  string
	Outcome    string
	Promoted   bool
}

// GetHit loads a single tracking hit by ID (ErrNotFound when absent).
func (s *Store) GetHit(c ctx, hitID string) (*Hit, error) {
	var h Hit
	err := s.db.QueryRow(c,
		`SELECT id::text, result_id::text, campaign_id::text, route,
		        coalesce(host(ip),''), coalesce(user_agent,''), outcome, promoted
		   FROM tracking_hits WHERE id=$1`, hitID).
		Scan(&h.ID, &h.ResultID, &h.CampaignID, &h.Route, &h.IP, &h.UserAgent, &h.Outcome, &h.Promoted)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &h, nil
}

// MarkHitPromoted flags a hit the operator promoted to a real interaction.
func (s *Store) MarkHitPromoted(c ctx, hitID string) error {
	_, err := s.db.Exec(c, `UPDATE tracking_hits SET promoted=TRUE WHERE id=$1`, hitID)
	return err
}
