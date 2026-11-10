-- +goose Up
-- Asset pipeline readiness: the upload handler records
-- the number of pipeline jobs it enqueued for an asset in
-- pending_jobs, and every successful job completion decrements it;
-- the completion that empties the counter stamps ready_at. That
-- completion is the single point where the job layer publishes the
-- asset.ready event, so the event fires exactly once per asset and
-- only when the whole pipeline is done.
--
-- The counter is the pinned readiness mechanism: the initial
-- pipeline was the metadata job alone, so
-- the upload inserts photo rows with pending_jobs 1 (and component
-- video rows with 0 -- they are refined by the metadata job but are
-- never the client-visible asset whose readiness the event announces).
-- The thumbnail job joins by bumping the uploader's count to 2
-- and completing through the same guarded decrement; no semantic
-- change is needed.
--
-- A crash between the upload transaction and the enqueue leaves the
-- counter positive forever: the asset is honestly not ready (its job
-- never ran). Recovery is a future requeue sweep, not a schema concern.
--
-- ready_at NULL means "not ready yet" for every asset that entered
-- through the upload endpoint; taken-at fallback semantics are
-- unchanged (the metadata job refines taken_at only when EXIF carries
-- a capture time; see the exif queries).

ALTER TABLE assets
    ADD COLUMN pending_jobs integer NOT NULL DEFAULT 1,
    ADD COLUMN ready_at timestamptz;

-- +goose Down
ALTER TABLE assets
    DROP COLUMN IF EXISTS ready_at,
    DROP COLUMN IF EXISTS pending_jobs;
