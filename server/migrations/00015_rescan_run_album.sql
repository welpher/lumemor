-- +goose Up
-- The rescan's run row needs its subject: a `job_runs` row of
-- the album_rescan kind belongs to one directory album, and the
-- endpoint's in-flight guard ("a rescan of THIS album is already
-- running -- join it, start nothing") must find that album's run
-- among every run of the kind. The kind column alone answers "a
-- rescan is running somewhere", which would serialize every
-- album's scan behind one unrelated scan -- the wrong scope.
--
-- album_id is nullable and carries a value only on the rescan
-- kind's rows: every other kind (model downloads, ML reruns, the
-- places re-resolution) is instance-scoped and keeps NULL. The
-- reference is SET NULL on the album's deletion -- the run is a
-- historical record of work that happened, not a live attachment
-- to the album: the album going away (v1 has no directory-album
-- delete flow; the column's strategy still pins the answer for
-- whenever one lands) leaves the row with its counts and a NULL
-- subject, the same discipline the timeline preference rows apply
-- in reverse.

ALTER TABLE job_runs
    ADD COLUMN album_id uuid REFERENCES albums(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE job_runs
    DROP COLUMN IF EXISTS album_id;
