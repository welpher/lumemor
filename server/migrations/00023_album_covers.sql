-- +goose Up
-- The cover's two stores. A cover is a SET of picked frames (four on
-- the album card, six on the memory band), not one photo: the owner's
-- picks are an ordered list whose order is the render order, and the
-- automatic group is a per-(album, ISO week) sampling frozen so the
-- week's group cannot move under a new upload.
--
-- The pinned set replaces the single albums.cover_asset_id column:
-- the migration carries every existing pin over as the first pick (a
-- one-frame pin is a partial set the reader fills), then the column
-- goes -- one truth, no dangling second store.
--
-- The week store is row-per-frame for the same reason the picks are:
-- an asset fold (the account removal's duplicate merge) rewrites a
-- single asset id without touching the rest of the group, and the
-- (album, week, position) primary key keeps a position single-valued.
-- A week row group is only ever the SERVER's sampling; the owner's
-- picks never land here.

CREATE TABLE album_cover_picks (
    album_id uuid NOT NULL REFERENCES albums (id) ON DELETE CASCADE,
    position smallint NOT NULL CHECK (position >= 0 AND position < 6),
    asset_id uuid NOT NULL REFERENCES assets (id) ON DELETE CASCADE,
    PRIMARY KEY (album_id, position),
    UNIQUE (album_id, asset_id)
);

INSERT INTO album_cover_picks (album_id, position, asset_id)
SELECT id, 0, cover_asset_id
FROM albums
WHERE cover_asset_id IS NOT NULL;

ALTER TABLE albums DROP COLUMN cover_asset_id;

CREATE TABLE album_cover_week_picks (
    album_id uuid NOT NULL REFERENCES albums (id) ON DELETE CASCADE,
    iso_week text NOT NULL,
    position smallint NOT NULL CHECK (position >= 0 AND position < 6),
    asset_id uuid NOT NULL REFERENCES assets (id) ON DELETE CASCADE,
    PRIMARY KEY (album_id, iso_week, position),
    UNIQUE (album_id, iso_week, asset_id)
);

-- +goose Down
DROP TABLE IF EXISTS album_cover_week_picks;
ALTER TABLE albums ADD COLUMN cover_asset_id uuid;
UPDATE albums
SET cover_asset_id = picks.asset_id
FROM album_cover_picks picks
WHERE picks.album_id = albums.id AND picks.position = 0;
DROP TABLE IF EXISTS album_cover_picks;