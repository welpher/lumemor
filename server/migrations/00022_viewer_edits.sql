-- +goose Up
-- The viewer's edit arms: the manual capture time (its display zone
-- rides the instant), the manual location (the coordinates the owner
-- pinned, resolved into a place like the metadata job's GPS step),
-- and the hand-drawn face (a face row without a detector -- model_id
-- goes nullable so a drawn face carries no model).
--
-- taken_at_zone is the zone the DISPLAYED capture time resolves in:
-- the EXIF chain's offset when the pipeline refined the time (the
-- metadata job writes it beside the instant), the PATCH's offset when
-- the owner wrote the time by hand. It is display state that rides
-- the instant, never a second truth.
--
-- location_lat/location_lon exist ONLY for a manual edit: they are
-- the owner's coordinates, the pair the info panel renders and the
-- place resolution runs on. The places re-resolution walk skips rows
-- that carry them (the owner's pin is the newer decision), and the
-- metadata refinement's place write keeps its hand too.

ALTER TABLE assets ADD COLUMN taken_at_zone text;
ALTER TABLE assets ADD COLUMN location_lat double precision
    CHECK (location_lat BETWEEN -90 AND 90);
ALTER TABLE assets ADD COLUMN location_lon double precision
    CHECK (location_lon BETWEEN -180 AND 180);

-- A manual face has no detector to name: model_id is the detection
-- run's model, and a hand-drawn face never went through one.
ALTER TABLE faces ALTER COLUMN model_id DROP NOT NULL;

-- +goose Down
-- The manual faces must go before the NOT NULL returns: they are the
-- rows the column change allowed.
DELETE FROM faces WHERE model_id IS NULL;
ALTER TABLE faces ALTER COLUMN model_id SET NOT NULL;

ALTER TABLE assets DROP COLUMN location_lon;
ALTER TABLE assets DROP COLUMN location_lat;
ALTER TABLE assets DROP COLUMN taken_at_zone;
