-- Places reference-data queries: the idempotent
-- startup import of the baked GeoNames rows and the read surface of
-- the caller-visible city grouping.
--
-- places/place_names are reference tables in the schema's sense:
-- read-only imports with no audit columns (the schema exempts them
-- deliberately -- no application-side updates, the import is the only
-- writer). The import replaces wholesale: the DELETE pair empties the
-- tables inside the import transaction, the bulk COPY lands the new
-- rows, and the data-version marker in instance_settings records
-- which dataset the rows ARE. The marker is the skip key -- an
-- unchanged dataset (the normal reboot) imports nothing; a changed
-- one (a rebuilt image) replaces the tables atomically, and because
-- assets.place_id carries ON DELETE SET NULL, the replacement
-- releases every stale resolution, the state the admin
-- re-resolution run then refills.
--
-- The read surface's domain is the caller-visible asset union
-- (caller_asset_visible) narrowed to the library faces' display
-- representative (asset_library_representative): the city grouping
-- counts and covers only what the calling account can see, once per
-- content -- a photo uploaded and mirrored by a directory album is
-- one photo of the city, not two. Place display names join through
-- place_names at read time in the request's locale, falling back to
-- the canonical name -- the assets rows never carry locale strings,
-- so a locale switch touches no asset row.

-- name: GetGeodataVersion :one
-- The data-version marker of the loaded bake: the sha256 digest of
-- the exact staged bytes, stored as a JSON string. No row means the
-- import never ran.
SELECT (value #>> '{}')::text AS version
FROM instance_settings
WHERE key = $1;

-- name: SetGeodataVersion :exec
-- Records (or replaces) the data-version marker. The import writes
-- it inside its transaction, so the marker and the rows it describes
-- commit together. The value is the digest as a JSON string.
INSERT INTO instance_settings (key, value)
VALUES (sqlc.arg('key'), to_jsonb(sqlc.arg('version')::text))
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;

-- name: DeleteAllPlaceNames :exec
-- The import's replace step, first half: the locale names go before
-- the places they address (place_names deliberately carries no
-- foreign key -- admin1 entities join without places rows -- but the
-- delete order keeps the semantic parent-first shape anyway).
DELETE FROM place_names;

-- name: DeleteAllPlaces :exec
-- The import's replace step, second half: deleting the places rows
-- releases every assets.place_id reference through the schema's ON
-- DELETE SET NULL -- resolutions belong to the data version that
-- produced them.
DELETE FROM places;

-- name: ListPlaceCentroids :many
-- The index's working set: every places row as a centroid. The
-- startup load builds the in-memory grid FROM THE DATABASE rows (not
-- the parsed bake), so the resolver can only ever answer place ids
-- the places table actually carries -- what keeps the metadata job's
-- writes inside the foreign key.
SELECT id, lat, lon
FROM places;

-- name: ListCallerPlaces :many
-- The city grouping of GET /places: every place holding at least one
-- CALLER-VISIBLE live asset (the album union plus the timeline's
-- live-asset filters), with the honest per-caller count and capture
-- span. The display name is the locale join: the request's locale row
-- of the place when place_names carries one, the canonical name
-- otherwise -- a locale switch changes the join, never an asset row.
-- The place's own coordinate rides along: the map face pins it, and it
-- is the place record's coordinate -- never a photo's. Ordered by the
-- latest capture time descending (the most recently photographed city
-- first) with the place id ascending as the deterministic tiebreak.
SELECT p.id,
       COALESCE(pn.name, p.name) AS name,
       p.country_code,
       p.lat,
       p.lon,
       count(*) AS asset_count,
       min(a.taken_at)::timestamptz AS taken_at_from,
       max(a.taken_at)::timestamptz AS taken_at_to
FROM assets a
JOIN places p ON p.id = a.place_id
LEFT JOIN place_names pn
       ON pn.geoname_id = p.id AND pn.locale = sqlc.arg('locale')
WHERE a.deleted_at IS NULL
  AND a.is_hidden = false
  AND a.is_component = false
  AND asset_library_representative(a.id)
  AND caller_asset_visible(sqlc.arg('user_id'), a.id)
GROUP BY p.id, p.name, pn.name, p.country_code, p.lat, p.lon
ORDER BY taken_at_to DESC, p.id ASC;

-- name: ListPlaceCoverCandidates :many
-- The derived covers of the given places: each one's four most
-- recent caller-visible live members, taken_at descending with the
-- asset id as the tiebreak -- the wall's cover discipline, applied to
-- the city groups. The window keeps the result bounded at four rows
-- per place however many members a place holds.
SELECT ranked.place_id,
       ranked.asset_id
FROM (
    SELECT a.place_id,
           a.id AS asset_id,
           row_number() OVER (
               PARTITION BY a.place_id
               ORDER BY a.taken_at DESC, a.id DESC
           ) AS recency_rank
    FROM assets a
    WHERE a.place_id = ANY(sqlc.arg('place_ids')::integer[])
      AND a.deleted_at IS NULL
      AND a.is_hidden = false
      AND a.is_component = false
      AND asset_library_representative(a.id)
      AND caller_asset_visible(sqlc.arg('user_id'), a.id)
) ranked
WHERE ranked.recency_rank <= 4
ORDER BY ranked.place_id, ranked.recency_rank;

-- name: CountGpsAssets :one
-- The scope size of the places re-resolution run's full mode: every
-- asset whose exif row carries GPS coordinates (the rows the
-- in-memory grid can resolve), whatever the row's lifecycle state --
-- place resolution is metadata, not visibility. The manual-location
-- guard is the walk's own predicate (see ListGpsAssetPage): a row the
-- owner located by hand belongs to that decision, so the count and the
-- walk cover the same rows and a run converges to done = total.
SELECT count(*)
FROM assets a
JOIN exif e ON e.asset_id = a.id
WHERE e.lat IS NOT NULL
  AND e.lon IS NOT NULL
  AND a.location_lat IS NULL;

-- name: CountMissingPlaceAssets :one
-- The fill-in scope's size: the GPS-bearing rows whose place is
-- absent -- the subset a missing-only re-resolution walks, exactly
-- the same row predicate as CountGpsAssets narrowed by place_id
-- (the manual-location guard included, so the count and the walk
-- cover the same rows).
SELECT count(*)
FROM assets a
JOIN exif e ON e.asset_id = a.id
WHERE e.lat IS NOT NULL
  AND e.lon IS NOT NULL
  AND a.location_lat IS NULL
  AND a.place_id IS NULL;

-- name: ListMissingPlaceAssetPage :many
-- One page of the fill-in scope, keyset-ordered by the asset id (the
-- same walk shape as ListGpsAssetPage): only the rows whose place is
-- absent, so a missing-only run rewrites nothing that already
-- resolved. The manual-location rows stay outside, the walk's own
-- discipline (the owner's newer decision owns those rows).
SELECT a.id,
       e.lat,
       e.lon
FROM assets a
JOIN exif e ON e.asset_id = a.id
WHERE e.lat IS NOT NULL
  AND e.lon IS NOT NULL
  AND a.location_lat IS NULL
  AND a.place_id IS NULL
  AND a.id > sqlc.arg('cursor')
ORDER BY a.id
LIMIT sqlc.arg('limit_count');

-- name: ListGpsAssetPage :many
-- One page of the re-resolution scope, keyset-ordered by the asset
-- id (the cursor is the previous page's last id; the zero value
-- starts the walk). lat/lon ride along as the resolution input.
SELECT a.id,
       e.lat,
       e.lon
FROM assets a
JOIN exif e ON e.asset_id = a.id
WHERE e.lat IS NOT NULL
  AND e.lon IS NOT NULL
  -- A manual location is the owner's newer decision: the walk
  -- leaves those rows alone (the metadata refinement keeps the
  -- same hand), so a re-resolution rewrites only the rows the
  -- pipeline's own GPS step still owns.
  AND a.location_lat IS NULL
  AND a.id > sqlc.arg('cursor')
ORDER BY a.id
LIMIT sqlc.arg('limit_count');

-- name: SetAssetPlace :exec
-- One re-resolution write: the resolved place of the asset's GPS
-- (NULL when the grid answers nothing within the threshold -- the
-- same honest semantics the metadata job's step applies).
UPDATE assets SET
    place_id = sqlc.narg('place_id'),
    updated_at = now()
WHERE id = sqlc.arg('id');
