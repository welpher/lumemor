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
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/db"
	"github.com/welpher/lumemor/server/internal/ml"
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
// map's tile source URLs, the empty strings when the instance has
// none.
func (s *server) GetTilesSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}
	settings, err := db.TileSettingsOf(r.Context(), s.queries)
	if err != nil {
		s.logger.Error("read the tile settings; answering unconfigured", "error", err)
	}
	s.writeJSON(w, http.StatusOK, apigen.TilesSettingsResponse{
		TilejsonUrl:     settings.TileJSONURL,
		DarkTilejsonUrl: settings.DarkTileJSONURL,
	})
}

// UpdateTilesSettings implements PUT /api/v1/admin/settings/tiles:
// validate both URLs' shapes and persist the pair. The empty string
// clears a source; a non-empty value must be an absolute http(s) URL.
// The dark style is optional in the payload -- an omitted or empty
// dark URL means the dark theme reuses the light source. The save
// cannot judge reachability -- the browser loads the sources -- and
// it deliberately admits LAN addresses (a home tile server).
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
	// The dark style is optional on the wire (the schema does not
	// require it): an omitted field and an empty string both mean
	// unconfigured, which is what an empty document save does.
	darkTilejsonURL := ""
	if req.DarkTilejsonUrl != nil {
		darkTilejsonURL = *req.DarkTilejsonUrl
	}
	if darkTilejsonURL != "" && !validWebURL(darkTilejsonURL) {
		s.writeValidationError(w, "darkTilejsonUrl must be an absolute http or https URL", []string{"darkTilejsonUrl"})
		return
	}
	settings := db.TileSettings{
		TileJSONURL:     req.TilejsonUrl,
		DarkTileJSONURL: darkTilejsonURL,
	}
	if err := db.SaveTileSettings(r.Context(), s.queries, settings); err != nil {
		s.internalError(w, r, "save the tile settings", err)
		return
	}
	s.writeJSON(w, http.StatusOK, apigen.TilesSettingsResponse{
		TilejsonUrl:     settings.TileJSONURL,
		DarkTilejsonUrl: settings.DarkTileJSONURL,
	})
}

// GetModelSettings implements GET /api/v1/admin/settings/models: the
// model store's automatic download switch plus the download sources
// (the allowed hosts and the mirror base the footer's panel edits).
func (s *server) GetModelSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}
	settings, err := db.ModelSettingsOf(r.Context(), s.queries)
	if err != nil {
		s.logger.Error("read the model settings; answering the defaults", "error", err)
	}
	s.writeJSON(w, http.StatusOK, modelSettingsResponse(settings))
}

// modelSettingsResponse maps the stored policy onto the contract
// shape; the response's host slice is a copy, so the encoder never
// aliases the stored value.
func modelSettingsResponse(settings db.ModelSettings) apigen.ModelSettingsResponse {
	return apigen.ModelSettingsResponse{
		AutoDownload:  settings.AutoDownload,
		AllowedHosts:  append([]string(nil), settings.AllowedHosts...),
		MirrorBaseUrl: settings.MirrorBaseUrl,
		Params:        modelParamsResponse(settings.Params),
	}
}

// modelParamsResponse maps the stored pipeline params onto the
// contract shape (E41): the two knobs, as float32s of the generated
// type.
func modelParamsResponse(params db.MlModelParams) apigen.ModelParams {
	return apigen.ModelParams{
		FaceDetectMinScore:     float32(params.FaceDetectMinScore),
		FaceRecognizeThreshold: float32(params.FaceRecognizeThreshold),
	}
}

// UpdateModelSettings implements PUT /api/v1/admin/settings/models:
// every field is contract-required (an omitted one would decode as
// its zero value and silently change the policy) and the values take
// effect on the next dispatch. The allowed hosts must be hostnames
// with an optional `*.` wildcard label; a mirror base, when
// non-empty, must name a host -- and that host joins the saved list
// automatically, so a mirror the list refuses would never fail on
// its own first hop. The params must sit in the closed 0-1 domain --
// the values the ML jobs read at their own dispatch.
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
	missing := make([]string, 0, 4)
	for _, field := range []string{"autoDownload", "allowedHosts", "mirrorBaseUrl", "params"} {
		if _, ok := presence[field]; !ok {
			missing = append(missing, field)
		}
	}
	if len(missing) > 0 {
		s.writeValidationError(w, "required fields missing: "+strings.Join(missing, ", "), missing)
		return
	}
	// The params' sub-fields are contract-required too: the nested
	// object's own presence does not answer for its two knobs, whose
	// silent zero would read as "accept nothing" (the detection's
	// score floor) and "cluster everything apart" (the threshold).
	paramPresence := make(map[string]json.RawMessage)
	_ = json.Unmarshal(presence["params"], &paramPresence)
	paramMissing := make([]string, 0, 2)
	for _, field := range []string{"faceDetectMinScore", "faceRecognizeThreshold"} {
		if _, ok := paramPresence[field]; !ok {
			paramMissing = append(paramMissing, "params."+field)
		}
	}
	if len(paramMissing) > 0 {
		s.writeValidationError(w, "required fields missing: "+strings.Join(paramMissing, ", "), paramMissing)
		return
	}
	// The closed 0-1 domain: out of bounds answers 400 naming the
	// field, nothing written. NaN cannot arrive through
	// encoding/json (a parse error above); the range check is the
	// whole domain.
	params := db.MlModelParams{
		FaceDetectMinScore:     float64(req.Params.FaceDetectMinScore),
		FaceRecognizeThreshold: float64(req.Params.FaceRecognizeThreshold),
	}
	invalid := make([]string, 0, 2)
	if params.FaceDetectMinScore < 0 || params.FaceDetectMinScore > 1 {
		invalid = append(invalid, "params.faceDetectMinScore")
	}
	if params.FaceRecognizeThreshold < 0 || params.FaceRecognizeThreshold > 1 {
		invalid = append(invalid, "params.faceRecognizeThreshold")
	}
	if len(invalid) > 0 {
		s.writeValidationError(w, "params must sit in the closed 0-1 domain", invalid)
		return
	}
	hosts := make([]string, 0, len(req.AllowedHosts))
	for _, host := range req.AllowedHosts {
		normalized, err := normalizeAllowedHost(host)
		if err != nil {
			s.writeValidationError(w,
				"allowedHosts must be hostnames, with an optional leading *. wildcard and an optional port",
				[]string{"allowedHosts"})
			return
		}
		hosts = append(hosts, normalized)
	}
	mirror := req.MirrorBaseUrl
	if mirror != "" {
		normalized, err := normalizeMirrorBase(mirror)
		if err != nil {
			s.writeValidationError(w,
				"mirrorBaseUrl must be a host, or an absolute http(s) URL whose host is kept",
				[]string{"mirrorBaseUrl"})
			return
		}
		mirror = normalized
		// The mirror's host name (port stripped) joins the saved
		// list, so a mirror the list refuses never fails on its own
		// first hop.
		mirrorHost := strings.Split(mirror, ":")[0]
		if !ml.HostAllowed(hosts, mirrorHost) {
			hosts = append(hosts, mirrorHost)
		}
	}
	if err := db.SaveModelSettings(r.Context(), s.queries, db.ModelSettings{
		AutoDownload:  req.AutoDownload,
		AllowedHosts:  hosts,
		MirrorBaseUrl: mirror,
		Params:        params,
	}); err != nil {
		s.internalError(w, r, "save the model settings", err)
		return
	}
	s.writeJSON(w, http.StatusOK, modelSettingsResponse(db.ModelSettings{
		AutoDownload:  req.AutoDownload,
		AllowedHosts:  hosts,
		MirrorBaseUrl: mirror,
		Params:        params,
	}))
}

// normalizeAllowedHost validates one host entry: a hostname with an
// optional leading `*.` wildcard label and an optional port (the port
// is stripped -- the gate matches hostnames, ports never decide).
// Anything else is refused.
func normalizeAllowedHost(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", errors.New("empty host")
	}
	if parsed, err := url.Parse(value); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		// A full URL carries its host; the scheme's presence marks it.
		value = parsed.Host
	}
	if strings.HasPrefix(value, "*.") {
		// The wildcard label is the one star an entry may carry: the
		// rest must be a plain hostname.
		rest := value[2:]
		if rest == "" || strings.ContainsAny(rest, "*?/ ") {
			return "", errors.New("bad wildcard host")
		}
		return strings.ToLower(value), nil
	}
	if strings.ContainsAny(value, "*?/ ") {
		return "", errors.New("bad host")
	}
	if strings.Contains(value, ":") {
		host, _, err := net.SplitHostPort(value)
		if err != nil {
			return "", errors.New("bad host or port")
		}
		value = host
	}
	if value == "" || value == "*" {
		return "", errors.New("bad host")
	}
	return strings.ToLower(value), nil
}

// normalizeMirrorBase validates the mirror base: a bare host (with an
// optional port, which the mirror rewrite uses), or an absolute
// http(s) URL whose host is taken verbatim.
func normalizeMirrorBase(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", errors.New("empty mirror base")
	}
	if parsed, err := url.Parse(value); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return "", errors.New("bad mirror base")
		}
		return strings.ToLower(parsed.Host), nil
	}
	if strings.ContainsAny(value, "/ ") {
		return "", errors.New("bad mirror base")
	}
	return strings.ToLower(value), nil
}

// GetMapConfig implements GET /api/v1/map/config: the map's
// client-side tile sources -- the light style and the optional dark
// one, each null when the instance has none. The read is the browser's
// only source of the URLs; the tiles themselves never pass through the
// server. The dark style follows the same rule: a configured one is
// the dark theme's base layer, an unset one means the dark theme
// reuses the light source.
func (s *server) GetMapConfig(w http.ResponseWriter, r *http.Request) {
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}
	settings, err := db.TileSettingsOf(r.Context(), s.queries)
	if err != nil {
		// The accessor's corrupt-row answer is the unconfigured
		// state: the map states the absence instead of loading a
		// source the row cannot be trusted to name.
		s.logger.Error("read the tile settings; answering unconfigured", "error", err)
	}
	response := apigen.MapConfigResponse{}
	if settings.TileJSONURL != "" {
		tilejsonURL := settings.TileJSONURL
		response.TilejsonUrl = &tilejsonURL
	}
	if settings.DarkTileJSONURL != "" {
		darkTilejsonURL := settings.DarkTileJSONURL
		response.DarkTilejsonUrl = &darkTilejsonURL
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
