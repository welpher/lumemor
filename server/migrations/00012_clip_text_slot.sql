-- +goose Up
-- The CLIP text tower's slot (M3d Task 3): the model registry gains
-- the clip-text functional slot the search's semantic path resolves
-- its query encoder through.
--
-- The enum carried stt unclaimed from the start ("the enum is built
-- once in full, zero-cost reserved" -- the data model's own words),
-- but no value for a text tower: the search milestone is the first
-- slot addition, and it cannot ride clip_embed -- the registry pins
-- AT MOST ONE ACTIVE MODEL PER SLOT (the models_active_slot_uq
-- partial unique index), and the vision tower must stay active for
-- the embedding pipeline to keep writing the rows the semantic arm
-- searches. A dedicated slot is the only honest home: the text tower
-- registers, activates, lists on the model admin page and degrades
-- (no active model -> the semantic path answers empty) exactly like
-- every other slot.
--
-- ALTER TYPE ... ADD VALUE cannot run inside the transaction block a
-- goose migration runs in before PostgreSQL 12; the deployment floor
-- is PostgreSQL 17 (the pinned image), where the statement is
-- transactional as long as the new value is not USED in the same
-- transaction -- this migration only adds it.

ALTER TYPE model_slot ADD VALUE 'clip_text';

-- +goose Down
-- Enum values cannot be removed once created; the honest Down is the
-- empty statement (the value stays unused when the slot's models are
-- gone, and re-applying Up is a no-op).
