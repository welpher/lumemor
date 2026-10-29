<div align="center">

# lumemor

**Lume (light) + Memor (memory)** — a self-hosted photo & video library designed for families.

[![License: AGPL v3](https://img.shields.io/badge/License-AGPL_v3-blue.svg)](LICENSE)
[![Status](https://img.shields.io/badge/status-development%20in%20progress-yellow)](#status)

</div>

> [!WARNING]
> **Status: development in progress.** The M1 skeleton is in place — Go server with first-run setup, OpenAPI contract with generated clients, docker compose deployment — and backend development is under way. Do not store irreplaceable photos in lumemor until a stable release exists. And even then, keep a [3-2-1 backup](https://www.backblaze.com/blog/the-3-2-1-backup-strategy/) of everything you love.

> [!NOTE]
> lumemor is built around one privacy rule: **your photos never touch lumemor-operated infrastructure — not even ciphertext.** All AI (faces, semantic search, OCR) runs locally on your own hardware.

## Principles

| | |
|---|---|
| 🏠 **Family-first** | Multi-user, shared timeline, album-level permissions — designed for the whole family, not for sysadmins |
| 🔒 **Privacy absolute** | Local-only ML inference; zero cloud calls, zero telemetry |
| 📦 **Single container** | One Go binary (web UI embedded) + PostgreSQL; no Redis, no ML sidecar, no microservices |
| 🀄 **Chinese as a first-class citizen** | pg_jieba full-text search, bilingual CLIP |
| 📶 **Remote without exposure** | P2P direct connections first (IPv6 / NAT hole punching); encrypted-byte relay as paid fallback |
| ☁️ **Backup to *your* cloud** | E2E chunk-encrypted push to your own S3 / OSS / WebDAV account — lumemor never stores photos |

## Features

| Features | Mobile | Web |
|---|---|---|
| Multi-user family timeline (own + shared albums, deduplicated) | Yes | Yes |
| Auto backup with selective albums | Yes | N/A |
| Optional delete-from-phone after verified upload | Yes | N/A |
| Offline timeline cache | Yes | N/A |
| First-run pairing (mDNS / QR / manual URL) | Yes | N/A |
| Source albums (sync) & directory albums (in-place reference) | Yes | Yes |
| Virtual albums mixed from anything you can see | Yes | Yes |
| Album-level per-user visibility, lock & trash | Yes | Yes |
| Unified search: filters + text + semantic | Yes | Yes |
| Face clustering with per-user private naming | Yes | Yes |
| On this day (memories) | Yes | Yes |
| Places (city grouping) | Yes | Yes |
| Tags | Yes | Yes |
| Hide from timeline & trash with retention | Yes | Yes |
| LAN share links (token) | Yes | Yes |
| E2E-encrypted backup to your own cloud (v1.x, subscription) | Yes | Yes |
| Administration (users, models, jobs) | No | Yes |
| Directory album creation (server filesystem) | N/A | Yes |

This table tracks the v1 surface; full scope and staging live in the design docs.

## Translations

`en` (source) and `zh-Hans` are officially maintained. Translations are flat JSON in [`i18n/`](i18n/) — contributions welcome once development starts (guide coming soon).

## Repository layout

```
lumemor/
├── server/     # Go: chi + pgx + sqlc + goose + River + onnxruntime/libvips/ffmpeg/exiftool
├── web/        # SvelteKit (adapter-static → embedded into the Go binary via embed.FS)
├── mobile/     # Flutter (Riverpod + drift)
├── openapi/    # OpenAPI spec-first: the single source of truth for all three clients
├── i18n/       # Flat JSON (en + zh-Hans), shared by web and mobile
├── docker/     # Multi-stage builds + PG image with pgvector + pg_jieba
└── docs/       # Deployment guide, CONTRIBUTING, API reference
```

## Contributing

- **No CLA** — contributions merge under AGPL-3.0 and contributors keep their copyright
- See [CONTRIBUTING.md](CONTRIBUTING.md); translations: [`i18n/`](i18n/)
- License: **[AGPL-3.0](LICENSE)**

## Contributors

<a href="https://github.com/welpher/lumemor/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=welpher/lumemor" />
</a>
