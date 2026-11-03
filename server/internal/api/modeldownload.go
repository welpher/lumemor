// The model download endpoint (M3a Task 3): the admin surface's
// trigger of the default model group's download. The route group
// proves the admin role; the handler's own discipline is idempotence
// -- a run that is still in flight (state queued or running) answers
// 200 with its identifier, and so does the most recent finished run,
// so repeated button clicks never pile up downloads (the job's own
// River retries own the recovery of a failed run). Only a first call
// -- no run of this kind at all -- creates one, answering 202. The
// request body is deliberately empty: the default group is the only
// group v1 downloads.
package api

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
)

// modelDownloadKind is the job_runs kind of the model download runs;
// the jobs package pins the same vocabulary (ModelDownloadKind).
const modelDownloadKind = "model_download"

// DownloadModels implements POST /api/v1/admin/models/download: start
// (or join) the default model group's download.
func (s *server) DownloadModels(w http.ResponseWriter, r *http.Request) {
	// The route group already proved the admin role; the authContext
	// read keeps the handler's shape uniform with every other one.
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}
	ctx := r.Context()

	// A run still in flight answers 200 with its identifier: the click
	// joins the running download instead of starting a second one.
	if row, err := s.queries.FindInFlightJobRunByKind(ctx, modelDownloadKind); err == nil {
		s.writeJSON(w, http.StatusOK, modelDownloadResponseFrom(row.ID))
		return
	} else if !errors.Is(err, pgx.ErrNoRows) {
		s.internalError(w, r, "find the in-flight model download run", err)
		return
	}

	// So does the most recent finished run: re-downloading a group
	// that already landed -- or whose job's own retries own the
	// recovery -- is not what a repeated click asks for.
	if row, err := s.queries.LatestJobRunByKind(ctx, modelDownloadKind); err == nil {
		s.writeJSON(w, http.StatusOK, modelDownloadResponseFrom(row.ID))
		return
	} else if !errors.Is(err, pgx.ErrNoRows) {
		s.internalError(w, r, "find the latest model download run", err)
		return
	}

	// The first call creates the run. A surface without the queue
	// wired, or an enqueue that fails, answers the surface's own
	// failure code: the download could not be started (the download
	// itself, once running, reports its failures through its row).
	if s.modelDownloads == nil {
		s.logger.Error("model download requested without a job queue")
		s.writeError(w, http.StatusInternalServerError, apigen.MODELDOWNLOADFAILED,
			"the model download could not be started", nil)
		return
	}
	jobRunID, err := s.modelDownloads.EnqueueModelDownload(ctx)
	if err != nil {
		s.logger.Error("enqueue the model download", "error", err)
		s.writeError(w, http.StatusInternalServerError, apigen.MODELDOWNLOADFAILED,
			"the model download could not be started", nil)
		return
	}
	s.writeJSON(w, http.StatusAccepted, apigen.ModelDownloadResponse{JobRunId: jobRunID})
}

// modelDownloadResponseFrom maps one job_runs row identifier onto the
// response shape.
func modelDownloadResponseFrom(id pgtype.UUID) apigen.ModelDownloadResponse {
	return apigen.ModelDownloadResponse{JobRunId: uuid.UUID(id.Bytes)}
}
