-- Scanner-canary correlation. The sending engine hides an invisible canary link
-- in every HTML email. Mail-security scanners fetch EVERY link in a message, so a
-- canary hit identifies a scanned recipient no matter WHEN the scan happens —
-- unlike a fixed post-send grace window, which breaks down when delivery itself
-- is spread over minutes (drip sends, queueing, greylisting). A real link/form
-- hit inside the cluster window of a canary probe is treated as part of the same
-- automated crawl: the result status does not advance, and the suppressed hit is
-- logged as a suspected_* event so nothing is silently hidden from the operator.
ALTER TABLE campaign_results ADD COLUMN IF NOT EXISTS scanner_probe_at TIMESTAMPTZ;

-- New event types for hits suppressed by the canary correlation.
ALTER TABLE events DROP CONSTRAINT IF EXISTS events_type_check;
ALTER TABLE events ADD CONSTRAINT events_type_check CHECK (type IN
    ('email_sent','email_opened','link_clicked','data_submitted','training_completed','error',
     'suspected_click','suspected_submit'));
