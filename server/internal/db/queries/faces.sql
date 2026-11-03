-- Face queries (M3a Task 6): the face detection job's write surface.
-- A detection run replaces the asset's faces OF THE MODEL IT RAN (the
-- delete-then-insert below runs in the worker's transaction) while the
-- faces of every other model survive untouched -- the model-switch
-- coexistence policy: products partition by model_id, queries filter
-- by the active model, and switching back revives the old rows
-- without recomputing anything. The recognition job (Task 7) owns the
-- embedding half of the table.
--
-- Reserved query names of Task 7 (face recognition) -- do not claim
-- them elsewhere:
--   UpdateFaceEmbedding          (embedding + embed_model_id of a face)
--   ListFacesNeedingEmbedding    (the active detect model's faces of an
--                                 asset whose embedding is still NULL)

-- name: InsertFace :exec
-- Stores one detected face of one model. embedding and cluster_id
-- start NULL: the recognition job fills the embedding, the clustering
-- step assigns the cluster. The per-mille CHECK bounds are the table's
-- own; the decoder clamps before it gets here.
INSERT INTO faces (id, asset_id, bbox_x, bbox_y, bbox_w, bbox_h, score, model_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: DeleteFacesByAsset :exec
-- Removes the asset's faces of ONE model: the detection (re)run's
-- replace step for that model. Faces of other models survive -- see
-- the coexistence note above.
DELETE FROM faces WHERE asset_id = $1 AND model_id = $2;

-- name: ListFacesByAsset :many
-- The asset's faces of one model, strongest first. The recognition
-- job walks them for the active detect model; the people surfaces of
-- the next milestone read them through the same filter.
SELECT * FROM faces WHERE asset_id = $1 AND model_id = $2
ORDER BY score DESC;
