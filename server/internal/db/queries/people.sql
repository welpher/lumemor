-- People surface queries: the read side of the named
-- region, the unnamed region's keyset pages and the cluster detail.
--
-- The per-user-state semantics run through every query: face clusters are GLOBAL
-- rows (face_clusters / faces), while names and the hidden switch are
-- PRIVATE per user (user_face_labels, keyed (user_id, cluster_id) --
-- never an instance-level name). A caller's view of a cluster is
-- therefore computed, not stored: which region the cluster sits in,
-- its label state and its face count all come from the caller's own
-- rows joined onto the global structure.
--
-- Caller visibility is the album union: an asset is
-- visible to the caller when caller_asset_visible says so -- the
-- caller's own sync and directory albums plus every album shared to
-- them -- AND it is live (not trashed, not hidden, no Live Photo
-- component row), the same live-asset filters the timeline applies.
-- A cluster is visible to the caller when at least one of its faces
-- sits on a visible asset; face counts count ONLY those faces (honest
-- counts per caller, so the same cluster can show different counts
-- to different accounts). The card crop rides the same union: the
-- best caller-visible live face (score DESC, face id ASC), never the
-- global representative -- a representative on an asset the caller
-- cannot see would render a dead crop URL (the card-face rule).
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
-- name case-insensitively, cluster id as the tiebreak. The card face
-- is the cluster's best caller-visible live member (the correlated
-- pick: score DESC, face id ASC), which the listing cluster proves
-- exists -- the join below already found a visible face.
SELECT fc.id AS cluster_id,
       l.name,
       l.is_hidden,
       (
           SELECT f2.id
           FROM faces f2
           JOIN assets a2 ON a2.id = f2.asset_id
           WHERE f2.cluster_id = fc.id
             AND a2.deleted_at IS NULL
             AND a2.is_hidden = false
             AND a2.is_component = false
             AND caller_asset_visible(sqlc.arg('user_id'), a2.id)
           ORDER BY f2.score DESC, f2.id
           LIMIT 1
       ) AS best_face_id,
       count(*) AS face_count
FROM face_clusters fc
JOIN user_face_labels l ON l.cluster_id = fc.id AND l.user_id = sqlc.arg('user_id')
JOIN faces f ON f.cluster_id = fc.id
JOIN assets a ON a.id = f.asset_id
WHERE a.deleted_at IS NULL
  AND a.is_hidden = false
  AND a.is_component = false
  AND caller_asset_visible(sqlc.arg('user_id'), a.id)
GROUP BY fc.id, l.name, l.is_hidden
ORDER BY lower(l.name), fc.id;

-- name: ListUnnamedClusters :many
-- The unnamed region, one keyset page at a time: clusters with at
-- least one face on a caller-visible asset and NO label row of the
-- caller (the NOT EXISTS -- another account's label never moves a
-- cluster for this caller). Ordered by face count descending, the
-- cluster id descending as the tiebreak; the cursor is the
-- (face_count, cluster_id) pair of the previous page's last row,
-- NULL on the first page. The card face is the best caller-visible
-- live member, computed inside the counts subquery beside the count
-- it belongs to.
SELECT cluster_id, best_face_id, face_count
FROM (
    SELECT fc.id AS cluster_id,
           (
               SELECT f2.id
               FROM faces f2
               JOIN assets a2 ON a2.id = f2.asset_id
               WHERE f2.cluster_id = fc.id
                 AND a2.deleted_at IS NULL
                 AND a2.is_hidden = false
                 AND a2.is_component = false
                 AND caller_asset_visible(sqlc.arg('user_id'), a2.id)
               ORDER BY f2.score DESC, f2.id
               LIMIT 1
           ) AS best_face_id,
           count(*) AS face_count
    FROM face_clusters fc
    JOIN faces f ON f.cluster_id = fc.id
    JOIN assets a ON a.id = f.asset_id
    WHERE a.deleted_at IS NULL
      AND a.is_hidden = false
      AND a.is_component = false
      AND caller_asset_visible(sqlc.arg('user_id'), a.id)
      AND NOT EXISTS (
          SELECT 1 FROM user_face_labels l
          WHERE l.user_id = sqlc.arg('user_id') AND l.cluster_id = fc.id
      )
    GROUP BY fc.id
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
-- another account's assets or a share the caller lost" -- the double
-- meaning the detail endpoint answers with one indistinguishable
-- 404. This gate also fronts the label writes (a sharee names the
-- people they see; the label row stays the caller's own).
SELECT count(*) AS face_count
FROM faces f
JOIN assets a ON a.id = f.asset_id
WHERE f.cluster_id = sqlc.arg('cluster_id')
  AND a.deleted_at IS NULL
  AND a.is_hidden = false
  AND a.is_component = false
  AND caller_asset_visible(sqlc.arg('user_id'), a.id);

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
  AND a.deleted_at IS NULL
  AND a.is_hidden = false
  AND a.is_component = false
  AND caller_asset_visible(sqlc.arg('user_id'), a.id)
ORDER BY f.score DESC, f.id;

-- The write side. The label and hidden writes are
-- per-caller upserts on the (user_id, cluster_id) pair -- the
-- per-user-state discipline's writes: they never read or touch another account's
-- row. The structural writes (merge, face move, from-face creation,
-- detach) stick to the
-- cluster-maintenance discipline for every
-- faces.cluster_id change: the member moves land inside ONE
-- transaction together with the representative maintenance and the
-- dissolution of the vacated clusters (whose labels cascade with
-- the row), never as bare member moves that would strand a stale
-- representative or an empty cluster.

-- name: UpsertUserFaceLabel :one
-- The caller's label write: an insert for a first naming, a rename
-- for a repeat (the upsert arm updates ONLY the name, so the hidden
-- switch keeps its state -- a rename is not an unhide). The row is
-- the caller's own: the (user_id, cluster_id) primary key scopes
-- every read to it.
INSERT INTO user_face_labels (user_id, cluster_id, name)
VALUES (sqlc.arg('user_id'), sqlc.arg('cluster_id'), sqlc.arg('name'))
ON CONFLICT (user_id, cluster_id)
DO UPDATE SET name = EXCLUDED.name, updated_at = now()
RETURNING name, is_hidden;

-- name: DeleteUserFaceLabel :exec
-- Removes the caller's label row: the person returns to the
-- caller's unnamed region. Deleting without a label is a zero-row
-- no-op, not an error -- the unlabel is idempotent.
DELETE FROM user_face_labels
WHERE user_id = sqlc.arg('user_id') AND cluster_id = sqlc.arg('cluster_id');

-- name: SetUserFaceLabelHidden :one
-- Flips the hidden switch on the caller's label row. The label row
-- IS the hidden state's home (name NOT NULL), so zero rows here
-- mean the caller has not named the cluster yet -- the
-- person_hide_requires_label case the handler maps to a 400.
UPDATE user_face_labels
SET is_hidden = sqlc.arg('is_hidden'), updated_at = now()
WHERE user_id = sqlc.arg('user_id') AND cluster_id = sqlc.arg('cluster_id')
RETURNING name, is_hidden;

-- name: CountVisibleClusters :one
-- How many of the given clusters the caller holds CURATION over:
-- at least one face on an asset the caller OWNS and that is live.
-- Deliberately owner-scoped: the merge is cluster
-- curation, and curation is the owner's alone -- a sharee who sees
-- a cluster through a shared album reads it and labels it, but the
-- structural faces never re-shape the shared clusters under their
-- cursor. The merge endpoint's visibility gate: one read settles
-- every source plus the target, and a count below the input length
-- names no id (anti-enumeration).
SELECT count(DISTINCT f.cluster_id)
FROM faces f
JOIN assets a ON a.id = f.asset_id
WHERE f.cluster_id = ANY(sqlc.arg('cluster_ids')::uuid[])
  AND a.owner_id = sqlc.arg('user_id')
  AND a.deleted_at IS NULL
  AND a.is_hidden = false
  AND a.is_component = false;

-- name: MoveFacesToCluster :exec
-- The merge's member move: every face of the source clusters
-- re-homes onto the target cluster. It runs INSIDE the merge
-- transaction between the representative maintenance and the
-- vacated-cluster dissolution (see the discipline note above) --
-- never alone.
UPDATE faces
SET cluster_id = sqlc.arg('target_cluster_id'), updated_at = now()
WHERE cluster_id = ANY(sqlc.arg('source_cluster_ids')::uuid[]);

-- name: MoveFaceToCluster :exec
-- The single face's move onto one cluster: the face-move
-- endpoint's member write, wrapped in the same transaction
-- discipline as the merge.
UPDATE faces
SET cluster_id = sqlc.arg('target_cluster_id'), updated_at = now()
WHERE id = sqlc.arg('face_id');

-- name: SeedFaceCluster :exec
-- The from-face person creation's birth write: one cluster row
-- whose representative is the seeding face (a sole member is the
-- strongest member by definition). Runs inside the creation
-- transaction BEFORE the face re-homes onto the row -- the circular
-- faces/face_clusters foreign keys require the cluster to exist
-- first -- and the vacated source's maintenance rides the same
-- transaction.
INSERT INTO face_clusters (id, representative_face_id)
VALUES (sqlc.arg('id'), sqlc.arg('representative_face_id'));

-- name: DetachFaceFromCluster :exec
-- The single-face detach's member write: the face's cluster
-- assignment goes NULL, the same state a freshly detected face
-- holds before the clustering assigns it. Wrapped in the same
-- transaction discipline as the move: the vacated source's
-- representative maintenance and possible dissolution ride the
-- same transaction, never a bare member release.
UPDATE faces
SET cluster_id = NULL, updated_at = now()
WHERE id = sqlc.arg('face_id');

-- name: RepointClusterRepresentative :exec
-- Points one cluster at its best member (score DESC, face id ASC --
-- the deterministic ordering faces.sql's sweep applies): the
-- merge's and the face move's target maintenance, run after the
-- member writes in the same transaction so the representative
-- always carries the strongest member of the current membership.
UPDATE face_clusters fc
SET representative_face_id = best.id, updated_at = now()
FROM (
    SELECT f.id FROM faces f
    WHERE f.cluster_id = sqlc.arg('cluster_id')
    ORDER BY f.score DESC, f.id
    LIMIT 1
) AS best
WHERE fc.id = sqlc.arg('cluster_id');

-- name: RepointVacatedClusterRepresentatives :exec
-- The vacated side of a face move: a cluster whose representative
-- is no longer among its members follows its best SURVIVING member
-- (faces.sql's RehomeClusterRepresentatives discipline, scoped by
-- cluster ids instead of assets -- a moved face is still a live
-- row, so what moved away is the membership, not the face). A
-- cluster with no member at all matches nothing here; the
-- dissolution below removes it.
UPDATE face_clusters fc
SET representative_face_id = best.id, updated_at = now()
FROM (
    SELECT DISTINCT ON (f.cluster_id) f.cluster_id, f.id
    FROM faces f
    WHERE f.cluster_id = ANY(sqlc.arg('cluster_ids')::uuid[])
    ORDER BY f.cluster_id, f.score DESC, f.id
) AS best
WHERE fc.id = best.cluster_id
  AND NOT EXISTS (
      SELECT 1 FROM faces f
      WHERE f.cluster_id = fc.id AND f.id = fc.representative_face_id
  );

-- name: DissolveVacatedClustersByID :exec
-- The dissolution of vacated clusters: a cluster of the given set
-- that holds no member loses its row (user_face_labels cascade
-- with the foreign key -- the honest person-without-faces
-- semantics, applied where the faces left). The NOT EXISTS guard
-- keeps only truly vacated clusters dying, whatever raced the
-- caller's gates.
DELETE FROM face_clusters fc
WHERE fc.id = ANY(sqlc.arg('cluster_ids')::uuid[])
  AND NOT EXISTS (
      SELECT 1 FROM faces f WHERE f.cluster_id = fc.id
  );

-- name: GetFaceForCaller :one
-- The face-move endpoint's face-side read and gate in one: the
-- face's current cluster, over an asset the caller OWNS and that is
-- live. Deliberately owner-scoped: the face move is
-- cluster curation, and curation is the owner's alone -- a sharee
-- never moves a face of an album shared to them. A miss (unknown
-- id, foreign or non-live asset) and a face the clustering has not
-- assigned to any cluster (cluster_id NULL) are both "no such
-- movable person face" -- the same 404 every people write keeps
-- indistinguishable.
SELECT f.id,
       f.cluster_id
FROM faces f
JOIN assets a ON a.id = f.asset_id
WHERE f.id = sqlc.arg('face_id')
  AND a.owner_id = sqlc.arg('user_id')
  AND a.deleted_at IS NULL
  AND a.is_hidden = false
  AND a.is_component = false;

-- name: GetFaceCropForCaller :one
-- The face crop endpoint's read and gate in one: the face's
-- bounding box (per-mille of the full image, addressing the view
-- derivative's own pixels) and its change signal (updated_at, the
-- ETag's input), over a caller-visible live asset -- the album
-- union (the caller's own arms plus every album shared to them).
-- A miss covers an unknown id and a face on an invisible asset
-- alike -- the same 404 the people surface answers everywhere; the
-- handler folds the derivative's absence into that same shape
-- (anti-enumeration).
SELECT f.id,
       f.asset_id,
       f.bbox_x,
       f.bbox_y,
       f.bbox_w,
       f.bbox_h,
       f.updated_at
FROM faces f
JOIN assets a ON a.id = f.asset_id
WHERE f.id = sqlc.arg('face_id')
  AND a.deleted_at IS NULL
  AND a.is_hidden = false
  AND a.is_component = false
  AND caller_asset_visible(sqlc.arg('user_id'), a.id);

-- name: ListAssetFacesForCaller :many
-- The asset detail's faces: every face of one asset
-- with the CALLER's label state of its cluster -- the
-- per-user-state discipline's read: clusters are global, labels are private, so the LEFT JOIN
-- picks the caller's row and another account's label never surfaces.
-- The asset itself was already read visibility-scoped by the
-- detail endpoint (the union gate answered its own 404 before this
-- query runs), so
-- the faces need no visibility predicate of their own: a trashed or
-- hidden asset's detail answers 200 and its faces ride along. The
-- cluster_id crosses as NULL for a face the clustering has not
-- assigned yet (label null with it -- nothing to label); the caller's
-- unnamed clusters read as null label columns. Ordered score
-- descending, the face id ascending as the tiebreak.
SELECT f.id,
       f.score,
       f.cluster_id,
       l.name AS label_name,
       l.is_hidden AS label_hidden
FROM faces f
LEFT JOIN user_face_labels l
       ON l.cluster_id = f.cluster_id AND l.user_id = sqlc.arg('user_id')
WHERE f.asset_id = sqlc.arg('asset_id')
ORDER BY f.score DESC, f.id;
