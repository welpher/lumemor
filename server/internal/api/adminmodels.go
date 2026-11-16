// The admin model management and job panel endpoints:
// the model listing (the registry's rows plus the activation flags,
// the files-present probe and the download run summary), the
// activation switch (the registry's transactional switch wrapped so
// the response reports the cross-dimension arm's facts), the rerun
// trigger (idempotent the way the download trigger is: an in-flight
// run of the slot's kind answers 200, everything else starts a fresh
// run), the custom model scan (registers, never activates), and the
// job run listing of the task panel. The route group proves the admin
// role; every handler keeps the authContext-read shape of the rest of
// the surface.
//
// The rerun's per-asset ML work is the workers'
// business: the endpoint starts the run and the jobs package's driver
// owns its row; the contract description states that boundary
// honestly.
package api

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	openapi_types "github.com/oapi-codegen/runtime/types"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/ml"
)

// The job panel listing's page bounds, mirroring the timeline's
// discipline: absent means the default, a value over the maximum is
// answered with the maximum, a value below 1 answers 400.
const (
	jobsDefaultLimit = 50
	jobsMaxLimit     = 200
)

// rerunKindOf maps each ML slot onto the job_runs kind of its rerun
// runs. The vocabulary is pinned (the jobs
// package pins the same strings; its endpoint-level test drives the
// two against each other so they cannot drift apart).
func rerunKindOf(slot dbgen.ModelSlot) (string, bool) {
	switch slot {
	case dbgen.ModelSlotFaceDetect:
		return "face_detect_rerun", true
	case dbgen.ModelSlotFaceRecognize:
		return "face_recognize_rerun", true
	case dbgen.ModelSlotClipEmbed:
		return "clip_embed_rerun", true
	case dbgen.ModelSlotOcr:
		return "ocr_rerun", true
	}
	return "", false
}

// ListModels implements GET /api/v1/admin/models: every registered
// model with its facts, its activation state and its files-present
// probe, plus the most recent model download run as the summary.
func (s *server) ListModels(w http.ResponseWriter, r *http.Request) {
	// The route group already proved the admin role; the authContext
	// read keeps the handler's shape uniform with every other one.
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}
	ctx := r.Context()

	if s.models == nil {
		s.logger.Error("model listing requested without a model registry")
		s.writeError(w, http.StatusInternalServerError, apigen.INTERNAL,
			"the model registry is not wired to this server", nil)
		return
	}
	models, err := s.models.List()
	if err != nil {
		s.internalError(w, r, "list the models", err)
		return
	}
	rows, err := s.queries.ListModels(ctx)
	if err != nil {
		s.internalError(w, r, "list the model rows", err)
		return
	}
	active, err := s.modelActivationMap(ctx)
	if err != nil {
		s.internalError(w, r, "read the models' activation flags", err)
		return
	}

	response := apigen.ModelListResponse{Models: make([]apigen.ModelResponse, 0, len(models))}
	for _, model := range models {
		response.Models = append(response.Models, modelResponseFrom(model, active[model.ID]))
	}

	// The per-slot state rows: the four states the page's cards
	// speak, derived from the registry's rows and the most recent
	// download run.
	if s.modelsDir != "" {
		var runRow *dbgen.JobRun
		if row, err := s.queries.LatestJobRunByKind(ctx, modelDownloadKind); err == nil {
			runRow = &row
		} else if !errors.Is(err, pgx.ErrNoRows) {
			s.internalError(w, r, "find the latest model download run", err)
			return
		}
		slots, err := s.slotStates(rows, runRow)
		if err != nil {
			s.internalError(w, r, "derive the slot states", err)
			return
		}
		response.Slots = slots
	} else {
		response.Slots = make([]apigen.SlotStateResponse, 0)
	}

	// The official list: the manifest's catalog with its download
	// markers.
	entries, err := s.models.Catalog()
	if err != nil {
		s.internalError(w, r, "read the official list", err)
		return
	}
	response.Official = make([]apigen.ModelCatalogEntry, 0, len(entries))
	for _, entry := range entries {
		catalog := apigen.ModelCatalogEntry{
			Id:         entry.ID,
			Slot:       apigen.ModelSlot(entry.Slot),
			Name:       entry.Name,
			License:    entry.License,
			Bytes:      int(entry.Bytes),
			Downloaded: entry.Downloaded,
		}
		if entry.Dim > 0 {
			dim := entry.Dim
			catalog.Dim = &dim
		}
		response.Official = append(response.Official, catalog)
	}

	// The download summary: the most recent run of the kind, whatever
	// its state -- the run the download endpoint's idempotence would
	// return -- or null before the first download was ever started.
	if row, err := s.queries.LatestJobRunByKind(ctx, modelDownloadKind); err == nil {
		summary := jobRunResponseFrom(row, pgtype.Text{})
		response.Download = &summary
	} else if !errors.Is(err, pgx.ErrNoRows) {
		s.internalError(w, r, "find the latest model download run", err)
		return
	}
	s.writeJSON(w, http.StatusOK, response)
}

// slotStates derives the four-state row of every pipeline slot from
// the models table's rows and the most recent download run. The states
// are the listing's own vocabulary: a slot whose active model's files
// are present runs (active); a slot with a download run in flight
// waits for it (downloading, the run's group progress is the card's
// bar); a slot whose weights are missing with the last run failed
// failed; everything else is honestly not_downloaded -- the mirror
// carries no weights, so an unfilled slot is the fresh instance's
// truth, not an error.
func (s *server) slotStates(rows []dbgen.Model, run *dbgen.JobRun) ([]apigen.SlotStateResponse, error) {
	registryModels, err := s.models.List()
	if err != nil {
		return nil, err
	}
	byID := make(map[string]ml.Model, len(registryModels))
	for _, model := range registryModels {
		byID[model.ID] = model
	}
	runInFlight := run != nil && (run.State == dbgen.JobStateQueued || run.State == dbgen.JobStateRunning)
	runFailed := run != nil && run.State == dbgen.JobStateFailed
	states := make([]apigen.SlotStateResponse, 0, 5)
	for _, slot := range []apigen.ModelSlot{
		apigen.FaceDetect, apigen.FaceRecognize,
		apigen.ClipEmbed, apigen.ClipText, apigen.Ocr,
	} {
		state := apigen.SlotStateResponse{Slot: slot}
		var activeRow *dbgen.Model
		for i := range rows {
			if rows[i].Slot == dbgen.ModelSlot(slot) && rows[i].IsActive {
				activeRow = &rows[i]
				break
			}
		}
		if activeRow == nil {
			state.ModelId = nil
			state.Source = nil
			if runInFlight {
				state.State = apigen.SlotStateDownloading
				state.Progress = slotProgressOf(run)
			} else if runFailed {
				state.State = apigen.SlotStateFailed
			} else {
				state.State = apigen.SlotStateNotDownloaded
			}
			states = append(states, state)
			continue
		}
		model := byID[activeRow.ID]
		state.ModelId = &activeRow.ID
		host, sha := s.models.SourceFacts(model)
		// The source row is the contract's inline object: kind, the
		// host and digest facts (null when the manifest is unreadable
		// or carries no source URL), the registration time.
		source := &struct {
			DownloadedAt time.Time                          `json:"downloadedAt"`
			Host         *string                            `json:"host"`
			Kind         apigen.SlotStateResponseSourceKind `json:"kind"`
			Sha256       *string                            `json:"sha256"`
		}{
			Kind:         apigen.SlotStateResponseSourceKindOfficial,
			Host:         nil,
			Sha256:       nil,
			DownloadedAt: activeRow.AddedAt.Time.UTC(),
		}
		if model.Source == "custom" {
			source.Kind = apigen.SlotStateResponseSourceKindCustom
		}
		if host != "" {
			source.Host = &host
		}
		if sha != "" {
			source.Sha256 = &sha
		}
		state.Source = source
		if ml.FilesPresent(model) {
			state.State = apigen.SlotStateActive
		} else if runInFlight {
			state.State = apigen.SlotStateDownloading
			state.Progress = slotProgressOf(run)
		} else if runFailed {
			state.State = apigen.SlotStateFailed
		} else {
			state.State = apigen.SlotStateNotDownloaded
		}
		states = append(states, state)
	}
	return states, nil
}

// slotProgressOf maps a download run's counts onto the slot row's
// progress object.
func slotProgressOf(run *dbgen.JobRun) *struct {
	Done  int `json:"done"`
	Total int `json:"total"`
} {
	return &struct {
		Done  int `json:"done"`
		Total int `json:"total"`
	}{Done: int(run.Done), Total: int(run.Total)}
}

// ActivateModel implements PATCH /api/v1/admin/models/{id}: the model
// becomes the active one of its slot. The registry's switch is the
// whole semantics -- same-dimension switches flip the flags only
// (nothing is deleted, switching back revives the old products),
// cross-dimension clip switches wipe and re-dimension the embeddings
// inside the same transaction -- and this handler maps the result
// onto the response, reporting the destructive arm's facts.
func (s *server) ActivateModel(w http.ResponseWriter, r *http.Request, id string) {
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}
	ctx := r.Context()

	if s.models == nil {
		s.logger.Error("model activation requested without a model registry")
		s.writeError(w, http.StatusInternalServerError, apigen.INTERNAL,
			"the model registry is not wired to this server", nil)
		return
	}

	// The slot comes from the model's own row: the activation request
	// is the identifier alone. The preload also fronts the 404.
	row, err := s.queries.ModelByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		s.writeError(w, http.StatusNotFound, apigen.MODELNOTFOUND,
			"no model with this identifier", nil)
		return
	}
	if err != nil {
		s.internalError(w, r, "load the model", err)
		return
	}

	report, err := s.models.Activate(ml.Slot(row.Slot), id)
	if errors.Is(err, ml.ErrModelNotFound) {
		s.writeError(w, http.StatusNotFound, apigen.MODELNOTFOUND,
			"no model with this identifier", nil)
		return
	}
	if errors.Is(err, ml.ErrInvalidModel) {
		s.writeError(w, http.StatusBadRequest, apigen.MODELINVALID,
			"the model's files are missing or do not match their pinned digests", nil)
		return
	}
	if err != nil {
		s.internalError(w, r, "activate the model", err)
		return
	}

	response := apigen.ModelActivateResponse{
		Activated: apigen.ModelRef{Id: id, Slot: apigen.ModelSlot(row.Slot)},
	}
	if report.DimensionChanged {
		response.DimensionChanged = &apigen.ModelDimensionChange{
			From: report.FromDim,
			To:   report.ToDim,
		}
		wiped := report.WipedEmbeddings
		response.WipedEmbeddings = &wiped
	}
	s.writeJSON(w, http.StatusOK, response)
}

// RerunModel implements POST /api/v1/admin/models/{id}/rerun: one run
// that redoes the ACTIVE model's slot over the whole library. Only
// the active model of a slot can be rerun (products attach to the
// active model); a run of the slot's kind still in flight answers 200
// with its identifier, anything else starts a fresh run -- the rerun
// is manual, a finished run never blocks it.
func (s *server) RerunModel(w http.ResponseWriter, r *http.Request, id string) {
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}
	ctx := r.Context()

	row, err := s.queries.ModelByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		s.writeError(w, http.StatusNotFound, apigen.MODELNOTFOUND,
			"no model with this identifier", nil)
		return
	}
	if err != nil {
		s.internalError(w, r, "load the model", err)
		return
	}
	if !row.IsActive {
		s.writeError(w, http.StatusBadRequest, apigen.MODELINVALID,
			"only the active model of a slot can be rerun", nil)
		return
	}
	kind, ok := rerunKindOf(row.Slot)
	if !ok {
		s.writeError(w, http.StatusBadRequest, apigen.MODELINVALID,
			"the model's slot has no rerun", nil)
		return
	}

	// A run of the kind still in flight answers 200 with its
	// identifier: the click joins the running rerun instead of
	// starting a second one.
	if run, err := s.queries.FindInFlightJobRunByKind(ctx, kind); err == nil {
		s.writeJSON(w, http.StatusOK, apigen.ModelRerunResponse{JobRunId: uuid.UUID(run.ID.Bytes)})
		return
	} else if !errors.Is(err, pgx.ErrNoRows) {
		s.internalError(w, r, "find the in-flight rerun run", err)
		return
	}

	if s.mlReruns == nil {
		s.logger.Error("model rerun requested without a job queue")
		s.writeError(w, http.StatusInternalServerError, apigen.INTERNAL,
			"the rerun could not be started", nil)
		return
	}
	jobRunID, err := s.mlReruns.EnqueueMlRerun(ctx, ml.Slot(row.Slot))
	if err != nil {
		s.logger.Error("enqueue the model rerun", "error", err)
		s.writeError(w, http.StatusInternalServerError, apigen.INTERNAL,
			"the rerun could not be started", nil)
		return
	}
	s.writeJSON(w, http.StatusAccepted, apigen.ModelRerunResponse{JobRunId: jobRunID})
}

// DeleteModel implements DELETE /api/v1/admin/models/{id}: the
// removal of a registration that is not its slot's active one. The
// row goes first (the foreign keys keep the still-referencing
// products resolvable), the model directory follows; a directory
// failure logs and still reports the deletion -- the registration is
// gone, the leftover directory is the next scan's cleanup.
func (s *server) DeleteModel(w http.ResponseWriter, r *http.Request, id string) {
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}
	ctx := r.Context()
	if s.models == nil || s.modelsDir == "" {
		s.logger.Error("model deletion requested without a model registry")
		s.writeError(w, http.StatusInternalServerError, apigen.INTERNAL,
			"the model registry is not wired to this server", nil)
		return
	}
	row, err := s.queries.ModelByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		s.writeError(w, http.StatusNotFound, apigen.MODELNOTFOUND,
			"no model with this identifier", nil)
		return
	}
	if err != nil {
		s.internalError(w, r, "load the model", err)
		return
	}
	if row.IsActive {
		s.writeError(w, http.StatusBadRequest, apigen.MODELINVALID,
			"the active model of a slot cannot be deleted; switch first", nil)
		return
	}
	if err := s.queries.DeleteModel(ctx, id); err != nil {
		s.internalError(w, r, "delete the model", err)
		return
	}
	// The model's directory: the weights and the manifest copy live
	// beside the registered file (the landing layout's one-dir-per-id
	// rule), so the removal is the registered file's parent.
	modelDir := filepath.Dir(filepath.Join(s.modelsDir, row.FilePath))
	if err := os.RemoveAll(modelDir); err != nil {
		// The registration is already gone; the leftover directory is
		// the next scan's business, and the answer stays honest.
		s.logger.Error("remove the deleted model's directory", "error", err, "model", id)
	}
	s.writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

// ScanModels implements POST /api/v1/admin/models/scan: every model
// directory under the models dir carrying a manifest is verified and
// registered with source custom. Scanning registers, it never
// activates.
func (s *server) ScanModels(w http.ResponseWriter, r *http.Request) {
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}
	ctx := r.Context()

	if s.models == nil || s.modelsDir == "" {
		s.logger.Error("model scan requested without a model registry")
		s.writeError(w, http.StatusInternalServerError, apigen.INTERNAL,
			"the model registry is not wired to this server", nil)
		return
	}
	registered, err := s.models.Scan(s.modelsDir)
	if errors.Is(err, ml.ErrInvalidModel) {
		s.writeError(w, http.StatusBadRequest, apigen.MODELINVALID,
			"a scanned model's files are missing or do not match their pinned digests", nil)
		return
	}
	if err != nil {
		s.internalError(w, r, "scan the models dir", err)
		return
	}
	active, err := s.modelActivationMap(ctx)
	if err != nil {
		s.internalError(w, r, "read the models' activation flags", err)
		return
	}

	response := apigen.ModelScanResponse{Registered: make([]apigen.ModelResponse, 0, len(registered))}
	for _, model := range registered {
		response.Registered = append(response.Registered, modelResponseFrom(model, active[model.ID]))
	}
	s.writeJSON(w, http.StatusOK, response)
}

// ListJobs implements GET /api/v1/admin/jobs: the task panel's runs,
// newest first, bounded by the limit parameter.
func (s *server) ListJobs(w http.ResponseWriter, r *http.Request, params apigen.ListJobsParams) {
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}

	// The timeline's limit discipline: absent means the default, a
	// value over the maximum is answered with the maximum, a value
	// below 1 answers 400.
	limit := jobsDefaultLimit
	if params.Limit != nil {
		if *params.Limit < 1 {
			s.writeValidationError(w, "limit must be at least 1", []string{"limit"})
			return
		}
		if *params.Limit > jobsMaxLimit {
			limit = jobsMaxLimit
		} else {
			limit = *params.Limit
		}
	}

	rows, err := s.queries.ListJobRunsWithSubject(r.Context(), int32(limit))
	if err != nil {
		s.internalError(w, r, "list the job runs", err)
		return
	}
	response := apigen.JobRunListResponse{Jobs: make([]apigen.JobRunResponse, 0, len(rows))}
	for _, row := range rows {
		// The listing row is the flat join of the run and its subject:
		// the run's own columns carry under the same names, the
		// subject rides beside them.
		response.Jobs = append(response.Jobs, jobRunResponseFrom(dbgen.JobRun{
			ID:            row.ID,
			Kind:          row.Kind,
			Total:         row.Total,
			Done:          row.Done,
			Failed:        row.Failed,
			State:         row.State,
			StartedAt:     row.StartedAt,
			FinishedAt:    row.FinishedAt,
			CreatedAt:     row.CreatedAt,
			UpdatedAt:     row.UpdatedAt,
			AlbumID:       row.AlbumID,
			ResumeFrom:    row.ResumeFrom,
			StopRequested: row.StopRequested,
		}, row.Subject))
	}
	s.writeJSON(w, http.StatusOK, response)
}

// CancelJobRun implements POST /api/v1/admin/jobs/{runId}/cancel: the
// task panel's cancel (E41). The request only flags the run -- its
// driver checks the flag between chunks and closes it cancelled with
// the committed work kept. A run that already finished answers 409
// JOB_NOT_CANCELLABLE (the row's own state is the honest answer); an
// unknown id answers 404 JOB_RUN_NOT_FOUND.
func (s *server) CancelJobRun(w http.ResponseWriter, r *http.Request, runId openapi_types.UUID) {
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}

	run, err := s.queries.GetJobRun(r.Context(), pgUUID(runId))
	if errors.Is(err, pgx.ErrNoRows) {
		s.writeError(w, http.StatusNotFound, apigen.JOBRUNNOTFOUND,
			"no job run carries this id", nil)
		return
	}
	if err != nil {
		s.internalError(w, r, "load the job run", err)
		return
	}
	if run.State != dbgen.JobStateQueued && run.State != dbgen.JobStateRunning {
		s.writeError(w, http.StatusConflict, apigen.JOBNOTCANCELLABLE,
			"this run has already finished and cannot be cancelled", nil)
		return
	}

	// The flag write: rows-affected zero means the run finished
	// between the read and the write (the driver closed it) -- the
	// re-read decides the honest answer instead of a silent 204.
	rows, err := s.queries.RequestJobRunCancel(r.Context(), pgUUID(runId))
	if err != nil {
		s.internalError(w, r, "mark the job run asked-to-cancel", err)
		return
	}
	if rows == 0 {
		if _, err := s.queries.GetJobRun(r.Context(), pgUUID(runId)); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			s.internalError(w, r, "re-read the finished job run", err)
			return
		}
		s.writeError(w, http.StatusConflict, apigen.JOBNOTCANCELLABLE,
			"this run has already finished and cannot be cancelled", nil)
		return
	}

	// A run still QUEUED closes here, at once: no chunk ever started,
	// so there is no boundary to wait for. The flag write above is
	// the race safety -- if the driver's pickup interlocked with this
	// close, the re-read below answers running and the flag alone is
	// the contract (the driver's own first boundary check closes the
	// run). The guarded pickup (MarkJobRunRunning's state='queued'
	// guard) can never resurrect a run this close ended.
	run, err = s.queries.GetJobRun(r.Context(), pgUUID(runId))
	if err != nil {
		s.internalError(w, r, "re-read the cancelled job run", err)
		return
	}
	if run.State == dbgen.JobStateQueued {
		if err := s.queries.FinishJobRun(r.Context(), dbgen.FinishJobRunParams{
			ID:     pgUUID(runId),
			State:  dbgen.JobStateCancelled,
			Done:   run.Done,
			Failed: run.Failed,
		}); err != nil {
			s.internalError(w, r, "close the queued job run", err)
			return
		}
		// The close announces itself on the same stream every other
		// run close rides: the task panel converges without a poll.
		// The payload's shape is jobs.jobEventPayload's (the driver
		// side pins the same keys); the api side repeats it here for
		// the one close the drivers cannot own.
		if s.events != nil {
			s.events.Publish("job.finished", map[string]string{
				"jobRunId": runId.String(),
				"kind":     run.Kind,
				"total":    strconv.Itoa(int(run.Total)),
				"done":     strconv.Itoa(int(run.Done)),
				"failed":   strconv.Itoa(int(run.Failed)),
				"state":    string(dbgen.JobStateCancelled),
			})
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

// modelActivationMap reads every model's activation flag in one
// query: the registry's Model carries no activation state on purpose
// (it is a fact of the table, the admin surface's business).
func (s *server) modelActivationMap(ctx context.Context) (map[string]bool, error) {
	rows, err := s.queries.ListModels(ctx)
	if err != nil {
		return nil, err
	}
	active := make(map[string]bool, len(rows))
	for _, row := range rows {
		active[row.ID] = row.IsActive
	}
	return active, nil
}

// modelResponseFrom maps one registry model and its activation flag
// onto the API shape. dim, inputSize and the provenance notice are
// null when the model declares none (or its manifest is unreadable);
// filesPresent is the presence probe, never the digest verification.
func modelResponseFrom(model ml.Model, isActive bool) apigen.ModelResponse {
	response := apigen.ModelResponse{
		Id:           model.ID,
		Slot:         apigen.ModelSlot(model.Slot),
		Version:      model.Version,
		Source:       apigen.ModelSource(model.Source),
		License:      model.License,
		IsActive:     isActive,
		FilesPresent: ml.FilesPresent(model),
		Params:       model.Params,
	}
	if model.Notice != nil {
		response.Notice = model.Notice
	}
	if response.Params == nil {
		response.Params = map[string]interface{}{}
	}
	if model.Dim > 0 {
		dim := model.Dim
		response.Dim = &dim
	}
	if model.Width > 0 && model.Height > 0 {
		response.InputSize = &struct {
			Height int `json:"height"`
			Width  int `json:"width"`
		}{Width: model.Width, Height: model.Height}
	}
	return response
}

// jobRunResponseFrom maps one job_runs row onto the API shape. The
// subject (E41) is the run's subject album's title, when the run
// carries one -- a rescan's rows name their album, every other kind
// answers null.
func jobRunResponseFrom(row dbgen.JobRun, subject pgtype.Text) apigen.JobRunResponse {
	response := apigen.JobRunResponse{
		Id:      uuid.UUID(row.ID.Bytes),
		Kind:    apigen.JobRunKind(row.Kind),
		State:   apigen.JobRunState(row.State),
		Subject: nil,
		Total:   int(row.Total),
		Done:    int(row.Done),
		Failed:  int(row.Failed),
	}
	if subject.Valid {
		title := subject.String
		response.Subject = &title
	}
	if row.StartedAt.Valid {
		started := row.StartedAt.Time.UTC()
		response.StartedAt = &started
	}
	if row.FinishedAt.Valid {
		finished := row.FinishedAt.Time.UTC()
		response.FinishedAt = &finished
	}
	return response
}
