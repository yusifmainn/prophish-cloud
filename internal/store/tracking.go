package store

import (
	"errors"

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
}

func (s *Store) TrackedByToken(c ctx, token string) (*TrackedFull, error) {
	var t TrackedFull
	err := s.db.QueryRow(c,
		`SELECT r.id, r.campaign_id, r.status,
		        coalesce(tg.first_name,''), coalesce(tg.last_name,''), coalesce(tg.position,''),
		        lp.name, lp.html_body, coalesce(lp.redirect_url,'')
		   FROM campaign_results r
		   JOIN campaigns c   ON c.id = r.campaign_id
		   JOIN targets   tg  ON tg.id = r.target_id
		   JOIN landing_pages lp ON lp.id = c.landing_page_id
		  WHERE r.result_token = $1`, token).
		Scan(&t.ResultID, &t.CampaignID, &t.Status,
			&t.FirstName, &t.LastName, &t.Position,
			&t.PageName, &t.PageHTML, &t.RedirectURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &t, err
}
