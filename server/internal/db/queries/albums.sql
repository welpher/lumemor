-- Album queries of the sync-album wiring (M2c Task 2): every user
-- owns exactly one sync album -- the core backup stream the timeline
-- aggregates -- and every upload lands in it. The creation points are
-- the account-creation transactions; the upsert below also lets the
-- upload transaction recover the stream of an account whose rows
-- predate the wiring (a user row without its sync album), because
-- failing the upload over a recoverable invariant would trade away the
-- upload's idempotency guarantees for no benefit.

-- name: EnsureSyncAlbum :one
-- The idempotent sync-album creator. The insert arm serves every
-- account-creation point; the ON CONFLICT arm serves concurrent
-- creators: the loser of the albums_sync_owner_uq race blocks on the
-- winner's index entry, then locks and returns the winner's row
-- through the (value-identical) no-op UPDATE -- the classic upsert
-- whose RETURNING always answers the album that exists afterwards,
-- never "no row". The title is the fixed system label; sync albums
-- are not user-facing artifacts of any v1 API.
INSERT INTO albums (id, owner_id, title, type)
VALUES ($1, $2, 'Sync', 'sync')
ON CONFLICT (owner_id) WHERE type = 'sync'
DO UPDATE SET updated_at = albums.updated_at
RETURNING id;

-- name: AddAssetToAlbum :exec
-- The materialized membership insert. ON CONFLICT DO NOTHING keeps
-- re-adding an asset that already sits in the album a no-op: album
-- membership is a set, and the callers converge to the same row
-- instead of failing. (The upload path is the only caller: the
-- trash-restore path needs no re-insertion -- soft delete never
-- removed the membership rows, and the live views' deleted_at filter
-- is the only thing that hid the asset while it sat in the trash, so
-- clearing the mark restores the timeline presence wholesale.)
INSERT INTO album_items (album_id, asset_id, added_by)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING;
