// The release check surface: the cached state behind GET
// /server/update-check, the manual run behind POST /server/update-check
// and the three settings behind GET/PUT /admin/settings/update-check.
//
// The route group proves the admin role (this face is the instance
// owner's business, not a member's: it names the release the instance
// runs and the feed it reaches); the handlers keep the authContext
// read of every other surface.
//
// The handlers own no network discipline of their own: the checker
// (the updatecheck package) carries the bounded request, the proxy
// environment, the internal-target refusal and the redirect refusal,
// and it owns the persisted answer. The handlers map its snapshot
// onto the contract's shape and validate the settings' shape -- the
// save never probes, so it cannot judge whether the endpoint is
// reachable, only whether it is a URL the probe could be pointed at.
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/db"
	"github.com/welpher/lumemor/server/internal/updatecheck"
)

// GetUpdateCheck implements GET /api/v1/server/update-check: the
// cached state of the release check -- never a network access.
func (s *server) GetUpdateCheck(w http.ResponseWriter, r *http.Request) {
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}
	if s.updateChecks == nil {
		s.logger.Error("release check requested without a checker")
		s.writeError(w, http.StatusInternalServerError, apigen.INTERNAL,
			"the release check is not wired to this server", nil)
		return
	}
	s.writeJSON(w, http.StatusOK, updateCheckResponse(s.updateChecks.Snapshot(r.Context())))
}

// RunUpdateCheck implements POST /api/v1/server/update-check: one
// probe now, answering the fresh state. A failed probe answers
// `unknown` with 200 -- the same silence the scheduled check keeps;
// a switched-off check answers `disabled` and probes nothing.
func (s *server) RunUpdateCheck(w http.ResponseWriter, r *http.Request) {
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}
	if s.updateChecks == nil {
		s.logger.Error("release check requested without a checker")
		s.writeError(w, http.StatusInternalServerError, apigen.INTERNAL,
			"the release check is not wired to this server", nil)
		return
	}
	s.writeJSON(w, http.StatusOK, updateCheckResponse(s.updateChecks.CheckNow(r.Context())))
}

// updateCheckResponse maps the checker's snapshot onto the contract's
// shape: the two optional fields are pointers there, so the snapshot's
// empty values become JSON null.
func updateCheckResponse(snapshot updatecheck.Snapshot) apigen.UpdateCheckResponse {
	response := apigen.UpdateCheckResponse{
		Enabled:        snapshot.Enabled,
		Status:         apigen.UpdateCheckStatus(snapshot.Status),
		CurrentVersion: snapshot.CurrentVersion,
	}
	if snapshot.LatestVersion != "" {
		latest := snapshot.LatestVersion
		response.LatestVersion = &latest
	}
	if snapshot.CheckedAt != nil {
		checkedAt := *snapshot.CheckedAt
		response.CheckedAt = &checkedAt
	}
	return response
}

// GetUpdateCheckSettings implements GET
// /api/v1/admin/settings/update-check: the switch, the feed and the
// cadence. The accessor's corrupt-row discipline answers the defaults
// with the corruption logged.
func (s *server) GetUpdateCheckSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}
	settings, err := db.UpdateCheckSettingsOf(r.Context(), s.queries)
	if err != nil {
		s.logger.Error("read the release check settings; answering the defaults", "error", err)
	}
	s.writeJSON(w, http.StatusOK, updateCheckSettingsResponse(settings))
}

// UpdateUpdateCheckSettings implements PUT
// /api/v1/admin/settings/update-check: validate the three fields and
// persist them. All three are contract-required: an omitted `enabled`
// would silently switch the check off, an omitted interval would
// turn it into a 1-hour one.
func (s *server) UpdateUpdateCheckSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeValidationError(w, "request body could not be read", []string{"body"})
		return
	}
	var req apigen.UpdateCheckSettingsUpdateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		s.writeValidationError(w, "request body must be valid JSON", []string{"body"})
		return
	}
	var presence map[string]json.RawMessage
	_ = json.Unmarshal(body, &presence)
	if missing := missingFields(presence, "enabled", "endpoint", "intervalHours"); len(missing) > 0 {
		s.writeValidationError(w, fmt.Sprintf("required fields missing: %s", strings.Join(missing, ", ")), missing)
		return
	}
	var invalid []string
	if !validWebURL(req.Endpoint) {
		invalid = append(invalid, "endpoint")
	}
	if req.IntervalHours < db.UpdateCheckIntervalMin || req.IntervalHours > db.UpdateCheckIntervalMax {
		invalid = append(invalid, "intervalHours")
	}
	if len(invalid) > 0 {
		s.writeValidationError(w, "invalid release check settings", invalid)
		return
	}
	settings := db.UpdateCheckSettings{
		Enabled:       req.Enabled,
		Endpoint:      req.Endpoint,
		IntervalHours: req.IntervalHours,
	}
	if err := db.SaveUpdateCheckSettings(r.Context(), s.queries, settings); err != nil {
		s.internalError(w, r, "save the release check settings", err)
		return
	}
	s.writeJSON(w, http.StatusOK, updateCheckSettingsResponse(settings))
}

// updateCheckSettingsResponse maps the stored settings onto the
// contract's shape (the field names are the same on both sides).
func updateCheckSettingsResponse(settings db.UpdateCheckSettings) apigen.UpdateCheckSettingsResponse {
	return apigen.UpdateCheckSettingsResponse{
		Enabled:       settings.Enabled,
		Endpoint:      settings.Endpoint,
		IntervalHours: settings.IntervalHours,
	}
}

// GetTilesSettings implements GET /api/v1/admin/settings/tiles: the
// map's tile source URL, the empty string when the instance has none.
func (s *server) GetTilesSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}
	tilejsonURL, err := db.TileSettings(r.Context(), s.queries)
	if err != nil {
		s.logger.Error("read the tile settings; answering unconfigured", "error", err)
	}
	s.writeJSON(w, http.StatusOK, apigen.TilesSettingsResponse{TilejsonUrl: tilejsonURL})
}

// UpdateTilesSettings implements PUT /api/v1/admin/settings/tiles:
// validate the URL's shape and persist it. The empty string clears
// the source; a non-empty value must be an absolute http(s) URL. The
// save cannot judge reachability -- the browser loads the source --
// and it deliberately admits LAN addresses (a home tile server).
func (s *server) UpdateTilesSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeValidationError(w, "request body could not be read", []string{"body"})
		return
	}
	var req apigen.TilesSettingsUpdateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		s.writeValidationError(w, "request body must be valid JSON", []string{"body"})
		return
	}
	var presence map[string]json.RawMessage
	_ = json.Unmarshal(body, &presence)
	if _, ok := presence["tilejsonUrl"]; !ok {
		s.writeValidationError(w, "required fields missing: tilejsonUrl", []string{"tilejsonUrl"})
		return
	}
	if req.TilejsonUrl != "" && !validWebURL(req.TilejsonUrl) {
		s.writeValidationError(w, "tilejsonUrl must be an absolute http or https URL", []string{"tilejsonUrl"})
		return
	}
	if err := db.SaveTileSettings(r.Context(), s.queries, req.TilejsonUrl); err != nil {
		s.internalError(w, r, "save the tile settings", err)
		return
	}
	s.writeJSON(w, http.StatusOK, apigen.TilesSettingsResponse{TilejsonUrl: req.TilejsonUrl})
}

// GetModelSettings implements GET /api/v1/admin/settings/models: the
// model store's automatic download switch.
func (s *server) GetModelSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}
	settings, err := db.ModelSettingsOf(r.Context(), s.queries)
	if err != nil {
		s.logger.Error("read the model settings; answering the defaults", "error", err)
	}
	s.writeJSON(w, http.StatusOK, apigen.ModelSettingsResponse{AutoDownload: settings.AutoDownload})
}

// UpdateModelSettings implements PUT /api/v1/admin/settings/models:
// the switch is contract-required (an omitted boolean would decode as
// off and silently change the policy) and takes effect on the next AI
// job's readiness check.
func (s *server) UpdateModelSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeValidationError(w, "request body could not be read", []string{"body"})
		return
	}
	var req apigen.ModelSettingsUpdateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		s.writeValidationError(w, "request body must be valid JSON", []string{"body"})
		return
	}
	var presence map[string]json.RawMessage
	_ = json.Unmarshal(body, &presence)
	if _, ok := presence["autoDownload"]; !ok {
		s.writeValidationError(w, "required fields missing: autoDownload", []string{"autoDownload"})
		return
	}
	if err := db.SaveModelSettings(r.Context(), s.queries, db.ModelSettings{AutoDownload: req.AutoDownload}); err != nil {
		s.internalError(w, r, "save the model settings", err)
		return
	}
	s.writeJSON(w, http.StatusOK, apigen.ModelSettingsResponse{AutoDownload: req.AutoDownload})
}

// GetMapConfig implements GET /api/v1/map/config: the map's
// client-side tile source, null when the instance has none. The read
// is the browser's only source of the URL; the tiles themselves never
// pass through the server.
func (s *server) GetMapConfig(w http.ResponseWriter, r *http.Request) {
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}
	tilejsonURL, err := db.TileSettings(r.Context(), s.queries)
	if err != nil {
		// The accessor's corrupt-row answer is the unconfigured
		// state: the map states the absence instead of loading a
		// source the row cannot be trusted to name.
		s.logger.Error("read the tile settings; answering unconfigured", "error", err)
	}
	response := apigen.MapConfigResponse{}
	if tilejsonURL != "" {
		response.TilejsonUrl = &tilejsonURL
	}
	s.writeJSON(w, http.StatusOK, response)
}

// validWebURL reports whether raw is an absolute http(s) URL with a
// host -- the shape both configured outbound URLs are validated
// against. It says nothing about reachability or about the address
// being public: the check's own probe refuses internal ranges, and
// the tile source is the browser's to load.
func validWebURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	return parsed.Host != ""
}

// missingFields lists the required fields absent from the body's
// presence map, in the caller's order -- the settings saves' shared
// guard against a zero value standing in for an omitted field.
func missingFields(presence map[string]json.RawMessage, fields ...string) []string {
	var missing []string
	for _, field := range fields {
		if _, ok := presence[field]; !ok {
			missing = append(missing, field)
		}
	}
	return missing
}
