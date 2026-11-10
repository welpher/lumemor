-- OCR text queries: the OCR job's write surface. The
-- table keys on asset_id alone -- one text per asset, the active
-- model's -- so the upsert overwrites the row on every re-run and
-- rerun. The language column is deliberately not written: the default
-- group's model reads a mixed Chinese-English corpus, and a NULL
-- language is the honest statement (no single tag describes the
-- corpus); a later milestone that narrows the model family can start
-- writing it.

-- name: UpsertOcrText :exec
-- Stores one asset's extracted text as the active OCR model's
-- product.
INSERT INTO ocr_texts (asset_id, model_id, text)
VALUES ($1, $2, $3)
ON CONFLICT (asset_id) DO UPDATE SET
    model_id = EXCLUDED.model_id,
    text = EXCLUDED.text,
    updated_at = now();
