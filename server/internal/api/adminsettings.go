// The admin settings handlers (the storage-template feature and the
// settings feature): the storage template's read and three-stage
// validated save of GET/PUT /admin/settings/storage, the trash
// retention's GET/PUT /admin/settings/trash, and the video
// transcoding profile's GET/PUT /admin/settings/transcoding. The
// permission model is the admin role itself: every route of this
// surface sits behind requireSession AND requireAdmin (see the
// router), so a signed-in caller without the role answers 403
// AUTH_FORBIDDEN before any handler runs.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/db"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/storage"
)

// storageSampleOwner is the fixed owner the sample paths render
// under: the preview answers a concrete stored relative path, exactly
// the placement a real upload of the sample file would land at.
const storageSampleOwner = "0192f8c1-0000-7000-8000-000000000001"

// storageSampleAssets is the fixed sample asset set every saved
// template must render: an album-less image (the common backup case,
// and the one that catches album-only templates rendering empty), an
// album-grouped image with its date bounds (the album code path), and
// a video (the VID family). Sample one doubles as the preview asset.
func storageSampleAssets() []storage.RenderVars {
	image := storage.RenderVars{
		Filename:  "IMG_1234.jpg",
		Ext:       "jpg",
		TakenAt:   time.Date(2023, 7, 14, 10, 15, 30, 123000000, time.FixedZone("", 5*3600+30*60)),
		Make:      "Acme",
		Model:     "Alpha 5",
		LensModel: "24-70mm f/2.8",
		AssetID:   "0192f8a1-0b1c-2d3e-4f5a-6b7c8d9e0f10",
	}
	return []storage.RenderVars{
		image,
		{
			Filename:   "beach.jpg",
			Ext:        "jpg",
			TakenAt:    time.Date(2023, 8, 1, 9, 0, 0, 0, time.FixedZone("", 2*3600)),
			Album:      "Summer trip",
			AlbumStart: time.Date(2023, 6, 30, 0, 0, 0, 0, time.UTC),
			AlbumEnd:   time.Date(2023, 8, 15, 0, 0, 0, 0, time.UTC),
			AssetID:    "0192f8a2-1c2d-3e4f-5a6b-7c8d9e0f1121",
		},
		{
			Filename: "VID_0001.mov",
			Ext:      "mov",
			Video:    true,
			TakenAt:  time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC),
			AssetID:  "0192f8a3-2d3e-4f5a-6b7c-8d9e0f112233",
		},
	}
}

// GetStorageSettings implements GET /api/v1/admin/settings/storage:
// the instance's storage template, its sample render and the built-in
// presets. A corrupt setting row answers the default template with
// the corruption logged server-side -- the read never fails over a
// row the save endpoint's validation should have kept honest.
func (s *server) GetStorageSettings(w http.ResponseWriter, r *http.Request) {
	template, err := db.StorageTemplate(r.Context(), s.queries)
	if err != nil {
		s.logger.Error("read the storage template setting; answering the default", "error", err)
	}
	response, err := s.storageSettingsResponse(template)
	if err != nil {
		// Only a template that cannot render its own sample reaches
		// here, and the accessor's shape checks plus the save's
		// validation guard make that a corrupted-row state.
		s.internalError(w, r, "render storage settings sample", err)
		return
	}
	s.writeJSON(w, http.StatusOK, response)
}

// UpdateStorageSettings implements PUT /api/v1/admin/settings/storage:
// the three-stage validated save. A template that fails any stage
// answers 400 VALIDATION_FAILED with the specific reason in the
// message and nothing written; a template that passes is stored and
// answers the saved state.
func (s *server) UpdateStorageSettings(w http.ResponseWriter, r *http.Request) {
	var req apigen.StorageSettingsUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		message := "request body must be valid JSON"
		var fields []interface{}
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			message = "request body exceeds the size limit"
			fields = []interface{}{"body"}
		}
		var details *map[string]interface{}
		if fields != nil {
			details = &map[string]interface{}{"fields": fields}
		}
		s.writeError(w, http.StatusBadRequest, apigen.VALIDATIONFAILED, message, details)
		return
	}
	if req.Template == "" {
		s.writeValidationError(w, "template is required", []string{"template"})
		return
	}

	// Stage one: the strict compile (unknown variables, malformed
	// conditionals, control runes, the length bound). Stage two and
	// three: the sample render over the fixed asset set, whose
	// post-pass owns the structural refusals (traversal, depth,
	// length, the empty render).
	if err := validateStorageTemplate(req.Template); err != nil {
		s.writeValidationError(w, err.Error(), []string{"template"})
		return
	}

	if err := s.queries.SetInstanceSetting(r.Context(), dbgen.SetInstanceSettingParams{
		Key:   db.StorageTemplateKey,
		Value: req.Template,
	}); err != nil {
		s.internalError(w, r, "save storage template", err)
		return
	}

	response, err := s.storageSettingsResponse(req.Template)
	if err != nil {
		// The template just passed the same render; this is not a
		// client state worth answering specifically.
		s.internalError(w, r, "render storage settings sample", err)
		return
	}
	s.writeJSON(w, http.StatusOK, response)
}

// validateStorageTemplate runs the save's validation pipeline over
// one candidate template: the strict compile, then the render over
// every sample asset (the structural post-pass runs inside every
// render). A nil error means the template is safe to store.
func validateStorageTemplate(template string) error {
	compiled, err := storage.CompileTemplate(template)
	if err != nil {
		return err
	}
	for _, vars := range storageSampleAssets() {
		if _, err := compiled.Render(vars); err != nil {
			return err
		}
	}
	return nil
}

// storageSettingsResponse builds the settings response of one
// template: the value itself, the first sample asset's rendered
// placement (the preview line), and the built-in presets.
func (s *server) storageSettingsResponse(template string) (apigen.StorageSettingsResponse, error) {
	compiled, err := storage.CompileTemplate(template)
	if err != nil {
		return apigen.StorageSettingsResponse{}, fmt.Errorf("compile template %q: %w", template, err)
	}
	sample := storageSampleAssets()[0]
	rendered, err := compiled.Render(sample)
	if err != nil {
		return apigen.StorageSettingsResponse{}, fmt.Errorf("render the sample of template %q: %w", template, err)
	}
	presets := make([]string, len(storage.TemplatePresets))
	copy(presets, storage.TemplatePresets)
	return apigen.StorageSettingsResponse{
		Template:   template,
		SamplePath: sampleStoredPath(rendered, sample.Ext),
		Presets:    presets,
	}, nil
}

// sampleStoredPath assembles the stored relative path of one sample
// render: the per-owner originals root, the rendered segment, the
// normalized extension -- the same assembly the placement performs.
func sampleStoredPath(rendered, ext string) string {
	return "originals/" + storageSampleOwner + "/" + rendered + "." + storage.NormalizeExt(ext)
}

// GetTrashSettings implements GET /api/v1/admin/settings/trash: the
// instance's trash retention in days. The accessor's corrupt-row
// discipline (default + logged error) keeps the read from failing
// over a row the save endpoint's validation should have kept
// honest.
func (s *server) GetTrashSettings(w http.ResponseWriter, r *http.Request) {
	retention, err := db.TrashRetentionDays(r.Context(), s.queries)
	if err != nil {
		s.logger.Error("read the trash retention setting; answering the default", "error", err)
	}
	s.writeJSON(w, http.StatusOK, apigen.TrashSettingsResponse{RetentionDays: retention})
}

// UpdateTrashSettings implements PUT /api/v1/admin/settings/trash:
// validate the 0-365 bound, persist the retention as a jsonb number
// (the raw writer -- the expiry job and the trash listing unmarshal
// an integer), answer the saved value. The write takes effect for
// the expiry job's NEXT run: the run reads the setting at its start.
func (s *server) UpdateTrashSettings(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeValidationError(w, "request body could not be read", []string{"body"})
		return
	}
	var req apigen.TrashSettingsUpdateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		message := "request body must be valid JSON"
		var fields []interface{}
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			message = "request body exceeds the size limit"
			fields = []interface{}{"body"}
		}
		var details *map[string]interface{}
		if fields != nil {
			details = &map[string]interface{}{"fields": fields}
		}
		s.writeError(w, http.StatusBadRequest, apigen.VALIDATIONFAILED, message, details)
		return
	}
	// The field is contract-required: an omitted retentionDays decodes
	// to 0 (a VALID value meaning immediate permanent delete) — the
	// presence check guards the destructive silent default.
	var presence map[string]json.RawMessage
	_ = json.Unmarshal(body, &presence)
	if _, ok := presence["retentionDays"]; !ok {
		s.writeValidationError(w, "retentionDays is required", []string{"retentionDays"})
		return
	}
	if req.RetentionDays < 0 || req.RetentionDays > db.TrashRetentionMax {
		s.writeValidationError(w, "retentionDays must be between 0 and 365", []string{"retentionDays"})
		return
	}
	if err := db.SaveTrashRetentionDays(r.Context(), s.queries, req.RetentionDays); err != nil {
		s.internalError(w, r, "save the trash retention", err)
		return
	}
	s.writeJSON(w, http.StatusOK, apigen.TrashSettingsResponse(req))
}

// GetTranscodingSettings implements GET /api/v1/admin/settings/transcoding:
// the instance's video transcoding profile. The accessor's
// corrupt-row discipline (per-field default + logged error) keeps
// the read honest over a broken row.
func (s *server) GetTranscodingSettings(w http.ResponseWriter, r *http.Request) {
	profile, err := db.TranscodingSettings(r.Context(), s.queries)
	if err != nil {
		s.logger.Error("read the transcoding settings; answering the default", "error", err)
	}
	s.writeJSON(w, http.StatusOK, transcodingSettingsResponse(profile))
}

// UpdateTranscodingSettings implements PUT
// /api/v1/admin/settings/transcoding: validate every field against
// its enum, persist the profile as one JSON object, answer the
// saved value. The profile affects NEW ingests only -- see the
// contract's scope pin; videos already transcoded are never
// re-encoded.
func (s *server) UpdateTranscodingSettings(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeValidationError(w, "request body could not be read", []string{"body"})
		return
	}
	var req apigen.TranscodingSettingsUpdateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		message := "request body must be valid JSON"
		var fields []interface{}
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			message = "request body exceeds the size limit"
			fields = []interface{}{"body"}
		}
		var details *map[string]interface{}
		if fields != nil {
			details = &map[string]interface{}{"fields": fields}
		}
		s.writeError(w, http.StatusBadRequest, apigen.VALIDATIONFAILED, message, details)
		return
	}
	// All four fields are contract-required: the generated types use
	// zero values for absent fields (false for the booleans, the
	// zero-enum for the preset), silently writing a destructive or
	// nonsensical profile.
	var presence map[string]json.RawMessage
	_ = json.Unmarshal(body, &presence)
	var missing []string
	for _, field := range []string{"preset", "concurrency", "thumbnailFrame", "keepOriginal"} {
		if _, ok := presence[field]; !ok {
			missing = append(missing, field)
		}
	}
	if len(missing) > 0 {
		s.writeValidationError(w, fmt.Sprintf("required fields missing: %s", strings.Join(missing, ", ")), missing)
		return
	}
	var invalid []string
	if !db.ValidTranscodingPreset(string(req.Preset)) {
		invalid = append(invalid, "preset")
	}
	if !db.ValidTranscodingConcurrency(int(req.Concurrency)) {
		invalid = append(invalid, "concurrency")
	}
	if len(invalid) > 0 {
		s.writeValidationError(w, "invalid transcoding settings", invalid)
		return
	}
	profile := db.TranscodingProfile{
		Preset:         string(req.Preset),
		Concurrency:    int(req.Concurrency),
		ThumbnailFrame: req.ThumbnailFrame,
		KeepOriginal:   req.KeepOriginal,
	}
	if err := db.SaveTranscodingSettings(r.Context(), s.queries, profile); err != nil {
		s.internalError(w, r, "save the transcoding settings", err)
		return
	}
	s.writeJSON(w, http.StatusOK, transcodingSettingsResponse(profile))
}

// transcodingSettingsResponse maps the stored profile onto the API
// shape (the struct's JSON tags mirror the contract's field names,
// so the mapping is the enum widening alone).
func transcodingSettingsResponse(profile db.TranscodingProfile) apigen.TranscodingSettingsResponse {
	return apigen.TranscodingSettingsResponse{
		Preset:         apigen.TranscodePreset(profile.Preset),
		Concurrency:    apigen.TranscodingSettingsResponseConcurrency(profile.Concurrency),
		ThumbnailFrame: profile.ThumbnailFrame,
		KeepOriginal:   profile.KeepOriginal,
	}
}

// GetMountRootsSettings and UpdateMountRootsSettings once lived
// here: the mount-root allowlist pair of E4's albums-as-sources,
// the directory-album trust boundary's read and wholesale save.
// The E6 boundary ruling retired the layer whole -- the compose
// file's mounts ARE the boundary (the operator controls what the
// container sees; the application layer neither can nor needs to
// restrict the operator further), and mounting an existing photo
// directory became the administrator's own act, guarded at the
// creation and completion faces instead of confined by a list. The
// handlers, their contract paths, their settings row accessors and
// their validation all went together -- no half-retired surface
// answers a route the contract no longer names.
