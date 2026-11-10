-- +goose Up
-- The unified search's full-text plane: the trigram
-- substring search over the asset's own text sources.
--
-- The engine is the proven shape for this workload: pg_trgm's GIN
-- trigram indexes accelerate unaccented ILIKE substring matching,
-- which is language-agnostic by construction (containment needs no
-- segmentation, no stemming, no dictionary -- the v1 boundary; a
-- tokenizing engine is the recorded v1.x upgrade path). Both
-- extensions are PostgreSQL contrib modules the stock database image
-- carries, so this migration is the whole deployment: no image
-- change, no external engine.
--
-- unaccent() itself is STABLE (it reads the dictionary from disk),
-- so an index expression may not call it directly: the IMMUTABLE
-- f_unaccent wrapper below pins the dictionary by OID instead, the
-- standard workaround -- the index and the queries agree on exactly
-- one accent rule, and a dictionary swap cannot silently drift
-- between them.
--
-- The indexed expressions cover the three ASSET-OWNED text sources
-- (the original filename and the description on the assets rows, the
-- extracted OCR text on the ocr_texts rows -- a separate table, so
-- it carries its own index and joins at query time). The per-user
-- relational vocabularies (the caller's tags, the titles of albums
-- the caller can see, the caller's person labels) are deliberately
-- NOT indexed anywhere: they are per-user state, joined at query
-- time per the data model's search design -- no shared index can
-- scope rows by caller.

CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS unaccent;

-- The immutable unaccent wrapper the indexes and the search queries
-- share. The regdictionary cast pins the dictionary at parse time,
-- which is what makes the function honestly immutable: the rule set
-- cannot change under a built index.
CREATE FUNCTION f_unaccent(source text)
RETURNS text
LANGUAGE sql
IMMUTABLE
PARALLEL SAFE
STRICT
AS $$
    SELECT public.unaccent('public.unaccent'::regdictionary, source)
$$;

COMMENT ON FUNCTION f_unaccent(text) IS 'The immutable unaccent wrapper of the search plane: the shared accent rule of the trigram indexes and the search queries (dictionary pinned by OID).';

-- The trigram indexes over the owned text sources. Each index serves
-- the unaccented ILIKE '%...%' probe of its column; the search query
-- unions the probes.
CREATE INDEX assets_search_filename_trgm
    ON assets USING GIN (f_unaccent(original_filename) gin_trgm_ops);
CREATE INDEX assets_search_description_trgm
    ON assets USING GIN (f_unaccent(description) gin_trgm_ops);
CREATE INDEX ocr_texts_search_trgm
    ON ocr_texts USING GIN (f_unaccent(text) gin_trgm_ops);

-- +goose Down
DROP INDEX IF EXISTS ocr_texts_search_trgm;
DROP INDEX IF EXISTS assets_search_description_trgm;
DROP INDEX IF EXISTS assets_search_filename_trgm;
DROP FUNCTION IF EXISTS f_unaccent(text);
-- The contrib extensions stay: other surfaces may depend on them and
-- dropping an extensionCASCADEs destructively.
