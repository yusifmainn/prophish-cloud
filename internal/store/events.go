package store

import (
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// RecordEvent appends an immutable event. IMPORTANT: this never stores submitted
// field values or credentials — only metadata (which campaign/result, type, ip, ua).
func (s *Store) RecordEvent(c ctx, campaignID string, resultID *string, eventType, ip, ua string) error {
	_, err := s.db.Exec(c,
		`INSERT INTO events (campaign_id, result_id, type, ip, user_agent)
		 VALUES ($1,$2,$3,$4,$5)`,
		campaignID, resultID, eventType, nullIfEmpty(ip), nullIfEmpty(ua))
	return err
}

// HasRecentEvent reports whether an event of the given type was recorded for
// this result within the last `since`. It dedups suspected_* events so a
// scanner replaying a link cannot flood the events table.
func (s *Store) HasRecentEvent(c ctx, resultID, eventType string, since time.Duration) (bool, error) {
	var exists bool
	err := s.db.QueryRow(c,
		`SELECT EXISTS(SELECT 1 FROM events
		          WHERE result_id=$1 AND type=$2
		            AND created_at > now() - $3::bigint * interval '1 microsecond')`,
		resultID, eventType, since.Microseconds()).Scan(&exists)
	return exists, err
}

// CountedHit is a link_clicked/data_submitted event a later canary probe may
// need to retract as part of the same scanner crawl.
type CountedHit struct {
	Type      string
	CreatedAt time.Time
}

// LastCountedHit returns the most recent counted link/submit event for a
// result (nil when none), for the canary handler's retrospective retraction.
func (s *Store) LastCountedHit(c ctx, resultID string) (*CountedHit, error) {
	var h CountedHit
	err := s.db.QueryRow(c,
		`SELECT type, created_at FROM events
		  WHERE result_id=$1 AND type IN ('link_clicked','data_submitted')
		  ORDER BY created_at DESC LIMIT 1`, resultID).
		Scan(&h.Type, &h.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &h, nil
}
