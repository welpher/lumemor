-- +goose Up
-- The download preferences gain their fourth switch: whether a
-- download hands over the motion an Android-style "motion photo"
-- carries INSIDE its own file as a separate video as well.
--
-- The column defaults to false on purpose. The still's original
-- already carries the embedded motion (that is what the camera wrote),
-- so the extra copy is a duplicate by default; a paired iPhone Live
-- Photo is the other shape -- its motion is a file of its own, always
-- rides along, and this switch says nothing about it. Existing rows
-- take the documented default.
ALTER TABLE users
    ADD COLUMN download_include_embedded_videos boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS download_include_embedded_videos;