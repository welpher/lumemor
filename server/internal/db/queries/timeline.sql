-- Timeline queries: the read side of the aggregated
-- timeline and the per-album timeline, both keyset-paginated.
--
-- The aggregate is the union of the albums that feed the caller's
-- timeline: the caller's sync album (the core stream, never
-- excludable), the caller's own directory albums, and every album
-- shared to the caller regardless of type (memory albums included).
-- Each non-sync album carries a per-user "keep out of the timeline"
-- switch; a missing preference row means kept in. The preference's
-- scope is the own directory albums and the shared albums -- the sync
-- arm is deliberately exempt from it, so a stray preference row on
-- the sync album excludes nothing (the core stream is the stream).
-- The caller's own memory albums never feed their own timeline
-- (in-stream curation would be pure duplication), which falls out of
-- the three arms below: an own memory album is neither sync nor
-- directory nor shared to its owner.
--
-- The three arms below are the arms of caller_asset_visible (the
-- one physical home of the union, migration 00010): this query keeps
-- its own inline EXISTS because the aggregation formula folds
-- the per-user preference INSIDE the existential -- an asset kept in
-- through ANY feeding album stays in even when another of its albums
-- is switched out -- and that conjunct cannot compose outside the
-- existential over the album rows. The arms themselves are
-- structurally identical between the two homes -- keep them in
-- sync.
--
-- Deduplication is the EXISTS semi-join: one asset visible through
-- several albums yields one row, and an asset kept in through ANY
-- feeding album stays in even when another of its albums is switched
-- out -- the union semantics of the formula. Every item is live by
-- construction: archived, trashed and Live Photo component rows are
-- filtered before the semi-join runs. The album narrow below is the
-- one exception, by design: an album is a reference container that
-- keeps its archived members (the archive removes an asset from the
-- family flow, never from the album that holds it), while a memory --
-- a family-flow surface of its own -- drops them like this
-- aggregate does.
--
-- The keyset predicate is the row-value comparison on the ordering
-- key (taken_at DESC, id DESC): it is index-consistent with the
-- assets_timeline_idx partial index, and the id tiebreak keeps pages
-- stable across rows sharing one taken_at. The cursor pair is passed
-- as nullable arguments: NULL on the first page, where the predicate
-- short-circuits to TRUE.

-- name: ListTimelineAssets :many
-- The per-user is_favorite flag rides along as an EXISTS: the
-- caller's own favorites row for each asset, computed here so the
-- timeline items carry the per-user state (the same asset is a
-- favorite for one viewer and a plain tile for another) without a
-- second roundtrip.
SELECT sqlc.embed(a),
       EXISTS (
           SELECT 1 FROM user_favorites uf
           WHERE uf.asset_id = a.id AND uf.user_id = sqlc.arg('user_id')
       ) AS is_favorite
FROM assets a
WHERE a.deleted_at IS NULL
  AND a.is_hidden = false
  AND a.is_component = false
  AND (
      sqlc.narg('cursor_taken_at')::timestamptz IS NULL
      OR (a.taken_at, a.id) < (sqlc.narg('cursor_taken_at')::timestamptz, sqlc.narg('cursor_id')::uuid)
  )
  AND EXISTS (
      SELECT 1
      FROM album_items ai
      JOIN albums al ON al.id = ai.album_id
      LEFT JOIN album_shares s
             ON s.album_id = al.id AND s.user_id = sqlc.arg('user_id')
      LEFT JOIN user_album_prefs p
             ON p.album_id = al.id AND p.user_id = sqlc.arg('user_id')
      WHERE ai.asset_id = a.id
        AND (
            (al.type = 'sync' AND al.owner_id = sqlc.arg('user_id'))
            OR (
                (
                    (al.type = 'directory' AND al.owner_id = sqlc.arg('user_id'))
                    OR s.user_id IS NOT NULL
                )
                AND (p.in_timeline IS NULL OR p.in_timeline)
            )
        )
  )
ORDER BY a.taken_at DESC, a.id DESC
LIMIT sqlc.arg('limit_count');

-- name: ListAlbumTimelineAssets :many
-- The album's own timeline with the same is_favorite computation as
-- the aggregate: per-user state of the requesting viewer, whatever
-- the album or the asset's owner. The member predicate is the album
-- container's: trashed and component rows are out, archived members
-- stay -- except on a memory, which is a family-flow surface and
-- drops them (the is_hidden arm below, gated on the album type).
SELECT sqlc.embed(a),
       EXISTS (
           SELECT 1 FROM user_favorites uf
           WHERE uf.asset_id = a.id AND uf.user_id = sqlc.arg('user_id')
       ) AS is_favorite
FROM assets a
JOIN album_items ai ON ai.asset_id = a.id
JOIN albums al ON al.id = ai.album_id
WHERE al.id = sqlc.arg('album_id')
  -- The visibility check of the in-album timeline: the caller owns
  -- the album or it is shared to the caller. An album that fails both
  -- resolves to no rows -- the empty list, indistinguishable from an
  -- unknown id, so album existence cannot be probed. The per-user
  -- timeline preference does not apply here: it scopes the aggregate,
  -- not the album's own view.
  AND (
      al.owner_id = sqlc.arg('user_id')
      OR EXISTS (
          SELECT 1 FROM album_shares s
          WHERE s.album_id = al.id AND s.user_id = sqlc.arg('user_id')
      )
  )
  AND a.deleted_at IS NULL
  AND (al.type <> 'memory' OR a.is_hidden = false)
  AND a.is_component = false
  -- The memory's shared-view rule rides the same arm as the detail's
  -- pages, the counts and the collages: a sharee's walk carries the
  -- live members the memory's owner CURRENTLY sees (the re-share
  -- ground, revalidated at read time -- a photo whose source
  -- visibility died leaves the walk, no snapshot residue), while the
  -- owner's own walk is the plain live membership. Directory and sync
  -- albums take the first arm and never reach the ground.
  AND (
      al.type <> 'memory'
      OR al.owner_id = sqlc.arg('user_id')
      OR caller_asset_visible(al.owner_id, a.id)
  )
  AND (
      sqlc.narg('cursor_taken_at')::timestamptz IS NULL
      OR (a.taken_at, a.id) < (sqlc.narg('cursor_taken_at')::timestamptz, sqlc.narg('cursor_id')::uuid)
  )
ORDER BY a.taken_at DESC, a.id DESC
LIMIT sqlc.arg('limit_count');

-- name: ListPlaceTimelineAssets :many
-- The place domain's photo wall: the live assets resolved to one of
-- the given places, with the aggregate's is_favorite computation,
-- ordering and keyset predicate. The narrow is a plain place_id
-- membership over the caller-visible asset union -- the same domain
-- the city grouping counts -- so a place id outside the caller's
-- visible assets contributes nothing (the empty list, never an
-- error). The map panel reads the union of a bubble's places through
-- the same query.
SELECT sqlc.embed(a),
       EXISTS (
           SELECT 1 FROM user_favorites uf
           WHERE uf.asset_id = a.id AND uf.user_id = sqlc.arg('user_id')
       ) AS is_favorite
FROM assets a
WHERE a.place_id = ANY(sqlc.arg('place_ids')::integer[])
  AND a.deleted_at IS NULL
  AND a.is_hidden = false
  AND a.is_component = false
  AND caller_asset_visible(sqlc.arg('user_id'), a.id)
  AND (
      sqlc.narg('cursor_taken_at')::timestamptz IS NULL
      OR (a.taken_at, a.id) < (sqlc.narg('cursor_taken_at')::timestamptz, sqlc.narg('cursor_id')::uuid)
  )
ORDER BY a.taken_at DESC, a.id DESC
LIMIT sqlc.arg('limit_count');

-- name: ListPersonTimelineAssets :many
-- The person detail's photo wall: one face cluster's own photos with
-- the aggregate's is_favorite computation, ordering and keyset
-- predicate. The cluster arm is an EXISTS semi-join over its faces,
-- so an asset carrying two faces of the same person is one item --
-- the wall is a photo wall, not a face wall -- and every member face
-- is read through the same caller-visible live predicate the people
-- surface applies (a cluster's assets all in other accounts' hands
-- resolves to no rows: the empty list, indistinguishable from an
-- unknown id, so cluster ids cannot be probed here either).
SELECT sqlc.embed(a),
       EXISTS (
           SELECT 1 FROM user_favorites uf
           WHERE uf.asset_id = a.id AND uf.user_id = sqlc.arg('user_id')
       ) AS is_favorite
FROM assets a
WHERE a.deleted_at IS NULL
  AND a.is_hidden = false
  AND a.is_component = false
  AND EXISTS (
      SELECT 1
      FROM faces f
      WHERE f.asset_id = a.id
        AND f.cluster_id = sqlc.arg('cluster_id')
        AND caller_asset_visible(sqlc.arg('user_id'), a.id)
  )
  AND (
      sqlc.narg('cursor_taken_at')::timestamptz IS NULL
      OR (a.taken_at, a.id) < (sqlc.narg('cursor_taken_at')::timestamptz, sqlc.narg('cursor_id')::uuid)
  )
ORDER BY a.taken_at DESC, a.id DESC
LIMIT sqlc.arg('limit_count');
