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

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

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
	active, err := s.modelActivationMap(ctx)
	if err != nil {
		s.internalError(w, r, "read the models' activation flags", err)
		return
	}

	response := apigen.ModelListResponse{Models: make([]apigen.ModelResponse, 0, len(models))}
	for _, model := range models {
		response.Models = append(response.Models, modelResponseFrom(model, active[model.ID]))
	}

	// The download summary: the most recent run of the kind, whatever
	// its state -- the run the download endpoint's idempotence would
	// return -- or null before the first download was ever started.
	if row, err := s.queries.LatestJobRunByKind(ctx, modelDownloadKind); err == nil {
		summary := jobRunResponseFrom(row)
		response.Download = &summary
	} else if !errors.Is(err, pgx.ErrNoRows) {
		s.internalError(w, r, "find the latest model download run", err)
		return
	}
	s.writeJSON(w, http.StatusOK, response)
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

	rows, err := s.queries.ListJobRuns(r.Context(), int32(limit))
	if err != nil {
		s.internalError(w, r, "list the job runs", err)
		return
	}
	response := apigen.JobRunListResponse{Jobs: make([]apigen.JobRunResponse, 0, len(rows))}
	for _, row := range rows {
		response.Jobs = append(response.Jobs, jobRunResponseFrom(row))
	}
	s.writeJSON(w, http.StatusOK, response)
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

// jobRunResponseFrom maps one job_runs row onto the API shape.
func jobRunResponseFrom(row dbgen.JobRun) apigen.JobRunResponse {
	response := apigen.JobRunResponse{
		Id:     uuid.UUID(row.ID.Bytes),
		Kind:   apigen.JobRunKind(row.Kind),
		State:  apigen.JobRunState(row.State),
		Total:  int(row.Total),
		Done:   int(row.Done),
		Failed: int(row.Failed),
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
