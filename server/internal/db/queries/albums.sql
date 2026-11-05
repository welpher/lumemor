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

-- Album reads (M3c Task 1): the wall, the cover candidates and the
-- detail face. The live-member predicate runs through all of them:
-- deleted_at IS NULL AND is_hidden = false AND is_component = false,
-- the timeline's own definition of a live asset row (A5 -- a hidden
-- asset is invisible to the whole family, and a Live Photo's video
-- component is never a tile of its own).

-- name: ListWallAlbumsForCaller :many
-- The wall of the calling user in one statement: the own group (the
-- caller's virtual and directory albums, updated_at descending) UNION
-- ALL the shared group (the albums shared to the caller, the
-- caller's grant time descending), each group breaking ties on the
-- album id descending. The arm_rank column is the group
-- discriminator the ORDER BY pins -- own first, shared second -- and
-- the per-arm sort key rides as sort_at (the album's updated_at for
-- own cards, the caller's granted_at for shared ones).
--
-- Sync albums never appear in either arm: the sync album is the
-- account's timeline, not a wall artifact -- the caller's own sync
-- album is invisible on every album face, and a sync album another
-- account shared feeds the timeline's shared arm, not the wall.
SELECT sqlc.embed(al),
       'owner'::text AS caller_role,
       NULL::uuid AS owner_user_id,
       NULL::text AS owner_email,
       EXISTS (
           SELECT 1 FROM album_shares s WHERE s.album_id = al.id
       ) AS is_shared,
       (
           SELECT count(*)
           FROM album_items ai
           JOIN assets a ON a.id = ai.asset_id
           WHERE ai.album_id = al.id
             AND a.deleted_at IS NULL
             AND a.is_hidden = false
             AND a.is_component = false
       ) AS item_count,
       0::int AS arm_rank,
       al.updated_at AS sort_at
FROM albums al
WHERE al.owner_id = sqlc.arg('user_id')
  AND al.type IN ('virtual', 'directory')

UNION ALL

SELECT sqlc.embed(al),
       'viewer'::text AS caller_role,
       u.id AS owner_user_id,
       u.email AS owner_email,
       -- The shared arm's is_shared holds true by construction: the
       -- caller's own share row is the reason this row exists.
       true AS is_shared,
       (
           SELECT count(*)
           FROM album_items ai
           JOIN assets a ON a.id = ai.asset_id
           WHERE ai.album_id = al.id
             AND a.deleted_at IS NULL
             AND a.is_hidden = false
             AND a.is_component = false
       ) AS item_count,
       1::int AS arm_rank,
       s.granted_at AS sort_at
FROM albums al
JOIN album_shares s ON s.album_id = al.id AND s.user_id = sqlc.arg('user_id')
JOIN users u ON u.id = al.owner_id
WHERE al.type IN ('virtual', 'directory')

ORDER BY arm_rank, sort_at DESC, id DESC;

-- name: ListAlbumCoverCandidates :many
-- The derived covers of the given albums: each one's four most
-- recent live members, taken_at descending with the asset id as the
-- tiebreak -- the exact rows the wall renders for an album without a
-- pinned cover asset. The window keeps the result bounded at four
-- rows per album however many members an album holds.
SELECT ranked.album_id,
       ranked.asset_id
FROM (
    SELECT ai.album_id,
           ai.asset_id,
           row_number() OVER (
               PARTITION BY ai.album_id
               ORDER BY a.taken_at DESC, a.id DESC
           ) AS recency_rank
    FROM album_items ai
    JOIN assets a ON a.id = ai.asset_id
    WHERE ai.album_id = ANY(sqlc.arg('album_ids')::uuid[])
      AND a.deleted_at IS NULL
      AND a.is_hidden = false
      AND a.is_component = false
) ranked
WHERE ranked.recency_rank <= 4
ORDER BY ranked.album_id, ranked.recency_rank;

-- name: GetAlbumForCaller :one
-- The album the caller may see, with the caller's role and the live
-- member count. The visibility check is the whole point: the caller
-- owns the album or an album_shares row grants it, and the album is
-- no sync album (the sync album is the timeline's, whoever owns it
-- -- the detail face answers 404 for it through the handler's
-- no-row mapping, which is what keeps unknown, invisible and sync
-- ids indistinguishable).
SELECT sqlc.embed(al),
       CASE
           WHEN al.owner_id = sqlc.arg('user_id') THEN 'owner'::text
           ELSE 'viewer'::text
       END AS caller_role,
       (
           SELECT count(*)
           FROM album_items ai
           JOIN assets a ON a.id = ai.asset_id
           WHERE ai.album_id = al.id
             AND a.deleted_at IS NULL
             AND a.is_hidden = false
             AND a.is_component = false
       ) AS item_count
FROM albums al
WHERE al.id = sqlc.arg('album_id')
  AND al.type IN ('virtual', 'directory')
  AND (
      al.owner_id = sqlc.arg('user_id')
      OR EXISTS (
          SELECT 1 FROM album_shares s
          WHERE s.album_id = al.id AND s.user_id = sqlc.arg('user_id')
      )
  );

-- name: ListAlbumItemsForCaller :many
-- One page of the album's items: the live members in the timeline's
-- own ordering (taken_at DESC, id DESC), keyset-paginated with the
-- same row-value predicate and the same NULL-init first page. The
-- caller's visibility gate is NOT here: the handler runs this only
-- after GetAlbumForCaller returned the album, so the rows it lists
-- are rows of an album the caller already proved they may see.
SELECT ai.asset_id,
       ai.added_at,
       a.taken_at,
       a.original_filename,
       a.media_kind
FROM album_items ai
JOIN assets a ON a.id = ai.asset_id
WHERE ai.album_id = sqlc.arg('album_id')
  AND a.deleted_at IS NULL
  AND a.is_hidden = false
  AND a.is_component = false
  AND (
      sqlc.narg('cursor_taken_at')::timestamptz IS NULL
      OR (a.taken_at, a.id) < (sqlc.narg('cursor_taken_at')::timestamptz, sqlc.narg('cursor_id')::uuid)
  )
ORDER BY a.taken_at DESC, a.id DESC
LIMIT sqlc.arg('limit_count');
