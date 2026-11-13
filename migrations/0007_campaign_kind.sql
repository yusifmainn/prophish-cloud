-- Campaign "kind" splits campaigns into two delivery styles:
--   'local'  — the classic flow: a click serves a landing page hosted by the
--              public server (optionally with a fact-only form), exactly as before.
--   'public' — a click is tracked and then 302-redirected to a real external
--              website (campaigns.redirect_url). No landing page is hosted and no
--              form is shown, so there is nothing to submit. The open pixel and
--              click tracking are unchanged (both live on our own public server),
--              so sent → opened → clicked still work identically.
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS kind text NOT NULL DEFAULT 'local';

-- Destination for a 'public' campaign. Empty for 'local' campaigns.
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS redirect_url text NOT NULL DEFAULT '';

-- 'public' campaigns have no landing page, so landing_page_id must be nullable.
-- Existing rows are all 'local' with a landing page set, so this only relaxes
-- the column; it does not rewrite any data.
ALTER TABLE campaigns ALTER COLUMN landing_page_id DROP NOT NULL;

-- Integrity guard: a local campaign must have a landing page; a public campaign
-- must have a redirect URL (and, by construction, no landing page). All existing
-- rows satisfy the local branch (landing_page_id was NOT NULL until now).
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'campaigns_kind_destination_chk'
  ) THEN
    ALTER TABLE campaigns ADD CONSTRAINT campaigns_kind_destination_chk CHECK (
      (kind = 'local'  AND landing_page_id IS NOT NULL) OR
      (kind = 'public' AND redirect_url <> '')
    );
  END IF;
END $$;
