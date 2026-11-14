-- Derivative rebuild queries: the image card's explicit re-derivation
-- run walks the image assets (the media_kind='image' rows the
-- derivation serves; video assets carry a poster frame the rebuild
-- does not touch), keyset-ordered by the asset id, and persists its
-- cursor on the job_runs row so a window edge or a pause resumes at
-- the breakpoint.

-- name: CountImageAssets :one
-- The rebuild scope's size: every live image asset, trashed
-- included -- place- and lifecycle-wise the serving copies are
-- metadata; a restored asset must find its derivations in place.
SELECT count(*)
FROM assets
WHERE media_kind = 'image'
  AND deleted_at IS NULL;

-- name: ListImageAssetPage :many
-- One page of the rebuild scope, keyset-ordered by the asset id (the
-- cursor is the previous page's last id; the zero value starts the
-- walk). The file path and storage kind ride along: the driver
-- resolves the original against the data root.
SELECT id,
       file_path,
       storage_kind
FROM assets
WHERE media_kind = 'image'
  AND deleted_at IS NULL
  AND id > sqlc.arg('cursor')
ORDER BY id
LIMIT sqlc.arg('page_size');

-- name: CountImageAssetsAfter :one
-- The resumed walk's remaining scope: the image assets past the run's
-- cursor. The re-entered walk re-derives its total from this -- the
-- row's total keeps counting the WHOLE scope, the resumed work is the
-- remainder, and the panel's done/total ratio stays honest.
SELECT count(*)
FROM assets
WHERE media_kind = 'image'
  AND deleted_at IS NULL
  AND id > $1;
