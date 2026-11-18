-- Per-campaign switch for open tracking. When disabled, the sending engine
-- does not inject the 1x1 open-tracking pixel into the emails, so no
-- "opened" events are recorded. Useful where mail-security gateways
-- pre-fetch and cache images, making open stats meaningless; click and
-- submission tracking are unaffected. Existing campaigns keep tracking opens.
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS track_opens boolean NOT NULL DEFAULT TRUE;
