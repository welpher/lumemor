-- +goose Up
-- The derivative rebuild's resume protocol on the job_runs rows: the
-- run's walk over the image assets persists its cursor (the id of the
-- last asset it finished) so a window edge or a pause stops the run
-- AT THE BREAKPOINT -- the next window re-enters past everything
-- already re-derived, never from zero -- and the task panel's stop
-- asks the driver to close the run between assets, never mid-asset.
-- Both columns are nullable/defaulted: every other kind ignores them.
ALTER TABLE job_runs
    ADD COLUMN resume_from uuid,
    ADD COLUMN stop_requested boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE job_runs
    DROP COLUMN IF EXISTS resume_from,
    DROP COLUMN IF EXISTS stop_requested;
