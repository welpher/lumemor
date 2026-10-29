-- +goose Up
-- Vector similarity support for CLIP embeddings. The Chinese text-search
-- tokenizer extension is deferred until full-text search lands.
CREATE EXTENSION IF NOT EXISTS vector;

-- +goose Down
DROP EXTENSION IF EXISTS vector;
