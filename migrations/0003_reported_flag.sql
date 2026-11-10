-- PROPhish 0003 — make "reported" an independent fact, not a status that can
-- override 'submitted'/'clicked'. A target recognising the phish (good) must
-- never erase the record that they had clicked or submitted (bad).

ALTER TABLE campaign_results ADD COLUMN IF NOT EXISTS reported     BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE campaign_results ADD COLUMN IF NOT EXISTS reported_at  TIMESTAMPTZ;

-- Migrate any rows that used the old status='reported' into the flag, and move
-- them back to a real lifecycle status (they were at least 'sent').
UPDATE campaign_results SET reported = TRUE, reported_at = now() WHERE status = 'reported';
UPDATE campaign_results SET status = 'sent' WHERE status = 'reported';
