-- +goose Up
-- Albums-as-sources readiness (E4): the scan snapshot column of the
-- rescan diff. The albums side of the milestone needed no new
-- columns -- root_path and last_scanned_at landed with the album
-- schema (00005), the album_type enum already carries 'directory'
-- (00002), and assets.storage_kind already carries 'external'
-- (00002) -- so the only schema work left is the snapshot the
-- directory rescan compares a file's freshness against: the
-- filesystem mtime. The size half of the snapshot rides the existing
-- assets.file_size column (the data-model schema's asset row has no
-- second size): the rescan stamps both at ingest and compares both
-- on the next run, deciding whether an external asset's metadata
-- pipeline needs a rerun (the checksum is read only when the
-- snapshot changed, never pre-read).
--
-- The honest default for existing rows is NULL: "never
-- scan-snapshotted". Every existing asset is a blob upload (external
-- assets arrive with the rescan job, which stamps the pair at
-- ingest), and blob assets are never rescan-diffed, so no existing
-- row ever owes a snapshot value -- NULL is the truthful state, not
-- a missing one.

ALTER TABLE assets
    ADD COLUMN file_mtime timestamptz;

-- +goose Down
ALTER TABLE assets
    DROP COLUMN IF EXISTS file_mtime;
