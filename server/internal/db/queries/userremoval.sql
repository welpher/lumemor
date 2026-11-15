-- The account disposal queries of the admin user management surface:
-- one account's deprovision, the single transaction behind
-- DELETE /admin/users/{id}. An account owns rows that no cascade may
-- take with it (its photos, its albums, the tags it made, the
-- authorship its album memberships carry, the grants on its albums),
-- and holds rows that live with the account alone (its sessions, its
-- favorites, its per-album preferences, the private names it gave
-- shared face clusters, the album shares granted TO it). The queries
-- below settle the first group onto a surviving owner in each
-- disposition arm; the second group dies through the schema's own
-- cascades when the users row goes.
--
-- Both arms lock the departing row and the instance's administrator
-- rows in one transaction, so the last-administrator guard cannot
-- race a concurrent disposal into an instance with no administrator.

-- name: LockUserForRemoval :one
-- The departing account's row, locked for the disposal. The lock is
-- what makes the read-and-settle sequence one atomic decision: a
-- concurrent disposal of the same account waits here and then finds
-- no row.
SELECT id, name, email, role, is_disabled, created_at
FROM users
WHERE id = $1
FOR UPDATE;

-- name: LockAdministratorIDs :many
-- The instance's administrator rows, locked in id order (a stable
-- order keeps two concurrent disposals from deadlocking on the same
-- pair of rows). The caller counts them: one administrator is the
-- line the disposal may not cross.
SELECT id
FROM users
WHERE role = 'admin'
ORDER BY id
FOR UPDATE;

-- name: DisposalSyncAlbum :one
-- The account's own stream, resolved for the transfer arm's fold.
SELECT id
FROM albums
WHERE owner_id = $1
  AND type = 'sync';

-- name: FoldSyncAlbumMembership :exec
-- The transfer arm's stream continuity: every membership of the
-- departing account's stream becomes a membership of the receiving
-- account's stream, so the photos keep their place in the family
-- timeline instead of falling out of it when the departing stream
-- goes. A photo the receiver already carries is one membership
-- either way.
INSERT INTO album_items (album_id, asset_id, added_by)
SELECT sqlc.arg('album_id'), ai.asset_id, sqlc.arg('added_by')
FROM album_items ai
WHERE ai.album_id = sqlc.arg('source_album_id')
ON CONFLICT (album_id, asset_id) DO NOTHING;

-- name: ReassignAlbumOwner :execrows
-- The transfer arm's single-album owner change (the stream of an
-- account whose receiver carries none of its own).
UPDATE albums
SET owner_id = sqlc.arg('owner_id'), updated_at = now()
WHERE id = sqlc.arg('id');

-- name: DeleteSyncAlbumOfOwner :execrows
-- The transfer arm's folded stream: its memberships have moved to
-- the receiver's stream, so the row goes (its shares and preferences
-- cascade).
DELETE FROM albums
WHERE owner_id = sqlc.arg('owner_id')
  AND type = 'sync';

-- name: TransferAlbumOwnership :exec
-- The transfer arm's container handover: every album and memory the
-- departing account owned becomes the receiver's. The stream is
-- already settled (folded or re-owned) above, so it is out of scope
-- here.
UPDATE albums
SET owner_id = sqlc.arg('to_owner'), updated_at = now()
WHERE owner_id = sqlc.arg('from_owner')
  AND type <> 'sync';

-- name: DeleteOwnedAlbums :execrows
-- The trash arm's container disposal: the departing account's albums
-- and memories go with it -- their memberships, shares and per-user
-- preferences cascade. The photos themselves survive as trash items
-- of the caller's.
DELETE FROM albums
WHERE owner_id = sqlc.arg('owner_id');

-- name: DuplicateOwnedAssets :many
-- The collision set of BOTH disposal arms: every departing row whose
-- content the destination owner already holds, in ANY state pair --
-- live/live, live/trashed, trashed/live and trashed/trashed.
-- assets_owner_checksum_uq is a TOTAL unique index on (owner_id,
-- checksum) over live rows AND trashed rows alike (00008), so the
-- ownership write collides whatever the two rows' states are; a
-- live-only predicate here misses three of the four pairs and lets
-- the write trip the index (a 500 the caller cannot read as anything).
-- The same index makes the pair 1:1 per checksum on each side: one
-- account holds at most one row of a given content, in either state,
-- so one departing row meets at most one destination row.
-- Component rows (a Live Photo's video half) stay out of the fold on
-- both sides: a component is not content of its own -- it rides its
-- photo's decision, and the fold purges it together with the photo it
-- belongs to. (The exclusion is scoped to the fold's intent, not a
-- completeness claim: two colliding component checksums -- identical
-- video bytes under different photo halves -- are outside this
-- statement and ride the handover untouched.)
SELECT d.id AS departing_id,
       (d.deleted_at IS NULL)::boolean AS departing_live,
       d.deleted_at AS departing_deleted_at,
       k.id AS existing_id,
       k.deleted_at AS existing_deleted_at
FROM assets d
JOIN assets k
  ON k.owner_id = sqlc.arg('to_owner')
 AND k.checksum = d.checksum
WHERE d.owner_id = sqlc.arg('from_owner')
  AND NOT d.is_component
  AND NOT k.is_component
ORDER BY d.id;

-- name: MoveAssetAlbumMemberships :exec
-- A folded duplicate's album memberships join the surviving row, so
-- no album loses the photo: the membership's author rides along.
INSERT INTO album_items (album_id, asset_id, added_by)
SELECT ai.album_id, sqlc.arg('to_asset'), ai.added_by
FROM album_items ai
WHERE ai.asset_id = sqlc.arg('from_asset')
ON CONFLICT (album_id, asset_id) DO NOTHING;

-- name: MoveAssetTagAssignments :exec
-- A folded duplicate's tags join the surviving row: the label set of
-- the photo is the union of the two copies'.
INSERT INTO asset_tags (asset_id, tag_id, created_at)
SELECT sqlc.arg('to_asset'), at.tag_id, at.created_at
FROM asset_tags at
WHERE at.asset_id = sqlc.arg('from_asset')
ON CONFLICT (asset_id, tag_id) DO NOTHING;

-- name: MoveAssetFavorites :exec
-- A folded duplicate's favorites join the surviving row: every
-- account that starred the photo keeps it starred.
INSERT INTO user_favorites (user_id, asset_id, created_at)
SELECT uf.user_id, sqlc.arg('to_asset'), uf.created_at
FROM user_favorites uf
WHERE uf.asset_id = sqlc.arg('from_asset')
ON CONFLICT (user_id, asset_id) DO NOTHING;

-- A folded duplicate that sits in a cover set hands its cell to the
-- surviving row: the pin's store and the week store both name asset
-- ids directly. The colliding-cell statements run FIRST and drop the
-- duplicate's cell in every set that already holds the survivor (the
-- two copies of one photo cannot hold two positions of the same
-- set), so the moves that follow never trip the set's uniqueness.

-- name: DropCollidingAlbumCoverPicks :exec
DELETE FROM album_cover_week_picks drops
WHERE drops.asset_id = sqlc.arg('from_asset')
  AND EXISTS (
      SELECT 1 FROM album_cover_week_picks kept
      WHERE kept.album_id = drops.album_id
        AND kept.iso_week = drops.iso_week
        AND kept.asset_id = sqlc.arg('to_asset')
  );

-- name: DropCollidingAlbumCoverPinnedPicks :exec
DELETE FROM album_cover_picks drops
WHERE drops.asset_id = sqlc.arg('from_asset')
  AND EXISTS (
      SELECT 1 FROM album_cover_picks kept
      WHERE kept.album_id = drops.album_id
        AND kept.asset_id = sqlc.arg('to_asset')
  );

-- name: MoveAlbumCoverPicks :exec
UPDATE album_cover_picks
SET asset_id = sqlc.arg('to_asset')
WHERE asset_id = sqlc.arg('from_asset');

-- name: MoveAlbumCoverWeekPicks :exec
UPDATE album_cover_week_picks
SET asset_id = sqlc.arg('to_asset')
WHERE asset_id = sqlc.arg('from_asset');

-- name: ExtendAssetTrashMark :exec
-- The both-trashed fold's mark: the surviving row keeps the LATER of
-- the two deletion marks. The family reading is the generous one --
-- content stays recoverable as long as either copy had left, so the
-- countdown restarts from the later mark instead of shrinking to the
-- surviving row's own. GREATEST over the row's current value keeps a
-- survivor whose own mark is later untouched, which is what makes the
-- write order-free.
UPDATE assets
SET deleted_at = GREATEST(deleted_at, sqlc.arg('deleted_at')),
    updated_at = now()
WHERE id = sqlc.arg('id');

-- name: ReleaseArrivingDeviceIdentity :exec
-- device_asset_id is an upload-IDEMPOTENCY key, not content: the
-- partial index assets_owner_device_asset_id_live_uq allows one LIVE
-- row per (owner, device asset), so a live departing row arriving at
-- a destination that already holds a live row of the same device
-- identity trips it -- and the checksum fold cannot cover that pair,
-- because the two rows are DIFFERENT content: a device may re-render a
-- file under the same library identity (an edited photo keeps its
-- device-side id while its bytes change), and the two backups then
-- carry one id with two checksums. The arriving row releases the
-- identity instead: the destination's own copy is the live row that
-- governs what that device dedupes against, and the departing account
-- is about to be gone, so no idempotency anchor is lost. The transfer
-- arm only -- the trash arm's write lands every arriving row TRASHED
-- in the same statement, and the index covers live rows alone.
UPDATE assets departing
SET device_asset_id = NULL,
    updated_at = now()
FROM assets existing
WHERE departing.owner_id = sqlc.arg('from_owner')
  AND existing.owner_id = sqlc.arg('to_owner')
  AND departing.device_asset_id IS NOT NULL
  AND existing.device_asset_id = departing.device_asset_id
  AND departing.deleted_at IS NULL
  AND existing.deleted_at IS NULL;

-- name: TransferAssetOwnership :exec
-- The transfer arm's photo handover: every row the departing account
-- owned -- live, trashed, hidden, components alike -- becomes the
-- receiver's. The collision set is settled first over EVERY state pair
-- (the uniqueness index is total), so no row survives here that the
-- receiver already holds in any state; what is left is content the
-- receiver does not have at all.
UPDATE assets
SET owner_id = sqlc.arg('to_owner'), updated_at = now()
WHERE owner_id = sqlc.arg('from_owner');

-- name: TrashOwnedAssets :exec
-- The trash arm's photo disposal, the rows the per-file move does not
-- address (blob rows of both states, already-trashed external rows):
-- every remaining row becomes the caller's and carries a deletion
-- mark. The COALESCE keeps an existing mark: a photo that was already
-- in the departing account's trash keeps its own countdown. The
-- collision set is settled first over EVERY state pair (the total
-- (owner, checksum) index covers trashed rows too, so a caller who
-- already holds the same content -- in any state -- would otherwise
-- trip it), and the partial device index needs no pass here: this
-- statement moves every row it writes into the trashed state in the
-- same update, and that index covers live rows alone.
UPDATE assets
SET owner_id = sqlc.arg('to_owner'),
    deleted_at = COALESCE(deleted_at, now()),
    updated_at = now()
WHERE owner_id = sqlc.arg('from_owner');

-- name: TrashExternalAssetForOwner :execrows
-- The trash arm's external row, whose file the caller moved into the
-- server's trash area BEFORE this write (the move needs the operating
-- system's permission; a failure there failed the whole disposal and
-- nothing was written). The write records the truth of that move and
-- hands the row to the caller in one statement.
UPDATE assets
SET owner_id = sqlc.arg('to_owner'),
    deleted_at = now(),
    original_path = file_path,
    trash_path = sqlc.arg('trash_path'),
    file_path = sqlc.arg('trash_path'),
    updated_at = now()
WHERE id = sqlc.arg('id')
  AND owner_id = sqlc.arg('from_owner')
  AND storage_kind = 'external'
  AND deleted_at IS NULL;

-- name: ReassignAlbumItemAuthorship :exec
-- A membership the departing account added to ANOTHER account's album
-- keeps its place: the surviving owner carries the authorship, which
-- is what the foreign key can hold once the account is gone.
UPDATE album_items
SET added_by = sqlc.arg('to_user')
WHERE added_by = sqlc.arg('from_user');

-- name: ReassignShareGrants :exec
-- The grants the departing account issued keep their authority: the
-- surviving owner carries them. On the transfer arm these are the
-- shares of albums that just changed hands (their grantees keep their
-- access); on the trash arm the albums themselves are gone, so this
-- arms holds nothing.
UPDATE album_shares
SET granted_by = sqlc.arg('to_user')
WHERE granted_by = sqlc.arg('from_user');

-- name: MergeDuplicateTags :exec
-- The user-made labels ride the content: where the surviving owner
-- already carries a tag of the same name, the departing one's links
-- re-point at it, so a photo keeps both labels as one.
INSERT INTO asset_tags (asset_id, tag_id, created_at)
SELECT at.asset_id, keep.id, at.created_at
FROM asset_tags at
JOIN tags moving
  ON moving.id = at.tag_id
 AND moving.owner_id = sqlc.arg('from_owner')
JOIN tags keep
  ON keep.owner_id = sqlc.arg('to_owner')
 AND keep.name = moving.name
ON CONFLICT (asset_id, tag_id) DO NOTHING;

-- name: DropMergedTags :exec
-- The folded duplicates of the merge above (their links went with
-- them: the surviving tag holds them now).
DELETE FROM tags moving
USING tags keep
WHERE moving.owner_id = sqlc.arg('from_owner')
  AND keep.owner_id = sqlc.arg('to_owner')
  AND keep.name = moving.name;

-- name: TransferTagOwnership :exec
-- The remaining labels change hands whole: the tags are attached to
-- photos the surviving owner now holds.
UPDATE tags
SET owner_id = sqlc.arg('to_owner'), updated_at = now()
WHERE owner_id = sqlc.arg('from_owner');

-- name: DeleteUserRow :execrows
-- The last step of both arms: the account row goes, and the schema's
-- cascades take what lives with the account alone -- its sessions,
-- API keys and pairing tokens (it can never authenticate again), its
-- favorites, its per-album timeline preferences, the private names it
-- gave shared face clusters, and every album share granted TO it.
DELETE FROM users
WHERE id = $1;
-- name: ListOwnedAlbumIDs :many
-- The departing account's containers with their types, read before
-- they move or go: the transfer arm announces the albums and memories
-- it handed over (the stream is folded away, not handed over), the
-- trash arm announces the rows it removed.
SELECT id, type
FROM albums
WHERE owner_id = $1
ORDER BY id;

-- name: ListLiveOwnedAssets :many
-- The departing account's live rows, read before the trash arm marks
-- them: the asset events announce the state transition, and a
-- component video row stays silent (no view ever listed one).
SELECT id, media_kind, is_component
FROM assets
WHERE owner_id = $1
  AND deleted_at IS NULL
ORDER BY id;

-- name: ListLiveExternalAssets :many
-- The departing account's live external rows: the ones whose SOURCE
-- file the trash arm moves into the server's trash area ahead of the
-- row write (the move needs the operating system's permission, so it
-- is the one step the transaction cannot do itself).
SELECT id, file_path
FROM assets
WHERE owner_id = $1
  AND deleted_at IS NULL
  AND storage_kind = 'external'
ORDER BY id;
