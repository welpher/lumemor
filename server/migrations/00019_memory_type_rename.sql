-- +goose Up
-- The album type value 'virtual' retires whole into 'memory': the
-- word carried a container metaphor the shape never had. A row of
-- this type holds no photos -- it is a curated set of references to
-- photos that live in other albums, and 'memory' is the concept's
-- final name. Nothing about the rows changes: RENAME VALUE keeps
-- every existing row on the renamed value (the deployments that
-- carry such rows keep their sets exactly as they were), and the
-- deployments that never created one -- the creation face had been
-- retired for two blocks -- see no data change at all. The retired
-- value is gone from the enum: no new row can ever carry it.
ALTER TYPE album_type RENAME VALUE 'virtual' TO 'memory';

-- +goose Down
-- The rename in reverse: rows on 'memory' become 'virtual' again,
-- and the enum admits the old value. Honest as a shape restore --
-- the data rides the value wherever it sits.
ALTER TYPE album_type RENAME VALUE 'memory' TO 'virtual';
