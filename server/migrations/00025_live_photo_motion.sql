-- +goose Up
-- How a Live Photo's motion came to exist, as ONE boolean: true when
-- the server carved the motion out of the still's own file (the
-- Android "motion photo" shape), false when the motion arrived as a
-- file of its own (the paired-upload shape -- an iPhone's MOV, or a
-- later half matched by its content identifier).
--
-- The distinction is what the download's embedded-motion switch
-- turns on: an extracted motion is a second copy of bytes the still
-- already carries, so handing it out again is opt-in; a motion that
-- arrived as its own file is part of the pair and always rides
-- along. Existing rows take false -- every motion in the library
-- today came in as its own file.
ALTER TABLE assets
    ADD COLUMN motion_extracted boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE assets DROP COLUMN IF EXISTS motion_extracted;