-- PROPhish 0002 — feature parity additions.

-- API keys for operators (REST API auth).
ALTER TABLE users ADD COLUMN IF NOT EXISTS api_key TEXT UNIQUE;

-- Campaign scheduling + lifecycle.
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS send_by      TIMESTAMPTZ; -- gradual-send window end
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS completed_at TIMESTAMPTZ;
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS archived     BOOLEAN NOT NULL DEFAULT FALSE;

-- Custom email headers on templates and sending profiles (merged at send time).
ALTER TABLE email_templates  ADD COLUMN IF NOT EXISTS headers JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE sending_profiles ADD COLUMN IF NOT EXISTS headers JSONB NOT NULL DEFAULT '{}'::jsonb;

-- Email attachments per template.
CREATE TABLE IF NOT EXISTS template_attachments (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id  UUID NOT NULL REFERENCES email_templates(id) ON DELETE CASCADE,
    filename     TEXT NOT NULL,
    content_type TEXT NOT NULL,
    content      BYTEA NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_attach_template ON template_attachments(template_id);

-- Global IMAP mailbox config for ingesting user-reported phishing emails.
CREATE TABLE IF NOT EXISTS imap_config (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    host         TEXT NOT NULL,
    port         INT  NOT NULL DEFAULT 993,
    username     TEXT NOT NULL,
    password_enc BYTEA,
    use_tls      BOOLEAN NOT NULL DEFAULT TRUE,
    folder       TEXT NOT NULL DEFAULT 'INBOX',
    enabled      BOOLEAN NOT NULL DEFAULT FALSE,
    poll_seconds INT NOT NULL DEFAULT 60,
    last_uid     BIGINT NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Emails employees reported as suspicious (the good outcome).
CREATE TABLE IF NOT EXISTS reported_emails (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id    UUID REFERENCES campaigns(id) ON DELETE SET NULL,
    result_id      UUID REFERENCES campaign_results(id) ON DELETE SET NULL,
    reporter_email TEXT,
    subject        TEXT,
    reported_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_reported_campaign ON reported_emails(campaign_id);

-- Allow new event types and a 'reported' result status.
ALTER TABLE events DROP CONSTRAINT IF EXISTS events_type_check;
ALTER TABLE events ADD CONSTRAINT events_type_check CHECK (type IN
  ('email_sent','email_opened','link_clicked','data_submitted','training_completed','email_reported','error'));

ALTER TABLE campaign_results DROP CONSTRAINT IF EXISTS campaign_results_status_check;
ALTER TABLE campaign_results ADD CONSTRAINT campaign_results_status_check CHECK (status IN
  ('scheduled','sent','opened','clicked','submitted','reported','error'));
