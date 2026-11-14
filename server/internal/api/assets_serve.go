// Asset serving handlers: the JSON detail of an owned
// asset, the derived thumbnail (or video poster) as a cacheable
// static asset, and the original file as a download with Range
// support. The video play stream adds the transcoded
// web-safe MP4 served inline with the same static-asset discipline;
// the Live Photo component video extends it -- its playable
// copy arrives through its own transcode job. The detail also
// carries the asset's faces (caller-scoped label state, the
// people section's read).
// The permission model is the caller-visible union:
// every read funnels through the union-gated sqlc query -- the
// caller's own sync and directory arms plus every album shared to
// them -- and a miss answers the anti-enumeration 404,
// byte-identical for unknown and invisible ids.
package api

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	openapi_types "github.com/oapi-codegen/runtime/types"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/db"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/media"
	"github.com/welpher/lumemor/server/internal/storage"
)

// mediaCacheControl is the caching discipline of every media
// response: the bytes are a pure function of the immutable,
// content-addressed original (the derivation is deterministic), so a
// fetched copy stays current for as long as the client keeps it. The
// assets are private to the owner, hence private, not public.
const mediaCacheControl = "private, max-age=31536000, immutable"

// GetAsset implements GET /api/v1/assets/{id}: the asset detail -- the
// identity and storage fields of the upload response plus the
// pipeline's refinements (dimensions, duration, blurhash, the EXIF hot
// fields, readiness), the asset's faces with the caller's label
// state of each face's cluster (the people section's read), the
// caller-visible albums holding the asset and the resolved place
// (the info panel's two deep-link rows).
func (s *server) GetAsset(w http.ResponseWriter, r *http.Request, id openapi_types.UUID, params apigen.GetAssetParams) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	// The locale parameter: the contract's enum is the instance's
	// language set, and the generated binder binds the raw string, so
	// the membership check lives here -- an out-of-set locale is a
	// client bug the 400 names (the places face's own check).
	locale := apigen.GetAssetParamsLocaleEn
	if params.Locale != nil {
		if !params.Locale.Valid() {
			s.writeValidationError(w, "locale must be one of: en, zh-Hans", []string{"locale"})
			return
		}
		locale = *params.Locale
	}

	// The caller-visible union in the query is the permission check
	//: the owner's own arms -- their sync album holds
	// every upload, whatever the row's state -- plus every album
	// shared to the caller. An id none of those cover resolves to no
	// row, exactly like an unknown one, and the answer below is the
	// byte-identical 404 of the anti-enumeration discipline.
	row, err := s.queries.GetAssetDetail(r.Context(), dbgen.GetAssetDetailParams{
		ID:     pgUUID(id),
		Viewer: pgUUID(requester.userID),
		Locale: string(locale),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		s.writeAssetNotFound(w)
		return
	}
	if err != nil {
		s.internalError(w, r, "load asset detail", err)
		return
	}

	// The faces and the caller's tags ride on the detail's own
	// permission: the row above proved the caller can see the asset,
	// so its faces (each joined with the caller's label state) and
	// its tags (the caller's own rows only) need no gate of
	// their own.
	faces, err := s.assetFacesForCaller(r.Context(), pgUUID(id), pgUUID(requester.userID))
	if err != nil {
		s.internalError(w, r, "load asset faces", err)
		return
	}
	tags, err := s.assetTagsForCaller(r.Context(), pgUUID(id), pgUUID(requester.userID))
	if err != nil {
		s.internalError(w, r, "load asset tags", err)
		return
	}
	// The albums face rides the same permission: the visible
	// membership rows only, never an album the caller cannot see.
	albums, err := s.assetAlbumsForCaller(r.Context(), pgUUID(id), pgUUID(requester.userID))
	if err != nil {
		s.internalError(w, r, "load asset albums", err)
		return
	}

	s.writeJSON(w, http.StatusOK, assetDetailFrom(row, faces, tags, albums))
}

// GetAssetThumbnail implements GET /api/v1/assets/{id}/thumbnail: the
// static serving copy of the asset -- the AVIF of the requested size
// for an image, the poster frame for a video -- with the immutable
// caching and ETag discipline of a derived static asset.
func (s *server) GetAssetThumbnail(w http.ResponseWriter, r *http.Request, id openapi_types.UUID, params apigen.GetAssetThumbnailParams) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}
	// The generated binder binds the raw string into the enum type
	// without checking membership, so the membership check lives here:
	// one of the three serving sizes or the 400 of the contract.
	if !params.Size.Valid() {
		s.writeValidationError(w, "invalid thumbnail size", []string{"size"})
		return
	}

	row, err := s.queries.GetVisibleAsset(r.Context(), dbgen.GetVisibleAssetParams{
		ID:     pgUUID(id),
		Viewer: pgUUID(requester.userID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		s.writeAssetNotFound(w)
		return
	}
	if err != nil {
		s.internalError(w, r, "load asset for thumbnail", err)
		return
	}

	// The serving file of this request: an image serves the
	// derivative of the requested size at the instance's configured
	// encoding; a video serves its poster frame (a JPEG) whatever
	// the size says -- the poster is the video's thumbnail, while
	// the playable stream is the transcoded copy the play endpoint
	// serves.
	path, contentType := s.derivedMediaPath(r.Context(), row, string(params.Size))
	file, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		// The asset exists but the derived file does not: the
		// thumbnail job has not produced it yet (or failed and is
		// retrying). ASSET_NOT_READY, not ASSET_NOT_FOUND -- the
		// client renders a placeholder and waits for asset.ready
		// instead of treating the asset as gone.
		s.writeError(w, http.StatusNotFound, apigen.ASSETNOTREADY,
			"asset media is not ready yet", nil)
		return
	}
	if err != nil {
		s.internalError(w, r, "open derived media", err)
		return
	}
	defer func() { _ = file.Close() }()

	s.serveMediaFile(w, r, file, row.Checksum, contentType, time.Time{})
}

// GetAssetOriginal implements GET /api/v1/assets/{id}/original: the
// stored original bytes as a download, with byte ranges for video
// playback and seeking. The original is the uploaded file itself, so
// it is servable immediately after the upload -- readiness gates only
// the derived files.
func (s *server) GetAssetOriginal(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	row, err := s.queries.GetVisibleAsset(r.Context(), dbgen.GetVisibleAssetParams{
		ID:     pgUUID(id),
		Viewer: pgUUID(requester.userID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		s.writeAssetNotFound(w)
		return
	}
	if err != nil {
		s.internalError(w, r, "load asset for original", err)
		return
	}

	// The stored file_path names where the bytes live: a blob row's
	// portable relative form resolved against the data root (the
	// originals-tree guard refusing anything that would leave it),
	// or an external row's real absolute path (the directory album's
	// source tree) -- the one kind-aware resolver every reader of
	// file_path goes through.
	path, err := storage.ResolveAssetPath(s.dataDir, row.FilePath, string(row.StorageKind))
	if err != nil {
		s.internalError(w, r, "resolve original file", err)
		return
	}
	file, err := os.Open(path)
	if err != nil {
		// A row whose blob is gone is a server-side storage
		// inconsistency, not a state the client can retry into: 500,
		// logged, never a 404 (the asset does exist).
		s.internalError(w, r, "open original file", err)
		return
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		s.internalError(w, r, "stat original file", err)
		return
	}

	// The download is always an attachment with the RFC 5987 escaped
	// original file name: no byte of the client-side name reaches the
	// header unescaped, so nothing can inject parameters and no
	// browser renders the file inline (SVG is not in the upload
	// whitelist either, so the rule covers every stored format).
	w.Header().Set("Content-Disposition",
		"attachment; filename*=UTF-8''"+escapeExtFilename(row.OriginalFilename))

	// The real modtime rides along: ServeContent then also answers
	// If-Modified-Since, and the content-addressed blob never changes
	// under its name.
	s.serveMediaFile(w, r, file, row.Checksum, row.MimeType, info.ModTime())
}

// PlayAsset implements GET /api/v1/assets/{id}/play: the transcoded
// playable copy of a video asset -- a standalone video, or a Live
// Photo's component video (the component counts a transcode of
// its own, so the pair's motion plays through this endpoint by the
// component's id) -- streamed inline for playback (the client's
// video element), with byte ranges for seeking. The stream is a
// pipeline product, so it exists only once the transcode job
// produced it -- unlike the original, which is servable immediately.
func (s *server) PlayAsset(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	row, err := s.queries.GetVisibleAsset(r.Context(), dbgen.GetVisibleAssetParams{
		ID:     pgUUID(id),
		Viewer: pgUUID(requester.userID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		s.writeAssetNotFound(w)
		return
	}
	if err != nil {
		s.internalError(w, r, "load asset for play", err)
		return
	}

	// A playable stream exists only for the video family. An image
	// has none -- it answers the asset's own anti-enumeration 404, the
	// same envelope, byte for byte, as an unknown or a foreign id,
	// rather than a distinct error that would leak the kind's shape
	// for ids the caller may not even own. A Live Photo's component
	// video row DOES have one (the component gate that 404ed it was
	// reversed): the component counts a transcode job of its own, so its
	// playable copy arrives exactly like any video's and the pair's
	// motion plays through this endpoint by the component's id --
	// HEVC components re-encoded to the web-safe pair every browser
	// plays. (The component's original also stays downloadable by its
	// id through /original.)
	if row.MediaKind != dbgen.MediaKindVideo {
		s.writeAssetNotFound(w)
		return
	}

	// The transcoded copy the pipeline left behind
	// (storage/encoded/<assetID>/play.mp4). A miss is the not-ready
	// state of every derived file: the transcode job has not produced
	// it yet (or failed and is retrying), so the client renders the
	// poster (a standalone video) or keeps the pair's still photo (a
	// Live Photo component, whose readiness the photo announces on
	// its own schedule) and retries instead of treating the asset as
	// gone.
	file, err := os.Open(storage.EncodedPath(uuid.UUID(row.ID.Bytes).String()))
	if errors.Is(err, fs.ErrNotExist) {
		s.writeError(w, http.StatusNotFound, apigen.ASSETNOTREADY,
			"asset media is not ready yet", nil)
		return
	}
	if err != nil {
		s.internalError(w, r, "open transcoded media", err)
		return
	}
	defer func() { _ = file.Close() }()

	// Inline playback, deliberately NOT a download: no
	// Content-Disposition (the video element consumes the stream
	// directly), and the fixed video/mp4 content type -- the
	// transcoded container is an MP4 by construction, whatever the
	// original's stored MIME type was. The original download stays
	// on /original. The zero modtime keeps the ETag the single
	// freshness authority, like every derived file.
	s.serveMediaFile(w, r, file, row.Checksum, "video/mp4", time.Time{})
}

// serveMediaFile writes one media file to the response through
// http.ServeContent with the shared static-asset discipline: the
// strong checksum-prefix ETag, the immutable private cache policy,
// and the caller's content type (ServeContent only sniffs when the
// header is unset). ServeContent supplies the conditional and range
// semantics: If-None-Match against the ETag answers 304 with no body,
// a satisfiable single range answers 206 with Content-Range, multiple
// ranges a multipart/byteranges body, an unsatisfiable range 416, and
// a malformed Range header is ignored (200, full body). A zero
// modtime suppresses Last-Modified and If-Modified-Since -- the ETag
// is the single freshness authority of derived files, which a
// thumbnail re-run rewrites with a new mtime but the same bytes.
func (s *server) serveMediaFile(w http.ResponseWriter, r *http.Request, file *os.File, checksum, contentType string, modtime time.Time) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("ETag", assetETag(checksum))
	w.Header().Set("Cache-Control", mediaCacheControl)
	http.ServeContent(w, r, "", modtime, file)
}

// assetETag builds the strong entity tag of an asset's media: the
// first 16 hex characters of the checksum, quoted. The checksum
// addresses the immutable source bytes and the derivation is
// deterministic, so the tag is stable across re-runs of the pipeline;
// the prefix keeps the header short (the full digest buys nothing:
// two derivations of the same bytes are the same entity).
func assetETag(checksum string) string {
	if len(checksum) > 16 {
		checksum = checksum[:16]
	}
	return `"` + checksum + `"`
}

// derivativeContentType maps a derivative encoding onto its content
// type: the extension IS the format, and both maps are closed over
// the two encodings the pipeline writes.
func derivativeContentType(ext string) string {
	if ext == media.FormatWebp {
		return "image/webp"
	}
	return "image/avif"
}

// derivedMediaPath resolves the serving file of a thumbnail request:
// an image asset serves the derivative of the requested size at the
// instance's configured encoding -- falling back to the shipped AVIF
// when the configured encoding's file is absent (a library that
// pre-dates an encoding change stays servable until a rebuild
// rewrites it) -- and a video asset serves its poster frame (a JPEG)
// whatever the requested size is. A corrupt image settings row
// answers the defaults with the corruption logged.
func (s *server) derivedMediaPath(ctx context.Context, row dbgen.Asset, size string) (path string, contentType string) {
	assetID := uuid.UUID(row.ID.Bytes).String()
	if row.MediaKind == dbgen.MediaKindVideo {
		return storage.PosterPath(assetID), "image/jpeg"
	}
	settings, err := db.ImageSettingsOf(ctx, s.queries)
	if err != nil {
		s.logger.Error("resolve a thumbnail's encoding; answering the default", "error", err)
	}
	path = storage.ThumbPath(assetID, size, settings.Format)
	if _, err := os.Stat(path); err != nil && settings.Format != media.FormatAvif {
		path = storage.ThumbPath(assetID, size, media.FormatAvif)
		return path, derivativeContentType(media.FormatAvif)
	}
	return path, derivativeContentType(settings.Format)
}

// escapeExtFilename escapes a file name for the filename* parameter
// of Content-Disposition per RFC 5987: every UTF-8 byte outside the
// attribute character set (ASCII letters, digits and
// "!#$&+-.^_`|~") is percent-encoded as uppercase %XX. Non-ASCII
// names therefore survive every transport, and no byte of the
// client-supplied name -- quotes, semicolons, CR, LF -- can break out
// of the parameter or the header.
func escapeExtFilename(name string) string {
	var escaped strings.Builder
	escaped.Grow(len(name))
	for i := 0; i < len(name); i++ {
		if isAttrChar(name[i]) {
			escaped.WriteByte(name[i])
		} else {
			fmt.Fprintf(&escaped, "%%%02X", name[i])
		}
	}
	return escaped.String()
}

// isAttrChar reports whether the byte is an RFC 5987 attribute
// character, the one set that may appear unescaped inside a
// filename* value.
func isAttrChar(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return true
	}
	switch c {
	case '!', '#', '$', '&', '+', '-', '.', '^', '_', '`', '|', '~':
		return true
	}
	return false
}

// writeAssetNotFound answers the anti-enumeration 404 shared by every
// asset endpoint: an unknown id and a foreign id produce the same
// status, code and message, byte for byte, so asset ids cannot be
// probed through any of them.
func (s *server) writeAssetNotFound(w http.ResponseWriter) {
	s.writeError(w, http.StatusNotFound, apigen.ASSETNOTFOUND,
		"no asset with this identifier", nil)
}

// assetFacesForCaller reads one asset's faces joined with the CALLER's
// label state of each face's cluster -- the read both detail-serving
// endpoints (GET and PATCH) share, so the PATCH response carries the
// faces exactly as the GET serves them. The caller scoping is the
// query's LEFT JOIN: another account's label of the same global
// cluster never surfaces. Callers must have proven the asset
// caller-visible first; the faces carry no gate of their own.
func (s *server) assetFacesForCaller(ctx context.Context, assetID, callerID pgtype.UUID) ([]apigen.AssetFaceSummary, error) {
	rows, err := s.queries.ListAssetFacesForCaller(ctx, dbgen.ListAssetFacesForCallerParams{
		UserID:  callerID,
		AssetID: assetID,
	})
	if err != nil {
		return nil, err
	}
	// A non-nil empty slice: the fixed-shape contract wants an array,
	// never JSON null, for a faceless asset.
	faces := make([]apigen.AssetFaceSummary, 0, len(rows))
	for _, row := range rows {
		faceID := uuid.UUID(row.ID.Bytes)
		face := apigen.AssetFaceSummary{
			Id:        faceID,
			Score:     row.Score,
			ClusterId: uuidValue(row.ClusterID),
			Thumbnail: faceThumbnailPath(faceID),
			Label:     nil,
		}
		if row.LabelName.Valid {
			face.Label = &apigen.PersonLabel{Name: row.LabelName.String}
		}
		// The person entity's birth date rides every face: the chip
		// computes the capture-time age from it (the same value for
		// every account -- it is the person's, not the caller's).
		if row.BirthDate.Valid {
			birthDate := openapi_types.Date{Time: row.BirthDate.Time.UTC()}
			face.BirthDate = &birthDate
		}
		faces = append(faces, face)
	}
	return faces, nil
}

// assetAlbumsForCaller reads the albums holding one asset as the
// CALLER sees them -- the info panel's where-does-this-live rows.
// The read both detail-serving endpoints (GET and PATCH) share, so
// the PATCH response carries the albums exactly as the GET serves
// them. Callers must have proven the asset caller-visible first;
// the array carries no gate of its own.
func (s *server) assetAlbumsForCaller(ctx context.Context, assetID, callerID pgtype.UUID) ([]apigen.AssetAlbumRef, error) {
	rows, err := s.queries.ListAssetAlbumsForCaller(ctx, dbgen.ListAssetAlbumsForCallerParams{
		Viewer:  callerID,
		AssetID: assetID,
	})
	if err != nil {
		return nil, err
	}
	// A non-nil empty slice: the fixed-shape contract wants an array,
	// never JSON null, for an asset no visible album holds.
	albums := make([]apigen.AssetAlbumRef, 0, len(rows))
	for _, row := range rows {
		role := apigen.AssetAlbumRefCallerRoleViewer
		if row.IsOwner {
			role = apigen.AssetAlbumRefCallerRoleOwner
		}
		albums = append(albums, apigen.AssetAlbumRef{
			Id:         uuid.UUID(row.ID.Bytes),
			Title:      row.Title,
			Type:       apigen.AssetAlbumRefType(row.Type),
			CallerRole: role,
		})
	}
	return albums, nil
}

// assetDetailFrom maps the detail read onto the API shape. The
// pipeline-dependent values cross as null while the jobs have not
// produced them: width/height stay 0 in the row until the metadata
// job refines them (a real image is never 0 pixels wide), duration
// and blurhash are nullable columns, and the exif hot fields are null
// until the sidecar carries the extraction. The two owner-level
// editable attributes (description, isHidden) cross as the row
// carries them -- the PATCH response reuses this mapper, so the
// applied values surface exactly as the GET endpoint serves them,
// faces, tags, albums and place included (the caller passes the same
// reads' answers). The place face rides the read's own locale join:
// a null place_id (no GPS, no baked data) answers null.
func assetDetailFrom(row dbgen.GetAssetDetailRow, faces []apigen.AssetFaceSummary, tags []apigen.AssetTagSummary, albums []apigen.AssetAlbumRef) apigen.AssetDetail {
	detail := apigen.AssetDetail{
		Id:               uuid.UUID(row.ID.Bytes),
		OwnerId:          uuid.UUID(row.OwnerID.Bytes),
		Kind:             apigen.AssetDetailKind(row.MediaKind),
		OriginalFilename: row.OriginalFilename,
		FileSize:         row.FileSize,
		MimeType:         row.MimeType,
		Checksum:         row.Checksum,
		CreatedAt:        row.CreatedAt.Time.UTC(),
		TakenAt:          row.TakenAt.Time.UTC(),
		TakenAtSource:    apigen.AssetDetailTakenAtSource(row.TakenAtSource),
		IsTrashed:        row.DeletedAt.Valid,
		IsHidden:         row.IsHidden,
		Description:      row.Description,
		IsReady:          row.ReadyAt.Valid,
		Width:            dimensionValue(row.Width),
		Height:           dimensionValue(row.Height),
		DurationMs:       int4Value(row.DurationMs),
		Blurhash:         textValue(row.Blurhash),
		LivePhotoVideoId: uuidValue(row.LivePhotoVideoID),
		Exif:             exifSummaryFrom(row),
		Faces:            faces,
		Tags:             tags,
		Albums:           albums,
	}
	if row.PlaceID.Valid {
		// The locale join's exact discipline (the `GET /places`
		// read's own): the locale's place_names row when the place
		// data carries one, the canonical name otherwise. The
		// canonical column is NOT NULL on a place row, so a valid
		// place_id always answers a name.
		name := row.CanonicalName.String
		if row.LocaleName.Valid {
			name = row.LocaleName.String
		}
		detail.Place = &apigen.AssetPlaceRef{
			PlaceId: row.PlaceID.Int32,
			Name:    name,
		}
	}
	return detail
}

// exifSummaryFrom flattens the joined exif hot fields onto the API
// shape; the join's NULLs stay null.
func exifSummaryFrom(row dbgen.GetAssetDetailRow) apigen.ExifSummary {
	return apigen.ExifSummary{
		CameraMake:  textValue(row.CameraMake),
		CameraModel: textValue(row.CameraModel),
		Lens:        textValue(row.Lens),
		Iso:         int4Value(row.Iso),
		Aperture:    float4Value(row.Aperture),
		Shutter:     textValue(row.Shutter),
		Focal:       float4Value(row.Focal),
		Lat:         float8Value(row.Lat),
		Lon:         float8Value(row.Lon),
		Alt:         float4Value(row.Alt),
		TzOffset:    textValue(row.TzOffset),
	}
}

// dimensionValue maps a NOT NULL dimension column whose 0 means
// "not refined yet" (the upload's placeholder) onto the nullable API
// field.
func dimensionValue(v int32) *int {
	if v == 0 {
		return nil
	}
	value := int(v)
	return &value
}

// int4Value maps a nullable int column onto the nullable API field.
func int4Value(v pgtype.Int4) *int {
	if !v.Valid {
		return nil
	}
	value := int(v.Int32)
	return &value
}

// float4Value maps a nullable real column onto the nullable API field.
func float4Value(v pgtype.Float4) *float32 {
	if !v.Valid {
		return nil
	}
	return &v.Float32
}

// float8Value maps a nullable double column onto the nullable API field.
func float8Value(v pgtype.Float8) *float64 {
	if !v.Valid {
		return nil
	}
	return &v.Float64
}

// textValue maps a nullable text column onto the nullable API field.
func textValue(v pgtype.Text) *string {
	if !v.Valid {
		return nil
	}
	return &v.String
}

// uuidValue maps a nullable uuid column onto the nullable API field.
func uuidValue(v pgtype.UUID) *openapi_types.UUID {
	if !v.Valid {
		return nil
	}
	value := openapi_types.UUID(v.Bytes)
	return &value
}
