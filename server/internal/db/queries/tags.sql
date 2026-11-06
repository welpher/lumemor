-- Tag queries (M3d Task 2): the tags domain's read and write plane.
-- Tags are per-user state (B4, the favorites' privacy shape): every
-- query here scopes owner_id to the calling user, and the count
-- subqueries join the assignment rows to CALLER-VISIBLE live assets
-- -- the album union (caller_asset_visible) plus the timeline's live
-- filters -- the honest-counts discipline of the people and places
-- surfaces. The assignment resolution deliberately drops the live
-- filters (the favorites arms' discipline): the assignment row is
-- per-user state written whatever the asset's live state, and the
-- counts and views filter it, exactly like a favorite row.

-- name: ListCallerTags :many
-- The caller's whole vocabulary with the honest per-tag count,
-- sorted by name case-insensitively with the tag id as the
-- deterministic tiebreak -- the named-people listing's own ordering
-- discipline, applied to the vocabulary. A tag whose every member
-- is trashed or hidden still lists, with its count of zero (the
-- assignment rows survive underneath, exactly like favorite rows;
-- the count returns with the restore or unhide).
SELECT sqlc.embed(t),
       (
           SELECT count(*)
           FROM asset_tags atg
           JOIN assets a ON a.id = atg.asset_id
           WHERE atg.tag_id = t.id
             AND a.deleted_at IS NULL
             AND a.is_hidden = false
             AND a.is_component = false
             AND caller_asset_visible(sqlc.arg('user_id'), a.id)
       ) AS asset_count
FROM tags t
WHERE t.owner_id = sqlc.arg('user_id')
ORDER BY lower(t.name), t.id;

-- name: CreateTag :one
-- The idempotent tag creator. The insert arm serves the fresh name;
-- the ON CONFLICT arm serves every other case -- the duplicate ask
-- (the set semantics of the assignment face applied to the
-- vocabulary) and the concurrent-creator race, whose loser blocks on
-- the winner's index entry and returns the winner's row through the
-- value-identical no-op UPDATE, the EnsureSyncAlbum discipline.
-- RETURNING always answers the id of the tag that exists
-- afterwards; the handler compares it with the id it generated to
-- tell the fresh row (201) from the convergence (200).
INSERT INTO tags (id, owner_id, name)
VALUES (sqlc.arg('id'), sqlc.arg('owner_id'), sqlc.arg('name'))
ON CONFLICT (owner_id, name)
DO UPDATE SET updated_at = tags.updated_at
RETURNING id;

-- name: GetTagSummary :one
-- One tag's summary row: the tag itself plus the honest count, the
-- shape every tags-face response answers. The owner scope is the
-- gate: a tag another account owns answers no row, exactly like an
-- unknown id -- the single-shape 404's authority (the create and
-- rename responses, and both assignment endpoints' gates, read
-- through this one query).
SELECT sqlc.embed(t),
       (
           SELECT count(*)
           FROM asset_tags atg
           JOIN assets a ON a.id = atg.asset_id
           WHERE atg.tag_id = t.id
             AND a.deleted_at IS NULL
             AND a.is_hidden = false
             AND a.is_component = false
             AND caller_asset_visible(sqlc.arg('user_id'), a.id)
       ) AS asset_count
FROM tags t
WHERE t.id = sqlc.arg('tag_id')
  AND t.owner_id = sqlc.arg('user_id');

-- name: RenameTag :one
-- The rename. The owner scope is the gate (no row = the 404 of an
-- unknown or foreign tag); a name another of the caller's tags
-- carries trips the (owner_id, name) unique constraint, which the
-- handler maps to the 409. The updated_at arm is the set-semantics
-- guard: renaming to the row's own name writes nothing -- the
-- response is the row's current state, the no-op stays a no-op.
UPDATE tags
SET name = sqlc.arg('name'),
    updated_at = CASE
        WHEN tags.name = sqlc.arg('name') THEN tags.updated_at
        ELSE now()
    END
WHERE id = sqlc.arg('tag_id')
  AND owner_id = sqlc.arg('owner_id')
RETURNING id;

-- name: DeleteTag :execrows
-- The delete: the row goes and every asset_tags row follows through
-- the schema's ON DELETE CASCADE -- one statement, no orphans on any
-- account's view (the rows were the caller's own to begin with).
-- Zero rows is the 404 of a gone tag (a concurrent delete won the
-- race, the album delete's discipline).
DELETE FROM tags
WHERE id = sqlc.arg('tag_id')
  AND owner_id = sqlc.arg('owner_id');

-- name: AddTagAssets :execrows
-- The assignment add: resolution and insert in one statement, the
-- AddAlbumItems pattern with the resolution domain swapped for the
-- caller-visible union (the M3d widening -- the union deliberately
-- carries tags, so a sharee tags what they see with their OWN tag).
-- The WHERE arm IS the resolution: an unknown id, another account's
-- unshared asset, and any id the caller cannot see never produce
-- rows -- they are the skipped of the honest report, never errors.
-- No live-state filter (the favorites arms' discipline): the
-- assignment row is per-user state written whatever the asset's live
-- state, and the counts and views filter it. ON CONFLICT DO NOTHING
-- makes the already-tagged id the no-op of the idempotent add
-- (assignment is a set). The SELECT scans assets once, so a repeated
-- id in the array yields one row; the rows-affected count is the
-- report's added.
INSERT INTO asset_tags (asset_id, tag_id)
SELECT a.id, sqlc.arg('tag_id')
FROM assets a
WHERE a.id = ANY(sqlc.arg('asset_ids')::uuid[])
  AND caller_asset_visible(sqlc.arg('user_id'), a.id)
ON CONFLICT DO NOTHING;

-- name: RemoveTagAssets :execrows
-- The assignment remove: the caller's rows of this tag that address
-- the request's ids leave, and the ids that carry no assignment
-- simply match nothing. No visibility predicate -- the removal
-- tidies the caller's own set (a trashed member's assignment row
-- still leaves), and the tag scope keeps the statement inside the
-- caller's rows by construction.
DELETE FROM asset_tags
WHERE tag_id = sqlc.arg('tag_id')
  AND asset_id = ANY(sqlc.arg('asset_ids')::uuid[]);

-- name: ListAssetTagsForCaller :many
-- The asset detail's tags array: the CALLER's tags on the asset,
-- sorted by name case-insensitively with the tag id as the
-- deterministic tiebreak (the vocabulary listing's own ordering) --
-- the join runs through the caller's own tag rows, so another
-- account's tags on the same asset never surface (B4). No visibility
-- gate here: the read rides the asset detail's own permission,
-- exactly like the faces read (the owner reads a trashed asset's
-- tags like the trash view reads its faces).
SELECT t.id, t.name
FROM asset_tags atg
JOIN tags t ON t.id = atg.tag_id
WHERE atg.asset_id = sqlc.arg('asset_id')
  AND t.owner_id = sqlc.arg('user_id')
ORDER BY lower(t.name), t.id;
