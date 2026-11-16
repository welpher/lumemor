-- +goose Up
-- The position-row key fork: an uploaded (blob) row keeps the content
-- key, a directory-source (external) row takes the position key, and
-- the library faces dedup the DISPLAY instead of the storage.
--
-- Why the fork: a directory album mirrors its folder. Before this
-- migration the owner+checksum uniqueness answered a scan that met
-- content the owner already held by linking the EXISTING row into the
-- album -- so a photo uploaded first and scanned second appeared in
-- its read-only source album as the uploaded blob, and the album's
-- copy could not be deleted through the application (the source
-- would not lose a byte, yet the row's delete was the upload's).
-- Position is what a directory album holds: each walked file gets its
-- own row, keyed by where it lies.
--
-- The blob domain keeps 00008's total uniqueness unchanged: content
-- addressing means the physical file is the identity there, and the
-- trash re-upload three-state reasoning (a trashed row and a live
-- row may never share one file, or the expiry of the trashed one
-- deletes bytes the live one serves) still holds exactly as written.
-- The qualifier is only that it now speaks for the blob rows alone.
--
-- The external domain's key is (owner, file_path): one row per
-- position. The three-state shape it resolves is the same -- a
-- walked path that a trashed row still owns resolves through the
-- lookup, exactly as a re-upload of trashed content does -- while
-- two folders holding identical bytes are two positions, two rows.
--
-- Zero data rewrites: every existing row satisfies its domain's key
-- by construction (the previous total uniqueness was stricter), and
-- the directory albums' blob memberships -- the pre-fork scan's dedup
-- links -- are rebuilt idempotently by the next rescan's convergence
-- arm (the blob row keeps its sync membership and loses the
-- directory-album one).
DROP INDEX assets_owner_checksum_uq;

CREATE UNIQUE INDEX assets_owner_checksum_blob_uq
    ON assets (owner_id, checksum)
    WHERE storage_kind = 'blob';

CREATE UNIQUE INDEX assets_owner_file_path_external_uq
    ON assets (owner_id, file_path)
    WHERE storage_kind = 'external';

-- The owner+checksum lookup's own index, over every state and both
-- kinds: the upload's full-domain check (the content may live in a
-- scan's position row) and the library predicate's group probe both
-- read by content, not by the invariants above. The uniqueness no
-- longer implies it -- a partial unique index answers only the
-- queries that prove its predicate -- so it lands here.
CREATE INDEX assets_owner_checksum_idx
    ON assets (owner_id, checksum);

-- The library-representative predicate, centralized (the
-- caller_asset_visible precedent, migration 00010): a row represents
-- its content on the library faces -- timeline, My Photos, favorites,
-- archived, search, places, memories, people and their counts --
-- exactly when no better-ranked LIVE row shares its (owner, checksum)
-- group. The rank is the owner's plain words: a live blob row (the
-- upload, the app's own copy) beats every position row; among
-- position rows the earliest ingest wins; a dead row (trashed, or a
-- vanished source's orphan mark) never represents anything, so a
-- group's representative naturally changes when one is trashed or
-- restored with no maintenance step and no materialized flag.
--
-- The trash view is NOT a library face (it lists rows raw), and a
-- directory album's wall lists MEMBERS (its source is the truth --
-- every walked file is a tile), so neither filters through this.
-- A memory and My Photos are library faces despite being albums:
-- their member lists dedup like the timeline they feed.
--
-- STABLE/LANGUAGE sql keeps it a plan-time candidate for inlining;
-- each row's check is one probe of assets_owner_checksum_idx.
CREATE FUNCTION asset_library_representative(target uuid)
RETURNS boolean
STABLE
LANGUAGE sql
AS $$
    SELECT NOT EXISTS (
        SELECT 1
        FROM assets a
        JOIN assets x
          ON x.owner_id = a.owner_id
         AND x.checksum = a.checksum
         AND x.id <> a.id
         AND x.deleted_at IS NULL
         AND (
             x.storage_kind = 'blob'
             OR (
                 a.storage_kind = 'external'
                 AND (x.created_at, x.id) < (a.created_at, a.id)
             )
         )
        WHERE a.id = target
    )
$$;

-- +goose Down
-- The inverse rebuilds the previous model's single-row-per-content
-- state rather than merely restoring its index: the twins the fork
-- allowed (one uploaded blob plus a source folder's position copies)
-- fold back into the earliest-ranked row, and every album membership
-- a dropped position row fed re-links onto the keeper -- exactly the
-- linked-membership shape the pre-fork scan produced. Files are
-- untouched either way (a dropped position row never owned its
-- source file). Without this collapse the old total uniqueness could
-- not be created at all.
CREATE TEMPORARY TABLE position_row_keepers ON COMMIT DROP AS
SELECT id,
       first_value(id) OVER (
           PARTITION BY owner_id, checksum
           ORDER BY (storage_kind = 'blob') DESC, deleted_at NULLS FIRST, created_at, id
       ) AS keeper_id
FROM assets;

INSERT INTO album_items (album_id, asset_id, added_by, added_at)
SELECT ai.album_id, k.keeper_id, ai.added_by, ai.added_at
FROM album_items ai
JOIN position_row_keepers k ON k.id = ai.asset_id
WHERE k.keeper_id <> k.id
ON CONFLICT DO NOTHING;

DELETE FROM assets a
USING position_row_keepers k
WHERE a.id = k.id
  AND k.keeper_id <> k.id;

DROP FUNCTION IF EXISTS asset_library_representative(uuid);
DROP INDEX IF EXISTS assets_owner_checksum_idx;
DROP INDEX IF EXISTS assets_owner_file_path_external_uq;
DROP INDEX IF EXISTS assets_owner_checksum_blob_uq;

CREATE UNIQUE INDEX assets_owner_checksum_uq
    ON assets (owner_id, checksum);