// The task catalog's own read and the instance-wide rescan sweep's
// start: GET /admin/jobs/summary answers the fixed catalog's state in
// one document (the live queue counts per pipeline kind, the run
// budget's pause switches and the latest run of each batch kind), and
// POST /admin/albums/rescan starts or joins the one sweep run whose
// units are albums.
//
// THE CATALOG IS NOT A LISTING: the routes exist so the catalog rows
// can speak without reading every run -- the execution history keeps
// its own listing (GET /admin/jobs). The queue counts are read
// straight from the queue's job table (the worker pool's truth, not a
// shadow counter); a kind with no job answers zeros.
//
// The sweep's heavy work lives on the queue: this file's handler is
// one bounded row read plus one enqueue, exactly like the per-album
// rescan's.
package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/db"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// albumRescanAllKind is the job_runs kind of the instance-wide rescan
// sweeps (the jobs package's pinned vocabulary, mirrored here for the
// in-flight join the way the per-album kind mirrors its own).
const albumRescanAllKind = "album_rescan_all"

// folderSyncJobKind is the queue's periodic folder sweep's kind. The
// sweep records no job_runs row by design, so the catalog's slot for
// it answers null; the kind is still pinned here because the catalog
// asks for it the same way it asks for the other batch kinds.
const folderSyncJobKind = "folder_sync"

// pipelineKinds is the catalog's pipeline vocabulary, in the
// contract's key order: the three media jobs and the four machine
// learning jobs. The strings ARE the queue's job kinds.
var pipelineKinds = []string{
	"thumbnail",
	"metadata",
	"transcode",
	"face_detect",
	"face_recognize",
	"clip_embed",
	"ocr",
}

// GetJobsSummary implements GET /api/v1/admin/jobs/summary: the fixed
// task catalog's state -- the queue's per-kind counts, the budget's
// pause switches, and the latest run of each batch kind.
func (s *server) GetJobsSummary(w http.ResponseWriter, r *http.Request) {
	// The route group already proved the admin role; the authContext
	// read keeps the handler's shape uniform with every other one.
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}
	ctx := r.Context()

	counts, err := s.pipelineCounts(ctx)
	if err != nil {
		s.internalError(w, r, "read the pipeline queue counts", err)
		return
	}
	// A corrupt budget row answers the defaults (the read is logged):
	// the catalog must still say where the switches stand.
	budget, err := db.RunBudgetOf(ctx, s.queries)
	if err != nil {
		s.logger.Error("read the run budget for the task summary; answering the defaults", "error", err)
	}
	runs, err := s.latestBatchRuns(ctx)
	if err != nil {
		s.internalError(w, r, "read the latest batch runs", err)
		return
	}
	s.writeJSON(w, http.StatusOK, apigen.JobsSummaryResponse{
		Pipeline: counts,
		Paused: apigen.PausedJobsState{
			Thumbnail: budget.PausedJobs.Thumbnail,
			Metadata:  budget.PausedJobs.Metadata,
			Transcode: budget.PausedJobs.Transcode,
			Ml:        budget.PausedJobs.Ml,
			Places:    budget.PausedJobs.Places,
		},
		Runs: runs,
	})
}

// pipelineCounts reads the queue's unfinished work per pipeline kind.
// The job table is River's own (outside the sqlc schema), so the read
// is one bounded aggregate statement: waiting sums the available,
// scheduled and retryable states, active counts running, and failed
// counts discarded (the queue's final give-up).
func (s *server) pipelineCounts(ctx context.Context) (apigen.PipelineCounts, error) {
	counts := apigen.PipelineCounts{}
	rows, err := s.pool.Query(ctx, `
		SELECT kind, state::text, count(*)
		FROM river_job
		WHERE kind = ANY($1)
		GROUP BY kind, state`, pipelineKinds)
	if err != nil {
		return counts, err
	}
	defer rows.Close()
	for rows.Next() {
		var kind, state string
		var count int
		if err := rows.Scan(&kind, &state, &count); err != nil {
			return counts, err
		}
		target := pipelineCountsOf(&counts, kind)
		if target == nil {
			continue
		}
		switch state {
		case "available", "scheduled", "retryable", "pending":
			target.Waiting += count
		case "running":
			target.Active += count
		case "discarded":
			target.Failed += count
		}
	}
	return counts, rows.Err()
}

// pipelineCountsOf answers the slot of one pipeline kind, or nil for
// a kind outside the catalog's vocabulary (the guard keeps an
// unexpected job kind out of the totals).
func pipelineCountsOf(counts *apigen.PipelineCounts, kind string) *apigen.PipelineJobCounts {
	switch kind {
	case "thumbnail":
		return &counts.Thumbnail
	case "metadata":
		return &counts.Metadata
	case "transcode":
		return &counts.Transcode
	case "face_detect":
		return &counts.FaceDetect
	case "face_recognize":
		return &counts.FaceRecognize
	case "clip_embed":
		return &counts.ClipEmbed
	case "ocr":
		return &counts.Ocr
	}
	return nil
}

// latestBatchRuns reads the most recent run of each of the catalog's
// batch kinds. A kind the instance never ran (and the folder sweep,
// which records no row at all) answers null.
func (s *server) latestBatchRuns(ctx context.Context) (apigen.LatestJobRuns, error) {
	var runs apigen.LatestJobRuns
	targets := []struct {
		kind string
		slot **apigen.JobRunSummary
	}{
		{rebuildDerivativesKind, &runs.RebuildDerivatives},
		{placesRerunJobKind, &runs.PlacesRerun},
		{albumRescanAllKind, &runs.AlbumRescanAll},
		{folderSyncJobKind, &runs.FolderSync},
		{modelDownloadKind, &runs.ModelDownload},
	}
	for _, target := range targets {
		run, err := s.queries.LatestJobRunByKind(ctx, target.kind)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return runs, err
		}
		*target.slot = jobRunSummaryOf(run)
	}
	return runs, nil
}

// jobRunSummaryOf maps one run row onto the catalog's summary shape.
func jobRunSummaryOf(run dbgen.JobRun) *apigen.JobRunSummary {
	summary := &apigen.JobRunSummary{
		Id:     uuid.UUID(run.ID.Bytes),
		State:  apigen.JobRunState(run.State),
		Total:  int(run.Total),
		Done:   int(run.Done),
		Failed: int(run.Failed),
	}
	if run.StartedAt.Valid {
		startedAt := run.StartedAt.Time
		summary.StartedAt = &startedAt
	}
	if run.FinishedAt.Valid {
		finishedAt := run.FinishedAt.Time
		summary.FinishedAt = &finishedAt
	}
	return summary
}

// RescanAllAlbums implements POST /api/v1/admin/albums/rescan: one
// sweep over every directory album. A sweep of the kind still in
// flight answers 200 with its identifier (the request joins it,
// starts nothing); anything else enqueues a fresh run through the
// queue seam and answers 202.
func (s *server) RescanAllAlbums(w http.ResponseWriter, r *http.Request) {
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}
	ctx := r.Context()

	// The in-flight join is instance-wide (the sweep's own scope): a
	// running sweep answers with its id; the per-album runs never
	// match here, they are a different kind.
	if run, err := s.queries.FindInFlightJobRunByKind(ctx, albumRescanAllKind); err == nil {
		s.writeJSON(w, http.StatusOK, apigen.AlbumRescanAllResponse{JobRunId: uuid.UUID(run.ID.Bytes)})
		return
	} else if !errors.Is(err, pgx.ErrNoRows) {
		s.internalError(w, r, "find the in-flight album rescan sweep", err)
		return
	}

	if s.albumRescanAlls == nil {
		s.logger.Error("album rescan sweep requested without a job queue")
		s.writeError(w, http.StatusInternalServerError, apigen.INTERNAL,
			"the album rescan could not be started", nil)
		return
	}
	jobRunID, err := s.albumRescanAlls.EnqueueAlbumRescanAll(ctx)
	if err != nil {
		s.logger.Error("enqueue the album rescan sweep", "error", err)
		s.writeError(w, http.StatusInternalServerError, apigen.INTERNAL,
			"the album rescan could not be started", nil)
		return
	}
	s.writeJSON(w, http.StatusAccepted, apigen.AlbumRescanAllResponse{JobRunId: jobRunID})
}
