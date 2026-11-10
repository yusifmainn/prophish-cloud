package store

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"prophish/internal/models"
)

// CreateResultsForCampaign materializes one result row per target across all groups
// of the campaign's client that were selected. Here we take explicit group IDs.
func (s *Store) CreateResults(c ctx, campaignID string, targetIDs []string, tokens []string) error {
	if len(targetIDs) != len(tokens) {
		return errors.New("targetIDs/tokens length mismatch")
	}
	b := &pgx.Batch{}
	for i := range targetIDs {
		b.Queue(
			`INSERT INTO campaign_results (campaign_id, target_id, result_token) VALUES ($1,$2,$3)`,
			campaignID, targetIDs[i], tokens[i])
	}
	br := s.db.SendBatch(c, b)
	defer br.Close()
	for range targetIDs {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

// PendingResults returns results that still need an email sent, with target + campaign data.
func (s *Store) PendingResults(c ctx, campaignID string) ([]SendItem, error) {
	rows, err := s.db.Query(c,
		`SELECT r.id, r.result_token, t.email, coalesce(t.first_name,''), coalesce(t.last_name,''), coalesce(t.position,'')
		   FROM campaign_results r JOIN targets t ON t.id = r.target_id
		  WHERE r.campaign_id=$1 AND r.status='scheduled'`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SendItem
	for rows.Next() {
		var it SendItem
		if err := rows.Scan(&it.ResultID, &it.Token, &it.Email, &it.First, &it.Last, &it.Position); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (s *Store) MarkResultSent(c ctx, resultID string) error {
	_, err := s.db.Exec(c,
		`UPDATE campaign_results SET status='sent', sent_at=now() WHERE id=$1 AND status='scheduled'`, resultID)
	return err
}

func (s *Store) MarkResultError(c ctx, resultID string) error {
	_, err := s.db.Exec(c, `UPDATE campaign_results SET status='error' WHERE id=$1`, resultID)
	return err
}

// ResultByToken resolves a tracking token to its result + campaign (for the public server).
func (s *Store) ResultByToken(c ctx, token string) (*Tracked, error) {
	var t Tracked
	err := s.db.QueryRow(c,
		`SELECT r.id, r.campaign_id, r.status, c.landing_page_id
		   FROM campaign_results r JOIN campaigns c ON c.id = r.campaign_id
		  WHERE r.result_token=$1`, token).
		Scan(&t.ResultID, &t.CampaignID, &t.Status, &t.LandingPageID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &t, err
}

// statusRank defines the forward-only ordering of a result's lifecycle.
var statusRank = map[string]int{
	"scheduled": 0, "sent": 1, "opened": 2, "clicked": 3, "submitted": 4, "error": -1,
}

// AdvanceStatus only moves a result forward (scheduled<sent<opened<clicked<submitted),
// so a later "opened" ping can never downgrade a result that already "clicked".
// It returns whether the status actually changed, so callers can record a
// tracking event only on the FIRST transition into a stage (dedup / anti-flood).
func (s *Store) AdvanceStatus(c ctx, resultID, newStatus string) (bool, error) {
	tx, err := s.db.Begin(c)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(c)

	var cur string
	if err := tx.QueryRow(c, `SELECT status FROM campaign_results WHERE id=$1 FOR UPDATE`, resultID).Scan(&cur); err != nil {
		return false, err
	}
	changed := false
	if statusRank[newStatus] > statusRank[cur] {
		if _, err := tx.Exec(c, `UPDATE campaign_results SET status=$2 WHERE id=$1`, resultID, newStatus); err != nil {
			return false, err
		}
		changed = true
	}
	if err := tx.Commit(c); err != nil {
		return false, err
	}
	return changed, nil
}

// TargetEmailByToken returns the email address of the target a tracking token
// belongs to (used to verify a reported email actually came from that target).
func (s *Store) TargetEmailByToken(c ctx, token string) (string, error) {
	var email string
	err := s.db.QueryRow(c,
		`SELECT t.email FROM campaign_results r JOIN targets t ON t.id = r.target_id
		  WHERE r.result_token = $1`, token).Scan(&email)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return email, err
}

// MarkReportedByToken records that the target reported the phish — as an
// INDEPENDENT flag, never overwriting the lifecycle status, so a report can't
// erase the fact that the target also clicked or submitted. Idempotent: the
// event is logged only the first time the flag flips.
func (s *Store) MarkReportedByToken(c ctx, token, reporter string) error {
	t, err := s.ResultByToken(c, token)
	if err != nil {
		return err
	}
	rid := t.ResultID
	tag, err := s.db.Exec(c,
		`UPDATE campaign_results SET reported=TRUE, reported_at=now()
		   WHERE id=$1 AND reported=FALSE`, rid)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return nil // already reported; don't duplicate the event/row
	}
	_ = s.RecordEvent(c, t.CampaignID, &rid, "email_reported", "", "")
	_, err = s.db.Exec(c,
		`INSERT INTO reported_emails (campaign_id, result_id, reporter_email) VALUES ($1,$2,$3)`,
		t.CampaignID, rid, nullIfEmpty(reporter))
	return err
}

// SendItem carries everything the mailer needs for one recipient.
type SendItem struct {
	ResultID string
	Token    string
	Email    string
	First    string
	Last     string
	Position string
}

// Tracked is the public-server view of a tracking token.
type Tracked struct {
	ResultID      string
	CampaignID    string
	Status        string
	LandingPageID string
}

var _ = models.CampaignResult{}
