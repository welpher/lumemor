-- +goose Up
-- Long-running job progress surfaced on the admin task panel. Queue-internal
-- state lives in the job queue's own tables and is not part of this schema.

CREATE TABLE job_runs (
    id          uuid PRIMARY KEY,
    kind        text NOT NULL,
    total       integer NOT NULL DEFAULT 0,
    done        integer NOT NULL DEFAULT 0,
    failed      integer NOT NULL DEFAULT 0,
    state       job_state NOT NULL DEFAULT 'queued',
    started_at  timestamptz,
    finished_at timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS job_runs;
