-- Per-target gender, used to render the correct Azerbaijani address form in
-- email templates: 'm' -> "Hörmətli <ad> bəy,", 'f' -> "Hörmətli <ad> xanım,",
-- '' (legacy rows) -> gender-neutral "Hörmətli <ad>,". CSV imports carry it as
-- the 5th column (f/m); the UI offers a per-target selector.
ALTER TABLE targets ADD COLUMN IF NOT EXISTS gender TEXT NOT NULL DEFAULT ''
    CHECK (gender IN ('', 'f', 'm'));
