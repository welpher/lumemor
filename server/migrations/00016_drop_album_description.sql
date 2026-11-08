-- +goose Up
-- The album description column dies whole (the 2026-11-08 owner
-- evaluation): real-world fill rate near zero, the context already
-- carried by title / date-range+count / place / cover. The render
-- site (the album detail page's header line), the contract field
-- (the summary and detail shapes) and the PATCH face were removed
-- together in the same change -- no "viewable but uneditable"
-- half-state, no column the API can never read or write again.
ALTER TABLE albums
    DROP COLUMN description;

-- +goose Down
-- Recreates the column with its original shape (NOT NULL, the empty
-- string default), so the schema round-trips -- but honestly NOT the
-- data: every row comes back with the empty default, whatever text
-- the Up dropped. A description that existed before this migration
-- is gone for good; restoring it is a re-type, not a rollback.
ALTER TABLE albums
    ADD COLUMN description text NOT NULL DEFAULT '';
