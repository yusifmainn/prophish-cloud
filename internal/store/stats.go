package store

import "prophish/internal/models"

// GroupStat is a per-department (group) aggregate used in the client PDF report.
type GroupStat struct {
	Name      string
	Total     int
	Sent      int
	Clicked   int
	Submitted int
	Reported  int
}

// CampaignStatsByGroup breaks a campaign's results down by the target's group
// (department), so a client report can show which departments are weakest.
func (s *Store) CampaignStatsByGroup(c ctx, campaignID string) ([]GroupStat, error) {
	rows, err := s.db.Query(c,
		`SELECT g.name,
		        count(*),
		        count(*) FILTER (WHERE r.status IN ('sent','opened','clicked','submitted')),
		        count(*) FILTER (WHERE r.status IN ('clicked','submitted')),
		        count(*) FILTER (WHERE r.status = 'submitted'),
		        count(*) FILTER (WHERE r.reported)
		   FROM campaign_results r
		   JOIN targets t ON t.id = r.target_id
		   JOIN groups  g ON g.id = t.group_id
		  WHERE r.campaign_id = $1
		  GROUP BY g.id
		  ORDER BY g.name`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GroupStat
	for rows.Next() {
		var gs GroupStat
		if err := rows.Scan(&gs.Name, &gs.Total, &gs.Sent, &gs.Clicked, &gs.Submitted, &gs.Reported); err != nil {
			return nil, err
		}
		out = append(out, gs)
	}
	return out, rows.Err()
}

// CampaignStats aggregates per-result progress for reporting. Derived from the
// forward-only result status, so each stage is cumulative.
func (s *Store) CampaignStats(c ctx, campaignID string) (models.ResultStats, error) {
	var st models.ResultStats
	err := s.db.QueryRow(c,
		`SELECT
		   count(*)                                                                    AS total,
		   count(*) FILTER (WHERE status IN ('sent','opened','clicked','submitted'))   AS sent,
		   count(*) FILTER (WHERE status IN ('opened','clicked','submitted'))          AS opened,
		   count(*) FILTER (WHERE status IN ('clicked','submitted'))                   AS clicked,
		   count(*) FILTER (WHERE status = 'submitted')                               AS submitted,
		   count(*) FILTER (WHERE reported)                                            AS reported
		 FROM campaign_results WHERE campaign_id=$1`, campaignID).
		Scan(&st.Total, &st.Sent, &st.Opened, &st.Clicked, &st.Submitted, &st.Reported)
	return st, err
}
