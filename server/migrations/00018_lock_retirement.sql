-- +goose Up
-- The albums.locked column retires WHOLE (the 2026-11-10 owner
-- ruling): a lock can only stop the one account that can also lift
-- it -- the ability to unlock is exactly the ability to delete, both
-- the owner's alone -- so the freeze protected nobody and carried the
-- heaviest semantics of any album field (frozen rescan and deletes,
-- an unfrozen audience, the unlock-first ritual). The real safety net
-- sits in three layers that never needed the column: the deletability
-- setting (a UI affordance gate, not an execution gate), the trash
-- (every deletion recovers), and the read-only mount (the operating
-- system's own authority). The data goes with the column: every row
-- was unlocked or became unlocked the moment the column left.
ALTER TABLE albums DROP COLUMN locked;

-- +goose Down
-- The column returns with its default and NOT NULL, so the pre-00018
-- statement shapes keep working -- but the retired lock values are
-- gone: the column is history's shape, not its data (every restored
-- row answers unlocked, which is exactly what the retirement made
-- true).
ALTER TABLE albums ADD COLUMN locked boolean NOT NULL DEFAULT false;
