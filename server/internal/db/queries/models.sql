-- Model registry queries: the models table is the
-- authority for what is registered and which model is active per slot;
-- the sha256 authority is the per-model manifest on disk, read by the
-- registry, not the database. ActivateModel and DeactivateSlot run
-- inside the activation transaction the registry drives over the pool:
-- one transaction turns the old row off and the new row on, and -- for
-- a clip model whose dimension differs from the column's current one --
-- clears clip_embeddings and ALTERs the column type. The dimension
-- check and the DDL of that switch are executed directly over the
-- transaction in the registry (never through sqlc: the check reads
-- pg_attribute, which the schema-driven generator cannot resolve, and
-- ALTER is DDL); the catalog read is what keeps the dimension policy
-- from ever guessing.

-- name: ListModels :many
-- Every registered model, grouped by slot then id: the listing the
-- admin surface renders.
SELECT * FROM models ORDER BY slot ASC, id ASC;

-- name: ModelByID :one
SELECT * FROM models WHERE id = $1;

-- name: GetActiveModelBySlot :one
-- The one active model of the slot. No row means the slot is unfilled;
-- the registry answers that with ErrNoActiveModel.
SELECT * FROM models WHERE slot = $1 AND is_active;

-- name: InsertModel :exec
-- Registers a model: the download service writes source 'downloaded',
-- the directory scan writes 'custom'. The upsert makes both
-- re-runnable: re-registering refreshes the descriptive fields and
-- leaves is_active exactly as it was.
INSERT INTO models (id, slot, version, source, file_path, input_size, dim, params)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (id) DO UPDATE SET
    slot = EXCLUDED.slot,
    version = EXCLUDED.version,
    source = EXCLUDED.source,
    file_path = EXCLUDED.file_path,
    input_size = EXCLUDED.input_size,
    dim = EXCLUDED.dim,
    params = EXCLUDED.params,
    updated_at = now();

-- name: ActivateModel :exec
-- The activation half of the switch; the registry has already verified
-- the model exists (ModelByID) and its files match their pinned
-- checksums.
UPDATE models SET is_active = true, updated_at = now()
WHERE id = $1;

-- name: DeactivateSlot :exec
-- The deactivation half of the switch. Matching no row is the plain
-- success: the slot was unfilled.
UPDATE models SET is_active = false, updated_at = now()
WHERE slot = $1 AND is_active;

-- name: DeleteModel :exec
-- The removal of one registration: the models row goes first (the
-- handler deletes its directory after), and the still-referencing
-- products keep resolving through their own foreign keys.
DELETE FROM models WHERE id = $1;
