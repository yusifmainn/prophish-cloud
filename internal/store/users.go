package store

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/jackc/pgx/v5"
	"prophish/internal/models"
)

var ErrNotFound = errors.New("not found")

// hashAPIKey returns the at-rest representation of an API key. Keys are 192-bit
// random, so a single SHA-256 is sufficient and lets us store only the hash.
func hashAPIKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// CountUsers returns the number of operator accounts (used for first-run setup).
func (s *Store) CountUsers(c ctx) (int, error) {
	var n int
	err := s.db.QueryRow(c, `SELECT count(*) FROM users`).Scan(&n)
	return n, err
}

// CreateUser inserts a new operator with an already-hashed password.
func (s *Store) CreateUser(c ctx, email, passwordHash, role string) (string, error) {
	var id string
	err := s.db.QueryRow(c,
		`INSERT INTO users (email, password_hash, role) VALUES ($1,$2,$3) RETURNING id`,
		email, passwordHash, role).Scan(&id)
	return id, err
}

// GetUserByEmail returns the user and their password hash for login verification.
func (s *Store) GetUserByEmail(c ctx, email string) (*models.User, string, error) {
	var u models.User
	var hash string
	err := s.db.QueryRow(c,
		`SELECT id, email, password_hash, role, is_active, created_at, last_login_at
		   FROM users WHERE email = $1`, email).
		Scan(&u.ID, &u.Email, &hash, &u.Role, &u.IsActive, &u.CreatedAt, &u.LastLoginAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", ErrNotFound
	}
	return &u, hash, err
}

// GetUser returns a user by id.
func (s *Store) GetUser(c ctx, id string) (*models.User, error) {
	var u models.User
	var hash string
	err := s.db.QueryRow(c,
		`SELECT id, email, password_hash, role, is_active, created_at, last_login_at
		   FROM users WHERE id = $1`, id).
		Scan(&u.ID, &u.Email, &hash, &u.Role, &u.IsActive, &u.CreatedAt, &u.LastLoginAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &u, err
}

// TouchLogin records the last successful login time.
func (s *Store) TouchLogin(c ctx, id string) error {
	_, err := s.db.Exec(c, `UPDATE users SET last_login_at = now() WHERE id = $1`, id)
	return err
}

// ListUsers returns all operator accounts.
func (s *Store) ListUsers(c ctx) ([]models.User, error) {
	rows, err := s.db.Query(c,
		`SELECT id, email, role, is_active, created_at, last_login_at FROM users ORDER BY email`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Role, &u.IsActive, &u.CreatedAt, &u.LastLoginAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Store) DeleteUser(c ctx, id string) error {
	_, err := s.db.Exec(c, `DELETE FROM users WHERE id=$1`, id)
	return err
}

func (s *Store) CountOtherAdmins(c ctx, exceptID string) (int, error) {
	var n int
	err := s.db.QueryRow(c,
		`SELECT count(*) FROM users WHERE role='admin' AND id <> $1`, exceptID).Scan(&n)
	return n, err
}

func (s *Store) SetPassword(c ctx, id, passwordHash string) error {
	_, err := s.db.Exec(c, `UPDATE users SET password_hash=$2 WHERE id=$1`, id, passwordHash)
	return err
}

// GetPasswordHash returns the stored bcrypt hash for an operator (for verifying
// the current password before a change).
func (s *Store) GetPasswordHash(c ctx, id string) (string, error) {
	var h string
	err := s.db.QueryRow(c, `SELECT password_hash FROM users WHERE id=$1`, id).Scan(&h)
	return h, err
}

// SetAPIKey stores (or clears) an operator's REST API key. Only the SHA-256 hash
// of the key is persisted; the plaintext is shown to the operator once and never
// stored.
func (s *Store) SetAPIKey(c ctx, id, apiKey string) error {
	var stored any
	if apiKey != "" {
		stored = hashAPIKey(apiKey)
	}
	_, err := s.db.Exec(c, `UPDATE users SET api_key=$2 WHERE id=$1`, id, stored)
	return err
}

// HasAPIKey reports whether the operator currently has an API key set.
func (s *Store) HasAPIKey(c ctx, id string) (bool, error) {
	var set bool
	err := s.db.QueryRow(c, `SELECT api_key IS NOT NULL FROM users WHERE id=$1`, id).Scan(&set)
	return set, err
}

// UserByAPIKey resolves a presented API key to its operator by comparing hashes.
func (s *Store) UserByAPIKey(c ctx, apiKey string) (*models.User, error) {
	var u models.User
	err := s.db.QueryRow(c,
		`SELECT id, email, role, is_active, created_at, last_login_at
		   FROM users WHERE api_key=$1 AND is_active=TRUE`, hashAPIKey(apiKey)).
		Scan(&u.ID, &u.Email, &u.Role, &u.IsActive, &u.CreatedAt, &u.LastLoginAt)
	if err != nil {
		return nil, ErrNotFound
	}
	return &u, nil
}
