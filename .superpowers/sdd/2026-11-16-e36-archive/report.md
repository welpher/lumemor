# E36 — Archive: one word, one set of boundaries

Branch `e36-archive` off `origin/main` @ 9107567 (PR #57). Worktree
`/Users/welpher/coding/lumemor-e36`.

Four commits, one decision each:

1. `437c1be` feat(api): the archive rename -- isArchived and GET /assets/archived
2. `4fc6d00` feat(albums): an album keeps its archived members, a memory drops them
3. `ec4a6d4` feat(archive): the vocabulary sweep -- one word, its glyphs and its toasts
4. `8f26d4e` test: the sweep's own residue -- the wrappers' comments and the
   toolbar's link probe

PR: **#58** (https://github.com/welpher/lumemor/pull/58) -- open, not merged.

The final tree was re-verified after the last commit: `make web-check`
(878 tests), `make api-check` (zero drift, clean tree) and `git status`
clean (the only untracked path is this report directory).

## What landed per task

### Task 3 (contract, done first — contract-first)

- `PATCH /assets/{id}`: `isHidden` → **`isArchived`** (AssetPatch + AssetDetail).
- `GET /assets/hidden` → **`GET /assets/archived`**, `operationId getHidden` →
  `getArchived`; the handler (`GetHidden`→`GetArchived`), the router mount and the
  `ListHiddenAssets`→`ListArchivedAssets` query follow.
- `TimelineItem` gains **`isArchived`** (required boolean). Reason: the album
  container has to tell its own rows apart to offer the unarchive entry the spec
  requires ("在相册里能取消归档"); the archived view's rows are `true` by
  construction, every family-flow wall's are `false`. Documented in the schema,
  examples updated, api-lint green.
- DB column stays `is_hidden` — zero migration. The query comments describe it as
  the archive switch. Batch action names stay `hide` / `unhide` (the contract's
  enum; the design's endpoint table keeps them) — so the one vocabulary decision
  did not churn the protocol enum.
- Prose sweep of the contract: every archive-state mention now says archive;
  the person-label "non-hidden" residues (retired person hide) and the Live Photo
  "hidden motion" mentions were deliberately left (different domains).
- Generated artifacts regenerated (`openapi/generated/**`, `server/internal/api/gen`,
  `server/internal/db/gen`) — `make api-check` zero drift on a clean tree.
- `scripts/acceptance_m2c.sh`'s archived-view probe follows the new path.

### Task 1 (UI rename)

- Drawer item: **Archive** (`nav_archived`), `/archived`, lucide `archive` glyph.
- Page: `<title>Archive · lumemor</title>`, `h1` "Archive" + count, route folder
  moved `hidden/` → `archived/`.
- Batch bar: **Archive** (`asset_archive`) / **Unarchive** (`asset_unarchive`)
  on timeline / people wall / places wall; the archived page's bar is the
  unarchive action (+ add-to-memory). The unarchive button renders only when the
  checked rows hold an archived member (the album container's own entry); the
  archive button is the family-flow walls' normal case.
- Toasts (new — the implementation had no batch success toast at all): the
  archive pair confirms its landing, `archive_done` / `unarchive_done`
  ("已归档 {n} 张。" / "已取消归档 {n} 张。" — the sentence keeps 已, the label does
  not), fired from the shared wall with the server's applied count when reported.
- Empty state: 「没有归档的照片」/「归档的照片不在时间线里；全家都看不到它，你随时可以取消归档。」
  — the false "only you can see it" sentence is gone; English is the same claim.
- Info panel notice: `viewer_hidden_notice` → `viewer_archived_notice`
  ("这张照片已归档，不在时间线里。").
- i18n: `nav_archived`, `asset_archive`, `asset_unarchive`, `archive_bar_count`,
  `archive_loading`, `archive_empty_title/body`, `archive_done`,
  `unarchive_done`, `viewer_archived_notice` added in BOTH locales; `action_hide`,
  `action_unhide`, `hidden_title`, `hidden_bar_count`, `hidden_loading`,
  `hidden_empty_title/body`, `viewer_hidden_notice` retired with **no aliases**.
  `scripts/check_i18n.py`'s NON_ERROR_KEYS allowlist updated.
- Icon rider: the drawer / batch-bar eye-off glyphs became lucide `archive` in the
  same commit. The unarchive button keeps the `eye` glyph — the design sample
  (`ui/hidden.html`) draws 取消归档 with `eye`, and the rider names only the
  eye-off → archive swap; noted here in case the sample changes.
- No intermediate state: `grep -ri "已隐藏" ` over `web/src` + `i18n` returns
  nothing; the route `/hidden` is gone (v1 unreleased, old deep links 404 —
  accepted by the plan).

### Task 2 (boundaries)

Verified / pinned:

| surface | rule | where |
|---|---|---|
| timeline (aggregate) | excludes | `ListTimelineAssets` (A5, existing) |
| favorites | excludes | `ListFavoriteAssets` (existing) |
| people wall + counts | excludes | `people.sql` (existing) |
| places wall + counts | excludes | `places.sql` (existing) |
| map pins | excludes | the map reads the places/timeline queries (existing) |
| memories (memory albums) | excludes | new type-aware predicate (below) |
| counts | excludes for every family flow; an album card counts its container members | `ListWallAlbumsForCaller` (below) |
| search | excludes, **no "include archived" switch added** | `search/scope.go`, `textquery.go` (existing) |
| **albums (sync/directory)** | **keep** archived members | album reads: type-aware predicate |
| album wall unarchive | owner, archived selection only | album page `OWNER_WALL_ACTIONS = ['unhide','trash']` + the wall's gate |

The one predicate: album reads filter `is_hidden` **only for `type = 'memory'`**
albums — `(al.type <> 'memory' OR a.is_hidden = false)`. It runs through
`ListAlbumTimelineAssets` (the detail grid), `ListAlbumItemsForCaller`,
`GetAlbumForCaller` / `GetAlbumSummary` counts, the wall cards' counts, the cover
pool / picks and the cover-write validation. `AddMemoryItems` keeps its
family-flow filter (an archived photo never enters a memory it would be
invisible in — the archived page's add-to-memory therefore answers the honest
skip, pinned by m3-memories scene 7).

Client side: the timeline client's convergence learns the container rule — an
album-narrowed wall keeps an archived member (the detail's `albums` face names
the container and its type), a memory drops it. `runBatch('unhide')` on an album
wall clears the archived mark but keeps the tile.

**Judgment call recorded:** the plan lists 计数 as an exclusion while 相册内保留
says the album keeps its members. I read the container rule as governing the
album's own face: an album card's `itemCount`/`photoCount` and its covers count
the members it shows, otherwise the card would contradict its own wall
("4 photos" over 5 tiles). The family-flow counts (timeline months, people,
places, memories) still exclude. If the owner wants album cards live-only, it is
a one-line predicate revert per count.

### Task 4 (walkthrough + gates)

Walked on real stacks (see e2e block). Interface carries no "已隐藏"/"hidden".

## Gates

```
$ make web-check
prettier --check .            clean
eslint .                      clean
svelte-check                  0 errors, 0 warnings
vitest                        76 files passed, 878 tests passed (was 873; +5 new)
```

```
$ python3 scripts/check_i18n.py
i18n check OK: 2 file(s) [en, zh-Hans], 770 key(s), 33 error code(s)
```

```
$ make api-lint
No results with a severity of 'error' found!
```

```
$ make api-check          # clean tree
git diff --exit-code      # zero drift (generated Go/TS/Dart match the contract)
```

```
$ make docker-lint
gofmt -l / go vet / golangci-lint
0 issues.
```

```
$ make ci-local
=== ci-local 1/6: the contract lints ===
=== ci-local 2/6: the web suite, cold (ephemeral node container) ===
=== ci-local 3/6: the server suite, cold (dev container, no cache volumes) ===
=== ci-local 4/6: the API artifacts regenerate without drift (twice) ===
=== ci-local 5/6: the sqlc output regenerates without drift ===
=== ci-local 6/6: the locales agree ===
ci-local: all green
```

(The run was piped through `tail -40`, so the per-step bodies are not
re-quoted here; every step's failure prints and aborts, and the final line
is the harness's own "all green".)

## E2E (fresh stack per spec, port 8245 remap reverted before the final commits)

Each spec: `docker compose down --remove-orphans` + `docker volume rm -f
lumemor-e36_pgdata` + `up -d` + `/healthz` 200, then the spec.

| spec | result |
|---|---|
| `m2-full-chain.spec.ts` | **12 passed** (scene 8 reworded: batch archive → archived view → unarchive, toasts asserted) |
| `m3-people.spec.ts` | **10 passed** (scene 5: archive from the person wall, unarchive from `/archived`) |
| `m3-memories.spec.ts` | **15 passed** (scene 7 archived wall + the honest memory skip) |
| `m3-search.spec.ts` | **9 passed** |
| `m3-albums.spec.ts` | **20 passed** — incl. the new `scene 13c`: archive a member via the API, the timeline/archived views agree, the album wall keeps the tile AND its "2 photos" count, the Unarchive button appears only for the archived selection, the landing toasts, the tile stays, the row leaves the archived view |
| `e19-places.spec.ts` | **8 passed** |

Environment notes (not code):
- the first m3-people run failed because the model download was cold and had no
  outbound path (job stuck `running`); seeding `./lumemor/config/models` from the
  dogfood's warm store made the bootstrap the documented warm path (11.6s). The
  worktree's `./lumemor` was deleted afterwards.
- m3-search's scene 3 failed twice on the count (2 vs 3) while the embeddings
  caught up, then passed on the third fresh stack (the bootstrap-ready poll is
  `isReady`-based; the search's semantic arm needs the embedding). Pre-existing
  bootstrap race, untouched by this block (search code is unmodified).

## Deviations

1. **`TimelineItem.isArchived` added** (contract grew one required boolean). The
   plan's contract task names only the two renames; the album-side unarchive
   entry the spec requires cannot know its selection's state without it. Additive,
   documented, zero behaviour change on family-flow walls.
2. **Batch action names stay `hide` / `unhide`** (contract enum). The design's
   endpoint table keeps them; only the field and the path renamed.
3. **Batch success toasts were added** (they did not exist). The plan's toast
   strings are implemented; only the archive pair toasts (favorite/trash toasts
   remain E11's unlanded work).
4. **Album counts/covers follow the container rule** (see the judgment call).
5. **Unarchive glyph stays `eye`** (the design sample's choice for 取消归档); only
   the eye-off glyphs became `archive`.

## Found, belongs to another block (recorded, not fixed)

- `scripts/acceptance_m3b.sh` still exercises `PUT /people/{clusterId}/hidden`
  and `isHidden` on person rows — the person hide retired on 2026-11-11 and the
  endpoint no longer exists; the script is stale (people block / E37 cleanup).
- The archived page's "Add to memory" entry can never land (memories refuse
  archived rows by the boundary). The picker reports the honest skip, so it is
  not a dead button, but the affordance is arguably misleading — the design
  sample keeps it on the archived bar; a design decision, not this block's.
- `web/src/lib/events/index.ts` still parses person-event `hidden` state (the
  retired person hide's wire field); harmless but stale.