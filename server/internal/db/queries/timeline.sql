-- Timeline queries (M2c Task 2): the read side of the aggregated
-- timeline and the per-album timeline, both keyset-paginated.
--
-- The aggregate is the union of the albums that feed the caller's
-- timeline: the caller's sync album (the core stream, never
-- excludable), the caller's own directory albums, and every album
-- shared to the caller regardless of type (virtual albums included).
-- Each non-sync album carries a per-user "keep out of the timeline"
-- switch; a missing preference row means kept in. The preference's
-- scope is the own directory albums and the shared albums -- the sync
-- arm is deliberately exempt from it, so a stray preference row on
-- the sync album excludes nothing (the core stream is the stream).
-- The caller's own virtual albums never feed their own timeline
-- (in-stream curation would be pure duplication), which falls out of
-- the three arms below: an own virtual album is neither sync nor
-- directory nor shared to its owner.
--
-- The three arms below are the arms of caller_asset_visible (the
-- one physical home of the union, migration 00010): this query keeps
-- its own inline EXISTS because the spec's aggregation formula folds
-- the per-user preference INSIDE the existential -- an asset kept in
-- through ANY feeding album stays in even when another of its albums
-- is switched out -- and that conjunct cannot compose outside the
-- existential over the album rows. The arms themselves must stay
-- byte-identical between the two homes.
--
-- Deduplication is the EXISTS semi-join: one asset visible through
-- several albums yields one row, and an asset kept in through ANY
-- feeding album stays in even when another of its albums is switched
-- out -- the union semantics of the formula. Every item is live by
-- construction: hidden, trashed and Live Photo component rows are
-- filtered before the semi-join runs.
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
-- the album or the asset's owner.
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
  AND a.is_hidden = false
  AND a.is_component = false
  AND (
      sqlc.narg('cursor_taken_at')::timestamptz IS NULL
      OR (a.taken_at, a.id) < (sqlc.narg('cursor_taken_at')::timestamptz, sqlc.narg('cursor_id')::uuid)
  )
ORDER BY a.taken_at DESC, a.id DESC
LIMIT sqlc.arg('limit_count');
