-- CLIP embedding queries: the embedding job's write
-- surface. The upsert keys on (asset_id, model_id) -- the primary key
-- the schema pins -- so a re-run of the job overwrites the same row
-- and a rerun of the model converges without piling rows. The
-- coexistence policy needs no delete-by-model statement: switching
-- models keeps every model's rows (queries filter by the active
-- model), and the one destructive arm -- the cross-dimension switch
-- that clears the table before ALTERing the column -- runs as the
-- registry's own transactional DELETE (the catalog read and the DDL
-- are beyond sqlc's schema-driven analysis).

-- name: UpsertClipEmbedding :exec
-- Stores one asset's embedding as one model's product: the job's
-- per-asset write. The INSERT guard (the embedding's length must
-- equal the column's current dimension, read from the catalog by the
-- job) runs BEFORE this statement; the upsert itself makes re-runs
-- and reruns converge.
INSERT INTO clip_embeddings (asset_id, model_id, embedding)
VALUES ($1, $2, $3)
ON CONFLICT (asset_id, model_id) DO UPDATE SET
    embedding = EXCLUDED.embedding,
    updated_at = now();
