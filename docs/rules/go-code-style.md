# Go code style

The authority split: **`gofmt` owns formatting** (tabs, wrapping, spacing —
run it, never argue with it), **`golangci-lint` owns the lint layer** (run via
`make docker-lint`), and this file owns what machines cannot check: naming,
structure, error patterns, and the gotchas.

## Naming

- Exported `PascalCase`, package-internal `camelCase`; acronyms stay upper
  (`ID`, `URL`, `API`).
- Tests: `Test<Subject><Behavior>` (`TestBatchSetSemantics`,
  `TestAlbumRescanIdempotent`) — the name states the pin, the body proves it.
- Error values `ErrX`; sentinel errors live near their domain.
- Handler files carry their route's resource (`albumaudience.go`,
  `adminsettings.go`); queries follow the sqlc file-per-domain grouping.

## Structure

- **The dependency direction is one-way**: `api → jobs → db/storage →
  media/geodata/…`. Left may import right; right must never import left.
- Handlers own HTTP shape (decode, validate, write); domain logic lives in
  the jobs/db/media layers they call.
- Contract types come from the generated `apigen`/`dbgen` packages — never
  hand-duplicate a shape the contract already defines.

## Comments

Comments are **design records**, not narrator text: dense, stating WHY the
code is shaped this way, matching the surrounding file's voice. Read two
neighbors before writing a third. A comment that only restates the code is
noise; a comment that pins a decision is the point.

## Errors and reports

- Validation failures use the established helpers (`writeValidationError`
  with field names); distinct failure causes get distinct contract error
  codes with both locales in `i18n/`.
- Set-semantics writes answer honest reports (`{affected, missing}` style
  partitions), never silent partial success.

## Gotchas (each has bitten once)

- **µs truncation**: Postgres timestamps truncate to microseconds — truncate
  BOTH sides before comparing times.
- **`pgUUID` conversions**: wrap/unwrap at the query boundary; a raw
  `uuid.UUID` in a query arg is a bug.
- **Never `Update` without a `WHERE`** — the exemplar-update trap updates
  every row.
- **Migrations**: `NNNNN_snake_description.sql`, up and down in the same
  file; a `down` that cannot truly restore says so honestly in its comment.
