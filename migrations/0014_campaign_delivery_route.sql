-- Delivery route axis for the 4-mode delivery matrix (route × trust):
--
--   internal  — sender and recipient are in the same mail environment; delivery
--               takes seconds, so a REAL user can open the mail within seconds
--               of sending. Untrusted mode therefore uses SHORT scanner windows
--               (5s grace / 5s pair) so genuine opens are not discarded; trusted
--               mode uses no filtering at all.
--   external  — the message crosses an external SMTP relay/gateway; delivery
--               lags send by longer, so untrusted mode keeps the original LONG
--               windows (15s grace / 10s pair) that catch delivery-time
--               prefetch without a canary probe.
--
-- Default 'external' is the conservative choice (biased toward filtering
-- scanner noise rather than counting every fast open).
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS delivery_route TEXT NOT NULL DEFAULT 'external';

-- ADD CONSTRAINT has no IF NOT EXISTS in Postgres: guard via the catalog so a
-- re-run of the migrations stays idempotent.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'campaigns_delivery_route_chk') THEN
        ALTER TABLE campaigns ADD CONSTRAINT campaigns_delivery_route_chk
            CHECK (delivery_route IN ('internal','external'));
    END IF;
END $$;
