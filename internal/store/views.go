package store

import "time"

// AuthRow is an authorization joined with its client name, for selection UIs.
type AuthRow struct {
	ID         string
	ClientID   string
	ClientName string
	Scope      string
	ValidFrom  time.Time
	ValidTo    time.Time
}

func (s *Store) ListAllAuthorizations(c ctx) ([]AuthRow, error) {
	rows, err := s.db.Query(c,
		`SELECT a.id, a.client_id, cl.name, a.scope, a.valid_from, a.valid_to
		   FROM authorizations a JOIN clients cl ON cl.id = a.client_id
		  ORDER BY cl.name, a.valid_to DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AuthRow
	for rows.Next() {
		var a AuthRow
		if err := rows.Scan(&a.ID, &a.ClientID, &a.ClientName, &a.Scope, &a.ValidFrom, &a.ValidTo); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// GroupRow is a group joined with its client name and target count.
type GroupRow struct {
	ID         string
	ClientID   string
	ClientName string
	Name       string
	Targets    int
}

func (s *Store) ListAllGroups(c ctx) ([]GroupRow, error) {
	rows, err := s.db.Query(c,
		`SELECT g.id, g.client_id, cl.name, g.name, count(t.id)
		   FROM groups g JOIN clients cl ON cl.id = g.client_id
		   LEFT JOIN targets t ON t.group_id = g.id
		  GROUP BY g.id, cl.name ORDER BY cl.name, g.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GroupRow
	for rows.Next() {
		var g GroupRow
		if err := rows.Scan(&g.ID, &g.ClientID, &g.ClientName, &g.Name, &g.Targets); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// TargetIDsForGroups returns all target ids belonging to the given groups.
func (s *Store) TargetIDsForGroups(c ctx, groupIDs []string) ([]string, error) {
	rows, err := s.db.Query(c,
		`SELECT id FROM targets WHERE group_id = ANY($1)`, groupIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// CampaignView is the detail/report view joining human-readable names.
type CampaignView struct {
	ID            string
	Name          string
	Status        string
	ClientID      string
	ClientName    string
	TemplateName  string
	PageName      string
	ProfileName   string
	AuthScope     string
	AuthValidTo   time.Time
	CreatedAt     time.Time
}

func (s *Store) CampaignView(c ctx, id string) (*CampaignView, error) {
	var v CampaignView
	err := s.db.QueryRow(c,
		`SELECT c.id, c.name, c.status, c.client_id, cl.name,
		        et.name, lp.name, sp.name, a.scope, a.valid_to, c.created_at
		   FROM campaigns c
		   JOIN clients cl        ON cl.id = c.client_id
		   JOIN email_templates et ON et.id = c.template_id
		   JOIN landing_pages lp   ON lp.id = c.landing_page_id
		   JOIN sending_profiles sp ON sp.id = c.sending_profile_id
		   JOIN authorizations a   ON a.id = c.authorization_id
		  WHERE c.id = $1`, id).
		Scan(&v.ID, &v.Name, &v.Status, &v.ClientID, &v.ClientName,
			&v.TemplateName, &v.PageName, &v.ProfileName, &v.AuthScope, &v.AuthValidTo, &v.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &v, nil
}
