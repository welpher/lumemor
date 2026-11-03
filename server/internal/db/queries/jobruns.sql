-- Job run queries (M3a Task 3): the job_runs rows back the admin task
-- panel -- long-running jobs surface their progress there and on the
-- activity stream (job.started / job.progress / job.finished). The
-- model download job of Task 3 is the first producer; the rerun jobs
-- of the later admin surface reuse the same statements with their own
-- kinds.
--
-- kind is plain text, pinned by the milestone plan's vocabulary
-- (model_download, face_detect_rerun and friends); state is the
-- job_state enum. total/done carry the run's work units -- pinned
-- bytes for the model download, asset counts for the reruns -- and
-- failed the failed work count; int4 comfortably holds the default
-- group's ~400 MiB of pinned bytes.

-- name: InsertJobRun :exec
-- Creates the row the enqueue announces: state queued, zero progress,
-- the run's total. The enqueue inserts the queue's job in the SAME
-- transaction, so a half-failed enqueue leaves neither behind.
INSERT INTO job_runs (id, kind, total)
VALUES ($1, $2, $3);

-- name: FindInFlightJobRunByKind :one
-- The most recent run of the kind that is still in flight (state
-- queued or running). The download endpoint's idempotence lookup: an
-- in-flight run answers 200 with its identifier instead of a second
-- enqueue.
SELECT id, kind, total, done, failed, state, started_at, finished_at, created_at, updated_at
FROM job_runs
WHERE kind = $1 AND state IN ('queued', 'running')
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: LatestJobRunByKind :one
-- The most recent run of the kind, whatever its state. The download
-- endpoint's second idempotence arm: when nothing is in flight, the
-- most recent finished run answers 200 the same way (a repeated click
-- neither re-downloads a landed group nor piles up runs; the job's own
-- retries own the recovery of a failed one).
SELECT id, kind, total, done, failed, state, started_at, finished_at, created_at, updated_at
FROM job_runs
WHERE kind = $1
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: MarkJobRunRunning :exec
-- Flips the run to running when its job picks it up. started_at keeps
-- the FIRST attempt's start (a retry does not move it): the task
-- panel's honest start time.
UPDATE job_runs
SET state = 'running', started_at = COALESCE(started_at, now()), updated_at = now()
WHERE id = $1;

-- name: UpdateJobRunProgress :exec
-- Records one progress report of the run's worker. The model download
-- reports its throttled cumulative byte count; the failed count is the
-- finish's business.
UPDATE job_runs
SET done = $2, updated_at = now()
WHERE id = $1;

-- name: FinishJobRun :exec
-- Closes the run: the final state with its converged counts and the
-- finish time.
UPDATE job_runs
SET state = $2, done = $3, failed = $4, finished_at = now(), updated_at = now()
WHERE id = $1;
