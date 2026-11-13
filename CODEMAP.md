# CODEMAP

One page so you can add a feature without spelunking.

## server/ (Go)

- `internal/api/` — the HTTP layer: route handlers (one file per resource,
  e.g. `albumaudience.go`, `batch.go`), the write/validation helpers, the
  generated contract types (`gen/` — never hand-edit).
- `internal/jobs/` — the River workers: the ingest pipeline fan-out
  (metadata/thumbnail/transcode), the scan job (`album_rescan.go`), the
  trash expiry, the models download. Job progress writes `job_runs`.
- `internal/db/` — sqlc: queries (`gen/`), migrations
  (`00001_extensions.sql` …), the settings accessors.
- `internal/media/` — the media toolchain wrappers (probe, transcode,
  thumbnails).
- `internal/storage/` — the blob tree (`originals/<userId>/…`, the storage
  template engine, `data/trash/` lifecycle).
- `internal/geodata/` — the baked GeoNames loader and the in-memory
  resolution grid.
- `internal/ml/` — the onnxruntime inference layer (model registry, slots).

## web/ (SvelteKit + Svelte 5)

- `src/lib/timeline/` — **the wall family**: `Wall.svelte` (the unified
  wall: rendering + month axis + selection flow), `FlatWall.svelte` (the
  query-mode rank wall), `CuratedWall.svelte` (favorites/hidden wrapper),
  `PhotoTile.svelte`, `MonthScrubber.svelte`, `BatchBar.svelte`.
- `src/lib/albums/` — the album surfaces: create/edit dialogs,
  `PathCompletion.svelte`, `AudienceField.svelte`.
- `src/lib/components/` — the primitives (`Button`, `Modal`, `Drawer`,
  `IconButton`, the `anchored` portal action).
- `src/lib/identity.ts` — the identity presentation rule shared by the
  walls, the attribution lines and the audience pickers: the display
  name first, the email as the fallback and the duplicate-name
  disambiguator.
- `src/lib/viewer/` — `Viewer.svelte` (the route-level viewer),
  `InfoDrawer.svelte` (the read-only info panel).
- `src/lib/auth/` — the session store (`session.svelte.ts`), pairing.
- `src/lib/theme/` — `theme.css` (the token single source, the `@theme`
  block), the z-ladder.
- `src/lib/i18n/` — the locale stores; `scripts/check_i18n.py` (repo root)
  enforces parity + the `NON_ERROR_KEYS` allowlist.
- `src/routes/(content)/` — the family shell: `timeline/`, `albums/`,
  `people/`, `places/`, `search/`, `favorites/`, `hidden/`, `trash/`,
  `a/[id]/` (the viewer route), `settings/`.
- `src/routes/(tool)/` — the admin shell: `admin/users`, `admin/settings`,
  `admin/models`, `admin/jobs` (the full-bleed rail layout).
- `src/routes/(auth)/` — login; the first-run wizard rides `/`.
- `e2e/` — the Playwright stories (one stack lifetime per spec).

## Root

- `openapi/lumemor.yaml` — the contract (hand-edited; the generated
  clients under `openapi/generated/` follow).
- `i18n/` — `en.json` + `zh-Hans.json`.
- `docker/server.Dockerfile` — the image build (the geodata bake, the
  model-manifest fetch machinery, the onnxruntime pin).
- `scripts/acceptance_m*.sh` — the serial acceptance chain (each script
  owns a compose stack; `acceptance_m3d.sh` chains the whole line).
- `AGENTS.md`, `docs/rules/`, `CONTRIBUTING.md` — the conventions stack.
