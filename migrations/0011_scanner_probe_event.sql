-- Log every scanner-canary probe as an event. The scanner_probe_at column only
-- keeps the LATEST probe, which erases the history needed to measure the real
-- interval between the canary fetch and the scanner's real-link fetch — the
-- number the canary cluster window must cover. With each probe timestamped in
-- the events log, that interval becomes measurable per episode (canary hit vs
-- the next link_clicked/suspected_click for the same result).
ALTER TABLE events DROP CONSTRAINT IF EXISTS events_type_check;
ALTER TABLE events ADD CONSTRAINT events_type_check CHECK (type IN
    ('email_sent','email_opened','link_clicked','data_submitted','training_completed','error',
     'suspected_click','suspected_submit','scanner_probe'));
