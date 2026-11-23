<div align="center">

# lumemor

A self-hosted photo & video library designed for families.

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
| Places (city grouping + map) | Yes | Yes |
| Tags | Yes | Yes |
| Hide from timeline & trash with retention | Yes | Yes |
| LAN share links (token) | Yes | Yes |
| E2E-encrypted backup to your own cloud (v1.x, subscription) | Yes | Yes |
| Administration (users, models, jobs) | No | Yes |
| Directory album creation (server filesystem) | N/A | Yes |

This table tracks the v1 surface; full scope and staging live in the design docs.

## Networking

lumemor is self-hosted and works offline. Three things reach the network, each on the instance's own terms:

- **Map tiles** — the browser loads the base map straight from the tile source the administrator configures (admin → settings → map tiles). Nothing ships as a default: a public tile service's usage policy does not allow carrying the traffic of an application's whole install base, and a source that works in one network may not work in another. The pins (your cities and their photo counts) always come from your own library and never cross the tile line. Set the URL of a source your network can reach, serving a style document (vector) or a TileJSON descriptor (raster) — the browser fetches it, so a LAN address is fine. Sources come in roughly three kinds:
  - a self-hosted tile server on your own machine or LAN (e.g. a planet tileset served by your own stack);
  - a self-hosted style and glyph mirror reachable from your network;
  - a hosted provider of your choosing (check its attribution and usage terms; the map shows the attribution the source declares).
  Leave the field empty and the map shows its pins over a stated missing base layer — never a blank canvas.
- **Release check** — the server asks a release feed for its newest version tag once a day (configurable, and switchable off in admin → settings). Only the version is compared; the request carries no photo, no library information and no instance identifier, private and loopback addresses are refused, and redirects are never followed. A failed check stays quiet. Set `LUMEMOR_UPDATE_CHECK_DISABLED=1` to turn it off at the deployment level, and `https_proxy` / `http_proxy` / `no_proxy` to route the request through your proxy.
- **Model downloads** — the AI model weights are fetched once, pinned by checksum, from the manifest's upstreams or your mirror (`LUMEMOR_MODELS_HF_MIRROR`). An instance that never downloads them simply runs without AI.

## Known limitations

The boundaries this release ships with on purpose — one family per
instance, an optional AI download, a tile source you choose, read-only
directory sources — are collected in
[`docs/known-limitations.md`](docs/known-limitations.md).

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
