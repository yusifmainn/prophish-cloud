package store

import (
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// TrackedFull resolves a tracking token to everything the public server needs to
// render a landing page for that specific target. It does NOT expose anything
// beyond the target's display name — and never stores what they submit.
type TrackedFull struct {
	ResultID    string
	CampaignID  string
	Status      string
	FirstName   string
	LastName    string
	Position    string
	PageName    string
	PageHTML    string
	RedirectURL string
	// CampaignKind is "local" (serve a hosted landing page) or "public" (redirect
	// the click to CampaignRedirectURL instead of showing any page).
	CampaignKind        string
	CampaignRedirectURL string
	// SentAt is when the message was sent, or nil if not yet sent. The public
	// server uses it to ignore scanner prefetches that fire moments after send.
	SentAt *time.Time
	// ScannerProbeAt is when the hidden canary link last fired, or nil if the
	// recipient was never scanner-probed. A real link/form hit inside the
	// cluster window after this moment is treated as the same automated crawl.
	ScannerProbeAt *time.Time
}

func (s *Store) TrackedByToken(c ctx, token string) (*TrackedFull, error) {
	var t TrackedFull
	// landing_pages is LEFT-joined: a 'public' campaign has no landing page
	// (landing_page_id IS NULL), so the page columns come back empty for it. The
	// public server decides what to do from CampaignKind.
	err := s.db.QueryRow(c,
		`SELECT r.id, r.campaign_id, r.status,
		        coalesce(tg.first_name,''), coalesce(tg.last_name,''), coalesce(tg.position,''),
		        coalesce(lp.name,''), coalesce(lp.html_body,''), coalesce(lp.redirect_url,''),
		        c.kind, coalesce(c.redirect_url,''), r.sent_at, r.scanner_probe_at
		   FROM campaign_results r
		   JOIN campaigns c   ON c.id = r.campaign_id
		   JOIN targets   tg  ON tg.id = r.target_id
		   LEFT JOIN landing_pages lp ON lp.id = c.landing_page_id
		  WHERE r.result_token = $1`, token).
		Scan(&t.ResultID, &t.CampaignID, &t.Status,
			&t.FirstName, &t.LastName, &t.Position,
			&t.PageName, &t.PageHTML, &t.RedirectURL,
			&t.CampaignKind, &t.CampaignRedirectURL, &t.SentAt, &t.ScannerProbeAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &t, err
}
