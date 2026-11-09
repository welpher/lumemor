-- +goose Up
-- The album autonomy facts (E6, the deletion chain of §7.9): two
-- columns on the directory album's row.
--
-- source_writable is the creation-time write probe's answer, stored
-- as a FACT the row owns forever: at creation the server writes one
-- probe file into the source directory, reads it back, removes it,
-- and records whether all three steps landed. The probe's failure is
-- NOT a creation failure (a readable but read-only directory is a
-- legitimate album source); the fact drives the deletability
-- affordance's render condition and the creation response's honest
-- "photos of this directory cannot be deleted through the
-- application" feedback. The truth is discovered exactly once -- no
-- runtime re-probing exists, by the same pin that keeps the mounts
-- free of ro/rw mode bits.
--
-- The honest default for rows that predate the probe is TRUE, the
-- E4-era behavior: those albums never ran a probe, and their deletes
-- were offered and answered by the operating system's own authority
-- (a read-only source surfaced at move time with the honest error,
-- and it still does -- the probe only front-loads the knowledge).
-- FALSE would silently retire delete affordances every existing
-- album still honestly has.
--
-- allow_app_deletion is the owner's deletability switch: on by
-- default (a writable source's deletions are the application's to
-- perform), turned off by the owner who manages the file layer
-- themselves. Together with source_writable and locked it derives
-- the wall's delete affordance (owner AND writable AND switch AND
-- unlocked), computed at read time; the switch is writable through
-- the property patch, gated by the lock like every property field.
ALTER TABLE albums
    ADD COLUMN source_writable boolean NOT NULL DEFAULT true,
    ADD COLUMN allow_app_deletion boolean NOT NULL DEFAULT true;

-- +goose Down
-- Round-trips the columns with their defaults; the probe facts and
-- the switch states the Up-migrated rows carried are lost -- the
-- re-migration re-probes nothing, so every album comes back
-- writable-and-deletable (the E4-era posture) until it is recreated.
ALTER TABLE albums
    DROP COLUMN IF EXISTS source_writable,
    DROP COLUMN IF EXISTS allow_app_deletion;
