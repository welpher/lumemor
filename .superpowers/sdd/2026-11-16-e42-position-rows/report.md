# E42 · Position rows — implementation report

Branch `e42-position-rows`, PR **#59** (opened, not merged).
Worktree `/Users/welpher/coding/lumemor-e42` (base `b29b64f`).

---

## 1. The model as landed (each pin → what was built)

### Pin 1 — the key fork (migration `00027_position_rows.sql`)

- `assets_owner_checksum_uq` (total over owner+checksum) dropped;
  `assets_owner_checksum_blob_uq` = `unique(owner_id, checksum) WHERE storage_kind='blob'`
  (total across live and trashed, as 00008 argued) and
  `assets_owner_file_path_external_uq` = `unique(owner_id, file_path) WHERE storage_kind='external'`.
- **Zero data rewrites.** The old total uniqueness was stricter than both new keys, so every
  existing row satisfies its domain's key. No row is touched by the up migration.
- 00008's three-state comment gained the "blob domain" qualifier (comment only; the applied SQL
  is untouched).
- The **down** is a real inverse, not an index swap: it folds every (owner, checksum) twin into
  the ranked keeper (blob first, live before dead, earliest ingest), re-links the album
  memberships the dropped row fed (`ON CONFLICT DO NOTHING`), then recreates the total unique
  index. Without the fold the previous index could not be created on data the fork legitimizes.
  `TestPositionRowsMigrationDown` pins up → insert a twin → down (one row, two memberships) → up.
- **Index support for the predicate:** `assets_owner_checksum_idx` (owner_id, checksum), full
  (both kinds, all states). It is both the predicate's group probe and the full-domain content
  lookup (a partial unique index only answers queries that prove its predicate, so the
  uniqueness no longer implies a usable lookup index). `EXPLAIN` on the stack shows the
  predicate's probe on `assets_owner_checksum_idx` and the timeline scan on
  `assets_timeline_idx`.

### Pin 2 — no cross-storage dedup on the scan, full-domain check on the upload

- **Scan (`album_rescan.go` addArm):** the walked file always attempts its own external row at
  its own path. The only collision the insert can meet is the position key, resolved in three
  states (`resolvePositionCollision`): LIVE row at the path → link this album's membership (an
  overlapping root or a racing scan); ORPHAN-MARKED row (dead, `trash_path IS NULL`) → revive it
  with the file's current identity (`ReviveExternalAsset`) and re-enqueue the pipeline; TRASHED
  WITH A COPY → unreachable by construction (its `file_path` names the trash area), logged and
  skipped.
- **Convergence (`ConvergeAlbumMembersToPositionRows`):** one statement per scan, after the walk
  and the vanish arm. It deletes this album's memberships whose asset has the same owner+checksum
  as one of the album's OWN live external members, restricted to candidates that are blob rows or
  orphan-marked dead rows. Two same-content files in a folder are both live external members and
  both stay. Trashed-with-copy members stay (their restore returns them to the album).
  Idempotent, and the run announces the membership move through the usual `album.items-changed`.
- **Upload (`assets.go`):** `lookupExistingUpload` walks the content lookups of both parts over
  the full domain (new `resolveContentLookups`), ordering live before trashed and blob before
  source; then the device identity. A live source row answers `duplicate`; a trashed row
  (either kind) answers `trashed` (the client's restore-or-save-as unchanged); an orphan-marked
  source row is purged through `purgeAssetRow` (faces re-home/dissolve, derivatives removed) and
  the upload proceeds to a fresh insert. `uploadUniquenessConstraints` now names
  `assets_owner_checksum_blob_uq`.
- **Motion (`jobs/motion.go`):** the reuse lookup narrowed to blob rows
  (`GetBlobAssetByOwnerChecksum`). A source file must never be adopted as a pair's component
  (the pair's deletion would reach a file the server never owned).

### Pin 3 — the library display predicate

One SQL function, the `caller_asset_visible` precedent (`00027`):
`asset_library_representative(target uuid) RETURNS boolean` — `NOT EXISTS` a same-group LIVE row
that ranks better (a live blob always; among sources an earlier `(created_at, id)`). STABLE /
LANGUAGE sql so it stays an inlining candidate.

| surface | query | applied |
|---|---|---|
| timeline | `ListTimelineAssets` | yes |
| My photos (sync album) | `ListAlbumTimelineAssets` / `ListAlbumItemsForCaller` / counts | yes (sync face) |
| favorites | `ListFavoriteAssets` | yes |
| archive | `ListArchivedAssets` | yes (see deviation 2) |
| search | `scope.newScope` + the text-match materialization + facets | yes |
| places | `ListCallerPlaces`, `ListPlaceCoverCandidates`, `ListPlaceTimelineAssets` | yes |
| memories | album timeline / items / counts / cover pool (memory face) | yes |
| people | `ListLabeledPeople`, `ListUnnamedClusters`, `CountUnnamedClusters`, `CountClusterFaces`, `ListClusterFacesForCaller`, `ListPersonTimelineAssets` | yes |
| counts | album `item_count`/`photo_count` (wall, detail), tag counts | yes (album counts split by type) |
| **trash view** | `ListTrash` | **no** (raw rows) |
| **directory album wall** | album timeline / items (`al.type = 'directory'`) | **no** (members) |

The album split is by face, not by endpoint: a directory album's timeline IS its wall, a sync
album and a memory are library views of what they hold (a memory holding both copies shows one
tile, and its count agrees with its wall).

### Pin 4 — the delete affordance is per copy

`Viewer.svelte`'s `canTrash` keeps its logic and loses the retired all-albums narrative: the sync
membership is the app's own blob (deletable), a directory album backs the row's trash exactly
when `sourceWritable`, a row reached only through a read-only source offers nothing. The duplicate
comment block the old narrative left was merged into one.

### Pin 5 — the twin note

- `AssetDetail.contentCopies` (required; GET and PATCH; examples 2 and 0) — the count of OTHER
  live rows of the same owner with the same checksum, computed in `GetAssetDetail`
  (`::bigint` sub-query), owner-scoped and row-agnostic.
- The viewer's delete toast: new `viewer_trashed_done` ("Moved to the trash.") plus
  `viewer_trashed_copies_note` when `contentCopies > 0`. Both locales; `scripts/check_i18n.py`
  allowlists the two UI keys.

### Pin 6 — the rename dissolution

Verified by the new e2e scene 6: a renamed file is one row's death and another's birth; the album
and the timeline keep it visible under the new name (and the new row represents the content once
the last live copy is the fresh one). No content dedup can hide the new path.

### The people domain

The people LISTS, their counts, the grids and the best-crop read filter through the predicate; a
hidden twin's face rows exist but never list. `m3-people` scene 11 proves it end to end (a
byte-identical folder copy of a seeded portrait: its own row, its own detected face, the person's
photo count and grid unchanged).

---

## 2. Gates (quoted)

- `make web-check` — format/lint/check/test green; `svelte-check found 0 errors and 0 warnings`,
  `Tests 880 passed (880)`.
- `make docker-lint` — `0 issues.` (gofmt + go vet + golangci-lint).
- `python3 scripts/check_i18n.py` — `i18n check OK: 2 file(s) [en, zh-Hans], 772 key(s), 33 error code(s)`.
- `make api-lint` — `No results with a severity of 'error' found!`
- `make api-check` (clean tree, run post-commit) — no diff; `make sqlc-gen` — no drift.
- `make ci-local` — **all green**:
  ```
  === ci-local 1/6: the contract lints ===        No results with a severity of 'error' found!
  === ci-local 2/6: the web suite, cold ===       All matched files use Prettier code style! … Tests 880 passed (880)
  === ci-local 3/6: the server suite, cold ===    0 issues.
  ok  …/internal/api    620.319s
  ok  …/internal/db      12.938s
  ok  …/internal/jobs   430.447s
  ok  …/internal/ml      47.614s   (all other packages ok)
  === ci-local 4/6: the API artifacts regenerate without drift (twice) ===
  === ci-local 5/6: the sqlc output regenerates without drift ===
  === ci-local 6/6: the locales agree ===
  ci-local: all green
  ```
  (The cold web stage needed a registry mirror through this network:
  `COLD_WEB_ENV='-e npm_config_registry=https://registry.npmmirror.com'` after three
  ECONNRESETs on the proxy path; the Go stage used the pinned `goproxy.cn`.)

---

## 3. E2E (fresh stack per spec: `docker compose down --remove-orphans && docker volume rm -f lumemor-e42_pgdata`)

| spec | result |
|---|---|
| **e42-position-rows** (new) | **6/6 passed** — upload → scan (own position row, one timeline tile, per-copy delete, twin note, survival, rename); scan → upload duplicate; vanish → replace |
| m2-full-chain | 12/12 |
| m3-albums | 20/20 |
| m3-people (incl. the new twin scene) | 11/11 |
| m3-search | 9/9 |
| e19-places | 8/8 |
| m3-memories | 15/15 |
| e22-viewer-panel | 9/9 |
| w3-users-alignment (the handover's collision set) | 5/5 |
| e28-month-axis | 6/6 |
| e20-admin-states | 5/5 |
| e33-derivative-jobs | 6/6 |
| e29-e30-live-photo | 5/5 |
| f1-tile-cache-hit | 3/3 |
| e5-map | 7/7 |
| m3-models | 7/7 |

The worktree stack ran on 8245: `docker-compose.yml`, `web/playwright.config.ts` and every
spec's local `BASE_URL` were edited for the run and **reverted before the commits** (the
committed tree carries 8244 only).

The owner's acceptance path (dogfood, same walk): upload a photo → mount a read-only folder
holding the same bytes as a directory album → the album's tile and "My photos"' tile are two
rows of one content → the album detail has no delete, My photos' has one → delete it: the toast
says one copy remains, the timeline tile survives, and it has no delete button either.

---

## 4. Deviations and decisions the plan left open

1. **`viewer_trashed_done` added.** The plan carried only the tail key; the design sample's toast
   has a base sentence ("已移入回收站。"). Added with that exact Chinese value.
2. **The note pluralizes (en).** The plan's literal en string "{n} more copies of this photo
   remain." is ungrammatical at the most common value (1). Shipped as an ICU plural
   (`=1 … # more copy … remains.` / `other … # more copies … remain.`); zh renders identically to
   the pinned sentence (`该内容还有 # 份副本会保留。`).
3. **The archive view carries the predicate** (the dispatch's list) though the plan's list omits
   it. Consequence recorded honestly: an archived source twin whose group still has a live
   upload is not listed in the archive (the content is still in the family flow), and the
   directory album's wall is where its state is unarchived (the wall keeps archived members).
4. **Album faces split by type** (deviation from a literal "album walls never filter"): a sync
   album and a memory are library faces, so their timelines, item pages, counts and cover pool
   dedup; only `directory` albums list raw members. This keeps each album's count equal to its
   wall on both faces.
5. **The convergence arm's candidate set** (the plan's wording would also evict a second
   same-content file in one folder): it evicts blob members and orphan-marked dead members only,
   so a folder genuinely holding two identical files keeps both tiles and a trashed-with-copy
   member still restores into its album.
6. **The handover's collision set** (`userremoval.sql`, blast radius the plan did not name): a
   departing row now collides by its own domain's key (blob by content, external by path). Pairing
   across domains would have handed a directory album's mirror to an upload row on a user
   transfer.
7. **The motion reuse narrowed to blob rows** — a semantic guard the fork required (see pin 2).
8. **The people per-asset face reads stay row-level** (`ListAssetFacesForCaller`, the face crop
   reads): the detail panel shows the faces that exist on the row it opened; only the lists,
   counts and grids speak for the library.
9. **The position lookup's index** is one full `(owner_id, checksum)` index rather than two
   partial ones: the full-domain lookup must also find trashed rows of both kinds.

---

## 5. Found, recorded, not fixed (other blocks)

1. **The acceptance scripts encode the pre-fork dedup.** `scripts/acceptance_m3c.sh` asserts the
   rescan *links* the first album's member (lines ~71-72, ~226, ~1637, ~1701-1714: "the removal
   album's membership must carry both files: the dedup link and the fresh ingest") and
   `scripts/acceptance_m3d.sh` asserts the four same-bytes files are "honest skips" with zero
   external rows (lines ~63, ~223, ~1265-1283). Both assertions describe the superseded model and
   would now fail; they belong to those blocks' harnesses (m3c/m3d own their scripts) and are not
   in E42's gate list. The web e2e specs that encode the same narrative were reworked here
   (`m3-albums` comments, the dual-home story in place of the old dedup scenes).
2. **Cross-album borrowed rows do not survive their first scan.** A pre-fork scan could link one
   external row into two albums. The first rescan of album A marks that row vanished (the walk of
   A does not see the path), which temporarily removes it from album B; B's next scan revives the
   same row through the position-key arm. Self-healing, not data-losing — recorded so a reviewer
   is not surprised by it.
3. **Archive semantics at the content level.** Archiving the uploaded copy hides the group from
   the family flows even when a source copy is live (the pin's literal ranking: a live blob
   represents, dead rows never do; "archived" is not "dead"). Deliberate, documented in the
   predicate's comment; if the owner later wants archive-per-content the predicate gains an
   `is_hidden` arm and the archived view loses it.
4. **`ui/` + `GUIDE.md` sync (design repo, dispatcher's side).** `ui/viewer.html`'s trash button
   sample carries the toast with the tail sentence in this block's design; the design repo's
   difference list and the "second wave #2" tracking entry are the dispatcher's same-day close
   (the plan's Task 4 item 5 and Task 6's design gates). This worktree did not touch the design
   repo.
5. **The compose image tag is shared.** The worktree build retagged `ghcr.io/welpher/lumemor:latest`
   (the dogfood stack's image name); the running dogfood container is unaffected, but the
   post-merge dogfood rebuild is the one that should be treated as authoritative.

---

## 6. File map (for the review)

- Migration: `server/migrations/00027_position_rows.sql` (+ comment qualifier in 00008).
- Queries: `assets.sql` (detail twin count, full-domain content lookup, blob-domain lookup),
  `rescan.sql` (position lookup, revive, convergence), `timeline.sql` / `favorites.sql` /
  `places.sql` / `people.sql` / `tags.sql` / `albums.sql` (the predicate), `userremoval.sql`
  (collision keys).
- Server: `api/assets.go` (upload arm), `api/assets_serve.go` (contentCopies), `jobs/album_rescan.go`
  (scan arms + convergence), `jobs/motion.go` (blob-domain reuse), `search/scope.go`,
  `search/textquery.go`.
- Contract: `openapi/lumemor.yaml` + regenerated Go/TS/Dart.
- Web: `lib/api/assets.ts`, `lib/viewer/Viewer.svelte`, `routes/(content)/a/[id]/+page.svelte`,
  both locales, `scripts/check_i18n.py` allowlist.
- Tests: `internal/db/positionrows_migration_test.go`, `internal/api/{assets,timeline,assets_serve}_test.go`,
  `internal/jobs/album_rescan_test.go`, `web/src/routes/(content)/a/[id]/viewer.test.ts`,
  `web/e2e/e42-position-rows.spec.ts`, `web/e2e/m3-people.spec.ts` (scene 11),
  `web/e2e/m3-albums.spec.ts` (comment rework).

---

**PR: #59** — https://github.com/welpher/lumemor/pull/59 (open, review pending).