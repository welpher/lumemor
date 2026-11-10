-- +goose Up
-- Upload idempotency: the device-side asset identity, and the
-- owner+checksum uniqueness completed to cover the trash.

-- device_asset_id is the stable identity the uploading device assigned
-- to the file (the backup app's idempotency key). It is device-side
-- metadata on an otherwise device-agnostic table: only the live photo
-- row of a device asset carries it -- the Live Photo pairing video is
-- a linked component row and never owns a device identity.
ALTER TABLE assets ADD COLUMN device_asset_id text;

-- One live row per (owner, device asset): a re-upload of a device
-- asset that is still live maps to the existing row (the duplicate
-- response). Trashed rows release their device identity, so a
-- restore-or-save-as decision stays with the user.
CREATE UNIQUE INDEX assets_owner_device_asset_id_live_uq
    ON assets (owner_id, device_asset_id)
    WHERE deleted_at IS NULL AND device_asset_id IS NOT NULL;

-- The owner+checksum uniqueness becomes total: live rows AND trashed
-- rows share one index. The live-only partial index it replaces let a
-- re-upload of a trashed asset insert a second row, so no database
-- state could ever answer "this content is already in the trash" --
-- the trashed half of the upload idempotency was unreachable. With
-- the total index the insert conflicts regardless of the existing
-- row's state, and the handler looks the row up after the violation
-- to answer duplicate (deleted_at IS NULL) or trashed (deleted_at
-- set) straight from the committed state -- no check-then-insert
-- race. The save-as-new outcome of the trash re-upload decision
-- stays reachable: it removes the trashed row before inserting the
-- replacement, in one transaction.
DROP INDEX assets_owner_checksum_live_uq;
CREATE UNIQUE INDEX assets_owner_checksum_uq
    ON assets (owner_id, checksum);

-- +goose Down
CREATE UNIQUE INDEX assets_owner_checksum_live_uq
    ON assets (owner_id, checksum)
    WHERE deleted_at IS NULL;
DROP INDEX IF EXISTS assets_owner_checksum_uq;
DROP INDEX IF EXISTS assets_owner_device_asset_id_live_uq;
ALTER TABLE assets DROP COLUMN IF EXISTS device_asset_id;
