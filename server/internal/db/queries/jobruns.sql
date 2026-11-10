-- Job run queries: the job_runs rows back the admin task
-- panel -- long-running jobs surface their progress there and on the
-- activity stream (job.started / job.progress / job.finished). The
-- model download job is the first producer; the rerun jobs
-- of the later admin surface reuse the same statements with their own
-- kinds.
--
-- kind is plain text, pinned by the shared vocabulary
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

-- name: InsertAlbumRescanRun :exec
-- The rescan run's row: the places-rerun row's own shape plus
-- the run's SUBJECT -- the directory album the walk reconciles.
-- The album_id column exists for exactly this kind (00015): the
-- endpoint's in-flight guard is scoped to the album, so one
-- album's scan never blocks another's while a global kind lookup
-- would. The enqueue inserts the River job in the SAME transaction
-- (one transaction, neither row nor job behind on a half-failure),
-- and the total carries the enqueue-time scope -- the album's live
-- external member count, the half of the diff the database knows
-- (the worker sizes the row to the full diff once the walk listed
-- the root).
INSERT INTO job_runs (id, kind, total, album_id)
VALUES ($1, $2, $3, $4);

-- name: FindInFlightAlbumRescan :one
-- The rescan endpoint's idempotence lookup, scoped to the album: a
-- run of the kind still in flight (state queued or running) FOR
-- THIS ALBUM answers 200 with its identifier instead of a second
-- enqueue. Another album's in-flight run matches nothing here --
-- the scans are independent, and the guard's scope is the pin (a
-- global kind lookup would serialize every album's scan behind one
-- unrelated one).
SELECT id, kind, total, done, failed, state, started_at, finished_at, created_at, updated_at, album_id
FROM job_runs
WHERE kind = $1
  AND album_id = $2
  AND state IN ('queued', 'running')
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: SetJobRunTotal :exec
-- The rescan worker's sizing write: the enqueue-time total was the
-- database's half of the diff (the live member count), and the
-- walk's listing of the source root completes it -- the row's
-- total becomes the full diff (walked entries plus vanished
-- members) so the panel's progress ratio stays meaningful as the
-- arms resolve entries. The write lands once, right after the walk
-- listed the root and before the first arm runs.
UPDATE job_runs
SET total = $2, updated_at = now()
WHERE id = $1;

-- name: FindInFlightJobRunByKind :one
-- The most recent run of the kind that is still in flight (state
-- queued or running). The download endpoint's idempotence lookup: an
-- in-flight run answers 200 with its identifier instead of a second
-- enqueue.
SELECT id, kind, total, done, failed, state, started_at, finished_at, created_at, updated_at, album_id
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
SELECT id, kind, total, done, failed, state, started_at, finished_at, created_at, updated_at, album_id
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

-- name: IncrementJobRunDone :exec
-- Counts one per-asset ML job's SUCCESS into its rerun run's row
-- (the rerun closure): the per-asset workers report their own
-- completions through this atomic increment while the rerun driver
-- waits for the counts to converge. The increment is atomic because
-- the jobs of one run complete concurrently on the ml queue.
UPDATE job_runs
SET done = done + 1, updated_at = now()
WHERE id = $1;

-- name: IncrementJobRunFailed :exec
-- Counts one per-asset ML job's FAILURE into its rerun run's row:
-- the failure report of the per-asset jobs (one attempt each -- a
-- failed asset counts once) and the direct count of a dispatch whose
-- enqueue itself failed (the asset's work never started).
UPDATE job_runs
SET failed = failed + 1, updated_at = now()
WHERE id = $1;

-- name: FinishJobRun :exec
-- Closes the run: the final state with its converged counts and the
-- finish time.
UPDATE job_runs
SET state = $2, done = $3, failed = $4, finished_at = now(), updated_at = now()
WHERE id = $1;

-- name: ListJobRuns :many
-- The task panel's listing: the newest runs across every kind, in
-- creation order (the uuidv7 id as the tiebreak). The limit is the
-- page bound the admin surface pins (default 50, maximum 200).
SELECT id, kind, total, done, failed, state, started_at, finished_at, created_at, updated_at, album_id
FROM job_runs
ORDER BY created_at DESC, id DESC
LIMIT $1;

-- name: GetJobRun :one
-- One run's row: the driver reads its total at pick-up (the started
-- event echoes the enqueue-time scope, the durable truth, not a
-- recount).
SELECT id, kind, total, done, failed, state, started_at, finished_at, created_at, updated_at, album_id
FROM job_runs
WHERE id = $1;
