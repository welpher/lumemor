-- +goose Up
-- The content identifier a Live Photo's two halves share: the EXIF
-- ContentIdentifier (iOS) or MediaGroupUUID (some Android cameras),
-- stored on the asset row so the pairing can find the other half
-- without walking sidecars.
--
-- The key is (owner, identifier, media kind): the same identifier is
-- never matched across accounts, and the two halves of a pair always
-- differ in kind. Rows without an identifier (every plain photo and
-- video) leave the column NULL and stay out of the index.
ALTER TABLE assets ADD COLUMN live_photo_cid text;

CREATE INDEX assets_owner_live_photo_cid_idx
    ON assets (owner_id, live_photo_cid)
    WHERE live_photo_cid IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS assets_owner_live_photo_cid_idx;
ALTER TABLE assets DROP COLUMN IF EXISTS live_photo_cid;