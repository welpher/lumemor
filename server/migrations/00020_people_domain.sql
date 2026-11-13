-- +goose Up
-- The people domain settles into two states -- named / unnamed -- and
-- the person ENTITY gains the attributes it lacked.
--
-- 1. user_face_labels.is_hidden drops. The switch was a per-user view
--    preference that never became one: a "hidden" person stayed in the
--    caller's list behind a badge, because the badge was the whole
--    mechanism. The retirement is approved (the two-state model has no
--    third bucket), so the stored values are DISCARDED, not migrated
--    anywhere -- there is no target state to carry, and the rows keep
--    their names untouched. The public replacement is removing the
--    caller's own name (DELETE /people/{clusterId}/label), which is
--    private by construction.
-- 2. face_clusters.birth_date joins: the person entity's birth date,
--    one value per cluster for every account (a family agrees on the
--    person's birthday), independent of naming. Null until someone
--    sets it.
-- 3. face_clusters.avatar_face_id joins: the avatar pick, one face per
--    cluster for every account. The foreign key carries ON DELETE SET
--    NULL on purpose -- a purged asset's face must never block the
--    purge, and a nulled pick simply falls back to the cluster's best
--    caller-visible face at read time. The pick is advisory to the
--    clustering: representative_face_id (the structural maintenance's
--    pointer) stays untouched, and a face that later leaves the
--    cluster stops counting at read time.
ALTER TABLE user_face_labels DROP COLUMN is_hidden;

ALTER TABLE face_clusters ADD COLUMN birth_date date;

ALTER TABLE face_clusters
    ADD COLUMN avatar_face_id uuid REFERENCES faces (id) ON DELETE SET NULL;

-- +goose Down
-- Honest as a shape restore: the two entity attributes go (their data
-- goes with them -- there is nowhere to keep it), and the retired
-- switch returns empty, defaulting to "not hidden" for every row --
-- the state the retirement discarded is not reconstructible, and a
-- false default is the truthful one for a switch nobody set since.
ALTER TABLE face_clusters DROP COLUMN avatar_face_id;
ALTER TABLE face_clusters DROP COLUMN birth_date;
ALTER TABLE user_face_labels ADD COLUMN is_hidden boolean NOT NULL DEFAULT false;