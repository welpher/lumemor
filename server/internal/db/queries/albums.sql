-- Album queries of the sync-album wiring (M2c Task 2): every user
-- owns exactly one sync album -- the core backup stream the timeline
-- aggregates -- and every upload lands in it. The creation points are
-- the account-creation transactions; the upsert below also lets the
-- upload transaction recover the stream of an account whose rows
-- predate the wiring (a user row without its sync album), because
-- failing the upload over a recoverable invariant would trade away the
-- upload's idempotency guarantees for no benefit.

-- name: EnsureSyncAlbum :one
-- The idempotent sync-album creator. The insert arm serves every
-- account-creation point; the ON CONFLICT arm serves concurrent
-- creators: the loser of the albums_sync_owner_uq race blocks on the
-- winner's index entry, then locks and returns the winner's row
-- through the (value-identical) no-op UPDATE -- the classic upsert
-- whose RETURNING always answers the album that exists afterwards,
-- never "no row". The title is the fixed system label; sync albums
-- are not user-facing artifacts of any v1 API.
INSERT INTO albums (id, owner_id, title, type)
VALUES ($1, $2, 'Sync', 'sync')
ON CONFLICT (owner_id) WHERE type = 'sync'
DO UPDATE SET updated_at = albums.updated_at
RETURNING id;

-- name: AddAssetToAlbum :exec
-- The materialized membership insert. ON CONFLICT DO NOTHING keeps
-- re-adding an asset that already sits in the album a no-op: album
-- membership is a set, and the callers converge to the same row
-- instead of failing. (The upload path is the only caller: the
-- trash-restore path needs no re-insertion -- soft delete never
-- removed the membership rows, and the live views' deleted_at filter
-- is the only thing that hid the asset while it sat in the trash, so
-- clearing the mark restores the timeline presence wholesale.)
INSERT INTO album_items (album_id, asset_id, added_by)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING;

-- Album reads (M3c Task 1): the wall, the cover candidates and the
-- detail face. The live-member predicate runs through all of them:
-- deleted_at IS NULL AND is_hidden = false AND is_component = false,
-- the timeline's own definition of a live asset row (A5 -- a hidden
-- asset is invisible to the whole family, and a Live Photo's video
-- component is never a tile of its own).

-- name: ListWallAlbumsForCaller :many
-- The wall of the calling user in one statement: the own group (the
-- caller's virtual and directory albums, updated_at descending) UNION
-- ALL the shared group (the albums shared to the caller, the
-- caller's grant time descending), each group breaking ties on the
-- album id descending. The arm_rank column is the group
-- discriminator the ORDER BY pins -- own first, shared second -- and
-- the per-arm sort key rides as sort_at (the album's updated_at for
-- own cards, the caller's granted_at for shared ones).
--
-- Sync albums never appear in either arm: the sync album is the
-- account's timeline, not a wall artifact -- the caller's own sync
-- album is invisible on every album face, and a sync album another
-- account shared feeds the timeline's shared arm, not the wall.
SELECT sqlc.embed(al),
       'owner'::text AS caller_role,
       NULL::uuid AS owner_user_id,
       NULL::text AS owner_email,
       EXISTS (
           SELECT 1 FROM album_shares s WHERE s.album_id = al.id
       ) AS is_shared,
       (
           SELECT count(*)
           FROM album_items ai
           JOIN assets a ON a.id = ai.asset_id
           WHERE ai.album_id = al.id
             AND a.deleted_at IS NULL
             AND a.is_hidden = false
             AND a.is_component = false
       ) AS item_count,
       0::int AS arm_rank,
       al.updated_at AS sort_at
FROM albums al
WHERE al.owner_id = sqlc.arg('user_id')
  AND al.type IN ('virtual', 'directory')

UNION ALL

SELECT sqlc.embed(al),
       'viewer'::text AS caller_role,
       u.id AS owner_user_id,
       u.email AS owner_email,
       -- The shared arm's is_shared holds true by construction: the
       -- caller's own share row is the reason this row exists.
       true AS is_shared,
       (
           SELECT count(*)
           FROM album_items ai
           JOIN assets a ON a.id = ai.asset_id
           WHERE ai.album_id = al.id
             AND a.deleted_at IS NULL
             AND a.is_hidden = false
             AND a.is_component = false
       ) AS item_count,
       1::int AS arm_rank,
       s.granted_at AS sort_at
FROM albums al
JOIN album_shares s ON s.album_id = al.id AND s.user_id = sqlc.arg('user_id')
JOIN users u ON u.id = al.owner_id
WHERE al.type IN ('virtual', 'directory')

ORDER BY arm_rank, sort_at DESC, id DESC;

-- name: ListAlbumCoverCandidates :many
-- The derived covers of the given albums: each one's four most
-- recent live members, taken_at descending with the asset id as the
-- tiebreak -- the exact rows the wall renders for an album without a
-- pinned cover asset. The window keeps the result bounded at four
-- rows per album however many members an album holds.
SELECT ranked.album_id,
       ranked.asset_id
FROM (
    SELECT ai.album_id,
           ai.asset_id,
           row_number() OVER (
               PARTITION BY ai.album_id
               ORDER BY a.taken_at DESC, a.id DESC
           ) AS recency_rank
    FROM album_items ai
    JOIN assets a ON a.id = ai.asset_id
    WHERE ai.album_id = ANY(sqlc.arg('album_ids')::uuid[])
      AND a.deleted_at IS NULL
      AND a.is_hidden = false
      AND a.is_component = false
) ranked
WHERE ranked.recency_rank <= 4
ORDER BY ranked.album_id, ranked.recency_rank;

-- name: GetAlbumForCaller :one
-- The album the caller may see, with the caller's role and the live
-- member count. The visibility check is the whole point: the caller
-- owns the album or an album_shares row grants it, and the album is
-- no sync album (the sync album is the timeline's, whoever owns it
-- -- the detail face answers 404 for it through the handler's
-- no-row mapping, which is what keeps unknown, invisible and sync
-- ids indistinguishable).
SELECT sqlc.embed(al),
       CASE
           WHEN al.owner_id = sqlc.arg('user_id') THEN 'owner'::text
           ELSE 'viewer'::text
       END AS caller_role,
       (
           SELECT count(*)
           FROM album_items ai
           JOIN assets a ON a.id = ai.asset_id
           WHERE ai.album_id = al.id
             AND a.deleted_at IS NULL
             AND a.is_hidden = false
             AND a.is_component = false
       ) AS item_count
FROM albums al
WHERE al.id = sqlc.arg('album_id')
  AND al.type IN ('virtual', 'directory')
  AND (
      al.owner_id = sqlc.arg('user_id')
      OR EXISTS (
          SELECT 1 FROM album_shares s
          WHERE s.album_id = al.id AND s.user_id = sqlc.arg('user_id')
      )
  );

-- name: ListAlbumItemsForCaller :many
-- One page of the album's items: the live members in the timeline's
-- own ordering (taken_at DESC, id DESC), keyset-paginated with the
-- same row-value predicate and the same NULL-init first page. The
-- caller's visibility gate is NOT here: the handler runs this only
-- after GetAlbumForCaller returned the album, so the rows it lists
-- are rows of an album the caller already proved they may see.
SELECT ai.asset_id,
       ai.added_at,
       a.taken_at,
       a.original_filename,
       a.media_kind
FROM album_items ai
JOIN assets a ON a.id = ai.asset_id
WHERE ai.album_id = sqlc.arg('album_id')
  AND a.deleted_at IS NULL
  AND a.is_hidden = false
  AND a.is_component = false
  AND (
      sqlc.narg('cursor_taken_at')::timestamptz IS NULL
      OR (a.taken_at, a.id) < (sqlc.narg('cursor_taken_at')::timestamptz, sqlc.narg('cursor_id')::uuid)
  )
ORDER BY a.taken_at DESC, a.id DESC
LIMIT sqlc.arg('limit_count');

-- Album writes (M3c Task 2): creation, the property patch, the
-- delete, the membership curation and the write face's authorization
-- gate. The reads' live-member predicate runs through the cover
-- validation and the add's resolution: the same three-part filter
-- (deleted_at IS NULL AND is_hidden = false AND is_component =
-- false).

-- name: GetAlbumForWrite :one
-- The write face's authorization gate: the album row plus whether
-- the caller holds a view share of it. Sync albums answer no row
-- (the timeline is their face, whoever owns them); the owner check
-- itself is NOT here -- the handler shapes the two-tier answer off
-- the row, the permanent delete's discipline (the refusal, not the
-- read, shapes the response): the owner or an administrator
-- proceeds, a sharee answers the honest 403, and everyone else
-- folds into the reads' byte-identical 404.
SELECT sqlc.embed(al),
       EXISTS (
           SELECT 1 FROM album_shares s
           WHERE s.album_id = al.id AND s.user_id = sqlc.arg('user_id')
       ) AS shared_to_caller
FROM albums al
WHERE al.id = sqlc.arg('album_id')
  AND al.type <> 'sync';

-- name: CreateVirtualAlbum :one
-- The virtual album creator: the write face's only creatable type
-- (directory albums arrive with their own milestone, sync albums are
-- born per account), with the id generated caller-side because the
-- table carries no default. The wall card's companions are the
-- creation-time constants -- no share, no member, no cover -- so the
-- handler reads the fresh row's full card through GetAlbumSummary
-- like every other write response.
INSERT INTO albums (id, owner_id, title, description, type)
VALUES (sqlc.arg('id'), sqlc.arg('owner_id'), sqlc.arg('title'), sqlc.arg('description'), 'virtual')
RETURNING id;

-- name: GetAlbumSummary :one
-- The album row with the wall card's honest companions: whether any
-- share row exists and the live member count. The write face's
-- responses build their cards off this row; the derived cover
-- candidates ride along through the wall's own follow-up query.
SELECT sqlc.embed(al),
       EXISTS (
           SELECT 1 FROM album_shares s WHERE s.album_id = al.id
       ) AS is_shared,
       (
           SELECT count(*)
           FROM album_items ai
           JOIN assets a ON a.id = ai.asset_id
           WHERE ai.album_id = al.id
             AND a.deleted_at IS NULL
             AND a.is_hidden = false
             AND a.is_component = false
       ) AS item_count
FROM albums al
WHERE al.id = sqlc.arg('album_id');

-- name: UpdateAlbumFields :one
-- The property patch: every field COALESCEs over the stored value,
-- so an absent (NULL) parameter keeps it and a present one replaces
-- it. The cover rides the same discipline -- a coverAssetId of JSON
-- null decodes as absent, the patch convention of the asset surface,
-- so the pin can change through this face but never clear.
-- updated_at moves with the write: the wall orders the own group by
-- it. No owner guard and no type guard here: the handler's gate
-- already resolved the caller's right to this write, and an album
-- deleted between the gate and the statement answers no row -- the
-- honest 404 of a gone album.
UPDATE albums
SET title = COALESCE(sqlc.narg('title'), title),
    description = COALESCE(sqlc.narg('description'), description),
    cover_asset_id = COALESCE(sqlc.arg('cover_asset_id'), cover_asset_id),
    locked = COALESCE(sqlc.narg('locked'), locked),
    updated_at = now()
WHERE id = sqlc.arg('album_id')
RETURNING id;

-- name: CountLiveAlbumMember :one
-- The cover validation (the spec's application-layer pin): does the
-- given asset sit in this album as a LIVE member? The membership row
-- must exist AND the asset must pass the live predicate -- a hidden,
-- trashed or Live Photo component member is a member no more for
-- cover purposes, and an asset outside the album never was one.
SELECT count(*) AS matches
FROM album_items ai
JOIN assets a ON a.id = ai.asset_id
WHERE ai.album_id = sqlc.arg('album_id')
  AND ai.asset_id = sqlc.arg('asset_id')
  AND a.deleted_at IS NULL
  AND a.is_hidden = false
  AND a.is_component = false;

-- name: DeleteVirtualAlbum :execrows
-- The virtual album delete: the row goes and every album_items and
-- album_shares row follows through the schema's ON DELETE CASCADE.
-- The type guard keeps the statement honest against the handler's
-- virtual-only verdict (directory removal is the directory album's
-- own flow), and zero rows -- a concurrent delete won the race -- is
-- the 404 of a gone album. No locked gate ever runs here: locking is
-- a curation freeze, not a tombstone.
DELETE FROM albums
WHERE id = sqlc.arg('album_id')
  AND type = 'virtual';

-- name: AddAlbumItems :execrows
-- The membership add: resolution and insert in one statement. The
-- WHERE arm IS the resolution -- every id must name a live asset the
-- CALLER owns (the owner-based live predicate; the union switchover
-- of the later task replaces it) -- so foreign, trashed, hidden,
-- component and unknown ids never produce rows: they are the
-- skipped of the honest report, never errors. ON CONFLICT DO NOTHING
-- makes the already-member id the no-op of the idempotent add
-- (membership is a set). The SELECT scans assets once, so a repeated
-- id in the array yields one row; the rows-affected count is the
-- report's added.
INSERT INTO album_items (album_id, asset_id, added_by)
SELECT sqlc.arg('album_id'), a.id, sqlc.arg('added_by')
FROM assets a
WHERE a.id = ANY(sqlc.arg('asset_ids')::uuid[])
  AND a.owner_id = sqlc.arg('owner_id')
  AND a.deleted_at IS NULL
  AND a.is_hidden = false
  AND a.is_component = false
ON CONFLICT DO NOTHING;

-- name: RemoveAlbumItems :execrows
-- The membership remove: the rows that exist leave, and the ids that
-- are no members simply match nothing. No live predicate -- the
-- removal tidies the set itself (a trashed member's row still
-- leaves), and the rows-affected count is the report's removed.
DELETE FROM album_items
WHERE album_id = sqlc.arg('album_id')
  AND asset_id = ANY(sqlc.arg('asset_ids')::uuid[]);

-- Album sharing (M3c Task 3): the view-ACL grant, revoke and the
-- owner's grantee list. The grant is the set discipline again: ON
-- CONFLICT DO NOTHING makes the re-grant of a current grantee a
-- no-op, so the rows-affected count is the transition signal the
-- handler's event discipline reads (a new row fires album.shared,
-- the idempotent no-op stays silent) -- and the PK is the race
-- guard: concurrent grants converge on one row.

-- name: GrantAlbumShare :execrows
-- The view-ACL grant: one album_shares row. The count answers the
-- transition -- 1 row landed a fresh grant, 0 rows mean the grantee
-- already holds the share (the handler then re-reads the existing
-- row for the response; the grantor and grant time of the original
-- stay untouched).
INSERT INTO album_shares (album_id, user_id, granted_by)
VALUES (sqlc.arg('album_id'), sqlc.arg('user_id'), sqlc.arg('granted_by'))
ON CONFLICT (album_id, user_id) DO NOTHING;

-- name: GetAlbumShare :one
-- One share row of one grantee: the honest post-write state of the
-- grant (the fresh row's now(), or the existing row's original
-- values when the grant was the idempotent no-op).
SELECT album_id, user_id, granted_at
FROM album_shares
WHERE album_id = sqlc.arg('album_id')
  AND user_id = sqlc.arg('user_id');

-- name: RevokeAlbumShare :execrows
-- The view-ACL revoke: the row leaves, and the count is the
-- transition signal -- 1 row left fires album.unshared, 0 rows (the
-- share never existed, already left, or the id names no account)
-- are the idempotent silence the 204 answers either way.
DELETE FROM album_shares
WHERE album_id = sqlc.arg('album_id')
  AND user_id = sqlc.arg('user_id');

-- name: ListAlbumShares :many
-- The owner's grantee list of one album: the share management
-- dialog's rows, the grant order (granted_at ascending). The join
-- carries the grantee's email -- the dialog lists people, not
-- uuids. Only the owner's detail ever renders this query's answer.
SELECT s.user_id, u.email, s.granted_at
FROM album_shares s
JOIN users u ON u.id = s.user_id
WHERE s.album_id = sqlc.arg('album_id')
ORDER BY s.granted_at ASC, s.user_id ASC;
