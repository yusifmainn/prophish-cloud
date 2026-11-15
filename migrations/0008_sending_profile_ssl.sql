-- Add implicit-TLS (SMTPS, port 465) support to sending profiles.
-- Existing profiles keep the STARTTLS behaviour (use_ssl = false): the mailer
-- connects in plaintext and upgrades with STARTTLS. When use_ssl is true the
-- connection is wrapped in TLS from the first byte (required on port 465).
ALTER TABLE sending_profiles
    ADD COLUMN IF NOT EXISTS use_ssl boolean NOT NULL DEFAULT false;
