package store

import (
	"errors"

	"github.com/jackc/pgx/v5"
)

// IMAPConfig is the single global mailbox SecDrill polls for reported emails.
type IMAPConfig struct {
	ID          string
	Host        string
	Port        int
	Username    string
	UseTLS      bool
	Folder      string
	Enabled     bool
	PollSeconds int
	LastUID     int64
	PasswordEnc []byte
}

func (s *Store) GetIMAPConfig(c ctx) (*IMAPConfig, error) {
	var m IMAPConfig
	err := s.db.QueryRow(c,
		`SELECT id, host, port, username, use_tls, folder, enabled, poll_seconds, last_uid, password_enc
		   FROM imap_config ORDER BY created_at LIMIT 1`).
		Scan(&m.ID, &m.Host, &m.Port, &m.Username, &m.UseTLS, &m.Folder, &m.Enabled, &m.PollSeconds, &m.LastUID, &m.PasswordEnc)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &m, err
}

// UpsertIMAPConfig keeps a single config row.
func (s *Store) UpsertIMAPConfig(c ctx, host string, port int, username string, passwordEnc []byte, useTLS bool, folder string, enabled bool, pollSeconds int) error {
	existing, err := s.GetIMAPConfig(c)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}
	if errors.Is(err, ErrNotFound) {
		_, e := s.db.Exec(c,
			`INSERT INTO imap_config (host, port, username, password_enc, use_tls, folder, enabled, poll_seconds)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			host, port, username, passwordEnc, useTLS, folder, enabled, pollSeconds)
		return e
	}
	// Keep the old password if a new one wasn't provided.
	pw := existing.PasswordEnc
	if len(passwordEnc) > 0 {
		pw = passwordEnc
	}
	_, e := s.db.Exec(c,
		`UPDATE imap_config SET host=$2, port=$3, username=$4, password_enc=$5, use_tls=$6, folder=$7, enabled=$8, poll_seconds=$9
		   WHERE id=$1`,
		existing.ID, host, port, username, pw, useTLS, folder, enabled, pollSeconds)
	return e
}

func (s *Store) SetIMAPLastUID(c ctx, id string, uid int64) error {
	_, err := s.db.Exec(c, `UPDATE imap_config SET last_uid=$2 WHERE id=$1`, id, uid)
	return err
}

// InsertReportedEmail records an employee-reported suspicious email.
func (s *Store) InsertReportedEmail(c ctx, reporter, subject string) error {
	_, err := s.db.Exec(c,
		`INSERT INTO reported_emails (reporter_email, subject) VALUES ($1,$2)`,
		nullIfEmpty(reporter), nullIfEmpty(subject))
	return err
}

type ReportedEmail struct {
	Reporter string
	Subject  string
	At       string
}

func (s *Store) ListReportedEmails(c ctx) ([]ReportedEmail, error) {
	rows, err := s.db.Query(c,
		`SELECT coalesce(reporter_email,''), coalesce(subject,''), to_char(reported_at,'YYYY-MM-DD HH24:MI')
		   FROM reported_emails ORDER BY reported_at DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ReportedEmail
	for rows.Next() {
		var r ReportedEmail
		if err := rows.Scan(&r.Reporter, &r.Subject, &r.At); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
