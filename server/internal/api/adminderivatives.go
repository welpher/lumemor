// The derivative actions: the image card's cost line (the summary
// read), the rebuild run's start and stop. The rebuild is a JOB -- it
// lists on the task panel with the four states, the pause and window
// rules of thumbnail-class work hold it, and the stop closes it
// between assets -- so these endpoints only create, join or flag the
// run: the heavy work lives on the queue.
//
// The rebuild kind and scopes are the contract's vocabulary (the
// JobRunKind and request enums); the jobs package's driver carries
// the same strings.
package api

import (
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/db"
)

// rebuildDerivativesKind is the job_runs kind of the derivative
// rebuild runs, pinned by the contract's JobRunKind enum.
const rebuildDerivativesKind = "rebuild_derivatives"

// estimateSecondsPerAsset is the declared estimate's per-asset cost:
// one second covers one decode plus the tile and view encodes at the
// shipped settings, divided by the dial's concurrency. It is a
// PLAN-OF-RECORD number the endpoint documents, not a measurement --
// the answer's copy says 约, and the NAS-class truth moves it.
const estimateSecondsPerAsset = 1.0

// GetDerivativesSummary implements GET /api/v1/admin/derivatives/summary:
// the image card's cost line -- the library's image count and the
// declared estimate of a full rebuild at the current settings and
// dial.
func (s *server) GetDerivativesSummary(w http.ResponseWriter, r *http.Request) {
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}
	count, err := s.queries.CountImageAssets(r.Context())
	if err != nil {
		s.internalError(w, r, "count the image assets", err)
		return
	}
	// The estimate divides the per-asset cost by the dial's current
	// step (a corrupt budget row answers the defaults, the read
	// logged) and floors at one minute: an empty library's rebuild
	// is still a run the admin starts on purpose.
	budget, budgetErr := db.RunBudgetOf(r.Context(), s.queries)
	if budgetErr != nil {
		s.logger.Error("read the run budget for the estimate; answering the defaults", "error", budgetErr)
	}
	estimate := int(math.Ceil(float64(count) * estimateSecondsPerAsset / float64(db.DialWorkers(budget.ConcurrencyPreset)) / 60.0))
	if estimate < 1 {
		estimate = 1
	}
	s.writeJSON(w, http.StatusOK, apigen.DerivativesSummaryResponse{
		ImageCount:      int(count),
		EstimateMinutes: estimate,
	})
}

// RebuildDerivatives implements POST /api/v1/admin/derivatives/rebuild:
// one run that re-derives the serving copies of every image asset at
// the current settings -- scope missing (only the assets whose
// current-format derivatives are absent) or all (everything, the
// other encoding's stale files swept). Idempotent the way the place
// re-resolution is: a run of the kind still in flight answers 200
// with its identifier, anything else starts a fresh run.
func (s *server) RebuildDerivatives(w http.ResponseWriter, r *http.Request) {
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeValidationError(w, "request body could not be read", []string{"body"})
		return
	}
	var req apigen.RebuildDerivativesRequest
	if err := json.Unmarshal(body, &req); err != nil {
		s.writeValidationError(w, "request body must be valid JSON", []string{"body"})
		return
	}
	if req.Scope != apigen.Missing && req.Scope != apigen.All {
		s.writeValidationError(w, "scope must be one of missing, all", []string{"scope"})
		return
	}

	// A run of the kind still in flight answers 200 with its
	// identifier: the request joins the running rebuild instead of
	// starting a second one.
	if run, err := s.queries.FindInFlightJobRunByKind(r.Context(), rebuildDerivativesKind); err == nil {
		s.writeJSON(w, http.StatusOK, apigen.RebuildDerivativesResponse{JobRunId: uuid.UUID(run.ID.Bytes)})
		return
	} else if !errors.Is(err, pgx.ErrNoRows) {
		s.internalError(w, r, "find the in-flight rebuild run", err)
		return
	}

	if s.derivativesRebuilds == nil {
		s.logger.Error("derivative rebuild requested without a job queue")
		s.writeError(w, http.StatusInternalServerError, apigen.INTERNAL,
			"the rebuild could not be started", nil)
		return
	}
	jobRunID, err := s.derivativesRebuilds.EnqueueDerivativesRebuild(r.Context(), string(req.Scope))
	if err != nil {
		// The queue refuses the enqueue of a scope outside the enum
		// exactly as it does for a broken transaction; the log line
		// carries which.
		s.logger.Error("enqueue the derivative rebuild", "error", err)
		s.writeError(w, http.StatusInternalServerError, apigen.INTERNAL,
			"the rebuild could not be started", nil)
		return
	}
	s.writeJSON(w, http.StatusAccepted, apigen.RebuildDerivativesResponse{JobRunId: jobRunID})
}

// StopDerivativesRebuild implements POST /api/v1/admin/derivatives/rebuild/stop:
// asks the in-flight rebuild to close at its next breakpoint (the
// driver checks the flag between assets). No rebuild in flight
// answers 409 REBUILD_NOT_RUNNING: the flag has nothing to land on.
func (s *server) StopDerivativesRebuild(w http.ResponseWriter, r *http.Request) {
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}
	run, err := s.queries.FindInFlightJobRunByKind(r.Context(), rebuildDerivativesKind)
	if errors.Is(err, pgx.ErrNoRows) {
		s.writeError(w, http.StatusConflict, apigen.REBUILDNOTRUNNING,
			"no derivative rebuild is in flight", nil)
		return
	}
	if err != nil {
		s.internalError(w, r, "find the in-flight rebuild run", err)
		return
	}
	if err := s.queries.RequestJobRunStop(r.Context(), pgUUID(uuid.UUID(run.ID.Bytes))); err != nil {
		s.internalError(w, r, "mark the rebuild run asked-to-stop", err)
		return
	}
	s.writeJSON(w, http.StatusOK, apigen.RebuildDerivativesResponse{JobRunId: uuid.UUID(run.ID.Bytes)})
}
