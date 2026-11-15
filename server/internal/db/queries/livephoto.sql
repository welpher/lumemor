-- Live Photo queries: the silent ingest chain's writers and the
-- maintenance sweep's reads.
--
-- The chain has three entrances and one shape (a still, a component
-- video row, and the still's live_photo_video_id pointing at it): the
-- paired upload (the handler's own transaction), the single-file
-- extraction (a motion photo whose video is embedded in the still),
-- and the content-identifier match (the two halves arrived as
-- separate uploads). These statements serve the last two and the
-- sweep; the paired upload keeps its own insert path.
--
-- VISIBILITY: a component row is never listed. Every listing surface
-- filters is_component = false, and the caller-visible union that
-- gates the media endpoints reads album membership instead -- which
-- is why the extraction links its new row into the owner's sync album
-- exactly like the paired upload's component.

-- name: GetAssetForMotionPhoto :one
-- The extraction's read of the still: what the new motion row copies
-- from it (the owner, the capture instant) and where its file goes
-- (the still's own directory and stem). live_photo_video_id rides
-- along so a re-run can tell an unchanged pair from a replaced one.
SELECT id, owner_id, file_path, original_filename, taken_at, taken_at_zone,
       storage_kind, media_kind, is_component, live_photo_video_id
FROM assets
WHERE id = $1;

-- name: CreateMotionAsset :one
-- The extraction's insert of the motion row: the counterpart of the
-- paired upload's component insert. It is born a component (never
-- listed), marked motion_extracted (so the download's
-- embedded-motion switch can tell it from a motion that arrived as
-- its own file), and it counts ONE pipeline job -- the transcode
-- that derives its playable copy, exactly like a paired component.
-- taken_at and taken_at_zone copy the still's: the motion is the same
-- instant, and the storage layout renders both from the still's date.
INSERT INTO assets (
    id, owner_id, storage_kind, file_path, checksum, original_filename,
    file_size, mime_type, media_kind, is_component, motion_extracted,
    width, height, taken_at, taken_at_source, taken_at_zone, pending_jobs
) VALUES (
    $1, $2, 'blob', $3, $4, $5, $6, 'video/mp4', 'video', true, true,
    0, 0, $7, $8, $9, 1
)
RETURNING *;

-- name: SetAssetLivePhotoVideo :exec
-- The link write: the still gains its motion, or (a NULL argument)
-- loses it. The explicit updated_at keeps the SSE convergence's
-- refetch meaningful -- the event only matters if the row changed.
UPDATE assets
SET live_photo_video_id = sqlc.narg('video_id'), updated_at = now()
WHERE id = $1;

-- name: MarkAssetAsLivePhotoMotion :exec
-- The reuse write: an ALREADY-STORED video whose bytes equal the
-- extracted motion becomes the pair's component -- hidden from every
-- listing (is_component) and marked as the extracted shape. The
-- bytes stay exactly where they were: no second copy on disk.
UPDATE assets
SET is_component = true, motion_extracted = true, updated_at = now()
WHERE id = $1;

-- name: MarkAssetAsComponent :exec
-- The pairing's hide write: a video the identifier matched becomes the
-- pair's motion, so it leaves every listing (is_component) while its
-- bytes stay exactly where they are. motion_extracted stays false --
-- this motion arrived as a file of its own, so a download always hands
-- it over.
UPDATE assets
SET is_component = true, updated_at = now()
WHERE id = $1;

-- name: CountLivePhotoReferences :one
-- How many stills point at this motion. The permanent-delete rule
-- reads it: the motion goes with the LAST still that references it,
-- never with the first (a burst or a duplicated pair shares one
-- motion by checksum).
SELECT count(*)::bigint FROM assets WHERE live_photo_video_id = $1;

-- name: CountLivePhotoReferencesExcluding :one
-- The same count with the row being deleted excluded: the delete
-- path counts BEFORE the row goes, so the caller passes its own id.
SELECT count(*)::bigint
FROM assets
WHERE live_photo_video_id = $1 AND id <> $2;

-- name: SetAssetLivePhotoCID :exec
-- The identifier write of the metadata refinement: ContentIdentifier
-- (iOS) or MediaGroupUUID (some Android cameras), NULL when the file
-- carries neither. It lands on every asset kind -- the identifier is
-- what lets a later arrival find this row.
UPDATE assets SET live_photo_cid = $2 WHERE id = $1;

-- name: GetLivePhotoCIDMatch :one
-- The counterpart hunt: the same owner's asset carrying the same
-- identifier with the OTHER media kind. The row being matched is
-- excluded by id, trashed rows never match (a restored half is a new
-- decision, not a silent repair), and the earliest row wins so the
-- answer is deterministic. A miss is the ordinary case -- the other
-- half has not arrived yet.
SELECT id, media_kind, is_component, live_photo_video_id
FROM assets
WHERE owner_id = $1
  AND live_photo_cid = $2
  AND media_kind = $3
  AND id <> $4
  AND deleted_at IS NULL
ORDER BY created_at, id
LIMIT 1;

-- name: ListLivePhotoComponents :many
-- The maintenance sweep's candidates: component rows old enough that
-- an in-flight extraction cannot be mistaken for an orphan (the
-- extraction creates the row and links it in the same job, so the
-- grace window only has to outlast one job). Bounded per run like the
-- trash expiry, so an hourly sweep never becomes an unbounded walk.
SELECT id, file_path, storage_kind, deleted_at
FROM assets
WHERE is_component
  AND created_at < now() - make_interval(secs => sqlc.arg('grace_seconds')::int)
ORDER BY created_at
LIMIT sqlc.arg('limit_count');

-- name: ListLivePhotoStillsOfMotion :many
-- The stills referencing one motion, live rows only: the sweep's
-- "does anything still need these bytes" question, and the delete
-- path's count. A trashed still keeps its link (a restore must find
-- its motion), so this answers across the trash state -- only the
-- PERMANENT delete detaches.
SELECT id, deleted_at FROM assets WHERE live_photo_video_id = $1;