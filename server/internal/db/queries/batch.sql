-- Batch action queries (the favorites arms widened to caller-visible assets):
-- the write side of the batch endpoint. The endpoint's contract is
-- set semantics over one transaction: resolve the request's ids
-- against the action's guard (the caller's ownership for the
-- curation arms, the caller-visible union for the favorites arms),
-- then apply one action per resolved id through the primitives here
-- (trash and restore reuse the trash lifecycle queries,
-- whose owner guards and pair predicates are the single-row
-- authority). Every write carries its guard itself so a foreign or
-- unknown id is a no-op, never an error -- the guards make items
-- no-ops, which is what lets the whole batch commit as one
-- transaction without a failure path over individual ids.

-- name: ResolveBatchAssets :many
-- The batch's id resolution for the OWNER-scoped actions (hide,
-- unhide, trash, restore): which of the request's ids address a
-- row the caller owns. The owner guard IS the permission check of
-- those curation actions -- a foreign id resolves to no row exactly
-- like an unknown one, and the handler reports both through the
-- response's missing list (the deliberate trade of the list surface:
-- the caller handed the ids over themselves). No state filter
-- applies here: the actions are set operations over owned rows
-- whatever their state (favoriting a trashed asset, trashing an
-- already-trashed one), because the state arms live in the action
-- queries themselves -- resolution answers "is this id the caller's
-- to act on", nothing more. media_kind rides along for the events
-- the handler publishes per changed id after the commit, so the
-- pair (id, kind) resolves in the same read the handler trusts for
-- the report.
SELECT id, media_kind
FROM assets
WHERE id = ANY(sqlc.arg('ids')::uuid[])
  AND owner_id = sqlc.arg('owner_id');

-- name: ResolveVisibleAssets :many
-- The batch's id resolution for the FAVORITES arms (favorite,
-- unfavorite): which of the request's ids address an
-- asset the CALLER CAN SEE -- the album union (their own sync and
-- directory arms plus every album shared to them), so a sharee
-- favorites what they see. Like the owner resolution it applies no
-- state filter: the favorites row is per-user state written
-- whatever the asset's live state, and the view filters. media_kind
-- rides along for the same uniform report shape.
SELECT id, media_kind
FROM assets a
WHERE a.id = ANY(sqlc.arg('ids')::uuid[])
  AND caller_asset_visible(sqlc.arg('viewer'), a.id);

-- name: FavoriteAsset :execrows
-- The caller's favorites row for one asset. The upsert arm is the
-- set semantics: favoriting a favorite is a no-op (zero rows), not
-- an error -- a repeat of the batch converges. The row carries no
-- state of the asset itself, so it is written whatever the asset's
-- live/archived/trashed state: the favorites view filters on the
-- asset's state, the favorite survives trash and archiving and
-- resurfaces with the restore/unarchive (an idempotent per-user
-- mark, not a copy of the asset).
INSERT INTO user_favorites (user_id, asset_id)
VALUES (sqlc.arg('user_id'), sqlc.arg('asset_id'))
ON CONFLICT DO NOTHING;

-- name: UnfavoriteAsset :execrows
-- The inverse: remove the caller's favorites row. A missing row is
-- the zero-row no-op of the set semantics, never an error.
DELETE FROM user_favorites
WHERE user_id = sqlc.arg('user_id')
  AND asset_id = sqlc.arg('asset_id');

-- name: SetAssetHidden :execrows
-- The owner-level archive switch of one asset row: archive or
-- unarchive in a single statement, the caller deciding the target
-- value. The is_hidden <> $3 arm is the set-semantics guard:
-- switching a row that already carries the target value matches
-- nothing, so the handler answers the idempotent repeat as the same
-- success and the event stays silent. Unlike the trash lifecycle
-- this write is deliberately single-row -- no Live Photo pair
-- predicate: the component video row never appears in any
-- client-visible surface (every view filters is_component), so its
-- is_hidden is inert, and the photo of the pair is the row that
-- carries the pair's state. No deleted_at arm either: archiving a
-- trashed row (or unarchiving one) simply coexists with the trash
-- mark -- the row re-enters the live views still archived once
-- restored, which is the documented behavior of the archived view.
-- The PATCH endpoint reuses this write as its isArchived arm: one
-- authority for the switch, both surfaces agreeing on the state of
-- an asset.
UPDATE assets SET
    is_hidden = sqlc.arg('hidden'),
    updated_at = now()
WHERE id = sqlc.arg('id')
  AND owner_id = sqlc.arg('owner_id')
  AND is_hidden <> sqlc.arg('hidden');
