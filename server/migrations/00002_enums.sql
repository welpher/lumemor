-- +goose Up
-- Native enums for every enumerated value in the schema. Values are pinned
-- by the milestone plan; adding a value later requires a new migration.

CREATE TYPE user_role AS ENUM ('admin', 'user');
CREATE TYPE media_kind AS ENUM ('image', 'video');
CREATE TYPE storage_kind AS ENUM ('blob', 'external');
CREATE TYPE album_type AS ENUM ('sync', 'directory', 'virtual');
CREATE TYPE taken_at_source AS ENUM ('exif', 'file', 'import');
CREATE TYPE session_kind AS ENUM ('web', 'mobile');
CREATE TYPE model_slot AS ENUM ('face_detect', 'face_recognize', 'clip_embed', 'ocr', 'stt');
CREATE TYPE model_source AS ENUM ('builtin', 'downloaded', 'custom');
CREATE TYPE job_state AS ENUM ('queued', 'running', 'succeeded', 'failed');

-- +goose Down
DROP TYPE job_state;
DROP TYPE model_source;
DROP TYPE model_slot;
DROP TYPE session_kind;
DROP TYPE taken_at_source;
DROP TYPE album_type;
DROP TYPE storage_kind;
DROP TYPE media_kind;
DROP TYPE user_role;
