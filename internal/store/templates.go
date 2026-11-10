package store

import (
	"encoding/json"

	"prophish/internal/models"
)

func (s *Store) CreateTemplate(c ctx, clientID *string, name, subject, html, text string, headers map[string]string) (string, error) {
	hb, _ := json.Marshal(orEmptyMap(headers))
	var id string
	err := s.db.QueryRow(c,
		`INSERT INTO email_templates (client_id, name, subject, html_body, text_body, headers)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		clientID, name, subject, html, nullIfEmpty(text), hb).Scan(&id)
	return id, err
}

func (s *Store) ListTemplates(c ctx) ([]models.EmailTemplate, error) {
	rows, err := s.db.Query(c,
		`SELECT id, client_id, name, subject, html_body, coalesce(text_body,''), created_at
		   FROM email_templates ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.EmailTemplate
	for rows.Next() {
		var t models.EmailTemplate
		if err := rows.Scan(&t.ID, &t.ClientID, &t.Name, &t.Subject, &t.HTMLBody, &t.TextBody, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// GetTemplate returns the template with its custom headers and attachments loaded.
func (s *Store) GetTemplate(c ctx, id string) (*models.EmailTemplate, error) {
	var t models.EmailTemplate
	var hb []byte
	err := s.db.QueryRow(c,
		`SELECT id, client_id, name, subject, html_body, coalesce(text_body,''), headers, created_at
		   FROM email_templates WHERE id=$1`, id).
		Scan(&t.ID, &t.ClientID, &t.Name, &t.Subject, &t.HTMLBody, &t.TextBody, &hb, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	t.Headers = map[string]string{}
	_ = json.Unmarshal(hb, &t.Headers)

	at, err := s.ListAttachments(c, id)
	if err != nil {
		return nil, err
	}
	t.Attachments = at
	return &t, nil
}

// UpdateTemplate edits a template's content (attachments are managed separately).
func (s *Store) UpdateTemplate(c ctx, id, name, subject, html, text string, headers map[string]string) error {
	hb, _ := json.Marshal(orEmptyMap(headers))
	_, err := s.db.Exec(c,
		`UPDATE email_templates SET name=$2, subject=$3, html_body=$4, text_body=$5, headers=$6 WHERE id=$1`,
		id, name, subject, html, nullIfEmpty(text), hb)
	return err
}

// CountCampaignsUsingTemplate reports how many campaigns reference this template.
// campaigns.template_id has no ON DELETE action, so a template still in use
// cannot be deleted — the caller checks this first to show a clear message
// instead of surfacing a raw foreign-key error.
func (s *Store) CountCampaignsUsingTemplate(c ctx, id string) (int, error) {
	var n int
	err := s.db.QueryRow(c, `SELECT count(*) FROM campaigns WHERE template_id=$1`, id).Scan(&n)
	return n, err
}

// DeleteTemplate removes a template; its attachments cascade. Call
// CountCampaignsUsingTemplate first — this errors if a campaign still uses it.
func (s *Store) DeleteTemplate(c ctx, id string) error {
	_, err := s.db.Exec(c, `DELETE FROM email_templates WHERE id=$1`, id)
	return err
}

func (s *Store) AddAttachment(c ctx, templateID, filename, contentType string, content []byte) error {
	_, err := s.db.Exec(c,
		`INSERT INTO template_attachments (template_id, filename, content_type, content)
		 VALUES ($1,$2,$3,$4)`, templateID, filename, contentType, content)
	return err
}

func (s *Store) ListAttachments(c ctx, templateID string) ([]models.Attachment, error) {
	rows, err := s.db.Query(c,
		`SELECT id, template_id, filename, content_type, content
		   FROM template_attachments WHERE template_id=$1 ORDER BY filename`, templateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Attachment
	for rows.Next() {
		var a models.Attachment
		if err := rows.Scan(&a.ID, &a.TemplateID, &a.Filename, &a.ContentType, &a.Content); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) DeleteAttachment(c ctx, id string) error {
	_, err := s.db.Exec(c, `DELETE FROM template_attachments WHERE id=$1`, id)
	return err
}

func orEmptyMap(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}
	return m
}
