package store

import "time"

// RecordAudit appends one append-only audit entry. It never stores secrets; the
// detail field is a short human description only. actorID / actorEmail may be ""
// for a pre-authentication action such as a failed login attempt.
func (s *Store) RecordAudit(c ctx, actorID, actorEmail, action, targetType, targetID, detail, ip string) error {
	_, err := s.db.Exec(c,
		`INSERT INTO audit_log (actor_id, actor_email, action, target_type, target_id, detail, ip)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		nullIfEmpty(actorID), nullIfEmpty(actorEmail), action,
		nullIfEmpty(targetType), nullIfEmpty(targetID), nullIfEmpty(detail), nullIfEmpty(ip))
	return err
}

// AuditRow is one audit-log line for display.
type AuditRow struct {
	CreatedAt  time.Time
	ActorEmail string
	Action     string
	TargetType string
	TargetID   string
	Detail     string
	IP         string
}

// ListAudit returns the most recent audit entries, newest first, capped.
func (s *Store) ListAudit(c ctx, limit int) ([]AuditRow, error) {
	if limit <= 0 || limit > 1000 {
		limit = 300
	}
	rows, err := s.db.Query(c,
		`SELECT created_at, coalesce(actor_email,''), action,
		        coalesce(target_type,''), coalesce(target_id,''), coalesce(detail,''),
		        coalesce(host(ip),'')
		   FROM audit_log ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AuditRow
	for rows.Next() {
		var a AuditRow
		if err := rows.Scan(&a.CreatedAt, &a.ActorEmail, &a.Action, &a.TargetType, &a.TargetID, &a.Detail, &a.IP); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
