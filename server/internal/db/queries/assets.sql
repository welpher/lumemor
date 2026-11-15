-- Asset queries of the upload endpoint, the device
-- idempotency lookup, the metadata pipeline, and the
-- attribute PATCH.
-- CreateAsset pins the server-side column values of a fresh upload:
-- storage_kind 'blob' (uploaded bytes live in the content-addressed
-- originals tree), taken_at_source 'import' (the device-reported
-- capture time, refined to 'exif' by the metadata job), and
-- width/height 0 (dimensions are the metadata job's refinement).
-- pending_jobs carries the pipeline count the uploader enqueues: 2
-- for photo rows (metadata + thumbnail), 3 for standalone video rows
-- (+ transcode), and -- since the Live Photo component transcode
-- -- 1 for a video-family component row (the transcode job that
-- derives its playable stream; an image-family component counts
-- none: no job of any kind can run for it). created_at/updated_at
-- default to now() on insert.

-- name: CreateAsset :one
-- The upload insert. live_photo_video_id links a Live Photo's image
-- row to its component video row (NULL otherwise); is_component marks
-- the component video rows that never appear in the timeline.
-- device_asset_id carries the device-side identity on photo rows only.
INSERT INTO assets (
    id, owner_id, storage_kind, file_path, checksum, original_filename,
    file_size, mime_type, media_kind, live_photo_video_id, is_component,
    device_asset_id, width, height, taken_at, taken_at_source, pending_jobs
) VALUES (
    $1, $2, 'blob', $3, $4, $5, $6, $7, $8, $9, $10, $11, 0, 0, $12, 'import', $13
)
RETURNING *;

-- name: GetAssetByOwnerChecksum :one
-- The three-state lookup behind the upload idempotency: the row of
-- this owner and checksum, live first. After a unique violation the
-- caller reads the row's deleted_at to answer duplicate (NULL) or
-- trashed (set).
SELECT * FROM assets
WHERE owner_id = $1 AND checksum = $2
ORDER BY deleted_at NULLS FIRST
LIMIT 1;

-- name: GetAssetByDeviceId :one
-- The device idempotency lookup of GET /assets/by-device-id/{id} and
-- the fallback after a device-identity unique violation. Owner-scoped;
-- the live row wins when live and trashed rows share the identity.
SELECT * FROM assets
WHERE owner_id = $1 AND device_asset_id = $2
ORDER BY deleted_at NULLS FIRST
LIMIT 1;

-- name: GetAsset :one
-- The owner-scoped row read behind the WRITE faces that gate on
-- ownership (the attribute PATCH and the trash restore): their
-- permission is ACL derivation -- the owner alone may edit and
-- restore -- so the owner filter IS the check, and a foreign id
-- resolves to no row, which the handlers answer exactly like an
-- unknown id -- 404, byte-identical -- so asset ids cannot be
-- enumerated. The READ faces (detail, thumbnail, original, play)
-- use GetAssetDetail / GetVisibleAsset instead: the caller-visible
-- union the switchover moved them onto.
SELECT * FROM assets
WHERE id = $1 AND owner_id = $2;

-- name: GetVisibleAsset :one
-- The caller-visible row read behind the media serving endpoints
-- (the thumbnail, the original download, the playable stream): the
-- album union -- the caller's own sync and directory arms plus
-- every album shared to them -- with no state filter, matching the
-- owner's long-standing shape (a trashed or hidden asset stays
-- servable at its known id; the listing surfaces are what filter
-- the states out). A miss answers the single-shape 404 the asset
-- endpoints share.
SELECT * FROM assets a
WHERE a.id = sqlc.arg('id')
  AND caller_asset_visible(sqlc.arg('viewer'), a.id);

-- name: GetAssetDetail :one
-- The detail read of GET /assets/{id}: the asset row joined with the
-- exif hot fields the response flattens, plus the asset's resolved
-- place (the metadata job's GPS resolution). The name rides as TWO
-- nullable columns -- the request locale's place_names row and the
-- canonical name -- because a place is optional on the asset: sqlc's
-- inference over a bare COALESCE marks the result non-null, and the
-- no-place rows would fail the scan. The handler folds the two with
-- the `GET /places` join's exact discipline (the locale row when the
-- place data carries one, the canonical name otherwise; places.name
-- is NOT NULL, so a valid place_id always answers a name). The gate
-- is the caller-visible union: the owner's own arms (their sync
-- album holds every upload, whatever the row's state) plus every
-- album shared to the caller. The join only attaches the sidecar of
-- the row that matched. The LEFT JOINs keep the detail answering
-- before and after the metadata job refined the sidecar: hot fields
-- are NULL until then, and an asset with no resolved place (no GPS,
-- or no baked place data) answers a null place face.
-- description and is_hidden ride along: the PATCH response reuses
-- this read after its writes, so it serves the same shape the GET
-- endpoint serves, applied values included.
SELECT a.id,
       a.owner_id,
       a.media_kind,
       a.original_filename,
       a.file_size,
       a.mime_type,
       a.checksum,
       a.created_at,
       a.taken_at,
       a.taken_at_source,
       a.taken_at_zone,
       a.width,
       a.height,
       a.duration_ms,
       a.blurhash,
       a.live_photo_video_id,
       a.deleted_at,
       a.ready_at,
       a.description,
       a.is_hidden,
       a.location_lat,
       a.location_lon,
       a.file_path,
       e.camera_make,
       e.camera_model,
       e.lens,
       e.iso,
       e.aperture,
       e.shutter,
       e.focal,
       e.lat,
       e.lon,
       e.alt,
       e.tz_offset,
       p.id AS place_id,
       pn.name AS locale_name,
       p.name AS canonical_name
FROM assets a
LEFT JOIN exif e ON e.asset_id = a.id
LEFT JOIN places p ON p.id = a.place_id
LEFT JOIN place_names pn
       ON pn.geoname_id = p.id AND pn.locale = sqlc.arg('locale')
WHERE a.id = sqlc.arg('id')
  AND caller_asset_visible(sqlc.arg('viewer'), a.id);

-- name: CreateExifPlaceholder :exec
-- The metadata sidecar row of a fresh asset: the exif jsonb carries
-- the raw device-side upload fields (deviceCreatedAt, deviceTimeZone)
-- for the metadata job, which replaces the placeholder with the
-- extracted EXIF.
INSERT INTO exif (asset_id, exif) VALUES ($1, $2);

-- name: GetAssetForMetadata :one
-- The pipeline jobs' shared read: everything a job needs to run its
-- derivation and resolve the taken-at fallback chain. exif_jsonb is
-- the placeholder payload of the upload (deviceCreatedAt /
-- deviceTimeZone) until the metadata job itself has run. No row
-- means the asset is gone; every job treats that as success so it
-- does not retry forever against a deleted asset. is_component rides
-- along for the completion discipline: the metadata and
-- thumbnail jobs must not consume a component row's counter (the
-- transcode alone owns it), and the transcode job completes the
-- counter but never announces for a component. storage_kind rides
-- along: an external asset's file_path is the file's real
-- absolute path (the directory album's source tree), not a
-- originals-relative form, so the path resolution branches on the
-- kind -- the one column every job needs to find the bytes.
SELECT a.file_path,
       a.storage_kind,
       a.media_kind,
       a.taken_at,
       a.taken_at_source,
       a.pending_jobs,
       a.is_component,
       e.exif
FROM assets a
LEFT JOIN exif e ON e.asset_id = a.id
WHERE a.id = $1;

-- name: UpdateExifFromMetadata :exec
-- The metadata job's write: replaces the placeholder sidecar with the
-- extracted metadata. exif is 1:1 with asset (asset_id is its primary
-- key), so the upsert makes re-runs converge to one row with the same
-- values: the ON CONFLICT arm updates in place, no second row can
-- ever exist. The exif jsonb is the raw exiftool output verbatim.
-- Null hot fields (a file without GPS or lens data) overwrite to NULL
-- deliberately: the row mirrors the file's actual metadata, not a
-- merge of successive extractions.
INSERT INTO exif (
    asset_id, exif, camera_make, camera_model, lens, iso, aperture,
    shutter, focal, lat, lon, alt, tz_offset
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
)
ON CONFLICT (asset_id) DO UPDATE SET
    exif         = EXCLUDED.exif,
    camera_make  = EXCLUDED.camera_make,
    camera_model = EXCLUDED.camera_model,
    lens         = EXCLUDED.lens,
    iso          = EXCLUDED.iso,
    aperture     = EXCLUDED.aperture,
    shutter      = EXCLUDED.shutter,
    focal        = EXCLUDED.focal,
    lat          = EXCLUDED.lat,
    lon          = EXCLUDED.lon,
    alt          = EXCLUDED.alt,
    tz_offset    = EXCLUDED.tz_offset,
    updated_at   = now();

-- name: RefineAssetFromMetadata :exec
-- The metadata job's refinement of the asset row. taken_at moves ONLY
-- when exif time is present (EXIF DateTimeOriginal resolved with an
-- offset); a NULL exif time keeps the upload's value and source, so
-- the upload response's initial taken_at never regresses (the fallback
-- chain deviceCreatedAt -> server now stays intact, source 'import').
-- The offset that resolved the instant is display state that rides it:
-- taken_at_zone lands with the same arm (the offset the metadata job
-- resolved, NULL when no zone did). The metadata job never passes the
-- 'file' source: a content-addressed
-- upload blob's mtime is the upload instant itself, so a file-mtime
-- step would churn values without information; 'file' stays reserved
-- for the directory-album importer that reads files in place.
-- Dimensions refine from the same extraction (COALESCE keeps the
-- existing value when the file reports none).
--
-- place_id is the GPS resolution's write: when the job resolved
-- the place (the in-memory grid is wired), the value -- the nearest
-- city within the threshold or NULL for no GPS / beyond threshold --
-- is the file's truth and lands unconditionally; when the grid is NOT
-- wired (a deployment without the baked place data), the gate flag
-- stays false and the column keeps whatever it held, so a
-- data-less boot never destroys resolutions a data-carrying boot
-- produced. A manual location (the owner's own coordinates, the
-- PATCH's write) is the newer decision and stays: the refinement
-- leaves the resolved place of a manually located row alone.
UPDATE assets SET
    taken_at        = COALESCE(sqlc.narg('taken_at'), taken_at),
    taken_at_source = CASE WHEN sqlc.narg('taken_at') IS NULL THEN taken_at_source
                           ELSE 'exif'::taken_at_source END,
    taken_at_zone   = COALESCE(sqlc.narg('taken_at_zone'), taken_at_zone),
    width           = COALESCE(sqlc.narg('width'), width),
    height          = COALESCE(sqlc.narg('height'), height),
    duration_ms     = COALESCE(sqlc.narg('duration_ms'), duration_ms),
    place_id        = CASE WHEN sqlc.arg('place_resolved')::boolean
                            AND location_lat IS NULL THEN sqlc.narg('place_id')
                           ELSE place_id END,
    updated_at      = now()
WHERE id = sqlc.arg('id');

-- name: CompleteAssetJob :one
-- One pipeline job completion: the guarded decrement of the readiness
-- counter. The CASE reads the OLD pending_jobs (unqualified columns
-- in UPDATE expressions reference the pre-update row): a job emptying
-- the counter (old value 1) stamps ready_at; a completion that did
-- not leaves it NULL. The WHERE guard makes the query a no-op for
-- rows with nothing pending (rows whose pipeline drained -- an
-- image-family Live Photo component counts none; re-runs of an
-- already completed pipeline, which is what keeps re-enqueued jobs
-- idempotent: no double decrement, no second ready_at) -- those
-- completions return NO ROW, and the job layer reads that as the
-- benign no-op (no asset.ready event) rather than an error. Which
-- completions may consume the counter at all is decided one layer
-- up, by the jobs: the metadata and thumbnail workers skip the call
-- for component rows entirely (the component's transcode alone owns
-- its counter). media_kind rides along
-- so the completion publishes the event with the asset's kind without
-- a second read.
UPDATE assets SET
    ready_at     = CASE WHEN pending_jobs = 1 THEN now() ELSE ready_at END,
    pending_jobs = pending_jobs - 1,
    updated_at   = now()
WHERE id = $1 AND pending_jobs > 0
RETURNING (ready_at IS NOT NULL)::boolean AS became_ready, media_kind;

-- name: UpdateAssetBlurhash :exec
-- The thumbnail job's write: stores the blurhash the job computed from
-- the tile (image assets) or the poster frame (video assets). The
-- value converges on re-runs (the same derivation over the same bytes
-- yields the same hash), so the update is idempotent.
UPDATE assets SET blurhash = $2, updated_at = now()
WHERE id = $1;

-- name: UpdateAssetDescription :execrows
-- The owner-scoped description write of PATCH /assets/{id}.
-- The description <> $1 arm is the set-semantics guard:
-- patching the text the row already carries matches nothing, so the
-- idempotent repeat is the same success, updated_at stays un-stamped
-- and the event stays silent -- the rows-affected count is the
-- handler's change signal. The owner guard IS the permission check
-- (asset attribute editing is owner-only, no admin override), so a
-- foreign id matches nothing exactly like an unknown one -- the same
-- anti-enumeration shape as GetAsset. No state filter: a trashed
-- asset is editable like it is readable, and the edit resurfaces
-- with the restore.
UPDATE assets SET
    description = sqlc.arg('description'),
    updated_at = now()
WHERE id = sqlc.arg('id')
  AND owner_id = sqlc.arg('owner_id')
  AND description <> sqlc.arg('description');

-- name: UpdateAssetTakenAt :execrows
-- The owner-scoped capture-time write of PATCH /assets/{id}: the
-- instant and the display zone that resolved it land together (the
-- offset IS the zone the info panel renders beside the time). The
-- pair <> arm is the set-semantics guard: a patch repeating both the
-- stored instant and the stored zone matches nothing, so the
-- idempotent repeat is the same success, updated_at stays un-stamped
-- and the event stays silent -- the rows-affected count is the
-- handler's change signal. (Both fields are compared: the instant
-- and the zone are DISTINCT from their stored values, so a
-- same-instant zone-only patch IS a change -- the zone is a served
-- field of its own.) The owner guard IS the permission check (asset
-- attribute editing is owner-only, no admin override), so a foreign
-- id matches nothing exactly like an unknown one. No state filter: a
-- trashed asset is editable like it is readable.
UPDATE assets SET
    taken_at = sqlc.arg('taken_at'),
    taken_at_zone = sqlc.arg('taken_at_zone'),
    updated_at = now()
WHERE id = sqlc.arg('id')
  AND owner_id = sqlc.arg('owner_id')
  AND (taken_at <> sqlc.arg('taken_at')
       OR taken_at_zone IS DISTINCT FROM sqlc.arg('taken_at_zone'));

-- name: UpdateAssetLocation :execrows
-- The owner-scoped manual-location write of PATCH /assets/{id}: the
-- coordinates land on the asset (the info panel's own pair) beside
-- the resolved place (the caller's resolved answer, NULL when the
-- point sits beyond every city or the deployment runs without the
-- baked place data -- the resolution flag in the handler decides).
-- The equality arm is the set-semantics guard: a patch repeating the
-- stored pair and place matches nothing -- the idempotent repeat is
-- the same success, no event. The owner guard IS the permission
-- check; no state filter, like the description arm.
UPDATE assets SET
    location_lat = sqlc.arg('lat'),
    location_lon = sqlc.arg('lon'),
    place_id = sqlc.narg('place_id'),
    updated_at = now()
WHERE id = sqlc.arg('id')
  AND owner_id = sqlc.arg('owner_id')
  AND (location_lat IS DISTINCT FROM sqlc.arg('lat')
       OR location_lon IS DISTINCT FROM sqlc.arg('lon')
       OR place_id IS DISTINCT FROM sqlc.narg('place_id'));

-- name: ListStuckAssets :many
-- The requeue sweep's stuck-row finder: assets whose
-- upload committed -- pending_jobs counted the jobs the uploader
-- meant to enqueue -- but whose readiness never arrived (ready_at
-- still NULL). These are the rows an enqueue insert failure or a
-- crash in the commit-to-enqueue window left with no (or missing)
-- jobs in the queue; a re-upload answers duplicate without
-- re-enqueueing, so the sweep is their only recovery. Since the
-- component transcode, a video-family component row counts one job
-- and matches here like any row. Soft-deleted assets are deliberately
-- excluded: a trashed asset is headed for expiry, not recovery.
-- Rows whose jobs are merely in flight (a live upload mid-pipeline)
-- match too -- the query cannot tell "lost jobs" from "busy jobs",
-- and it does not need to: the sweep's re-enqueue inserts carry
-- River unique keys over kind + asset id, so repeated or overlapping
-- sweeps do not pile up jobs, but the upload path's enqueues are
-- unkeyed and never join the unique index -- an in-flight upload row
-- gets one duplicate set from the sweep (two jobs for an image,
-- three for a video), and all of them run, converged by the pipeline
-- jobs' own idempotency and the guarded readiness decrement (the
-- extra completions no-op). The sweep never resets pending_jobs (the
-- counter already counts the missing jobs; the re-enqueued
-- completions drive it down). media_kind and is_component ride along
-- because the sweep re-enqueues by the row's kind and role: two jobs
-- for an image (metadata + thumbnail), three for a standalone video
-- since the transcode job split the video arm (metadata +
-- thumbnail + transcode, matching the upload's pending_jobs
-- initialization), and for a Live Photo component the transcode
-- ALONE (the one job its counter counts; its metadata refinement
-- rides the upload path and never touches the counter).
SELECT id, media_kind, is_component FROM assets
WHERE pending_jobs > 0 AND ready_at IS NULL AND deleted_at IS NULL;

-- name: CountAssetIDsByKind :one
-- The rerun scope of one media kind: the assets whose ML work a
-- rerun redoes -- finished their pipeline (ready_at, the ML jobs
-- belong after readiness) and not in the trash (a trashed asset's
-- products serve nothing; the permanent purge deletes them with the
-- row). The count becomes the rerun run's total at enqueue time.
SELECT count(*) FROM assets
WHERE media_kind = $1 AND ready_at IS NOT NULL AND deleted_at IS NULL;

-- name: ListAssetIDsByKind :many
-- The rerun scope's asset ids, paged in id order: the rerun driver
-- walks them one page at a time and dispatches each asset's work.
-- id order keeps the paging stable; assets becoming ready DURING a
-- run may join the tail (done can pass the enqueue-time total) --
-- they would otherwise miss both the seam and the rerun.
SELECT id FROM assets
WHERE media_kind = $1 AND ready_at IS NOT NULL AND deleted_at IS NULL
ORDER BY id ASC
LIMIT $2 OFFSET $3;
