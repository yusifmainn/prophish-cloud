package store

import "time"

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
