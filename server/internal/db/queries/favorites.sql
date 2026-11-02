-- Favorites and hidden view queries (M2c Task 4): the read side of
-- the two curated views. Both answer TimelineItem lists in the
-- timeline's ordering and keyset pagination, so both compute the
-- per-user is_favorite flag the timeline items carry (the EXISTS
-- form is the same computation the timeline queries use, evaluated
-- for the requesting user). Both filter to the live view the same
-- way: trashed rows, and the component video rows of Live Photos,
-- never appear -- the two views differ only in which assets they
-- select (the caller's favorites rows vs the caller's own hidden
-- rows).

-- name: ListFavoriteAssets :many
-- The favorites view: the caller's favorites rows joined to the live
-- assets behind them, newest capture first. The favorites row is the
-- selection; the asset's state is the filter -- a favorited asset
-- that was hidden or trashed leaves the view (it returns with the
-- unhide or the restore, the favorite row untouched). No owner
-- filter on the asset side: favorites are per-user state joined to
-- whatever live asset the row points at -- in this milestone the
-- batch actions are owner-scoped, so favorites rows of the caller's
-- own assets are all the API can produce, and the join stays the
-- honest shape for any future shared-content favorites. is_favorite
-- is true by construction here (the join selected the row); it is
-- computed anyway so the column carries the same meaning as in the
-- timeline queries.
SELECT sqlc.embed(a),
       EXISTS (
           SELECT 1 FROM user_favorites uf
           WHERE uf.asset_id = a.id AND uf.user_id = sqlc.arg('user_id')
       ) AS is_favorite
FROM assets a
JOIN user_favorites f ON f.asset_id = a.id AND f.user_id = sqlc.arg('user_id')
WHERE a.deleted_at IS NULL
  AND a.is_hidden = false
  AND a.is_component = false
  AND (
      sqlc.narg('cursor_taken_at')::timestamptz IS NULL
      OR (a.taken_at, a.id) < (sqlc.narg('cursor_taken_at')::timestamptz, sqlc.narg('cursor_id')::uuid)
  )
ORDER BY a.taken_at DESC, a.id DESC
LIMIT sqlc.arg('limit_count');

-- name: ListHiddenAssets :many
-- The hidden view: the caller's own rows whose owner-level
-- visibility switch is off, live ones only -- a trashed asset stays
-- in the trash view until restored or expired, and a restored one
-- returns here still hidden (the hide survived the trash, exactly
-- like a favorite did). Component video rows never appear (the Live
-- Photo pair is represented by its photo, whose is_hidden carries
-- the pair's visibility). is_favorite still reports the caller's
-- favorites state of a hidden asset: the favorites view filtered the
-- row out, the flag did not change.
SELECT sqlc.embed(a),
       EXISTS (
           SELECT 1 FROM user_favorites uf
           WHERE uf.asset_id = a.id AND uf.user_id = sqlc.arg('user_id')
       ) AS is_favorite
FROM assets a
WHERE a.owner_id = sqlc.arg('user_id')
  AND a.is_hidden = true
  AND a.deleted_at IS NULL
  AND a.is_component = false
  AND (
      sqlc.narg('cursor_taken_at')::timestamptz IS NULL
      OR (a.taken_at, a.id) < (sqlc.narg('cursor_taken_at')::timestamptz, sqlc.narg('cursor_id')::uuid)
  )
ORDER BY a.taken_at DESC, a.id DESC
LIMIT sqlc.arg('limit_count');
