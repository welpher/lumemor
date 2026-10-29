-- Asset queries of the upload endpoint (M2b Task 3) and the device
-- idempotency lookup. CreateAsset pins the server-side column values
-- of a fresh upload: storage_kind 'blob' (uploaded bytes live in the
-- content-addressed originals tree), taken_at_source 'import' (the
-- device-reported capture time, refined to 'exif' by the metadata
-- job), and width/height 0 (dimensions are the metadata job's
-- refinement). created_at/updated_at default to now() on insert.

-- name: CreateAsset :one
-- The upload insert. live_photo_video_id links a Live Photo's image
-- row to its component video row (NULL otherwise); is_component marks
-- the component video rows that never appear in the timeline.
-- device_asset_id carries the device-side identity on photo rows only.
INSERT INTO assets (
    id, owner_id, storage_kind, file_path, checksum, original_filename,
    file_size, mime_type, media_kind, live_photo_video_id, is_component,
    device_asset_id, width, height, taken_at, taken_at_source
) VALUES (
    $1, $2, 'blob', $3, $4, $5, $6, $7, $8, $9, $10, $11, 0, 0, $12, 'import'
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
