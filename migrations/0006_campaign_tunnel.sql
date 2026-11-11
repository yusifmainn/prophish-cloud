-- Per-campaign quick tunnels: tunnel_managed=true means the app spawns and
-- supervises its own cloudflared process for this campaign and keeps
-- public_base_url in sync with the current tunnel URL.
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS tunnel_managed boolean NOT NULL DEFAULT false;
