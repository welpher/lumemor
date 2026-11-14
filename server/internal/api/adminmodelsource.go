// The model source endpoints: the probe/switch pipeline behind the
// switch dialog, and the per-slot state rows of the listing.
//
// THE PIPELINE: a custom source import is four steps -- stage, probe,
// register, swap -- and the first three are the ml package's probe
// manager's (the URL download into staging, the sandbox load, the
// slot contract's checks, the content-named registration); the fourth
// is the registry's Activate under the slot's lock. The steps run in
// this order and fail independently: a failed probe leaves nothing
// saved, the slot keeps its previous model and its products.
//
// THE SLOT LOCK: one switch of a slot at a time (TryLock, the loser
// answers 409) -- the swap is seconds, but two overlapping switches
// would race the recalculation run's products. In-flight ML jobs keep
// their own model reference and finish their asset; the next job
// dispatch reads the new active model, so no asset is half-processed.
//
// THE RECALC: the switch's semantics decide whether a recalculation
// run is enqueued -- the face slots and the OCR slot recompute their
// slot's products after any switch (the names survive: the cluster
// labels are user data, the membership follows the model), the embed
// slot only after a cross-dimension switch (a same-dimension switch
// keeps both generations coexisting, the rerun stays manual). The
// response names the run so the dialog can hand off to the task
// panel.
package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/ml"
)

// sourceSlots lists the slots that carry custom imports: the semantic
// arm's vision tower. The face pair and OCR ride the official list
// only (a detector switch re-tunes the alignment, an OCR switch is
// the two-file pair the manifest pins), and the text tower follows
// the vision tower it shares the joint space with.
var sourceSlots = map[apigen.ModelSlot]bool{
	apigen.ClipEmbed: true,
}

// probeMaxHeaderBytes bounds the multipart header buffer of a probe
// POST; the file parts stream to disk without buffering.
const probeMaxHeaderBytes = 1 << 20

// parseSlot validates the path's slot value against the contract's
// enum: the machine value the listing returns.
func parseSlot(raw string) (apigen.ModelSlot, bool) {
	switch apigen.ModelSlot(raw) {
	case apigen.FaceDetect, apigen.FaceRecognize,
		apigen.ClipEmbed, apigen.ClipText, apigen.Ocr:
		return apigen.ModelSlot(raw), true
	}
	return "", false
}

// slotLocks serializes the source switches: one swap per slot at a
// time, the loser answered 409 without touching anything. The locks
// are a process-level surface on purpose: the swap is in-process, and
// a restart simply clears them.
type slotLocks struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

// modelSlotLocks is the process's switch serialization.
var modelSlotLocks = newSlotLocks()

func newSlotLocks() *slotLocks {
	return &slotLocks{locks: map[string]*sync.Mutex{}}
}

// tryLock answers whether the caller got the slot's lock, with the
// unlock func when it did.
func (l *slotLocks) tryLock(slot string) (unlock func(), ok bool) {
	l.mu.Lock()
	lock, existing := l.locks[slot]
	if !existing {
		lock = &sync.Mutex{}
		l.locks[slot] = lock
	}
	l.mu.Unlock()
	if !lock.TryLock() {
		return nil, false
	}
	return lock.Unlock, true
}

// ProbeModelSource implements POST /api/v1/admin/models/{slot}/probe:
// the stage+probe half of the pipeline. The multipart body carries
// the candidate (a URL to download, or the uploaded bytes) and the
// answer is 202 with the probe identifier the dialog polls.
func (s *server) ProbeModelSource(w http.ResponseWriter, r *http.Request, slot apigen.ModelSlot) {
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}
	slotVal, ok := parseSlot(string(slot))
	if !ok {
		s.writeError(w, http.StatusNotFound, apigen.MODELSLOTUNKNOWN,
			"no model slot with this name", nil)
		return
	}
	if s.probes == nil {
		s.logger.Error("a model source probe was requested without the probe manager")
		s.writeError(w, http.StatusInternalServerError, apigen.INTERNAL,
			"the source pipeline is not wired to this server", nil)
		return
	}
	if !sourceSlots[slotVal] {
		s.writeValidationError(w,
			"this slot takes the official list only; custom imports are the semantic arm's",
			[]string{"slot"})
		return
	}

	// The multipart discipline mirrors the upload endpoint's: text
	// fields first, then the file part, streamed -- never buffered.
	if err := r.ParseMultipartForm(probeMaxHeaderBytes); err != nil {
		s.writeValidationError(w, "request body must be multipart/form-data", []string{"body"})
		return
	}
	pinnedSHA := strings.TrimSpace(r.FormValue("sha256"))
	if pinnedSHA != "" && !isSHA256(pinnedSHA) {
		s.writeValidationError(w, "sha256 must be 64 hex chars", []string{"sha256"})
		return
	}
	switch strings.TrimSpace(r.FormValue("kind")) {
	case "url":
		rawURL := strings.TrimSpace(r.FormValue("url"))
		if rawURL == "" {
			s.writeValidationError(w, "url is required for kind url", []string{"url"})
			return
		}
		probeID, err := s.probes.Start(ml.Slot(slotVal), rawURL, pinnedSHA)
		if err != nil {
			s.writeValidationError(w, err.Error(), []string{"url"})
			return
		}
		s.writeJSON(w, http.StatusAccepted, apigen.ModelProbeAcceptedResponse{ProbeId: probeID})
	case "file":
		file, _, err := r.FormFile("file")
		if err != nil {
			s.writeValidationError(w, "file is required for kind file", []string{"file"})
			return
		}
		defer func() { _ = file.Close() }()
		probeID, err := s.probes.StartFile(ml.Slot(slotVal), file, pinnedSHA)
		if err != nil {
			s.writeValidationError(w, err.Error(), []string{"file"})
			return
		}
		s.writeJSON(w, http.StatusAccepted, apigen.ModelProbeAcceptedResponse{ProbeId: probeID})
	default:
		s.writeValidationError(w, "kind must be url or file", []string{"kind"})
	}
}

// isSHA256 answers whether raw is a 64-hex digest string.
func isSHA256(raw string) bool {
	if len(raw) != 64 {
		return false
	}
	for _, r := range raw {
		isHex := (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
		if !isHex {
			return false
		}
	}
	return true
}

// GetModelProbe implements GET /api/v1/admin/models/probes/{probeId}:
// the probe the dialog polls, whatever its current state.
func (s *server) GetModelProbe(w http.ResponseWriter, r *http.Request, probeId string) {
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}
	if s.probes == nil {
		s.logger.Error("a model probe was read without the probe manager")
		s.writeError(w, http.StatusInternalServerError, apigen.INTERNAL,
			"the source pipeline is not wired to this server", nil)
		return
	}
	record, ok := s.probes.Get(probeId)
	if !ok {
		s.writeError(w, http.StatusNotFound, apigen.MODELPROBENOTFOUND,
			"no probe with this identifier", nil)
		return
	}
	response := apigen.ModelProbeResponse{
		ProbeId:      record.ID,
		State:        apigen.ModelProbeResponseState(record.State),
		HostDeclared: record.Declared,
	}
	if record.State == ml.ProbeRunning {
		response.Progress = &struct {
			Done  int `json:"done"`
			Total int `json:"total"`
		}{Done: record.Done, Total: record.Total}
	}
	if record.State == ml.ProbeOK || record.State == ml.ProbeWarn {
		dims := record.Dims
		response.Dims = &dims
		sha := record.SHA256
		response.Sha256 = &sha
		if record.Width > 0 && record.Height > 0 {
			response.InputSize = &struct {
				Height int `json:"height"`
				Width  int `json:"width"`
			}{Height: record.Height, Width: record.Width}
		}
	}
	if record.Reason != "" {
		reason := record.Reason
		response.Reason = &reason
	}
	s.writeJSON(w, http.StatusOK, response)
}

// SwitchModelSource implements POST /api/v1/admin/models/{slot}/source:
// the register+swap half of the pipeline, and the official list's
// switch. 202 answers the switch's facts; the recalculation, when one
// is due, is the enqueued run the response names.
func (s *server) SwitchModelSource(w http.ResponseWriter, r *http.Request, slot apigen.ModelSlot) {
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}
	slotVal, ok := parseSlot(string(slot))
	if !ok {
		s.writeError(w, http.StatusNotFound, apigen.MODELSLOTUNKNOWN,
			"no model slot with this name", nil)
		return
	}
	if s.models == nil {
		s.logger.Error("a model source switch was requested without a model registry")
		s.writeError(w, http.StatusInternalServerError, apigen.INTERNAL,
			"the model registry is not wired to this server", nil)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeValidationError(w, "request body could not be read", []string{"body"})
		return
	}
	var req apigen.ModelSourceSwitchRequest
	if err := json.Unmarshal(body, &req); err != nil {
		s.writeValidationError(w, "request body must be valid JSON", []string{"body"})
		return
	}

	unlock, locked := modelSlotLocks.tryLock(string(slotVal))
	if !locked {
		s.writeError(w, http.StatusConflict, apigen.MODELSLOTLOCKED,
			"a switch of this slot is already in flight", nil)
		return
	}
	defer unlock()

	switch req.Kind {
	case "builtin":
		s.switchBuiltin(w, r, slotVal, req)
	case "url", "file":
		s.switchCustom(w, r, slotVal, req)
	default:
		s.writeValidationError(w, "kind must be builtin, url or file", []string{"kind"})
	}
}

// switchBuiltin switches a slot onto an official-list entry: the
// entry is the validated catalog, no probe rides it. The entry must
// be on disk (the dialog renders the downloaded marker); the
// activation is the registry's, and the recalc the slot's semantics
// demand is enqueued.
func (s *server) switchBuiltin(w http.ResponseWriter, r *http.Request, slotVal apigen.ModelSlot, req apigen.ModelSourceSwitchRequest) {
	if req.ModelId == nil || *req.ModelId == "" {
		s.writeValidationError(w, "modelId is required for kind builtin", []string{"modelId"})
		return
	}
	// The catalog validates the id: an unknown or non-matching entry
	// is the request's own error.
	entries, err := s.models.Catalog()
	if err != nil {
		s.internalError(w, r, "read the official list", err)
		return
	}
	var entry *ml.CatalogEntry
	for i := range entries {
		if entries[i].ID == *req.ModelId && entries[i].Slot == ml.Slot(slotVal) {
			entry = &entries[i]
			break
		}
	}
	if entry == nil {
		s.writeError(w, http.StatusNotFound, apigen.MODELNOTFOUND,
			"no official-list entry with this identifier for this slot", nil)
		return
	}
	if !entry.Downloaded {
		s.writeError(w, http.StatusBadRequest, apigen.MODELINVALID,
			"the official-list entry's files are not on disk; download it first", nil)
		return
	}
	// The row must exist before the activation flips it: an entry
	// whose files were landed by another path (the offline bundle, a
	// hand-laid directory) registers here, the download service's
	// own insert.
	if manifestEntry, known := ml.ManifestEntryByID(*req.ModelId); known {
		if err := s.models.EnsureRegistered(r.Context(), manifestEntry); err != nil {
			s.internalError(w, r, "register the official-list entry", err)
			return
		}
	}
	response := s.activateAndRecalc(w, r, slotVal, *req.ModelId, req.ConfirmWipe != nil && *req.ConfirmWipe)
	if response != nil {
		s.writeJSON(w, http.StatusAccepted, *response)
	}
}

// switchCustom registers the probe's staged bytes and swaps them in:
// the probeId must name a probe that finished ok, the same-bytes reuse
// activates an existing registration instead of storing a duplicate,
// and the recalc the slot's semantics demand is enqueued.
func (s *server) switchCustom(w http.ResponseWriter, r *http.Request, slotVal apigen.ModelSlot, req apigen.ModelSourceSwitchRequest) {
	if !sourceSlots[slotVal] {
		s.writeValidationError(w,
			"this slot takes the official list only; custom imports are the semantic arm's",
			[]string{"slot"})
		return
	}
	if req.ProbeId == nil || *req.ProbeId == "" {
		s.writeError(w, http.StatusBadRequest, apigen.MODELPROBEREQUIRED,
			"the source must carry a probe that passed", nil)
		return
	}
	if s.probes == nil {
		s.logger.Error("a custom source switch was requested without the probe manager")
		s.writeError(w, http.StatusInternalServerError, apigen.INTERNAL,
			"the source pipeline is not wired to this server", nil)
		return
	}
	record, ok := s.probes.Get(*req.ProbeId)
	if !ok {
		s.writeError(w, http.StatusBadRequest, apigen.MODELPROBEREQUIRED,
			"the probe is unknown or expired; probe the source again", nil)
		return
	}
	if record.State != ml.ProbeOK && record.State != ml.ProbeWarn {
		s.writeError(w, http.StatusBadRequest, apigen.MODELPROBEREQUIRED,
			"the probe did not finish ok; probe the source again", nil)
		return
	}

	// The warned path keeps its promise: an activation through a
	// host the settings never declared is recorded in the server's
	// log -- the copy says the use is logged, so the use is logged.
	// The record carries the digest and the slot; the host itself is
	// the probe reason's business (the warning names it).
	if !record.Declared {
		s.logger.Warn("a custom model source from an undeclared host was switched in",
			"slot", string(slotVal), "sha256", record.SHA256, "probe", record.ID)
	}

	// The same-bytes reuse: a registration whose primary digest
	// matches the staged bytes is the model to activate, and the
	// staged file is dropped -- the same bytes are never stored
	// twice.
	if record.SHA256 != "" {
		if existing, found := s.models.FindByPrimarySHA256(record.SHA256); found {
			s.probes.Discard(*req.ProbeId)
			if response := s.activateAndRecalc(w, r, slotVal, existing.ID, req.ConfirmWipe != nil && *req.ConfirmWipe); response != nil {
				s.writeJSON(w, http.StatusAccepted, *response)
			}
			return
		}
	}
	if _, err := s.probes.RegisterStaged(r.Context(), *req.ProbeId, s.models); err != nil {
		switch {
		case errors.Is(err, ml.ErrProbeNotPassed), errors.Is(err, ml.ErrProbeNotFound):
			s.writeError(w, http.StatusBadRequest, apigen.MODELPROBEREQUIRED,
				"the probe is unknown, expired or did not finish ok; probe the source again", nil)
		default:
			s.internalError(w, r, "register the staged model", err)
		}
		return
	}
	// The registered id is the content-named slug the register step
	// just landed: the slot's row whose id carries the probe's digest
	// short form.
	models, err := s.models.List()
	if err != nil {
		s.internalError(w, r, "list the models", err)
		return
	}
	var registered *ml.Model
	for i := range models {
		if models[i].Slot == ml.Slot(slotVal) && strings.HasSuffix(models[i].ID, "@"+record.SHA256[:8]) {
			registered = &models[i]
			break
		}
	}
	if registered == nil {
		s.internalError(w, r, "find the registered model", errors.New("the register step landed no row"))
		return
	}
	if response := s.activateAndRecalc(w, r, slotVal, registered.ID, req.ConfirmWipe != nil && *req.ConfirmWipe); response != nil {
		s.writeJSON(w, http.StatusAccepted, *response)
	}
}

func (s *server) activateAndRecalc(w http.ResponseWriter, r *http.Request, slotVal apigen.ModelSlot, modelID string, confirmWipe bool) *apigen.ModelSourceSwitchResponse {
	ctx := r.Context()

	// The destructive arm's gate: a cross-dimension embed switch
	// wipes the column inside the activation transaction, so the
	// confirmation is the request's own field. The current dimension
	// is the database catalog's truth, never an assumption -- and a
	// read that cannot be made answers an error, never a silent
	// pass: the gate must fail closed like the transaction it guards.
	var requireConfirm bool
	if slotVal == apigen.ClipEmbed {
		currentDim, err := ml.EmbeddingColumnDim(ctx, s.pool)
		if err != nil {
			s.internalError(w, r, "read the embedding dimension", err)
			return nil
		}
		if currentDim <= 0 {
			s.internalError(w, r, "the embedding dimension is absent", errors.New("dimension not present"))
			return nil
		}
		row, rowErr := s.queries.ModelByID(ctx, modelID)
		if rowErr != nil {
			s.internalError(w, r, "read the target model", rowErr)
			return nil
		}
		if row.Dim.Valid && int(row.Dim.Int32) != currentDim {
			requireConfirm = true
		}
	}
	if requireConfirm && !confirmWipe {
		s.writeError(w, http.StatusBadRequest, apigen.MODELCONFIRMREQUIRED,
			"this switch wipes the slot's products; confirm it first", nil)
		return nil
	}

	report, err := s.models.Activate(ml.Slot(slotVal), modelID)
	if errors.Is(err, ml.ErrModelNotFound) {
		s.writeError(w, http.StatusNotFound, apigen.MODELNOTFOUND,
			"no model with this identifier", nil)
		return nil
	}
	if errors.Is(err, ml.ErrInvalidModel) {
		s.writeError(w, http.StatusBadRequest, apigen.MODELINVALID,
			"the model's files are missing or do not match their pinned digests", nil)
		return nil
	}
	if err != nil {
		s.internalError(w, r, "activate the model", err)
		return nil
	}

	response := &apigen.ModelSourceSwitchResponse{
		Activated: apigen.ModelRef{Id: modelID, Slot: slotVal},
	}
	if report.DimensionChanged {
		response.DimensionChanged = &struct {
			From int `json:"from"`
			To   int `json:"to"`
		}{From: report.FromDim, To: report.ToDim}
		wiped := report.WipedEmbeddings
		response.WipedEmbeddings = &wiped
	}

	// The recalc the slot's semantics demand: the face slots and the
	// OCR slot recompute their products after any switch (the names
	// survive, the membership follows the model); the embed slot only
	// when the dimension changed. A same-dimension embed switch keeps
	// both generations -- the rerun stays manual, as the activation
	// switch always kept it.
	if slotVal != apigen.ClipEmbed || report.DimensionChanged {
		if _, hasRerun := rerunKindOf(dbgen.ModelSlot(slotVal)); hasRerun && s.mlReruns != nil {
			runID, err := s.mlReruns.EnqueueMlRerun(ctx, ml.Slot(slotVal))
			if err != nil {
				// The switch landed; the recalc run is the task
				// panel's manual button if the enqueue failed.
				s.logger.Error("enqueue the switch's recalculation", "error", err, "slot", string(slotVal))
			} else {
				runIDStr := runID.String()
				response.RecalcRunId = &runIDStr
			}
		}
	}
	return response
}
