package store

// RecordEvent appends an immutable event. IMPORTANT: this never stores submitted
// field values or credentials — only metadata (which campaign/result, type, ip, ua).
func (s *Store) RecordEvent(c ctx, campaignID string, resultID *string, eventType, ip, ua string) error {
	_, err := s.db.Exec(c,
		`INSERT INTO events (campaign_id, result_id, type, ip, user_agent)
		 VALUES ($1,$2,$3,$4,$5)`,
		campaignID, resultID, eventType, nullIfEmpty(ip), nullIfEmpty(ua))
	return err
}
