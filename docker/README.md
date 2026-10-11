# docker

- Multi-stage build: CGO builder (libvips + onnxruntime) → debian-slim runtime (ffmpeg + exiftool + bundled models), linux/amd64 + arm64
- Custom PostgreSQL image based on `pgvector/pgvector:pg17` with pg_jieba compiled in
- Target: `docker compose up` = lumemor + postgres, two containers, nothing else

Dockerfiles and compose files land here during implementation.
