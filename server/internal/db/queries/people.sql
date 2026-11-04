-- People surface queries (M3b Task 1): the read side of the named
-- region, the unnamed region's keyset pages and the cluster detail.
--
-- The B4 semantics run through every query: face clusters are GLOBAL
-- rows (face_clusters / faces), while names and the hidden switch are
-- PRIVATE per user (user_face_labels, keyed (user_id, cluster_id) --
-- never an instance-level name). A caller's view of a cluster is
-- therefore computed, not stored: which region the cluster sits in,
-- its label state and its face count all come from the caller's own
-- rows joined onto the global structure.
--
-- Caller visibility is owner-based while the library has no sharing:
-- an asset is visible to the caller when they own it and it is live
-- (not trashed, not hidden, no Live Photo component row) -- the same
-- live-asset filters the timeline applies. A cluster is visible to
-- the caller when at least one of its faces sits on a visible asset;
-- face counts count ONLY those faces (honest counts per caller, so
-- the same cluster can show different counts to different accounts).
--
-- The unnamed region's keyset predicate is the row-value comparison
-- on the ordering key (face_count DESC, cluster_id DESC): the cursor
-- carries the pair of the page's last row and the comparison mirrors
-- the timeline's (taken_at, id) pair. The pair is passed as nullable
-- arguments: NULL on the first page, where the predicate
-- short-circuits to TRUE. Because the ordering key is the aggregate
-- itself, the counts are computed in a subquery and the predicate
-- rides on the outside of it.

-- name: ListLabeledPeople :many
-- The named region: clusters that carry a label row OF THE CALLER
-- (the inner join) and contain at least one face on a caller-visible
-- asset (the faces join drops the rest -- a label on an invisible
-- cluster lists nowhere). Hidden people stay listed carrying their
-- is_hidden flag: the switch is a badge, not a filter. Ordered by
-- name case-insensitively, cluster id as the tiebreak.
SELECT fc.id AS cluster_id,
       l.name,
       l.is_hidden,
       fc.representative_face_id,
       count(*) AS face_count
FROM face_clusters fc
JOIN user_face_labels l ON l.cluster_id = fc.id AND l.user_id = sqlc.arg('user_id')
JOIN faces f ON f.cluster_id = fc.id
JOIN assets a ON a.id = f.asset_id
WHERE a.owner_id = sqlc.arg('user_id')
  AND a.deleted_at IS NULL
  AND a.is_hidden = false
  AND a.is_component = false
GROUP BY fc.id, l.name, l.is_hidden, fc.representative_face_id
ORDER BY lower(l.name), fc.id;

-- name: ListUnnamedClusters :many
-- The unnamed region, one keyset page at a time: clusters with at
-- least one face on a caller-visible asset and NO label row of the
-- caller (the NOT EXISTS -- another account's label never moves a
-- cluster for this caller). Ordered by face count descending, the
-- cluster id descending as the tiebreak; the cursor is the
-- (face_count, cluster_id) pair of the previous page's last row,
-- NULL on the first page.
SELECT cluster_id, representative_face_id, face_count
FROM (
    SELECT fc.id AS cluster_id,
           fc.representative_face_id,
           count(*) AS face_count
    FROM face_clusters fc
    JOIN faces f ON f.cluster_id = fc.id
    JOIN assets a ON a.id = f.asset_id
    WHERE a.owner_id = sqlc.arg('user_id')
      AND a.deleted_at IS NULL
      AND a.is_hidden = false
      AND a.is_component = false
      AND NOT EXISTS (
          SELECT 1 FROM user_face_labels l
          WHERE l.user_id = sqlc.arg('user_id') AND l.cluster_id = fc.id
      )
    GROUP BY fc.id, fc.representative_face_id
) counts
WHERE (
    sqlc.narg('cursor_face_count')::bigint IS NULL
    OR (counts.face_count, counts.cluster_id)
        < (sqlc.narg('cursor_face_count')::bigint, sqlc.narg('cursor_cluster_id')::uuid)
)
ORDER BY face_count DESC, cluster_id DESC
LIMIT sqlc.arg('limit_count');

-- name: CountClusterFaces :one
-- The cluster's face count over caller-visible assets ONLY. A count
-- of zero covers both "unknown cluster" and "every face sits on
-- another account's asset" -- the double meaning the detail endpoint
-- answers with one indistinguishable 404.
SELECT count(*) AS face_count
FROM faces f
JOIN assets a ON a.id = f.asset_id
WHERE f.cluster_id = sqlc.arg('cluster_id')
  AND a.owner_id = sqlc.arg('user_id')
  AND a.deleted_at IS NULL
  AND a.is_hidden = false
  AND a.is_component = false;

-- name: GetClusterForCaller :one
-- One cluster's label state for the caller: the caller's own label
-- via the LEFT JOIN (null columns = the unnamed state), whatever any
-- other account labeled. Visibility is not this query's concern --
-- the count gate runs first.
SELECT fc.id AS cluster_id,
       l.name AS label_name,
       l.is_hidden AS label_hidden
FROM face_clusters fc
LEFT JOIN user_face_labels l ON l.cluster_id = fc.id AND l.user_id = sqlc.arg('user_id')
WHERE fc.id = sqlc.arg('cluster_id');

-- name: ListClusterFacesForCaller :many
-- The cluster's faces on caller-visible assets, strongest score
-- first with the face id as the tiebreak: the detail grid's rows
-- with their minimal asset context.
SELECT f.id,
       f.asset_id,
       f.score,
       a.taken_at,
       a.original_filename
FROM faces f
JOIN assets a ON a.id = f.asset_id
WHERE f.cluster_id = sqlc.arg('cluster_id')
  AND a.owner_id = sqlc.arg('user_id')
  AND a.deleted_at IS NULL
  AND a.is_hidden = false
  AND a.is_component = false
ORDER BY f.score DESC, f.id;
