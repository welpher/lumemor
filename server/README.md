# server

Go module. Stack: chi v5 (HTTP), pgx v5 (PostgreSQL driver), sqlc (SQL-first), goose (migrations, auto-run at startup via embed.FS), River (PG-native queue), onnxruntime (local ML inference), libvips / ffmpeg / exiftool (media pipeline).

```
cmd/lumemor/          entrypoint
internal/api/         REST (/api/v1/*) + SSE (/api/v1/events)
internal/auth/        sessions / Argon2id / API keys / TOTP
internal/db/          sqlc-generated code (migrations live in server/migrations/)
internal/events/      in-process domain event bus (shared by SSE and plugin seams)
internal/jobs/        River job definitions, worker pools
internal/media/       ingest, thumbnails, transcoding, metadata
internal/ml/          model registry, inference, vectors (pgvector)
internal/storage/     file layout, content addressing, referenced directories
```
