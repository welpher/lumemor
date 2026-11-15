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
-- The transfer arm's collision set: the departing account's LIVE
-- rows whose content the receiver already holds live. One live copy
-- per owner is the storage rule, so these rows cannot change hands;
-- they fold into the receiver's copy instead (the fingerprint is the
-- checksum, and a trashed row of either side is not a collision --
-- the uniqueness is partial on live rows).
SELECT d.id AS duplicate_id,
       k.id AS surviving_id
FROM assets d
JOIN assets k
  ON k.owner_id = sqlc.arg('to_owner')
 AND k.checksum = d.checksum
 AND k.deleted_at IS NULL
WHERE d.owner_id = sqlc.arg('from_owner')
  AND d.deleted_at IS NULL;

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

-- name: MoveAlbumCovers :exec
-- A folded duplicate that pins an album's cover hands the pin to the
-- surviving row (covers carry no foreign key: a dangling pin would
-- outlive the row it names).
UPDATE albums
SET cover_asset_id = sqlc.arg('to_asset'), updated_at = now()
WHERE cover_asset_id = sqlc.arg('from_asset');

-- name: TransferAssetOwnership :exec
-- The transfer arm's photo handover: every row the departing account
-- owned -- live, trashed, hidden, components alike -- becomes the
-- receiver's. The collision set is settled first, so the storage
-- rule's one-live-copy-per-owner holds through the write.
UPDATE assets
SET owner_id = sqlc.arg('to_owner'), updated_at = now()
WHERE owner_id = sqlc.arg('from_owner');

-- name: TrashOwnedAssets :exec
-- The trash arm's photo disposal, the rows the per-file move does not
-- address (blob rows of both states, already-trashed external rows):
-- every remaining row becomes the caller's and carries a deletion
-- mark. The COALESCE keeps an existing mark: a photo that was already
-- in the departing account's trash keeps its own countdown.
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
