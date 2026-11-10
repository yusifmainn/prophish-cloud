package store

import (
	"github.com/jackc/pgx/v5"
	"prophish/internal/models"
)

// AddTarget inserts a single target, ignoring duplicates within the same group.
func (s *Store) AddTarget(c ctx, groupID, email, first, last, position string) error {
	_, err := s.db.Exec(c,
		`INSERT INTO targets (group_id, email, first_name, last_name, position)
		 VALUES ($1,$2,$3,$4,$5)
		 ON CONFLICT (group_id, email) DO NOTHING`,
		groupID, email, nullIfEmpty(first), nullIfEmpty(last), nullIfEmpty(position))
	return err
}

// AddTargetsBatch inserts many targets in a single transaction.
func (s *Store) AddTargetsBatch(c ctx, groupID string, rows [][4]string) (int, error) {
	tx, err := s.db.Begin(c)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(c)

	b := &pgx.Batch{}
	for _, r := range rows {
		b.Queue(
			`INSERT INTO targets (group_id, email, first_name, last_name, position)
			 VALUES ($1,$2,$3,$4,$5) ON CONFLICT (group_id, email) DO NOTHING`,
			groupID, r[0], nullIfEmpty(r[1]), nullIfEmpty(r[2]), nullIfEmpty(r[3]))
	}
	br := tx.SendBatch(c, b)
	inserted := 0
	for range rows {
		ct, err := br.Exec()
		if err != nil {
			br.Close()
			return 0, err
		}
		// ON CONFLICT DO NOTHING → RowsAffected is 1 for a real insert, 0 for a
		// skipped duplicate, so this is the true number of targets added.
		inserted += int(ct.RowsAffected())
	}
	if err := br.Close(); err != nil {
		return 0, err
	}
	if err := tx.Commit(c); err != nil {
		return 0, err
	}
	return inserted, nil
}

func (s *Store) ListTargets(c ctx, groupID string) ([]models.Target, error) {
	rows, err := s.db.Query(c,
		`SELECT id, group_id, email, coalesce(first_name,''), coalesce(last_name,''), coalesce(position,''), created_at
		   FROM targets WHERE group_id=$1 ORDER BY email`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Target
	for rows.Next() {
		var t models.Target
		if err := rows.Scan(&t.ID, &t.GroupID, &t.Email, &t.FirstName, &t.LastName, &t.Position, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) GetTarget(c ctx, id string) (*models.Target, error) {
	var t models.Target
	err := s.db.QueryRow(c,
		`SELECT id, group_id, email, coalesce(first_name,''), coalesce(last_name,''), coalesce(position,''), created_at
		   FROM targets WHERE id=$1`, id).
		Scan(&t.ID, &t.GroupID, &t.Email, &t.FirstName, &t.LastName, &t.Position, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// UpdateTarget edits one target. The UNIQUE(group_id,email) constraint still
// applies, so an edit to a duplicate email returns an error the caller surfaces.
func (s *Store) UpdateTarget(c ctx, id, email, first, last, position string) error {
	_, err := s.db.Exec(c,
		`UPDATE targets SET email=$2, first_name=$3, last_name=$4, position=$5 WHERE id=$1`,
		id, email, nullIfEmpty(first), nullIfEmpty(last), nullIfEmpty(position))
	return err
}

// CountResultsForTarget guards deletion: campaign_results.target_id has no cascade.
func (s *Store) CountResultsForTarget(c ctx, id string) (int, error) {
	var n int
	err := s.db.QueryRow(c, `SELECT count(*) FROM campaign_results WHERE target_id=$1`, id).Scan(&n)
	return n, err
}

func (s *Store) DeleteTarget(c ctx, id string) error {
	_, err := s.db.Exec(c, `DELETE FROM targets WHERE id=$1`, id)
	return err
}
