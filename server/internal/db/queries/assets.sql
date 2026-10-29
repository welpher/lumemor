-- Asset queries of the upload endpoint (M2b Task 3), the device
-- idempotency lookup, and the metadata pipeline (M2b Task 4).
-- CreateAsset pins the server-side column values of a fresh upload:
-- storage_kind 'blob' (uploaded bytes live in the content-addressed
-- originals tree), taken_at_source 'import' (the device-reported
-- capture time, refined to 'exif' by the metadata job), and
-- width/height 0 (dimensions are the metadata job's refinement).
-- pending_jobs carries the pipeline count the uploader enqueues: 1
-- for photo rows (the metadata job in Task 4; Task 5's thumbnail job
-- raises it to 2), 0 for component video rows (refined by the
-- metadata job but never the asset whose readiness the event
-- announces). created_at/updated_at default to now() on insert.

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

-- name: CreateExifPlaceholder :exec
-- The metadata sidecar row of a fresh asset: the exif jsonb carries
-- the raw device-side upload fields (deviceCreatedAt, deviceTimeZone)
-- for the metadata job, which replaces the placeholder with the
-- extracted EXIF.
INSERT INTO exif (asset_id, exif) VALUES ($1, $2);

-- name: GetAssetForMetadata :one
-- The metadata job's read: everything it needs to run exiftool and
-- resolve the taken-at fallback chain. exif_jsonb is the placeholder
-- payload of the upload (deviceCreatedAt / deviceTimeZone) until the
-- job itself has run. No row means the asset is gone; the job treats
-- that as success so it does not retry forever against a deleted
-- asset.
SELECT a.file_path,
       a.media_kind,
       a.taken_at,
       a.taken_at_source,
       a.pending_jobs,
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
-- The metadata job never passes the 'file' source: a content-addressed
-- upload blob's mtime is the upload instant itself, so a file-mtime
-- step would churn values without information; 'file' stays reserved
-- for the directory-album importer that reads files in place.
-- Dimensions refine from the same extraction (COALESCE keeps the
-- existing value when the file reports none).
UPDATE assets SET
    taken_at        = COALESCE(sqlc.narg('taken_at'), taken_at),
    taken_at_source = CASE WHEN sqlc.narg('taken_at') IS NULL THEN taken_at_source
                           ELSE 'exif'::taken_at_source END,
    width           = COALESCE(sqlc.narg('width'), width),
    height          = COALESCE(sqlc.narg('height'), height),
    duration_ms     = COALESCE(sqlc.narg('duration_ms'), duration_ms),
    updated_at      = now()
WHERE id = sqlc.arg('id');

-- name: CompleteAssetJob :one
-- One pipeline job completion: the guarded decrement of the readiness
-- counter. The CASE reads the OLD pending_jobs (unqualified columns
-- in UPDATE expressions reference the pre-update row): a job emptying
-- the counter (old value 1) stamps ready_at; a completion that did
-- not leaves it NULL. The WHERE guard makes the query a no-op for
-- rows with nothing pending (component videos; re-runs of an already
-- completed pipeline, which is what keeps re-enqueued jobs idempotent:
-- no double decrement, no second ready_at) -- those completions
-- return NO ROW, and the job layer reads that as the benign no-op
-- (no asset.ready event) rather than an error. media_kind rides along
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
