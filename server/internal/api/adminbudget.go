// The run budget's settings pair: the task panel's run-settings
// document -- the concurrency dial's preset, the night window, the
// five per-kind pause switches. The same GET/PUT shape as every other
// admin settings face: one GET answering the current document (a
// corrupt row answers the defaults, the corruption logged), one PUT
// that requires every field and validates each value before the
// whole-object write.
//
// THE NEVER-BLOCKS RULE is what this face cannot break: the budget
// governs the queue's WORKERS only. An upload request and a serving
// read never consult it, so no dial position, window shape or pause
// switch can hold up the family's foreground.
package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/db"
)

// GetRunBudgetSettings implements GET /api/v1/admin/settings/run-budget:
// the instance's background-work budget.
func (s *server) GetRunBudgetSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}
	budget, err := db.RunBudgetOf(r.Context(), s.queries)
	if err != nil {
		s.logger.Error("read the run budget; answering the defaults", "error", err)
	}
	s.writeJSON(w, http.StatusOK, runBudgetResponse(budget))
}

// runBudgetResponse maps the stored document onto the contract shape.
func runBudgetResponse(budget db.RunBudget) apigen.RunBudgetSettingsResponse {
	return apigen.RunBudgetSettingsResponse{
		ConcurrencyPreset: apigen.ConcurrencyPreset(budget.ConcurrencyPreset),
		NightlyEnabled:    budget.NightlyEnabled,
		NightlyStart:      budget.NightlyStart,
		NightlyEnd:        budget.NightlyEnd,
		PausedJobs: apigen.PausedJobsState{
			Thumbnail: budget.PausedJobs.Thumbnail,
			Metadata:  budget.PausedJobs.Metadata,
			Transcode: budget.PausedJobs.Transcode,
			Ml:        budget.PausedJobs.Ml,
			Places:    budget.PausedJobs.Places,
		},
	}
}

// UpdateRunBudgetSettings implements PUT /api/v1/admin/settings/run-budget:
// every field is contract-required (an omitted one would decode as
// its zero value and silently change the posture), and each value is
// checked: the preset against its three-step enum, the two times
// against the HH:MM form (and against each other -- an equal pair
// would define a window that never opens, which the OFF switch says
// honestly), and the five pause switches against their presence.
//
// The save takes effect immediately, without a restart: every job
// reads this document when it starts, and a job already held wakes
// within its hold granularity and re-reads. The switches never touch
// a job already running -- the one in flight finishes.
func (s *server) UpdateRunBudgetSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeValidationError(w, "request body could not be read", []string{"body"})
		return
	}
	var req apigen.RunBudgetSettingsUpdateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		s.writeValidationError(w, "request body must be valid JSON", []string{"body"})
		return
	}
	var presence map[string]json.RawMessage
	_ = json.Unmarshal(body, &presence)
	missing := make([]string, 0, 5)
	for _, field := range []string{"concurrencyPreset", "nightlyEnabled", "nightlyStart", "nightlyEnd", "pausedJobs"} {
		if _, ok := presence[field]; !ok {
			missing = append(missing, field)
		}
	}
	if len(missing) > 0 {
		s.writeValidationError(w, "required fields missing: "+strings.Join(missing, ", "), missing)
		return
	}
	if req.ConcurrencyPreset != apigen.ConcurrencyPresetLight &&
		req.ConcurrencyPreset != apigen.ConcurrencyPresetStandard &&
		req.ConcurrencyPreset != apigen.ConcurrencyPresetFull {
		s.writeValidationError(w, "concurrencyPreset must be one of light, standard, full", []string{"concurrencyPreset"})
		return
	}
	if !db.ValidWindowTime(req.NightlyStart) {
		s.writeValidationError(w, "nightlyStart must be a HH:MM time", []string{"nightlyStart"})
		return
	}
	if !db.ValidWindowTime(req.NightlyEnd) {
		s.writeValidationError(w, "nightlyEnd must be a HH:MM time", []string{"nightlyEnd"})
		return
	}
	if req.NightlyStart == req.NightlyEnd {
		s.writeValidationError(w, "nightlyStart and nightlyEnd must differ", []string{"nightlyStart", "nightlyEnd"})
		return
	}
	var pausedPresence map[string]json.RawMessage
	_ = json.Unmarshal(presence["pausedJobs"], &pausedPresence)
	missingPaused := make([]string, 0, 5)
	for _, field := range []string{"thumbnail", "metadata", "transcode", "ml", "places"} {
		if _, ok := pausedPresence[field]; !ok {
			missingPaused = append(missingPaused, field)
		}
	}
	if len(missingPaused) > 0 {
		s.writeValidationError(w, "pausedJobs required fields missing: "+strings.Join(missingPaused, ", "), missingPaused)
		return
	}
	budget := db.RunBudget{
		ConcurrencyPreset: string(req.ConcurrencyPreset),
		NightlyEnabled:    req.NightlyEnabled,
		NightlyStart:      req.NightlyStart,
		NightlyEnd:        req.NightlyEnd,
		PausedJobs: db.PausedJobs{
			Thumbnail: req.PausedJobs.Thumbnail,
			Metadata:  req.PausedJobs.Metadata,
			Transcode: req.PausedJobs.Transcode,
			Ml:        req.PausedJobs.Ml,
			Places:    req.PausedJobs.Places,
		},
	}
	if err := db.SaveRunBudget(r.Context(), s.queries, budget); err != nil {
		s.internalError(w, r, "save the run budget", err)
		return
	}
	s.writeJSON(w, http.StatusOK, runBudgetResponse(budget))
}
