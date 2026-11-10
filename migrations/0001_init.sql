-- ProPhish initial schema
-- Design principles:
--   * Multi-tenant: every client's data is scoped by client_id and isolated.
--   * Privacy-safe: we NEVER store credentials or form values submitted by targets.
--     Only the FACT that a submission happened is recorded (see events.type = 'data_submitted').
--   * Every campaign MUST reference a valid authorization (written client consent / RoE).

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Platform operators (your security team). NOT the phishing targets.
CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,           -- bcrypt; plaintext passwords never stored
    role          TEXT NOT NULL DEFAULT 'operator' CHECK (role IN ('admin','operator')),
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_login_at TIMESTAMPTZ
);

-- Client organizations you have a contract with.
CREATE TABLE IF NOT EXISTS clients (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL,
    contact_email TEXT,
    notes         TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Written authorization / Rules of Engagement for a client. A campaign cannot launch
-- without an authorization whose validity window covers the launch time.
CREATE TABLE IF NOT EXISTS authorizations (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id     UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    scope         TEXT NOT NULL,            -- allowed domains / targets description
    authorized_by TEXT NOT NULL,            -- name/title of the client signatory
    document_ref  TEXT,                     -- reference to the signed contract/RoE file
    valid_from    TIMESTAMPTZ NOT NULL,
    valid_to      TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (valid_to > valid_from)
);
CREATE INDEX IF NOT EXISTS idx_authorizations_client ON authorizations(client_id);

-- Target groups (e.g. a department) belonging to a client.
CREATE TABLE IF NOT EXISTS groups (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id  UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_groups_client ON groups(client_id);

-- Individual employees who will receive a simulated email.
CREATE TABLE IF NOT EXISTS targets (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id   UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    email      TEXT NOT NULL,
    first_name TEXT,
    last_name  TEXT,
    position   TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (group_id, email)
);
CREATE INDEX IF NOT EXISTS idx_targets_group ON targets(group_id);

-- Email templates. client_id NULL = shared/global template.
CREATE TABLE IF NOT EXISTS email_templates (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id  UUID REFERENCES clients(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    subject    TEXT NOT NULL,
    html_body  TEXT NOT NULL,
    text_body  TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Landing pages shown after a click.
-- IMPORTANT: there is intentionally NO column to store submitted credentials.
-- When a page has a form, we only record that a submission occurred, then redirect
-- (typically to an awareness/training page). No field VALUES are ever persisted.
CREATE TABLE IF NOT EXISTS landing_pages (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id     UUID REFERENCES clients(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    html_body     TEXT NOT NULL,
    redirect_url  TEXT,                     -- where the target is sent after interacting
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- SMTP sending profiles. password_enc is encrypted at the application layer.
CREATE TABLE IF NOT EXISTS sending_profiles (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL,
    smtp_host     TEXT NOT NULL,
    smtp_port     INT  NOT NULL DEFAULT 587,
    smtp_username TEXT,
    password_enc  BYTEA,                    -- app-encrypted; never plaintext
    from_address  TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS campaigns (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id          UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    authorization_id   UUID NOT NULL REFERENCES authorizations(id),
    name               TEXT NOT NULL,
    template_id        UUID NOT NULL REFERENCES email_templates(id),
    landing_page_id    UUID NOT NULL REFERENCES landing_pages(id),
    sending_profile_id UUID NOT NULL REFERENCES sending_profiles(id),
    status             TEXT NOT NULL DEFAULT 'draft'
                       CHECK (status IN ('draft','scheduled','running','completed','cancelled')),
    launch_at          TIMESTAMPTZ,
    created_by         UUID REFERENCES users(id),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_campaigns_client ON campaigns(client_id);

-- One row per target per campaign. result_token is the unique, unguessable id used
-- in tracking links/pixels so we never put an email address in a URL.
CREATE TABLE IF NOT EXISTS campaign_results (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id  UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    target_id    UUID NOT NULL REFERENCES targets(id),
    result_token TEXT NOT NULL UNIQUE,
    status       TEXT NOT NULL DEFAULT 'scheduled'
                 CHECK (status IN ('scheduled','sent','opened','clicked','submitted','error')),
    sent_at      TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_results_campaign ON campaign_results(campaign_id);

-- Immutable event log. Deliberately contains NO submitted field values / credentials.
CREATE TABLE IF NOT EXISTS events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    result_id   UUID REFERENCES campaign_results(id) ON DELETE CASCADE,
    type        TEXT NOT NULL CHECK (type IN
                 ('email_sent','email_opened','link_clicked','data_submitted','training_completed','error')),
    ip          INET,
    user_agent  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_events_campaign ON events(campaign_id);
CREATE INDEX IF NOT EXISTS idx_events_result ON events(result_id);
