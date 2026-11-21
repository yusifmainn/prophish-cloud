-- Raw request log: one row for EVERY tracking request (/p pixel, /t click, /c
-- canary, /s submit), with the full request fingerprint (IP, User-Agent) and —
-- crucially — the classification decision the public server made for it
-- (outcome + machine-readable reason). The events table only keeps deduplicated,
-- post-decision entries, so when a hit "didn't count" there was no way to audit
-- WHY. This table is that audit trail, and it is also the dataset for tuning the
-- scanner windows: every probe/hit interval is preserved here from day one.
--
-- outcome: 'counted'   — advanced the result status (first transition)
--          'duplicate' — human-looking but status already at/ past this stage
--          'suspected' — suppressed, but logged visibly (scanner pair window)
--          'ignored'   — dropped silently by UA filter or post-send grace window
--          'probe'     — canary hit; marks/extends the scanner probe for the result
-- reason:  machine token: 'counted' | 'duplicate' | 'ua' | 'grace_window' |
--          'pair_window' | 'canary'. Kept in English on purpose — it is a
--          diagnostic code, not user copy.
CREATE TABLE IF NOT EXISTS tracking_hits (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    result_id   UUID NOT NULL REFERENCES campaign_results(id) ON DELETE CASCADE,
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    route       TEXT NOT NULL CHECK (route IN ('open','click','canary','submit')),
    ip          INET,
    user_agent  TEXT,
    outcome     TEXT NOT NULL CHECK (outcome IN ('counted','duplicate','suspected','ignored','probe')),
    reason      TEXT NOT NULL DEFAULT '',
    promoted    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_tracking_hits_campaign ON tracking_hits(campaign_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_tracking_hits_result ON tracking_hits(result_id, created_at);
