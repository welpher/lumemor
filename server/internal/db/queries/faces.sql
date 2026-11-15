-- Face queries: the face detection job's write surface.
-- A detection run replaces the asset's faces OF THE MODEL IT RAN (the
-- delete-then-insert below runs in the worker's transaction) while the
-- faces of every other model survive untouched -- the model-switch
-- coexistence policy: products partition by model_id, queries filter
-- by the active model, and switching back revives the old rows
-- without recomputing anything.
--
-- Face recognition landed here as the architecture's brief
-- pinned it: the recognition worker re-detects the asset's landmarks
-- with the active detect model and matches them against the asset's
-- FULL row set of that model (ListFacesByAsset), so every matched
-- row is re-embedded idempotently; the reserved name
-- ListFacesNeedingEmbedding went unused -- a needing-embedding
-- pre-filter would silently skip same-model rows whose stored
-- geometry no longer redetects, leaving a stale embedding in place.

-- name: InsertFace :exec
-- Stores one detected face of one model. embedding and cluster_id
-- start NULL: the recognition job fills the embedding, the clustering
-- step assigns the cluster. The per-mille CHECK bounds are the table's
-- own; the decoder clamps before it gets here.
INSERT INTO faces (id, asset_id, bbox_x, bbox_y, bbox_w, bbox_h, score, model_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: InsertManualFace :one
-- Stores one hand-drawn face (POST /faces): the caller's drawn box
-- becomes the row as-is, with no detector to name (model_id NULL) and
-- no embedding (the recognition never saw it). The score is 1.0 --
-- the caller's own certainty, and the value that puts the drawn face
-- first among the detail's faces. The row returns: the caller's
-- assign flow needs the new id.
INSERT INTO faces (id, asset_id, bbox_x, bbox_y, bbox_w, bbox_h, score, model_id)
VALUES ($1, $2, $3, $4, $5, $6, 1.0, NULL)
RETURNING id, bbox_x, bbox_y, bbox_w, bbox_h, score;

-- name: DeleteFacesByAsset :exec
-- Removes the asset's faces of ONE model: the detection (re)run's
-- replace step for that model. Faces of other models survive -- see
-- the coexistence note above.
DELETE FROM faces WHERE asset_id = $1 AND model_id = $2;

-- name: ListFacesByAsset :many
-- The asset's faces of one model, strongest first. The recognition
-- job walks them for the active detect model: the matching needs the
-- identity, the box and the score. The columns are explicit on
-- purpose -- a bare SELECT * drags the embedding column along, and a
-- NULL vector does not scan into the generated Go type (the
-- embedding's readers select it explicitly, never through the star).
SELECT id, bbox_x, bbox_y, bbox_w, bbox_h, score FROM faces
WHERE asset_id = $1 AND model_id = $2
ORDER BY score DESC;

-- name: UpdateFaceEmbedding :exec
-- Stamps one face with the recognition model's embedding: the
-- recognition job's per-match write. The clustering step reads the
-- rows back through ListEmbeddedFacesByEmbedModel in the same
-- transaction, so the cluster pass always sees its own writes.
UPDATE faces SET embedding = $2, embed_model_id = $3, updated_at = now()
WHERE id = $1;

-- name: ListEmbeddedFacesByEmbedModel :many
-- Every face the given recognize model embedded, strongest first,
-- ties broken deterministically by id: the clustering step's input.
-- Rows of every other recognize model are invisible here -- the
-- model-switch coexistence policy on the embedding side -- and rows
-- without an embedding never enter the clustering.
SELECT id, cluster_id, score, embedding FROM faces
WHERE embed_model_id = $1 AND embedding IS NOT NULL
ORDER BY score DESC, id;

-- name: InsertCluster :exec
-- Creates one cluster row with its representative face: the
-- diff-apply's birth step, the representative being the cluster's
-- highest-score member.
INSERT INTO face_clusters (id, representative_face_id) VALUES ($1, $2);

-- name: UpdateClusterRepresentative :exec
-- Points one cluster at its new representative face: the diff-apply's
-- reuse step (a re-homed or re-converged cluster follows its current
-- best member).
UPDATE face_clusters SET representative_face_id = $2, updated_at = now()
WHERE id = $1;

-- name: DeleteCluster :exec
-- Removes one cluster row: the diff-apply's dissolution step for a
-- cluster the new clustering no longer populates (user_face_labels
-- cascade with it -- a person without faces is a person without a
-- name), and the deletion sweep's dissolve for a cluster whose
-- members are all going away.
DELETE FROM face_clusters WHERE id = $1;

-- name: ListClusters :many
-- Every cluster row, in id order: the diff-apply's existing-state
-- read (the deterministic order settles identity-match ties).
SELECT id, representative_face_id FROM face_clusters ORDER BY id;

-- name: ListClusterFaces :many
-- The member faces of one cluster, strongest first, ties broken by
-- id: the diff-apply's membership read (the identity match's majority
-- rule) and the deletion sweep's survivor selection.
SELECT id, score FROM faces WHERE cluster_id = $1
ORDER BY score DESC, id;

-- name: UpdateFaceCluster :exec
-- One face's cluster assignment of the clustering step: the cluster
-- it joined, or NULL to release it to noise. The step walks every
-- embedded face inside its own transaction (the batch lands whole;
-- sqlc's analyzer cannot express the unnest form of the same write,
-- and the per-family-scale cost sits inside the clustering run's
-- envelope).
UPDATE faces SET cluster_id = $2, updated_at = now() WHERE id = $1;

-- name: CountFaceClusters :one
-- The library's cluster total: the face.clustered event's honest
-- count, read inside the clustering transaction.
SELECT count(*) FROM face_clusters;

-- name: CountLabeledFaceClusters :one
-- The clusters carrying at least one user label: the face.clustered
-- event's labeled count.
SELECT count(DISTINCT cluster_id) FROM user_face_labels;

-- name: RehomeClusterRepresentatives :exec
-- The deletion sweep's first half, ahead of any statement that
-- removes the faces (the detection replace, the asset purges): a
-- cluster whose representative face is among the removed ones
-- (asset_id = $1, optionally scoped to model $2 -- NULL = every
-- model of the asset) re-points its representative to the cluster's
-- best surviving member, so the representative foreign key never
-- dangles.
UPDATE face_clusters fc
SET representative_face_id = best.id, updated_at = now()
FROM (
    SELECT DISTINCT ON (f.cluster_id) f.cluster_id, f.id
    FROM faces f
    WHERE f.cluster_id IS NOT NULL
      AND NOT (f.asset_id = $1 AND (sqlc.narg('model_id')::text IS NULL OR f.model_id = sqlc.narg('model_id')))
    ORDER BY f.cluster_id, f.score DESC, f.id
) AS best
WHERE fc.id = best.cluster_id
  AND fc.representative_face_id IN (
      SELECT f.id FROM faces f
      WHERE f.asset_id = $1 AND (sqlc.narg('model_id')::text IS NULL OR f.model_id = sqlc.narg('model_id'))
  );

-- name: DissolveVacatedClusters :exec
-- The deletion sweep's second half: a cluster whose representative
-- is among the removed faces AND that has no member surviving the
-- removal loses its row (user_face_labels cascade -- the honest
-- person-without-faces semantics, applied where the faces die).
DELETE FROM face_clusters fc
WHERE fc.representative_face_id IN (
    SELECT f.id FROM faces f
    WHERE f.asset_id = $1 AND (sqlc.narg('model_id')::text IS NULL OR f.model_id = sqlc.narg('model_id'))
)
AND NOT EXISTS (
    SELECT 1 FROM faces f
    WHERE f.cluster_id = fc.id
      AND NOT (f.asset_id = $1 AND (sqlc.narg('model_id')::text IS NULL OR f.model_id = sqlc.narg('model_id')))
);
