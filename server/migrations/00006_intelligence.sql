-- +goose Up
-- Intelligence domain: registered models, detected faces, face clusters,
-- per-user cluster labels, CLIP embeddings, and OCR text.
--
-- faces and face_clusters reference each other, so the cluster side of the
-- cycle is added with ALTER TABLE after both tables exist.

CREATE TABLE models (
    id         text PRIMARY KEY, -- slug
    slot       model_slot NOT NULL,
    version    text NOT NULL,
    source     model_source NOT NULL,
    file_path  text NOT NULL,
    input_size integer,
    dim        integer,
    params     jsonb NOT NULL DEFAULT '{}',
    is_active  boolean NOT NULL DEFAULT false,
    added_at   timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- At most one active model per slot.
CREATE UNIQUE INDEX models_active_slot_uq
    ON models (slot)
    WHERE is_active;

CREATE TABLE face_clusters (
    id                    uuid PRIMARY KEY,
    representative_face_id uuid NOT NULL,
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE faces (
    id             uuid PRIMARY KEY,
    asset_id       uuid NOT NULL REFERENCES assets (id) ON DELETE CASCADE,
    cluster_id     uuid REFERENCES face_clusters (id) ON DELETE SET NULL,
    -- Bounding box in per-mille units (0-1000) relative to the full image.
    bbox_x         integer NOT NULL CHECK (bbox_x BETWEEN 0 AND 1000),
    bbox_y         integer NOT NULL CHECK (bbox_y BETWEEN 0 AND 1000),
    bbox_w         integer NOT NULL CHECK (bbox_w BETWEEN 0 AND 1000),
    bbox_h         integer NOT NULL CHECK (bbox_h BETWEEN 0 AND 1000),
    score          real NOT NULL,
    model_id       text NOT NULL REFERENCES models (id) ON DELETE CASCADE,
    embedding      vector,
    embed_model_id text REFERENCES models (id),
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE face_clusters
    ADD CONSTRAINT face_clusters_representative_face_fkey
    FOREIGN KEY (representative_face_id) REFERENCES faces (id);

CREATE INDEX faces_asset_idx ON faces (asset_id);

CREATE TABLE user_face_labels (
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    cluster_id uuid NOT NULL REFERENCES face_clusters (id) ON DELETE CASCADE,
    name       text NOT NULL,
    is_hidden  boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, cluster_id)
);

-- Face recognition clusters faces in application memory, so no ANN index is
-- needed here.
CREATE TABLE clip_embeddings (
    asset_id  uuid NOT NULL REFERENCES assets (id) ON DELETE CASCADE,
    model_id  text NOT NULL REFERENCES models (id) ON DELETE CASCADE,
    embedding vector(512) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (asset_id, model_id)
);

-- pgvector requires a fixed dimension for HNSW; the registered clip-embed
-- model must therefore emit 512-dimensional vectors.
CREATE INDEX clip_embeddings_hnsw_idx
    ON clip_embeddings USING hnsw (embedding vector_cosine_ops);

CREATE TABLE ocr_texts (
    asset_id  uuid PRIMARY KEY REFERENCES assets (id) ON DELETE CASCADE,
    model_id  text NOT NULL REFERENCES models (id),
    text      text NOT NULL,
    language  text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS ocr_texts;
DROP TABLE IF EXISTS clip_embeddings;
DROP TABLE IF EXISTS user_face_labels;
-- CASCADE drops the circular foreign key from face_clusters.
DROP TABLE IF EXISTS faces CASCADE;
DROP TABLE IF EXISTS face_clusters;
DROP TABLE IF EXISTS models;
