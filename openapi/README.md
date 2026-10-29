# openapi

Spec-first API contract: `lumemor.yaml` is the single source of truth for the
Go server stubs and every generated client.

- oapi-codegen → Go server interface + types (`server/internal/api/gen`, config in `codegen/go.yaml`)
- openapi-typescript → TypeScript types (`generated/web/api.d.ts`)
- openapi-generator (dart-dio) → Dart client package (`generated/mobile`, config in `codegen/mobile.yaml`)

Workflow, run from the repository root:

- `make api-lint` — lint `lumemor.yaml` with Spectral (ruleset in the root `.spectral.yaml`)
- `make api-gen` — regenerate all three outputs; the artifacts are committed
- `make api-check` — regenerate and fail on any drift

Tool versions are pinned in the root `Makefile` (plus `openapitools.json`
for the generator) so regeneration is deterministic. Never hand-edit
generated files: change `lumemor.yaml` and run `make api-gen`.
