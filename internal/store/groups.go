package store

import "secdrill/internal/models"

func (s *Store) CreateGroup(c ctx, clientID, name string) (string, error) {
	var id string
	err := s.db.QueryRow(c,
		`INSERT INTO groups (client_id, name) VALUES ($1,$2) RETURNING id`, clientID, name).Scan(&id)
	return id, err
}

func (s *Store) ListGroups(c ctx, clientID string) ([]models.Group, error) {
	rows, err := s.db.Query(c,
		`SELECT g.id, g.client_id, g.name, g.created_at, count(t.id)
		   FROM groups g LEFT JOIN targets t ON t.group_id = g.id
		  WHERE g.client_id = $1
		  GROUP BY g.id ORDER BY g.name`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Group
	for rows.Next() {
		var g models.Group
		if err := rows.Scan(&g.ID, &g.ClientID, &g.Name, &g.CreatedAt, &g.Targets); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (s *Store) GetGroup(c ctx, id string) (*models.Group, error) {
	var g models.Group
	err := s.db.QueryRow(c,
		`SELECT id, client_id, name, created_at FROM groups WHERE id=$1`, id).
		Scan(&g.ID, &g.ClientID, &g.Name, &g.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

// UpdateGroup renames a group.
func (s *Store) UpdateGroup(c ctx, id, name string) error {
	_, err := s.db.Exec(c, `UPDATE groups SET name=$2 WHERE id=$1`, id, name)
	return err
}

// CountResultsForGroup guards deletion: groups→targets cascade on delete, but
// campaign_results.target_id has no cascade, so a group whose targets are already
// used in a campaign cannot be deleted. The caller checks this first.
func (s *Store) CountResultsForGroup(c ctx, id string) (int, error) {
	var n int
	err := s.db.QueryRow(c,
		`SELECT count(*) FROM campaign_results cr JOIN targets t ON t.id = cr.target_id WHERE t.group_id=$1`, id).Scan(&n)
	return n, err
}

// DeleteGroup removes a group; its targets cascade. Call CountResultsForGroup first.
func (s *Store) DeleteGroup(c ctx, id string) error {
	_, err := s.db.Exec(c, `DELETE FROM groups WHERE id=$1`, id)
	return err
}
