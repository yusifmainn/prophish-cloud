package store

// DashboardStats holds the headline numbers for the admin dashboard.
type DashboardStats struct {
	ActiveCampaigns int // running or scheduled, not archived
	Campaigns       int // total, not archived
	Tested          int // results actually delivered (sent and beyond)
	ClickRate       int // percent of delivered recipients who clicked
	ReportRate      int // percent of delivered recipients who reported the phish
}

// DashboardStats aggregates headline metrics across all non-archived campaigns.
// Rates are percentages of delivered recipients (guarded against divide-by-zero).
func (s *Store) DashboardStats(c ctx) (DashboardStats, error) {
	var d DashboardStats
	if err := s.db.QueryRow(c,
		`SELECT count(*) FILTER (WHERE NOT archived),
		        count(*) FILTER (WHERE NOT archived AND status IN ('running','scheduled'))
		   FROM campaigns`).Scan(&d.Campaigns, &d.ActiveCampaigns); err != nil {
		return d, err
	}

	var sent, clicked, reported int
	if err := s.db.QueryRow(c,
		`SELECT count(*) FILTER (WHERE r.status IN ('sent','opened','clicked','submitted')),
		        count(*) FILTER (WHERE r.status IN ('clicked','submitted')),
		        count(*) FILTER (WHERE r.reported)
		   FROM campaign_results r
		   JOIN campaigns ca ON ca.id = r.campaign_id AND NOT ca.archived`).
		Scan(&sent, &clicked, &reported); err != nil {
		return d, err
	}
	d.Tested = sent
	if sent > 0 {
		d.ClickRate = clicked * 100 / sent
		d.ReportRate = reported * 100 / sent
	}
	return d, nil
}
