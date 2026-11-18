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

// CountedHit is a link_clicked/data_submitted event a later canary probe may
// need to retract as part of the scanner's crawl.
type CountedHit struct {
	Type      string
	CreatedAt time.Time
}

// FirstProbeTime returns the earliest logged scanner_probe event for a result
// (nil when none). It marks the message's first scan ≈ delivery: no human click
// can precede it, which is what makes the unbounded pre-delivery lookback valid.
// Fallback for results probed before probe-event logging: the caller can use
// the scanner_probe_at column, which then holds the only probe.
func (s *Store) FirstProbeTime(c ctx, resultID string) (*time.Time, error) {
	var ts *time.Time
	err := s.db.QueryRow(c,
		`SELECT min(created_at) FROM events WHERE result_id=$1 AND type='scanner_probe'`,
		resultID).Scan(&ts)
	return ts, err
}

// CountedHitsAround returns tracked events that a probe arrival may need to
// retract: either stamped BEFORE the first probe (pre-delivery — no human
// could have produced them) or within the recent burst window before this
// probe (a slow sequential crawl that raced ahead of its canary). Opens are
// included: a scanner that prefetches images during its crawl gets its
// "opened" retracted just like its click.
func (s *Store) CountedHitsAround(c ctx, resultID string, firstProbe time.Time, recentWindow time.Duration) ([]CountedHit, error) {
	rows, err := s.db.Query(c,
		`SELECT type, created_at FROM events
		  WHERE result_id=$1 AND type IN ('link_clicked','data_submitted','email_opened')
		    AND (created_at < $2 OR created_at > now() - $3::bigint * interval '1 microsecond')`,
		resultID, firstProbe, recentWindow.Microseconds())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CountedHit
	for rows.Next() {
		var h CountedHit
		if err := rows.Scan(&h.Type, &h.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}
