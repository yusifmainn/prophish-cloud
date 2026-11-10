package store

import "prophish/internal/models"

func (s *Store) CreateLandingPage(c ctx, clientID *string, name, html, redirect string) (string, error) {
	var id string
	err := s.db.QueryRow(c,
		`INSERT INTO landing_pages (client_id, name, html_body, redirect_url)
		 VALUES ($1,$2,$3,$4) RETURNING id`,
		clientID, name, html, nullIfEmpty(redirect)).Scan(&id)
	return id, err
}

func (s *Store) ListLandingPages(c ctx) ([]models.LandingPage, error) {
	rows, err := s.db.Query(c,
		`SELECT id, client_id, name, html_body, coalesce(redirect_url,''), created_at
		   FROM landing_pages ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.LandingPage
	for rows.Next() {
		var p models.LandingPage
		if err := rows.Scan(&p.ID, &p.ClientID, &p.Name, &p.HTMLBody, &p.RedirectURL, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) GetLandingPage(c ctx, id string) (*models.LandingPage, error) {
	var p models.LandingPage
	err := s.db.QueryRow(c,
		`SELECT id, client_id, name, html_body, coalesce(redirect_url,''), created_at
		   FROM landing_pages WHERE id=$1`, id).
		Scan(&p.ID, &p.ClientID, &p.Name, &p.HTMLBody, &p.RedirectURL, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// UpdateLandingPage edits a page's content (client scoping left unchanged here).
func (s *Store) UpdateLandingPage(c ctx, id, name, html, redirect string) error {
	_, err := s.db.Exec(c,
		`UPDATE landing_pages SET name=$2, html_body=$3, redirect_url=$4 WHERE id=$1`,
		id, name, html, nullIfEmpty(redirect))
	return err
}

// CountCampaignsUsingLandingPage guards deletion: campaigns.landing_page_id has no
// ON DELETE action (mirrors templates/profiles).
func (s *Store) CountCampaignsUsingLandingPage(c ctx, id string) (int, error) {
	var n int
	err := s.db.QueryRow(c, `SELECT count(*) FROM campaigns WHERE landing_page_id=$1`, id).Scan(&n)
	return n, err
}

func (s *Store) DeleteLandingPage(c ctx, id string) error {
	_, err := s.db.Exec(c, `DELETE FROM landing_pages WHERE id=$1`, id)
	return err
}
