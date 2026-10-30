-- Trash queries (M2c Task 3): the write side of the trash lifecycle
-- -- soft delete, restore, the owner-scoped listing -- plus the
-- expiry machinery: the expired-row candidates, the blob reference
-- count, and the guarded delete. The reference-counted blob rule is
-- the M2b hard-gate handoff this closes: content addressing stores
-- two owners' identical uploads as ONE physical file referenced by
-- two rows, so no delete path may remove the file while any other
-- row points at it.
--
-- The Live Photo pair moves through the trash as one asset: every
-- write below addresses the pair through
-- (id = $1 OR live_photo_video_id = $1), so trashing or restoring
-- the photo carries its component video row along. Soft delete does
-- NOT touch album_items (the timeline and every live view filter on
-- deleted_at, so membership rows simply stop mattering while the
-- asset is trashed) -- which is also what makes restore a single
-- UPDATE: no membership re-insertion exists to do.

-- name: SoftDeleteAsset :execrows
-- The trash entry point (the batch handler of Task 4 calls this with
-- the caller's owner guard): stamps deleted_at on the asset and,
-- when the asset is a Live Photo, on its linked component video row.
-- The predicate addresses the pair from either side: by the photo id
-- (the subquery resolves the component the photo links to) or by the
-- component id (the live_photo_video_id arm resolves the photo that
-- links to it) -- the pair moves through the trash as ONE asset,
-- whichever row names it. The deleted_at IS NULL arm makes the
-- write a no-op on already-trashed rows -- the set semantics of the
-- batch actions: a repeat trash reports zero affected rows, which
-- the handler answers as the same 204 success. Only the rows
-- change: the blob stays in place (the file system half of the
-- soft-delete contract) and the pipeline state is left alone (a
-- trashed asset that finishes its pipeline is harmless; its
-- readiness just never surfaces anywhere).
UPDATE assets SET
    deleted_at = now(),
    updated_at = now()
WHERE (assets.id = $1
       OR assets.live_photo_video_id = $1
       OR assets.id = (SELECT src.live_photo_video_id FROM assets src WHERE src.id = $1))
  AND assets.deleted_at IS NULL
  AND assets.owner_id = $2;

-- name: ListTrash :many
-- The trash view of one owner: every trashed, non-component asset
-- newest deletion first, each with the countdown of whole days
-- until the expiry job removes it. The countdown is computed HERE,
-- against the database's own clock -- the deletion mark came from
-- now() on that same clock, so the two can never disagree about
-- "moments ago" (a host clock a fraction ahead or behind would
-- otherwise flip the days at the boundary). make_interval carries
-- the retention the caller read from the instance settings; CEIL
-- rounds up (an asset trashed moments ago with the default 30-day
-- retention shows 30, not 29), and the GREATEST arm clamps the
-- window between a passed deadline and the next hourly run to 0.
-- Component video rows never appear -- the Live Photo pair is
-- represented by its photo. The bound caps the response (the trash
-- is a recovery view for recent deletes; older items age out
-- through expiry on their own), and the ordering is what the
-- client's trash page renders.
SELECT a.*,
       GREATEST(
           CEIL(EXTRACT(epoch FROM (a.deleted_at + make_interval(days => sqlc.arg('retention_days')::int) - now())) / 86400),
           0
       )::int AS remaining_days
FROM assets a
WHERE a.owner_id = $1
  AND a.deleted_at IS NOT NULL
  AND a.is_component = false
ORDER BY a.deleted_at DESC
LIMIT sqlc.arg('limit_count');

-- name: RestoreAsset :execrows
-- Clears the trash mark on the asset and, when the asset is a Live
-- Photo, on its linked component video row -- the same pair
-- predicate as the soft delete, addressed from either side, because
-- the pair restores as ONE asset. The deleted_at IS NOT NULL arm is
-- the set-semantics guard of the plan: restoring an already-live row
-- is a no-op (zero affected rows), which the handler answers as the
-- same 204 -- the row is already in the desired state. The owner
-- guard is the permission check: a foreign id matches no row,
-- exactly like an unknown one. No membership re-insertion runs
-- anywhere: soft delete never removed the album_items rows, and the
-- timeline's deleted_at filter is what hid the asset, so clearing
-- the mark is the whole restore.
UPDATE assets SET
    deleted_at = NULL,
    updated_at = now()
WHERE (assets.id = $1
       OR assets.live_photo_video_id = $1
       OR assets.id = (SELECT src.live_photo_video_id FROM assets src WHERE src.id = $1))
  AND assets.deleted_at IS NOT NULL
  AND assets.owner_id = $2;

-- name: ListExpiredAssets :many
-- The expiry job's candidates: trashed rows whose retention window
-- closed before the cutoff, oldest first, bounded per run so one
-- hour's sweep can never turn into an unbounded delete loop (v1
-- scale: 500 rows per run; anything deeper waits for the next run).
-- file_path rides along because the blob reference check below
-- needs it before the row goes away; media_kind rides along for the
-- asset.deleted event the run publishes per purged row (the same
-- id-plus-kind payload every asset event carries).
SELECT id, file_path, media_kind FROM assets
WHERE deleted_at IS NOT NULL
  AND deleted_at < $1
ORDER BY deleted_at
LIMIT $2;

-- name: CountAssetFileReferences :one
-- The shared-blob reference count, the query the M2b handoff pinned
-- verbatim: how many OTHER asset rows reference the same physical
-- file. Content addressing makes another owner's row of the same
-- bytes -- or any other row at all, live or trashed -- point at the
-- same file_path, and a trashed row may still be restored, so every
-- row counts. The candidate's own id is excluded by the id <> $2
-- arm: the count is meaningful both before and after the row itself
-- is deleted, which is what lets the expiry job (and the explicit
-- delete) count first, delete the row, then decide about the file.
SELECT count(*) FROM assets
WHERE file_path = $1
  AND id <> $2;

-- name: ExpireAsset :execrows
-- The expiry job's guarded delete: the row goes only if it is still
-- trashed AND still past the cutoff inside the deleting statement
-- itself. This is the restore-vs-expiry race guard (Review Focus
-- 4): a row restored between the job's candidate listing and this
-- statement fails the deleted_at IS NOT NULL arm, the delete
-- matches nothing, and the job skips the row's files -- a restored
-- asset must never be purged. A concurrent expiry run that already
-- deleted the row answers zero rows the same way. The cascades
-- (album_items, exif, favorites) take the row's dependents with it.
DELETE FROM assets
WHERE id = $1
  AND deleted_at IS NOT NULL
  AND deleted_at < $2;

-- name: DeleteAssetPermanently :execrows
-- The explicit permanent delete of DELETE /assets/{id}: the row goes
-- whatever its state (the caller asked for the asset's destruction,
-- not its trash). No owner guard lives here because the permission
-- decision (owner or administrator) ran on the row the handler
-- already read; a concurrent delete that got there first matches
-- nothing and the handler answers 404 -- the asset no longer exists,
-- exactly like an unknown id.
DELETE FROM assets
WHERE id = $1;

-- name: GetAssetByID :one
-- The permission-decision read of the explicit delete: the full row
-- behind an id with NO owner filter, because the caller may be the
-- owner OR an administrator, and telling the two cases apart needs
-- the row's owner. A plain non-owner user resolves the row and is
-- refused by the handler with the same 404 an unknown id answers --
-- the anti-enumeration discipline holds because the refusal, not
-- the read, shapes the response.
SELECT * FROM assets
WHERE id = $1;
