-- +goose Up
-- The settings surface: the account's download preferences land as
-- columns on the users row (the appearance pair -- theme and locale
-- -- has existed since the core schema), and the sessions row gains
-- the creation-time address the device list renders. Existing rows
-- keep their behavior: the preference columns backfill with the
-- documented defaults, the address backfills as NULL (unknown, the
-- honest answer for a session created before the column existed).

ALTER TABLE users
    ADD COLUMN download_format text NOT NULL DEFAULT 'zip',
    ADD COLUMN download_include_videos boolean NOT NULL DEFAULT true,
    ADD COLUMN download_include_sidecars boolean NOT NULL DEFAULT false;

ALTER TABLE sessions
    ADD COLUMN ip_address text;

-- +goose Down
ALTER TABLE sessions DROP COLUMN IF EXISTS ip_address;
ALTER TABLE users
    DROP COLUMN IF EXISTS download_include_sidecars,
    DROP COLUMN IF EXISTS download_include_videos,
    DROP COLUMN IF EXISTS download_format;
