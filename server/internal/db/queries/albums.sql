-- Album queries of the sync-album wiring: every user
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

-- Album reads: the wall, the cover candidates and the
-- detail face. The live-member predicate runs through all of them:
-- deleted_at IS NULL AND is_hidden = false AND is_component = false,
-- the timeline's own definition of a live asset row (a hidden
-- asset is invisible to the whole family, and a Live Photo's video
-- component is never a tile of its own).

-- name: ListWallAlbumsForCaller :many
-- The wall of the calling user in one statement: the pinned sync
-- card (the caller's own core stream, the default "My photos"
-- album) UNION ALL the own group (the caller's virtual and
-- directory albums, updated_at descending) UNION ALL the shared
-- group (the albums shared to the caller, the caller's grant time
-- descending), each group breaking ties on the album id descending.
-- The arm_rank column is the group discriminator the ORDER BY pins
-- -- sync first (the pin is structural: no own album's updated_at
-- outranks it), own second, shared last -- and the per-arm sort key
-- rides as sort_at (the album's updated_at for the sync and own
-- cards, the caller's granted_at for shared ones).
--
-- A sync album of ANOTHER account never appears: the sync arm is
-- owner-scoped and the shared arm admits virtual and directory
-- albums only (a sync album a share row names -- nothing can
-- produce one -- feeds no wall card for the sharee).
--
-- in_timeline is the caller's raw timeline preference row of the
-- card's album (the exclusion toggle's stored state): NULL means no
-- row, which the handler maps to the default of feeding the timeline
-- -- except on the caller's own virtual cards, where the field stays
-- absent entirely (they never feed the caller's own timeline), and
-- on the sync card (the handler omits it there too: the core stream
-- is never excludable, so the stored value -- always NULL, the
-- preference face refuses sync -- never surfaces).
SELECT sqlc.embed(al),
       'owner'::text AS caller_role,
       NULL::uuid AS owner_user_id,
       NULL::text AS owner_email,
       -- The sync card's is_shared holds the same honest EXISTS the
       -- own arm computes: no share row can name a sync album (the
       -- sharing face refuses them), so the answer is false -- but
       -- a defensive row would report true rather than lie.
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
       p.in_timeline,
       0::int AS arm_rank,
       al.updated_at AS sort_at
FROM albums al
LEFT JOIN user_album_prefs p
       ON p.album_id = al.id AND p.user_id = sqlc.arg('user_id')
WHERE al.owner_id = sqlc.arg('user_id')
  AND al.type = 'sync'

UNION ALL

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
       p.in_timeline,
       1::int AS arm_rank,
       al.updated_at AS sort_at
FROM albums al
LEFT JOIN user_album_prefs p
       ON p.album_id = al.id AND p.user_id = sqlc.arg('user_id')
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
       p.in_timeline,
       2::int AS arm_rank,
       s.granted_at AS sort_at
FROM albums al
JOIN album_shares s ON s.album_id = al.id AND s.user_id = sqlc.arg('user_id')
JOIN users u ON u.id = al.owner_id
LEFT JOIN user_album_prefs p
       ON p.album_id = al.id AND p.user_id = sqlc.arg('user_id')
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
-- The album the caller may see, with the caller's role, the live
-- member count and the caller's timeline preference. The visibility
-- check is the whole point: the caller owns the album or an
-- album_shares row grants it -- and a sync album resolves ONLY for
-- its owner: the caller's own sync album is the default album's
-- read-only detail, while a sync album of another account (whatever
-- share rows name it) answers no row -- the handler's no-row
-- mapping keeps unknown, invisible and foreign-sync ids
-- indistinguishable. The WRITE faces keep their own gate
-- (GetAlbumForWrite answers no row for every sync album, whoever
-- owns it): this read admitting the caller's own sync opens no
-- write path.
-- in_timeline is the caller's raw preference row (NULL = no row);
-- the handler maps it to the excludable cards' field -- present for
-- the own directory albums and the shared albums, absent for the
-- caller's own virtual albums and for the caller's own sync album
-- (the core stream is never excludable), true when no row exists.
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
       ) AS item_count,
       p.in_timeline
FROM albums al
LEFT JOIN user_album_prefs p
       ON p.album_id = al.id AND p.user_id = sqlc.arg('user_id')
WHERE al.id = sqlc.arg('album_id')
  AND (al.type <> 'sync' OR al.owner_id = sqlc.arg('user_id'))
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

-- Album writes (reshaped by albums-as-sources): creation, the property
-- patch, the delete, the membership removal and the write face's
-- authorization
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

-- name: CreateDirectoryAlbum :one
-- The directory album creator (the albums-as-sources rework, widened
-- by album autonomy): the row references an existing filesystem
-- directory the server validated against the path matrix BEFORE
-- this statement ran -- exists, is a directory, readable, outside
-- the managed tree -- and root_path stores the RESOLVED absolute
-- path (symlinks evaluated at validation time), so the album can
-- never be silently repointed by a later symlink swap. The creation
-- is the administrator's act (the boundary ruling: mounting an
-- existing photo directory is a configuration act), and
-- source_writable carries the write probe's answer over that
-- directory -- a fact, not a gate: a read-only source is a
-- legitimate album whose photos cannot be deleted through the
-- application. The id is generated caller-side; the creation face
-- carries title, path and audience only. The wall card's companions
-- are the creation-time constants -- the caller's own audience
-- grants land right after -- so the handler reads the fresh row's
-- full card through GetAlbumSummary like every other write
-- response.
INSERT INTO albums (id, owner_id, title, type, root_path, source_writable)
VALUES (sqlc.arg('id'), sqlc.arg('owner_id'), sqlc.arg('title'), 'directory', sqlc.arg('root_path'), sqlc.arg('source_writable'))
RETURNING id;

-- name: GetAlbumSummary :one
-- The album row with the wall card's honest companions: whether any
-- share row exists, the live member count, and the caller's raw
-- timeline preference row (the write face's caller is the owner, so a
-- virtual album's card carries no preference field; the handler maps
-- NULL to the default the same way the wall does). The write face's
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
       ) AS item_count,
       p.in_timeline
FROM albums al
LEFT JOIN user_album_prefs p
       ON p.album_id = al.id AND p.user_id = sqlc.arg('user_id')
WHERE al.id = sqlc.arg('album_id');

-- name: UpdateAlbumFields :one
-- The property patch: every field COALESCEs over the stored value,
-- so an absent (NULL) parameter keeps it and a present one replaces
-- it. The cover rides the same discipline -- a coverAssetId of JSON
-- null decodes as absent, the patch convention of the asset surface,
-- so the pin can change through this face but never clear. The
-- deletability switch joins the property fields: present
-- updates the owner's allow-app-deletion posture of the directory
-- album, absent keeps it, and the lock's gate lives in the handler
-- (the property fields' own refusal) with the type arm that keeps
-- the switch off virtual albums. updated_at moves with the write:
-- the wall orders the own group by it. No owner guard and no type
-- guard here: the handler's gate already resolved the caller's
-- right to this write, and an album deleted between the gate and
-- the statement answers no row -- the honest 404 of a gone album.
UPDATE albums
SET title = COALESCE(sqlc.narg('title'), title),
    cover_asset_id = COALESCE(sqlc.arg('cover_asset_id'), cover_asset_id),
    locked = COALESCE(sqlc.narg('locked'), locked),
    allow_app_deletion = COALESCE(sqlc.narg('allow_app_deletion'), allow_app_deletion),
    updated_at = now()
WHERE id = sqlc.arg('album_id')
RETURNING id;

-- name: CountLiveAlbumMember :one
-- The cover validation (the application-layer check): does the
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


-- name: RemoveAlbumItems :execrows
-- The membership remove: the rows that exist leave, and the ids that
-- are no members simply match nothing. No live predicate -- the
-- removal tidies the set itself (a trashed member's row still
-- leaves), and the rows-affected count is the report's removed.
DELETE FROM album_items
WHERE album_id = sqlc.arg('album_id')
  AND asset_id = ANY(sqlc.arg('asset_ids')::uuid[]);

-- name: ListAlbumsOfAsset :many
-- The albums holding membership rows for one asset: the ids whose
-- LIVE membership a trash or restore just changed (the deletability
-- chain -- the membership row itself survives the trash, the live
-- views' deleted_at filter is the change), so the handler can
-- publish each album's items-changed convergence signal. The sync
-- album never appears (its membership is implicit, no rows), and a
-- trashed member still names its albums -- the row never left.
SELECT DISTINCT ai.album_id
FROM album_items ai
WHERE ai.asset_id = sqlc.arg('asset_id');

-- Album audience (the albums-as-sources rework): the
-- full-set write behind PUT /albums/{albumId}/audience replaces the
-- one-row grant/revoke pair. The set discipline is the batch
-- surface's: the requested ids that cannot be granted -- unknown or
-- disabled accounts, and the album's own owner -- are the skipped of
-- the honest report, never errors; the rest converge onto exactly
-- the requested audience. The PK is the race guard, and RETURNING is
-- the transition signal the handler's event discipline reads: the
-- ids that actually landed / actually left, so a concurrent writer
-- can never make an idempotent no-op fire an event.

-- name: AddAlbumAudienceGrants :many
-- The grant arm of the audience write: one album_shares row for
-- every REQUESTED id that names a usable account (known, not
-- disabled), inserted with the acting user as the grantor. ON
-- CONFLICT DO NOTHING makes the already-granted id the no-op of the
-- idempotent set write, so RETURNING carries exactly the rows this
-- write landed -- the ids whose album.shared events fire.
INSERT INTO album_shares (album_id, user_id, granted_by)
SELECT sqlc.arg('album_id'), u.id, sqlc.arg('granted_by')
FROM users u
WHERE u.id = ANY(sqlc.arg('user_ids')::uuid[])
  AND NOT u.is_disabled
ON CONFLICT (album_id, user_id) DO NOTHING
RETURNING user_id;

-- name: RemoveAlbumAudienceRows :many
-- The revoke arm of the audience write: every share row whose
-- grantee is NOT in the requested set leaves. An EMPTY array is the
-- honest "audience: none" request: ANY('{}') matches no id, so every
-- row leaves -- the owner's access is ownership, not a share row,
-- and stays untouched whatever the array holds. RETURNING carries
-- exactly the rows that left -- the ids whose album.unshared events
-- fire.
DELETE FROM album_shares
WHERE album_id = sqlc.arg('album_id')
  AND NOT (user_id = ANY(sqlc.arg('user_ids')::uuid[]))
RETURNING user_id;

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

-- Timeline preferences: the per-user "keep this album
-- out of my timeline" switch. The preference scopes the caller's own
-- view only (the aggregated timeline's formula folds it inside its
-- membership existential); the album's own items, the wall and every
-- sharee's view never read it. The row is the caller's: (user_id,
-- album_id) primary key, a missing row meaning kept in.

-- name: GetAlbumForTimelinePref :one
-- The preference face's gate: the album row with the caller's role.
-- Unlike GetAlbumForCaller this resolves EVERY album type -- the
-- caller's own sync and own virtual albums must reach the handler so
-- their 400 (not excludable) can answer, and a sync album a share
-- row names (nothing can produce one, but a defense holds anyway)
-- resolves as a viewer row the handler's type check refuses. No row
-- covers the unknown and the not-visible-alike -- the single-shape
-- 404 (anti-enumeration).
SELECT sqlc.embed(al),
       CASE
           WHEN al.owner_id = sqlc.arg('user_id') THEN 'owner'::text
           ELSE 'viewer'::text
       END AS caller_role
FROM albums al
WHERE al.id = sqlc.arg('album_id')
  AND (
      al.owner_id = sqlc.arg('user_id')
      OR EXISTS (
          SELECT 1 FROM album_shares s
          WHERE s.album_id = al.id AND s.user_id = sqlc.arg('user_id')
      )
  );

-- name: UpsertAlbumTimelinePreference :one
-- The preference write: an insert for a first toggle, an update for
-- every repeat (the upsert converges; the response always carries
-- the stored state). updated_at moves with a real change only, so a
-- repeat of the stored value leaves the row untouched -- the same
-- set-semantics guard the album writes apply.
INSERT INTO user_album_prefs (user_id, album_id, in_timeline)
VALUES (sqlc.arg('user_id'), sqlc.arg('album_id'), sqlc.arg('in_timeline'))
ON CONFLICT (user_id, album_id)
DO UPDATE SET
    in_timeline = EXCLUDED.in_timeline,
    updated_at  = CASE
        WHEN user_album_prefs.in_timeline = EXCLUDED.in_timeline
        THEN user_album_prefs.updated_at
        ELSE now()
    END
RETURNING in_timeline;

-- name: DeleteAlbumTimelinePreference :execrows
-- The preference reset: the row goes, the album returns to the
-- default of feeding the caller's timeline. Zero rows are the
-- idempotent no-op -- the caller asked for the row to be gone and it
-- already is.
DELETE FROM user_album_prefs
WHERE user_id = sqlc.arg('user_id')
  AND album_id = sqlc.arg('album_id');
