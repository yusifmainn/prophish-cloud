package store

import (
	"encoding/json"

	"secdrill/internal/models"
)

func (s *Store) CreateSendingProfile(c ctx, name, host string, port int, user, fromAddr string, passwordEnc []byte, headers map[string]string) (string, error) {
	hb, _ := json.Marshal(orEmptyMap(headers))
	var id string
	err := s.db.QueryRow(c,
		`INSERT INTO sending_profiles (name, smtp_host, smtp_port, smtp_username, from_address, password_enc, headers)
		 VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		name, host, port, nullIfEmpty(user), fromAddr, passwordEnc, hb).Scan(&id)
	return id, err
}

func (s *Store) ListSendingProfiles(c ctx) ([]models.SendingProfile, error) {
	rows, err := s.db.Query(c,
		`SELECT id, name, smtp_host, smtp_port, coalesce(smtp_username,''), from_address,
		        (password_enc IS NOT NULL AND length(password_enc) > 0), headers, created_at
		   FROM sending_profiles ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.SendingProfile
	for rows.Next() {
		var p models.SendingProfile
		var hb []byte
		if err := rows.Scan(&p.ID, &p.Name, &p.SMTPHost, &p.SMTPPort, &p.SMTPUser, &p.FromAddress, &p.HasPassword, &hb, &p.CreatedAt); err != nil {
			return nil, err
		}
		p.Headers = map[string]string{}
		_ = json.Unmarshal(hb, &p.Headers)
		out = append(out, p)
	}
	return out, rows.Err()
}

// UpdateSendingProfile edits a profile. When replacePassword is false the stored
// password is left untouched (the edit form leaves the field blank to keep it);
// when true, password_enc is set to passwordEnc — pass a nil blob to clear it.
func (s *Store) UpdateSendingProfile(c ctx, id, name, host string, port int, user, fromAddr string, passwordEnc []byte, replacePassword bool, headers map[string]string) error {
	hb, _ := json.Marshal(orEmptyMap(headers))
	if replacePassword {
		_, err := s.db.Exec(c,
			`UPDATE sending_profiles
			    SET name=$2, smtp_host=$3, smtp_port=$4, smtp_username=$5, from_address=$6, password_enc=$7, headers=$8
			  WHERE id=$1`,
			id, name, host, port, nullIfEmpty(user), fromAddr, passwordEnc, hb)
		return err
	}
	_, err := s.db.Exec(c,
		`UPDATE sending_profiles
		    SET name=$2, smtp_host=$3, smtp_port=$4, smtp_username=$5, from_address=$6, headers=$7
		  WHERE id=$1`,
		id, name, host, port, nullIfEmpty(user), fromAddr, hb)
	return err
}

// CountCampaignsUsingProfile reports how many campaigns reference this profile.
// campaigns.sending_profile_id has no ON DELETE action, so a profile still in
// use cannot be deleted — the caller checks this to show a clear message.
func (s *Store) CountCampaignsUsingProfile(c ctx, id string) (int, error) {
	var n int
	err := s.db.QueryRow(c, `SELECT count(*) FROM campaigns WHERE sending_profile_id=$1`, id).Scan(&n)
	return n, err
}

// DeleteSendingProfile removes a profile. Call CountCampaignsUsingProfile first —
// this errors if a campaign still references it.
func (s *Store) DeleteSendingProfile(c ctx, id string) error {
	_, err := s.db.Exec(c, `DELETE FROM sending_profiles WHERE id=$1`, id)
	return err
}

// GetSendingProfileWithSecret returns the profile, its custom headers, and the
// encrypted password blob.
func (s *Store) GetSendingProfileWithSecret(c ctx, id string) (*models.SendingProfile, []byte, error) {
	var p models.SendingProfile
	var enc, hb []byte
	err := s.db.QueryRow(c,
		`SELECT id, name, smtp_host, smtp_port, coalesce(smtp_username,''), from_address, headers, created_at, password_enc
		   FROM sending_profiles WHERE id=$1`, id).
		Scan(&p.ID, &p.Name, &p.SMTPHost, &p.SMTPPort, &p.SMTPUser, &p.FromAddress, &hb, &p.CreatedAt, &enc)
	if err != nil {
		return nil, nil, err
	}
	p.Headers = map[string]string{}
	_ = json.Unmarshal(hb, &p.Headers)
	return &p, enc, nil
}
