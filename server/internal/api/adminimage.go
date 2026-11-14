// The derivative settings surface: the image card's settings pair
// (the sizes and encoding every NEW derivation runs at) and, later in
// the file family, the run budget and the rebuild actions. The
// settings pair follows the admin settings shape exactly: one GET
// answering the current document (a corrupt row answers the defaults,
// the corruption logged), one PUT that requires every field and
// validates each against its enum before the whole-object write.
//
// THE COST STATEMENT IS STRUCTURAL: neither handler starts a
// derivation. The save writes the document; the serving copies
// already on disk stay as they are until the explicit rebuild run
// re-derives them.
package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/db"
)

// GetImageSettings implements GET /api/v1/admin/settings/image: the
// instance's derivative image settings, as the next derivation will
// use them.
func (s *server) GetImageSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}
	settings, err := db.ImageSettingsOf(r.Context(), s.queries)
	if err != nil {
		s.logger.Error("read the image settings; answering the defaults", "error", err)
	}
	s.writeJSON(w, http.StatusOK, imageSettingsResponse(settings))
}

// imageSettingsResponse maps the stored document onto the contract
// shape.
func imageSettingsResponse(settings db.ImageSettings) apigen.ImageSettingsResponse {
	return apigen.ImageSettingsResponse{
		TileSize:    apigen.ImageSettingsResponseTileSize(settings.TileSize),
		PreviewSize: apigen.ImageSettingsResponsePreviewSize(settings.PreviewSize),
		Format:      apigen.DerivativeFormat(settings.Format),
		FullSize:    settings.FullSize,
		RawEmbedded: settings.RawEmbedded,
	}
}

// UpdateImageSettings implements PUT /api/v1/admin/settings/image:
// every field is contract-required (an omitted one would decode as
// its zero value and silently change the posture), and each value is
// checked against its enum. The save writes the document and nothing
// else: no derivation starts, no existing serving file is touched,
// and a rebuild already running keeps its own starting parameters.
// The values bind at the next derivation -- the thumbnail job reads
// the settings at run time, so no restart is needed.
func (s *server) UpdateImageSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeValidationError(w, "request body could not be read", []string{"body"})
		return
	}
	var req apigen.ImageSettingsUpdateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		s.writeValidationError(w, "request body must be valid JSON", []string{"body"})
		return
	}
	var presence map[string]json.RawMessage
	_ = json.Unmarshal(body, &presence)
	missing := make([]string, 0, 5)
	for _, field := range []string{"tileSize", "previewSize", "format", "fullSize", "rawEmbedded"} {
		if _, ok := presence[field]; !ok {
			missing = append(missing, field)
		}
	}
	if len(missing) > 0 {
		s.writeValidationError(w, "required fields missing: "+strings.Join(missing, ", "), missing)
		return
	}
	if !db.ValidImageTileSize(int(req.TileSize)) {
		s.writeValidationError(w, "tileSize must be one of 250, 320", []string{"tileSize"})
		return
	}
	if !db.ValidImagePreviewSize(int(req.PreviewSize)) {
		s.writeValidationError(w, "previewSize must be one of 1080, 1440, 2048", []string{"previewSize"})
		return
	}
	if req.Format != apigen.Avif && req.Format != apigen.Webp {
		s.writeValidationError(w, "format must be one of avif, webp", []string{"format"})
		return
	}
	settings := db.ImageSettings{
		TileSize:    int(req.TileSize),
		PreviewSize: int(req.PreviewSize),
		Format:      string(req.Format),
		FullSize:    req.FullSize,
		RawEmbedded: req.RawEmbedded,
	}
	if err := db.SaveImageSettings(r.Context(), s.queries, settings); err != nil {
		s.internalError(w, r, "save the image settings", err)
		return
	}
	s.writeJSON(w, http.StatusOK, imageSettingsResponse(settings))
}
