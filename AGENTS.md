# AGENTS.md

Rules for AI coding agents working in the lumemor repository. Human
contributors should also read [CONTRIBUTING.md](CONTRIBUTING.md); this
file covers the mechanical ground rules that keep the repository
consistent no matter who (or what) writes the code.

## Language

- All repository content is **English only**: code, comments, commit
  messages, documentation, issues, and CI logs.

## Reference projects

- Self-hosted photo libraries other than lumemor are treated as
  **reference projects**. Reading their source to learn domain concepts
  is allowed and encouraged.
- The boundary is strict: **never copy** from reference projects — no
  code, no UI copy, no API shapes. Everything committed here is written
  from scratch, guided by lumemor's own design docs.
- **Never name reference projects** in any file, commit message, issue,
  or generated output. Always use the phrase "reference projects".

## Generated code

Never hand-edit generated output. Change the source of truth and
regenerate through the Makefile:

| Generated path | Source of truth | Regenerate |
|---|---|---|
| `server/internal/api/gen/` | `openapi/lumemor.yaml` | `make api-gen` |
| `openapi/generated/` | `openapi/lumemor.yaml` | `make api-gen` |
| `server/internal/db/gen/` | `server/sqlc.yaml` + `server/internal/db/queries/` | `make sqlc-gen` |

`make api-check` fails when a fresh regeneration differs from the
committed artifacts — run it after touching the OpenAPI contract, and
re-run `make sqlc-gen` (leaving the tree clean) after touching the
queries or schema.

## Conventions

- **Error codes map to i18n keys by lowercasing**:
  `SETUP_ALREADY_COMPLETED` → `setup_already_completed`. All
  translation keys, UI strings included, are lowercase snake_case.
- **Translations** live in `i18n/` as flat JSON. `en.json` is the
  source of truth; every other language file carries the same key set
  and the same `{placeholder}` names. Check with
  `python3 scripts/check_i18n.py`.
- **Default port: 8244.** The server binary, the compose deployment,
  and the acceptance script all assume it.
- **Commits** follow Conventional Commits (`feat:`, `fix:`, `docs:`,
  `chore:`, `refactor:`, ...) with an English imperative subject line.
- **Time assertions against DB values** must truncate to microseconds:
  PostgreSQL stores `timestamptz` at microsecond precision while Linux wall
  clocks carry nanoseconds — an untruncated Go time can never round-trip
  `Equal` against a database value on Linux (macOS clocks happen to be
  microsecond-resolution, which hides the bug locally and surfaces it on
  CI runners).
- **Toolchain versions** (sqlc, oapi-codegen, openapi-generator,
  openapi-typescript, spectral, golangci-lint) are pinned in the root
  `Makefile` and the dev image. Extend the Makefile instead of
  duplicating or restructuring its targets.

## Before reporting work as done

Run the same checks CI runs. Media-pipeline packages (libvips CGO,
exiftool, ffmpeg) are **containerized** — the host installs nothing; run
those through the dev image:

```sh
cd server
gofmt -l .            # empty output
go vet ./...
golangci-lint run     # pinned v2.15.0; or `make docker-lint` from the repo root
go test ./...         # needs a local Docker daemon (testcontainers);
                      # media-dependent tests skip with a message on hosts
                      # without the toolchain
go build ./...
cd ..
make api-lint
make api-check        # regenerates artifacts; the git tree must stay clean
python3 scripts/check_i18n.py
make docker-test      # authoritative full suite (dev image, Linux parity
                      # with CI — platform-dependent failures surface here
                      # before CI does)
```
