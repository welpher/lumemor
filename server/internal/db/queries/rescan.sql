-- Rescan queries: the three-arm diff's database half.
-- The rescan worker walks a directory album's source root on the
-- filesystem and reconciles it against the album's LIVE external
-- members -- the rows the previous scans (or this run's earlier
-- arms) left pointing at real files. Trashed members never enter
-- the diff: the orphan arm's pinned semantics (2026-11-08) hold
-- that a vanished file's row is marked deleted and STAYS marked --
-- no missing state, no revival -- so a deleted row is not a
-- candidate for matching, and a file that returns is ingested as a
-- NEW row by the add arm rather than matched to the old one.
--
-- The walk's matching key is the exact file path: the album row
-- stores the RESOLVED absolute root (symlinks evaluated at
-- validation time) and the walk never follows symlinks, so a
-- member's file_path and a walk entry's path are both plain
-- absolute paths of the same tree, and string equality IS identity.
--
-- The freshness comparison is the mtime+size snapshot (00014):
-- file_mtime landed as the scan's own column, the size half rides
-- the existing assets.file_size (the data-model schema's asset row
-- has no second size). The checksum is read only when the snapshot
-- changed -- never pre-read -- and re-lands on the row so the
-- owner+checksum identity and the serving ETags stay truthful
-- about the bytes the file now holds.

-- name: GetRescanAlbum :one
-- The rescan worker's subject read: the directory album's owner and
-- resolved root. The type arm is part of the query on purpose: the
-- enqueue's gate proved the type at request time, but the worker
-- runs later and against whatever the row is now -- a row that
-- stopped being a directory album (nothing in v1 can do this, the
-- delete face is memory-only) answers no row and the run closes
-- failed instead of walking a source the album no longer claims.
SELECT owner_id, root_path
FROM albums
WHERE id = $1
  AND type = 'directory';

-- name: CountLiveAlbumExternalMembers :one
-- The enqueue-time total of the rescan run: the album's live
-- external member count, the half of the diff the database already
-- knows. The walk's own half (the files on disk) is discovered by
-- the worker, which sizes the row's total to the full diff once
-- the walk has listed the root -- the started event echoes THIS
-- count (the durable truth the endpoint handed out), the progress
-- events carry the sized one.
SELECT count(*)
FROM album_items ai
JOIN assets a ON a.id = ai.asset_id
WHERE ai.album_id = $1
  AND a.deleted_at IS NULL
  AND a.storage_kind = 'external';

-- name: ListAlbumExternalMembers :many
-- The diff's database side: every live external member's scan
-- identity -- the path the walk matches on, the mtime+size snapshot
-- the freshness comparison reads, and the media kind the arms'
-- event payloads and pipeline dispatch carry. The checksum stays
-- unselected on purpose -- the pin is "never pre-read": the
-- comparison is the snapshot alone, and the checksum is read only
-- inside the change arm's full verification. One statement holds
-- the whole set: the diff's matching structure is a map keyed by
-- path, and a file on disk is "new" only when NO member matches, so
-- paging would only postpone the same resident set (a family-scale
-- album is tens of thousands of small rows, the ingest it stands
-- for is the real bound).
SELECT a.id, a.file_path, a.media_kind, a.file_size, a.file_mtime
FROM album_items ai
JOIN assets a ON a.id = ai.asset_id
WHERE ai.album_id = $1
  AND a.deleted_at IS NULL
  AND a.storage_kind = 'external';

-- name: CreateExternalAsset :one
-- The add arm's reference-style ingest insert: the row points AT
-- the file where it lies (storage_kind 'external', file_path the
-- real absolute path) -- zero copy, zero move. The column pins the
-- pipeline's shape the way CreateAsset pins the upload's: taken_at
-- is the file's own mtime with taken_at_source 'file' (the source
-- the schema reserved for exactly this importer -- the metadata job
-- refines it to 'exif' when the file's header resolves a capture
-- time), pending_jobs counts the pipeline the rescan enqueues (2
-- for an image: metadata + thumbnail; 3 for a video: + transcode),
-- and no device identity exists -- the file's location is its
-- identity. The caller computed the checksum by reading the file,
-- validated its format against the same whitelist the upload
-- receiver enforces, and derived the mime type and family from
-- that -- the insert itself trusts the pipeline's own gates.
INSERT INTO assets (
    id, owner_id, storage_kind, file_path, checksum, original_filename,
    file_size, mime_type, media_kind, is_component,
    width, height, taken_at, taken_at_source, pending_jobs,
    file_mtime
) VALUES (
    $1, $2, 'external', $3, $4, $5, $6, $7, $8, false,
    0, 0, $9, 'file', $10, $11
)
RETURNING *;

-- name: StampAlbumScanned :exec
-- The scan's own clock on the album row: the last run whose diff
-- changed the album (an ingest, a vanish, a snapshot refresh). The
-- column stays NULL through an unchanged re-scan on purpose -- the
-- idempotence pin (2026-11-08) holds that a second run over an
-- unchanged directory writes NOTHING, and a timestamp that moves on
-- every no-op run would break that contract for a value nobody
-- asked to refresh. updated_at stays out of the SET list: a scan is
-- not an album edit, and the wall's ordering of the own group is
-- the edit surface's own discipline (membership writes never moved
-- it either). The type arm keeps the write a no-op on any row that
-- stopped being a directory album.
UPDATE albums SET
    last_scanned_at = now()
WHERE id = $1
  AND type = 'directory';

-- name: MarkExternalAssetVanished :execrows
-- The orphan arm (the pinned decision, 2026-11-08): the file is
-- gone from disk, so the row is marked deleted_at directly -- the
-- file is not the server's and the trash has no meaning for it.
-- The guards make the arm idempotent: only a LIVE external row of
-- this owner matches (a row already marked, or restored by a
-- racing restore, matches nothing and the re-run of a driver retry
-- counts no second transition), and the album_items row is
-- deliberately NOT touched -- soft delete never removed membership
-- rows (the trash queries' own discipline), the live views filter
-- on deleted_at, and the next scan's add arm ingests a returning
-- file as a NEW row that lands its OWN membership.
UPDATE assets SET
    deleted_at = now(),
    updated_at = now()
WHERE id = $1
  AND owner_id = $2
  AND storage_kind = 'external'
  AND deleted_at IS NULL;

-- name: RefreshExternalSnapshot :execrows
-- The change arm: the file's mtime+size snapshot changed, so the
-- full read the arm just performed re-lands on the row -- the
-- checksum (the content identity the owner+checksum uniqueness and
-- the serving ETags read), the size half of the snapshot, and the
-- mtime half. The guards mirror the orphan arm's: only a live
-- external row of this owner matches, so a driver retry's re-walk
-- of an already-refreshed row (same snapshot now -- the comparison
-- no longer differs) never reaches this statement at all, and a
-- row that vanished or trashed mid-run matches nothing rather
-- than stamping a vanished file's truth onto a dead row. A unique
-- violation here (the changed file's new content exactly equals
-- another row's checksum of the same owner) is the caller's
-- honest failure count, never a silent skip.
UPDATE assets SET
    checksum = $2,
    file_size = $3,
    file_mtime = $4,
    updated_at = now()
WHERE id = $1
  AND owner_id = $5
  AND storage_kind = 'external'
  AND deleted_at IS NULL;

-- name: ListDirectoryAlbums :many
-- The folder sync's scope: every directory album the periodic
-- snapshot diff follows, with the root it walks. A directory album
-- IS a scan of the folder it references (the domain's own
-- definition), so "every directory album" and "every actively
-- tracked folder" are the same set -- there is no paused or
-- archived state for one to be in. The root arm keeps the walk
-- total: a row without a resolved path has nothing to compare and
-- is skipped by the query rather than by every caller.
SELECT id, root_path
FROM albums
WHERE type = 'directory'
  AND root_path IS NOT NULL
ORDER BY created_at, id;
