// The face crop thumbnail endpoint: the people surface's
// pixels. GET /faces/{faceId}/thumbnail crops one face out of its
// asset's view derivative at query time -- the bounding box the face
// row carries is in per-mille units that address the derivative's own
// pixels, because the derivative preserves the original's framing --
// and answers 256x256 AVIF bytes that are never stored: they are a
// pure function of the derivative and the face row, which is what
// makes the endpoint's immutable caching honest.
//
// The permission gate is the face's asset (the visibility predicate
// every people query applies); each miss answers the surface's single
// byte-identical 404. The derivative's absence folds into that same
// shape deliberately -- a not-yet-derived asset is indistinguishable
// from an invisible one here, so face ids cannot be probed and the
// web renders its skeleton placeholder until the pixels exist (the
// asset thumbnail endpoint keeps its own separate ASSET_NOT_READY
// code; this surface's grid can wait silently instead).
package api

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/welpher/lumemor/server/internal/db"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/media"
	"github.com/welpher/lumemor/server/internal/storage"
)

// GetFaceThumbnail implements GET /api/v1/faces/{faceId}/thumbnail:
// one face's crop, cropped from the asset's view derivative through
// the media package's pinned geometry and served with the
// static-asset discipline of the derived media endpoints -- the
// strong face-signal ETag, the immutable private cache policy, and
// ServeContent's conditional and range semantics (a matching
// If-None-Match answers 304 with no body).
func (s *server) GetFaceThumbnail(w http.ResponseWriter, r *http.Request, faceId uuid.UUID) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	// The gate and the crop's inputs in one read: no row means an
	// unknown face or one on an asset the caller cannot see -- the
	// 404 every people endpoint answers identically.
	face, err := s.queries.GetFaceCropForCaller(r.Context(), dbgen.GetFaceCropForCallerParams{
		FaceID: pgUUID(faceId),
		UserID: pgUUID(requester.userID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		s.writePersonNotFound(w)
		return
	}
	if err != nil {
		s.internalError(w, r, "load face for crop", err)
		return
	}

	// The source pixels: the view derivative, never the original.
	// Its absence is the not-ready state of this endpoint, folded
	// into the same 404 -- the asset exists but the crop cannot be
	// derived yet, and the caller learns nothing either way. The
	// encoding follows the instance's configured one, falling back
	// to the shipped AVIF when the configured file is absent (the
	// same discipline the asset thumbnail serves under).
	assetID := uuid.UUID(face.AssetID.Bytes).String()
	settings, settingsErr := db.ImageSettingsOf(r.Context(), s.queries)
	if settingsErr != nil {
		s.logger.Error("resolve a face crop's source encoding; answering the default", "error", settingsErr)
	}
	viewPath := storage.ThumbPath(assetID, media.SizeView, settings.Format)
	if _, err := os.Stat(viewPath); err != nil && settings.Format != media.FormatAvif {
		viewPath = storage.ThumbPath(assetID, media.SizeView, media.FormatAvif)
	}
	if _, err := os.Stat(viewPath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			s.writePersonNotFound(w)
			return
		}
		s.internalError(w, r, "stat view derivative", err)
		return
	}

	crop, err := media.FaceCropThumbnail(viewPath,
		int(face.BboxX), int(face.BboxY), int(face.BboxW), int(face.BboxH))
	if err != nil {
		s.internalError(w, r, "derive face crop", err)
		return
	}

	// The same static-asset discipline as the derived media
	// endpoints, over the in-memory crop: the zero modtime keeps the
	// ETag the single freshness authority (ServeContent then answers
	// If-None-Match with 304 and serves byte ranges off the reader).
	w.Header().Set("Content-Type", "image/avif")
	w.Header().Set("ETag", faceCropETag(faceId, face.UpdatedAt.Time))
	w.Header().Set("Cache-Control", mediaCacheControl)
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(crop))
}

// faceCropETag builds the strong entity tag of a face crop: the face
// id with the face row's updated_at in unix seconds. updated_at is
// the row's change signal -- a cluster move bumps it (the pixels do
// not change, but a re-render under a new tag is harmless), and
// re-detection replaces the face row outright, which changes the id
// and with it the URL. The derivation itself is deterministic, so
// one signal always renders the same bytes.
func faceCropETag(faceID uuid.UUID, updatedAt time.Time) string {
	return fmt.Sprintf(`"face-%s-%d"`, faceID, updatedAt.Unix())
}
