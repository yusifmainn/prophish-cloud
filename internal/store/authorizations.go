package store

import (
	"time"

	"secdrill/internal/models"
)

func (s *Store) CreateAuthorization(c ctx, clientID, scope, authorizedBy, docRef string, from, to time.Time) (string, error) {
	var id string
	err := s.db.QueryRow(c,
		`INSERT INTO authorizations (client_id, scope, authorized_by, document_ref, valid_from, valid_to)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		clientID, scope, authorizedBy, nullIfEmpty(docRef), from, to).Scan(&id)
	return id, err
}

func (s *Store) ListAuthorizations(c ctx, clientID string) ([]models.Authorization, error) {
	rows, err := s.db.Query(c,
		`SELECT id, client_id, scope, authorized_by, coalesce(document_ref,''), valid_from, valid_to, created_at
		   FROM authorizations WHERE client_id = $1 ORDER BY valid_to DESC`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Authorization
	for rows.Next() {
		var a models.Authorization
		if err := rows.Scan(&a.ID, &a.ClientID, &a.Scope, &a.AuthorizedBy, &a.DocumentRef,
			&a.ValidFrom, &a.ValidTo, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ActiveAuthorizationExists reports whether the client has an authorization whose
// validity window covers 'at'. A campaign cannot launch otherwise.
func (s *Store) ActiveAuthorizationExists(c ctx, authorizationID string, at time.Time) (bool, error) {
	var ok bool
	err := s.db.QueryRow(c,
		`SELECT exists(SELECT 1 FROM authorizations WHERE id=$1 AND valid_from <= $2 AND valid_to >= $2)`,
		authorizationID, at).Scan(&ok)
	return ok, err
}

func (s *Store) GetAuthorization(c ctx, id string) (*models.Authorization, error) {
	var a models.Authorization
	err := s.db.QueryRow(c,
		`SELECT id, client_id, scope, authorized_by, coalesce(document_ref,''), valid_from, valid_to, created_at
		   FROM authorizations WHERE id=$1`, id).
		Scan(&a.ID, &a.ClientID, &a.Scope, &a.AuthorizedBy, &a.DocumentRef, &a.ValidFrom, &a.ValidTo, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// UpdateAuthorization edits a RoE. The DB CHECK (valid_to > valid_from) still applies.
func (s *Store) UpdateAuthorization(c ctx, id, scope, authorizedBy, docRef string, from, to time.Time) error {
	_, err := s.db.Exec(c,
		`UPDATE authorizations SET scope=$2, authorized_by=$3, document_ref=$4, valid_from=$5, valid_to=$6 WHERE id=$1`,
		id, scope, authorizedBy, nullIfEmpty(docRef), from, to)
	return err
}

// CountCampaignsUsingAuthorization guards deletion: campaigns.authorization_id has
// no ON DELETE action, so an authorization still referenced by a campaign cannot
// be removed — the caller checks this first for a clear message.
func (s *Store) CountCampaignsUsingAuthorization(c ctx, id string) (int, error) {
	var n int
	err := s.db.QueryRow(c, `SELECT count(*) FROM campaigns WHERE authorization_id=$1`, id).Scan(&n)
	return n, err
}

func (s *Store) DeleteAuthorization(c ctx, id string) error {
	_, err := s.db.Exec(c, `DELETE FROM authorizations WHERE id=$1`, id)
	return err
}
