-- +goose Up
-- The cancelled state of a job run (E41): the task panel's cancel
-- asks an in-flight run to stop at its next chunk boundary, and the
-- run's driver -- which checks the stop flag between chunks -- closes
-- it cancelled. The work already committed stays (the completed
-- portion keeps its counts), so the ending is its own state and never
-- reads as a broken run: a stopped run is the admin's own deliberate
-- ending, not a failure.
ALTER TYPE job_state ADD VALUE IF NOT EXISTS 'cancelled' AFTER 'failed';

-- +goose Down
-- Enum values cannot be dropped in PostgreSQL; the state's removal
-- is a table rewrite this migration never promises. The Down target
-- is a no-op by design.
SELECT 1;
