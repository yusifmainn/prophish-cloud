package store

import "time"

// ResultDetail is a per-target row for the campaign report / CSV export.
type ResultDetail struct {
	Email     string
	FirstName string
	LastName  string
	Position  string
	Status    string
	SentAt    *time.Time
}

func (s *Store) ListResultsDetailed(c ctx, campaignID string) ([]ResultDetail, error) {
	rows, err := s.db.Query(c,
		`SELECT t.email, coalesce(t.first_name,''), coalesce(t.last_name,''), coalesce(t.position,''),
		        r.status, r.sent_at
		   FROM campaign_results r JOIN targets t ON t.id = r.target_id
		  WHERE r.campaign_id=$1
		  ORDER BY t.email`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ResultDetail
	for rows.Next() {
		var d ResultDetail
		if err := rows.Scan(&d.Email, &d.FirstName, &d.LastName, &d.Position, &d.Status, &d.SentAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// EventDetail is a single timeline entry (which target did what, when).
type EventDetail struct {
	Email     string
	Type      string
	IP        string
	CreatedAt time.Time
}

func (s *Store) ListEventsDetailed(c ctx, campaignID string) ([]EventDetail, error) {
	rows, err := s.db.Query(c,
		`SELECT coalesce(t.email,'(sistem)'), e.type, coalesce(host(e.ip),''), e.created_at
		   FROM events e
		   LEFT JOIN campaign_results r ON r.id = e.result_id
		   LEFT JOIN targets t ON t.id = r.target_id
		  WHERE e.campaign_id=$1
		  ORDER BY e.created_at DESC
		  LIMIT 500`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []EventDetail
	for rows.Next() {
		var d EventDetail
		if err := rows.Scan(&d.Email, &d.Type, &d.IP, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
