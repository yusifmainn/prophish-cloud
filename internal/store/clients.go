package store

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"secdrill/internal/models"
)

// ErrClientReferenced is returned by DeleteClient when the client cannot be
// removed because a template or landing page scoped to it is still referenced by
// another client's campaign (campaigns.template_id / landing_page_id are NO
// ACTION). The delete is refused as a whole, never partially applied.
var ErrClientReferenced = errors.New("client is referenced by another client's campaign")

func (s *Store) ListClients(c ctx) ([]models.Client, error) {
	rows, err := s.db.Query(c,
		`SELECT id, name, coalesce(contact_email,''), coalesce(notes,''), created_at
		   FROM clients ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Client
	for rows.Next() {
		var cl models.Client
		if err := rows.Scan(&cl.ID, &cl.Name, &cl.ContactEmail, &cl.Notes, &cl.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, cl)
	}
	return out, rows.Err()
}

func (s *Store) CreateClient(c ctx, name, contactEmail, notes string) (string, error) {
	var id string
	err := s.db.QueryRow(c,
		`INSERT INTO clients (name, contact_email, notes) VALUES ($1,$2,$3) RETURNING id`,
		name, nullIfEmpty(contactEmail), nullIfEmpty(notes)).Scan(&id)
	return id, err
}

// UpdateClient edits a client's display fields. The client_id-linked rows
// (authorizations, groups, templates, campaigns) are unaffected.
func (s *Store) UpdateClient(c ctx, id, name, contactEmail, notes string) error {
	_, err := s.db.Exec(c,
		`UPDATE clients SET name=$2, contact_email=$3, notes=$4 WHERE id=$1`,
		id, name, nullIfEmpty(contactEmail), nullIfEmpty(notes))
	return err
}

// DeleteClient removes a client and everything scoped to it — authorizations,
// groups (and their targets), the client's own templates/landing pages, and its
// campaigns (with their results and events) — all via ON DELETE CASCADE. Global
// templates/pages (client_id NULL) are left untouched. It can still be refused:
// if a client-scoped template or landing page is referenced by ANOTHER client's
// campaign, Postgres aborts the whole delete and this returns ErrClientReferenced
// so the caller shows a clear message instead of an opaque 500. Irreversible on
// success.
func (s *Store) DeleteClient(c ctx, id string) error {
	_, err := s.db.Exec(c, `DELETE FROM clients WHERE id=$1`, id)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" { // foreign_key_violation
		return ErrClientReferenced
	}
	return err
}

func (s *Store) GetClient(c ctx, id string) (*models.Client, error) {
	var cl models.Client
	err := s.db.QueryRow(c,
		`SELECT id, name, coalesce(contact_email,''), coalesce(notes,''), created_at
		   FROM clients WHERE id = $1`, id).
		Scan(&cl.ID, &cl.Name, &cl.ContactEmail, &cl.Notes, &cl.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &cl, nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
