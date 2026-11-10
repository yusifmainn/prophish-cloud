-- SecDrill 0004 — audit log.
--
-- An append-only record of security-relevant OPERATOR actions (who did what,
-- when, from where). This is distinct from the `events` table, which records
-- TARGET behaviour (opens/clicks). The audit log exists for accountability and
-- legal/forensic evidence: which operator launched, stopped, or deleted a
-- campaign, created an authorization (RoE), managed users, or signed in.
--
-- Privacy: this table NEVER stores secrets (no passwords, API keys, SMTP
-- credentials). `detail` is a short human description only.
CREATE TABLE IF NOT EXISTS audit_log (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- actor_id is set NULL if the operator account is later deleted; actor_email
    -- is denormalized so the actor is still identifiable after that happens, and
    -- so a pre-authentication action (a failed login) can still name the attempt.
    actor_id     UUID REFERENCES users(id) ON DELETE SET NULL,
    actor_email  TEXT,
    action       TEXT NOT NULL,   -- e.g. 'campaign.launch', 'authorization.create', 'auth.login_failed'
    target_type  TEXT,            -- e.g. 'campaign', 'user', 'client'
    target_id    TEXT,            -- id (or a human label) of the affected object
    detail       TEXT,            -- short human description; NEVER a secret value
    ip           INET,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_log(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_actor   ON audit_log(actor_id);
