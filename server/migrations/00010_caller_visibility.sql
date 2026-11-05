-- +goose Up
-- The caller-visible-asset predicate, centralized (M3c Task 4).
--
-- Every read gate of the server answers the same question: can this
-- account see this asset? Before album sharing the answer was the
-- ownership row (a.owner_id = $viewer plus each surface's live-asset
-- filters), duplicated query by query. The album union replaces it:
-- an asset is caller-visible when it is a member of at least one
-- album that feeds the caller's view --
--
--   * the caller's own sync album (the core backup stream: every
--     upload lands there atomically, which is what makes the sync
--     arm the single-user equivalent of ownership), or
--   * the caller's own directory albums (E4's referenced files), or
--   * any album shared to the caller through album_shares (view ACL,
--     every shareable type -- virtual and directory).
--
-- The caller's own virtual albums are deliberately NOT an arm: they
-- never feed the caller's own timeline (in-stream curation would be
-- pure duplication), so an asset only an own virtual album holds
-- stays visible to exactly the owner's other arms or nobody.
--
-- This function is the ONE physical home of the arms. The
-- STABLE/LANGUAGE sql shape keeps it a plan-time candidate for
-- inlining into the calling query (and the per-row evaluation is the
-- same cost class as the EXISTS semi-join each consumer query ran
-- before); the timeline's aggregation query keeps its own inline
-- EXISTS because the spec's formula folds the per-user timeline
-- preference INSIDE the existential (an asset kept in through ANY
-- feeding album stays in even when another of its albums is switched
-- out), a conjunct that cannot compose outside it -- the arms there
-- are these arms, and the comment above that query says so.
CREATE FUNCTION caller_asset_visible(viewer uuid, asset uuid)
RETURNS boolean
STABLE
LANGUAGE sql
AS $$
    SELECT EXISTS (
        SELECT 1
        FROM album_items vi
        JOIN albums val ON val.id = vi.album_id
        LEFT JOIN album_shares vs
               ON vs.album_id = val.id AND vs.user_id = viewer
        WHERE vi.asset_id = asset
          AND (
              (val.type = 'sync' AND val.owner_id = viewer)
              OR (val.type = 'directory' AND val.owner_id = viewer)
              OR vs.user_id IS NOT NULL
          )
    )
$$;

-- +goose Down
DROP FUNCTION IF EXISTS caller_asset_visible(uuid, uuid);
