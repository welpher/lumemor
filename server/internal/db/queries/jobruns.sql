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
SELECT id, kind, total, done, failed, state, started_at, finished_at, created_at, updated_at, album_id, resume_from, stop_requested
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
SELECT id, kind, total, done, failed, state, started_at, finished_at, created_at, updated_at, album_id, resume_from, stop_requested
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
SELECT id, kind, total, done, failed, state, started_at, finished_at, created_at, updated_at, album_id, resume_from, stop_requested
FROM job_runs
WHERE kind = $1
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: MarkJobRunRunning :execrows
-- Flips the run to running when its job picks it up. started_at keeps
-- the FIRST attempt's start (a retry does not move it): the task
-- panel's honest start time. The guard is the cancel's race-safety
-- (E41): a run the cancel closed while QUEUED matches no row here --
-- rows answers zero and the driver returns without starting (the
-- row stays cancelled; a running run is never re-closed by this
-- write, so the endpoint's queued close and the pickup cannot
-- interleave into a zombie).
UPDATE job_runs
SET state = 'running', started_at = COALESCE(started_at, now()), updated_at = now()
WHERE id = $1 AND state = 'queued';

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
SELECT id, kind, total, done, failed, state, started_at, finished_at, created_at, updated_at, album_id, resume_from, stop_requested
FROM job_runs
ORDER BY created_at DESC, id DESC
LIMIT $1;

-- name: GetJobRun :one
-- One run's row: the driver reads its total at pick-up (the started
-- event echoes the enqueue-time scope, the durable truth, not a
-- recount).
SELECT id, kind, total, done, failed, state, started_at, finished_at, created_at, updated_at, album_id, resume_from, stop_requested
FROM job_runs
WHERE id = $1;

-- name: SetJobRunResumeFrom :exec
-- The rebuild walk's breakpoint write: the id of the last asset the
-- run finished, so a window edge or a pause re-enters past
-- everything already re-derived. Other kinds never write it.
UPDATE job_runs
SET resume_from = $2, updated_at = now()
WHERE id = $1;

-- name: ClearJobRunResumeFrom :exec
-- The run's completion write: the cursor is the walk's own scratch,
-- cleared when the walk ends (succeeded or failed) so a finished row
-- carries no dangling resume state.
UPDATE job_runs
SET resume_from = NULL, updated_at = now()
WHERE id = $1;

-- name: RequestJobRunStop :exec
-- The task panel's stop: marks the in-flight run asked-to-stop. The
-- driver -- which checks the flag between assets -- closes the run at
-- the asset it just finished. Nothing happens when no run of the id
-- is in flight (the flag lands on a closed row harmlessly).
UPDATE job_runs
SET stop_requested = true, updated_at = now()
WHERE id = $1
  AND state IN ('queued', 'running');

-- name: GetJobRunStopRequested :one
-- The drivers' per-chunk stop check: one boolean read.
SELECT stop_requested
FROM job_runs
WHERE id = $1;

-- name: RequestJobRunCancel :execrows
-- The task panel's cancel (E41): marks the in-flight run
-- asked-to-stop, the same flag the rebuild's stop writes -- the
-- drivers check it between chunks (a walk page, a rescan batch) and
-- close the run cancelled with the committed work kept. RowsAffected
-- answers zero when the run finished between the caller's state read
-- and this write: the endpoint re-reads and answers 409, never a
-- silent no-op. A QUEUED run flags the same way -- its driver checks
-- before the first chunk and closes the run at once.
UPDATE job_runs
SET stop_requested = true, updated_at = now()
WHERE id = $1
  AND state IN ('queued', 'running');

-- name: GetJobRunSubject :one
-- One run's subject label (E41): the directory album a rescan run
-- walks, by title. The task panel's rows speak -- a rescan row names
-- its album; every other kind has no subject (the instance-wide
-- reruns never name one) and answers no row here.
SELECT a.title AS subject
FROM job_runs jr
JOIN albums a ON a.id = jr.album_id
WHERE jr.id = $1;

-- name: ListJobRunsWithSubject :many
-- The task panel's listing with the runs' subjects: the newest runs
-- across every kind, in creation order (the uuidv7 id as the
-- tiebreak), each joined to its subject album's title when the run
-- carries one (the album_rescan kind's rows; every other kind's
-- album_id is NULL and the join answers NULL). The limit is the page
-- bound the admin surface pins (default 50, maximum 200).
SELECT jr.id, jr.kind, jr.total, jr.done, jr.failed, jr.state, jr.started_at, jr.finished_at, jr.created_at, jr.updated_at, jr.album_id, jr.resume_from, jr.stop_requested,
       a.title AS subject
FROM job_runs jr
LEFT JOIN albums a ON a.id = jr.album_id
ORDER BY jr.created_at DESC, jr.id DESC
LIMIT $1;
