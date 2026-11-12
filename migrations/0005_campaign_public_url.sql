-- Per-campaign public base URL override. Empty string = fall back to the
-- global PROPHISH_PUBLIC_BASE_URL at send time. Needed because quick-tunnel
-- (trycloudflare) URLs change on tunnel restart, and campaigns created at
-- different times may legitimately point at different URLs.
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS public_base_url text NOT NULL DEFAULT '';
