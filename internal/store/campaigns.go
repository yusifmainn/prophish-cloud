package store

import (
	"time"

	"prophish/internal/models"
)

func (s *Store) CreateCampaign(c ctx, clientID, authID, name, templateID, pageID, profileID, createdBy, publicBaseURL, kind, redirectURL string, tunnelManaged bool) (string, error) {
	var id string
	err := s.db.QueryRow(c,
		`INSERT INTO campaigns (client_id, authorization_id, name, template_id, landing_page_id, sending_profile_id, created_by, public_base_url, tunnel_managed, kind, redirect_url)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`,
		clientID, authID, name, templateID, nullIfEmpty(pageID), profileID, nullIfEmpty(createdBy), publicBaseURL, tunnelManaged, kind, redirectURL).Scan(&id)
	return id, err
}

// ListCampaigns lists non-archived campaigns by default.
func (s *Store) ListCampaigns(c ctx, includeArchived bool) ([]CampaignRow, error) {
	q := `SELECT c.id, c.name, cl.name, c.status, c.created_at, c.kind
	        FROM campaigns c JOIN clients cl ON cl.id = c.client_id`
	if !includeArchived {
		q += ` WHERE c.archived = FALSE`
	}
	q += ` ORDER BY c.created_at DESC`
	rows, err := s.db.Query(c, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CampaignRow
	for rows.Next() {
		var r CampaignRow
		if err := rows.Scan(&r.ID, &r.Name, &r.ClientName, &r.Status, &r.CreatedAt, &r.Kind); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) GetCampaign(c ctx, id string) (*models.Campaign, error) {
	var m models.Campaign
	err := s.db.QueryRow(c,
		`SELECT id, client_id, authorization_id, name, template_id, coalesce(landing_page_id::text,''), sending_profile_id,
		        status, launch_at, send_by, completed_at, archived, created_at, public_base_url, tunnel_managed, kind, redirect_url
		   FROM campaigns WHERE id=$1`, id).
		Scan(&m.ID, &m.ClientID, &m.AuthorizationID, &m.Name, &m.TemplateID, &m.LandingPageID,
			&m.SendingProfileID, &m.Status, &m.LaunchAt, &m.SendBy, &m.CompletedAt, &m.Archived, &m.CreatedAt,
			&m.PublicBaseURL, &m.TunnelManaged, &m.Kind, &m.RedirectURL)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *Store) SetCampaignStatus(c ctx, id, status string) error {
	_, err := s.db.Exec(c, `UPDATE campaigns SET status=$2 WHERE id=$1`, id, status)
	return err
}

// StartCampaign atomically claims a campaign for sending: it flips the status to
// 'running' ONLY if it is still 'draft' or 'scheduled'. It returns true for the
// single caller that won the claim, and false for everyone else. This is the
// guard that makes Launch idempotent: a double-clicked "İşə sal", or the
// scheduler firing twice for the same campaign, can no longer send twice.
func (s *Store) StartCampaign(c ctx, id string) (bool, error) {
	tag, err := s.db.Exec(c,
		`UPDATE campaigns SET status='running' WHERE id=$1 AND status IN ('draft','scheduled')`, id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// CancelCampaign marks a currently-running campaign as 'cancelled'. It returns
// true only if a running campaign was actually transitioned (so the UI knows a
// stop request took effect). The in-flight send goroutine is signalled
// separately via Launcher.Cancel.
func (s *Store) CancelCampaign(c ctx, id string) (bool, error) {
	tag, err := s.db.Exec(c,
		`UPDATE campaigns SET status='cancelled', completed_at=now() WHERE id=$1 AND status='running'`, id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// RecoverStuckCampaigns resets campaigns left in 'running' by a crash/restart
// back to 'scheduled' and clears their launch time, so the scheduler will NOT
// auto-resume them — an operator must deliberately relaunch. A relaunch only
// sends to results still in 'scheduled' (never re-sends to targets already
// 'sent'), so recovery is safe. Returns the number recovered.
func (s *Store) RecoverStuckCampaigns(c ctx) (int64, error) {
	tag, err := s.db.Exec(c,
		`UPDATE campaigns SET status='scheduled', launch_at=NULL WHERE status='running'`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// MarkCampaignCompleted transitions a running campaign to 'completed'. It is
// guarded on status='running' so it can never overwrite a terminal status an
// operator Stop ('cancelled') already set in a race with the send loop finishing.
func (s *Store) MarkCampaignCompleted(c ctx, id string) error {
	_, err := s.db.Exec(c,
		`UPDATE campaigns SET status='completed', completed_at=now() WHERE id=$1 AND status='running'`, id)
	return err
}

// ScheduleCampaign sets a future launch time (and optional drip window end).
func (s *Store) ScheduleCampaign(c ctx, id string, launchAt time.Time, sendBy *time.Time) error {
	_, err := s.db.Exec(c,
		`UPDATE campaigns SET status='scheduled', launch_at=$2, send_by=$3 WHERE id=$1`,
		id, launchAt, sendBy)
	return err
}

// RenameCampaign changes only the campaign's display name. Its configuration
// (client, authorization, template, page, profile, targets) is unchanged; to
// reconfigure, delete the draft and recreate it.
func (s *Store) RenameCampaign(c ctx, id, name string) error {
	_, err := s.db.Exec(c, `UPDATE campaigns SET name=$2 WHERE id=$1`, id, name)
	return err
}

func (s *Store) SetCampaignArchived(c ctx, id string, archived bool) error {
	_, err := s.db.Exec(c, `UPDATE campaigns SET archived=$2 WHERE id=$1`, id, archived)
	return err
}

func (s *Store) DeleteCampaign(c ctx, id string) error {
	_, err := s.db.Exec(c, `DELETE FROM campaigns WHERE id=$1`, id)
	return err
}

// UpdateCampaignConfig reconfigures a DRAFT campaign. Guarded on status='draft' so
// a launched campaign's record stays faithful to what was actually sent. Returns
// true only if a draft was updated. Target groups are regenerated separately via
// DeleteResultsForCampaign + CreateResults.
func (s *Store) UpdateCampaignConfig(c ctx, id, name, authID, templateID, pageID, profileID, publicBaseURL, kind, redirectURL string, tunnelManaged bool) (bool, error) {
	tag, err := s.db.Exec(c,
		`UPDATE campaigns SET name=$2, authorization_id=$3, template_id=$4, landing_page_id=$5, sending_profile_id=$6, public_base_url=$7, tunnel_managed=$8, kind=$9, redirect_url=$10
		   WHERE id=$1 AND status='draft'`,
		id, name, authID, templateID, nullIfEmpty(pageID), profileID, publicBaseURL, tunnelManaged, kind, redirectURL)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// DeleteResultsForCampaign removes all per-target result rows of a campaign. Only
// used when reconfiguring a DRAFT campaign's target groups (nothing has been sent).
func (s *Store) DeleteResultsForCampaign(c ctx, campaignID string) error {
	_, err := s.db.Exec(c, `DELETE FROM campaign_results WHERE campaign_id=$1`, campaignID)
	return err
}

// CampaignGroupIDs returns the distinct group ids currently covered by a
// campaign's results, so the edit form can pre-check the right groups.
func (s *Store) CampaignGroupIDs(c ctx, campaignID string) ([]string, error) {
	rows, err := s.db.Query(c,
		`SELECT DISTINCT t.group_id FROM campaign_results r
		   JOIN targets t ON t.id = r.target_id WHERE r.campaign_id=$1`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// DueScheduledCampaigns returns scheduled campaigns whose launch time has arrived.
func (s *Store) DueScheduledCampaigns(c ctx, now time.Time) ([]string, error) {
	rows, err := s.db.Query(c,
		`SELECT id FROM campaigns
		  WHERE status='scheduled' AND launch_at IS NOT NULL AND launch_at <= $1 AND archived=FALSE`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ResetNonRespondersForResend re-queues (status back to 'scheduled') the results
// of targets who were sent the email but never opened it, so a follow-up send can
// be launched. Returns the number re-queued.
func (s *Store) ResetNonRespondersForResend(c ctx, campaignID string) (int64, error) {
	tag, err := s.db.Exec(c,
		`UPDATE campaign_results SET status='scheduled', sent_at=NULL
		   WHERE campaign_id=$1 AND status='sent'`, campaignID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// CampaignRow is a flattened list view.
type CampaignRow struct {
	ID         string
	Name       string
	ClientName string
	Status     string
	CreatedAt  time.Time
	Kind       string // "local" or "public"
}

// SetCampaignTunnel records a managed campaign's current tunnel URL (called by
// the tunnel manager every time cloudflared (re)starts and prints a new URL).
func (s *Store) SetCampaignTunnel(c ctx, id, url string) error {
	_, err := s.db.Exec(c,
		`UPDATE campaigns SET public_base_url=$2, tunnel_managed=true WHERE id=$1`, id, url)
	return err
}

// ListTunnelManagedCampaigns returns ids whose tunnels must be restarted on
// boot. Archived campaigns keep their stored URL but get no live tunnel — they
// no longer send, so a dead link there is harmless.
func (s *Store) ListTunnelManagedCampaigns(c ctx) ([]string, error) {
	rows, err := s.db.Query(c,
		`SELECT id FROM campaigns WHERE tunnel_managed=true AND archived=false`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
