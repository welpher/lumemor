// Asset handlers of the upload endpoint (M2b): the multipart upload
// with its idempotent three-state semantics, the Live Photo pairing,
// and the device idempotency lookup.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/storage"
)

const (
	// The multipart part names of the upload contract. The text fields
	// precede the file parts: the handler streams file parts to disk
	// and must know every field before the first file byte is
	// accepted, so the part order is part of the contract.
	uploadFieldDeviceAssetID   = "deviceAssetId"
	uploadFieldFileName        = "fileName"
	uploadFieldDeviceCreatedAt = "deviceCreatedAt"
	uploadFieldDeviceTimeZone  = "deviceTimeZone"
	uploadPartPhoto            = "photo"
	uploadPartLiveVideo        = "liveVideo"

	// maxUploadFieldBytes caps each text field of the multipart body:
	// the fields are small strings, and an unbounded "field" must
	// never become a streaming sink.
	maxUploadFieldBytes = 1 << 10

	// Value bounds of the upload fields, pinned by the contract.
	maxDeviceAssetIDRunes = 256
	maxFileNameRunes      = 512
	maxDeviceTimeZoneRuns = 64
)

// uploadUniquenessConstraints are the unique indexes whose violation
// IS the upload idempotency: the total owner+checksum index (live and
// trashed rows alike) and the owner+device identity. A violation of
// any of them routes the request into the three-state lookup instead
// of the error path.
var uploadUniquenessConstraints = map[string]bool{
	"assets_owner_checksum_uq":             true,
	"assets_owner_device_asset_id_live_uq": true,
}

// uploadForm is the normalized form of the upload's text fields.
type uploadForm struct {
	deviceAssetID      string
	fileName           string
	deviceCreatedAtRaw string
	deviceCreatedAt    time.Time
	deviceTimeZone     string
}

// committedUpload is one file part that reached the content-addressed
// originals tree: everything the database row needs, recorded while
// the part streamed.
type committedUpload struct {
	// partName is the file part's own file name; the component video
	// row keeps it as its original filename.
	partName string
	// relPath is the content-addressed storage path relative to the
	// data root, the portable form stored in file_path.
	relPath string
	// checksum is the sha256 hex digest of the payload.
	checksum string
	// size is the streamed byte count.
	size int64
	// format is the whitelisted format the extension mapped to.
	format storage.UploadFormat
}

// UploadAsset implements POST /api/v1/assets: the multipart backup
// upload with idempotent three-state semantics.
//
// The body streams through in one pass: text fields are read as they
// arrive (they precede the file parts per contract), every file part
// is handed to a fresh Receiver that stages, validates and
// content-addresses it, and the database insert -- one row for the
// photo, plus a linked component row for the Live Photo video --
// decides the outcome through the unique indexes: a clean insert
// answers 201 created; a violation routes into the lookup that
// answers 200 duplicate (the identical asset is live) or 200 trashed
// (it sits in the trash). The handler validates the fields before the
// first Receiver exists, so a request that fails validation reaches
// its terminal state without staging anything; a request rejected
// only after a file part legitimately committed leaves the
// content-addressed blob (which may be shared and must not be
// removed) but never a row.
func (s *server) UploadAsset(w http.ResponseWriter, r *http.Request) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	reader, err := r.MultipartReader()
	if err != nil {
		s.writeValidationError(w, "request body must be multipart/form-data", nil)
		return
	}

	fields := make(map[string]string)
	var photo, liveVideo *committedUpload

	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			s.writeValidationError(w, "malformed multipart body", nil)
			return
		}
		name := part.FormName()

		if uploadFilePartName(name) {
			// Every field must be known and valid before the first
			// file byte is staged: this is the structural gate the
			// receiver's contract rides on.
			if _, invalid := normalizeUploadForm(fields); len(invalid) > 0 {
				s.writeValidationError(w, "invalid upload fields", invalid)
				return
			}
			if part.FileName() == "" {
				s.writeValidationError(w, "file parts must carry a file name", []string{name})
				return
			}
			target := &photo
			if name == uploadPartLiveVideo {
				if photo == nil {
					// The pairing video follows its photo: streaming
					// cannot hold the video until a later photo part
					// arrives, so an early liveVideo is refused before
					// anything is staged.
					s.writeValidationError(w, "liveVideo must follow the photo part",
						[]string{uploadPartLiveVideo})
					return
				}
				target = &liveVideo
			}
			if *target != nil {
				s.writeValidationError(w, "duplicate file part", []string{name})
				return
			}
			committed, handled := s.receiveFilePart(w, r, part)
			if !handled {
				return
			}
			*target = &committed
			continue
		}

		if uploadTextFieldName(name) {
			if photo != nil || liveVideo != nil {
				s.writeValidationError(w, "multipart fields must precede file parts", []string{name})
				return
			}
			if _, seen := fields[name]; seen {
				s.writeValidationError(w, "duplicate multipart field", []string{name})
				return
			}
			value, err := readUploadField(part)
			if err != nil {
				s.writeValidationError(w, "multipart field is malformed or exceeds its size bound", []string{name})
				return
			}
			fields[name] = value
			continue
		}

		s.writeValidationError(w, "unknown multipart part", []string{name})
		return
	}

	// The form validates the same way at end of body: a request of
	// fields only (no photo part) fails here without having staged
	// anything.
	form, invalid := normalizeUploadForm(fields)
	if len(invalid) > 0 {
		s.writeValidationError(w, "invalid upload fields", invalid)
		return
	}
	if photo == nil {
		s.writeValidationError(w, "photo part is required", []string{uploadPartPhoto})
		return
	}
	// A Live Photo whose video is the photo's own bytes is client
	// nonsense: the two rows would collide inside one transaction,
	// so the pair is refused before anything is staged.
	if liveVideo != nil && liveVideo.checksum == photo.checksum {
		s.writeValidationError(w, "the Live Photo video must differ from the photo",
			[]string{uploadPartLiveVideo})
		return
	}

	asset, state, err := s.insertUpload(r, requester.userID, form, photo, liveVideo)
	if err != nil {
		s.internalError(w, r, "insert upload", err)
		return
	}
	if state == apigen.Created {
		s.publishAssetUploaded(asset)
		s.writeJSON(w, http.StatusCreated, assetResponseFrom(asset, state))
		return
	}
	s.logger.Info("upload matched an existing asset",
		"owner_id", requester.userID, "state", state,
		"checksum", photo.checksum, "asset_id", uuid.UUID(asset.ID.Bytes))
	s.writeJSON(w, http.StatusOK, assetResponseFrom(asset, state))
}

// GetAssetByDeviceId implements GET /api/v1/assets/by-device-id/{id}:
// the backup idempotency probe, owner-scoped, answering the asset
// with the state a re-upload of its content would produce.
func (s *server) GetAssetByDeviceId(w http.ResponseWriter, r *http.Request, deviceAssetId string) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	deviceAssetID := strings.TrimSpace(deviceAssetId)
	if deviceAssetID == "" || utf8.RuneCountInString(deviceAssetID) > maxDeviceAssetIDRunes {
		s.writeValidationError(w, "invalid deviceAssetId", []string{"deviceAssetId"})
		return
	}

	row, err := s.queries.GetAssetByDeviceId(r.Context(), dbgen.GetAssetByDeviceIdParams{
		OwnerID:       pgUUID(requester.userID),
		DeviceAssetID: pgtype.Text{String: deviceAssetID, Valid: true},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		s.writeError(w, http.StatusNotFound, apigen.ASSETNOTFOUND,
			"no asset with this device identifier", nil)
		return
	}
	if err != nil {
		s.internalError(w, r, "lookup asset by device id", err)
		return
	}

	s.writeJSON(w, http.StatusOK, assetResponseFrom(row, uploadStateOf(row)))
}

// receiveFilePart streams one file part into the content-addressed
// originals tree through a fresh Receiver. It reports whether the
// request was answered (a rejection writes the response itself), and
// the committed upload on success. The part's own file name carries
// the extension; a payload inconsistent with it, or a file over the
// per-file limit, aborts with the file part named in the details.
func (s *server) receiveFilePart(w http.ResponseWriter, r *http.Request, part *multipart.Part) (committedUpload, bool) {
	ext := normalizeExtension(part.FileName())
	receiver, err := storage.NewReceiver(s.uploadStagingDir(), s.uploadLimit, part)
	if err != nil {
		s.internalError(w, r, "stage upload", err)
		return committedUpload{}, false
	}
	relPath, err := receiver.Commit(ext)
	if err != nil {
		var rejected *storage.ErrRejected
		field := part.FormName()
		details := map[string]interface{}{"fields": []interface{}{field}}
		switch {
		case errors.Is(err, storage.ErrTooLarge):
			s.writeError(w, http.StatusRequestEntityTooLarge, apigen.ASSETTOOLARGE,
				"uploaded file exceeds the instance upload limit", &details)
		case errors.As(err, &rejected):
			s.writeValidationError(w, "uploaded file rejected: "+rejected.Reason, []string{field})
		default:
			s.internalError(w, r, "commit upload", err)
		}
		return committedUpload{}, false
	}

	format, ok := storage.LookupUploadFormat(ext)
	if !ok {
		// Commit has already validated the extension against the
		// whitelist, so this cannot happen; the guard keeps the
		// invariant explicit rather than trusting a faraway table.
		s.internalError(w, r, "resolve upload format",
			fmt.Errorf("extension %q left the whitelist", ext))
		return committedUpload{}, false
	}

	return committedUpload{
		partName: part.FileName(),
		relPath:  relPath,
		checksum: receiver.Checksum(),
		size:     receiver.Size(),
		format:   format,
	}, true
}

// insertUpload persists the upload: the Live Photo video component
// row first, then the photo row referencing it, then the EXIF
// placeholders -- one transaction, all rows or none. The taken-at
// value is the device-reported capture time or, absent one, the
// upload time; the metadata pipeline refines it later. A unique index
// violation is not a failure but the idempotency at work: the
// transaction rolls back and the conflict resolves through the lookup
// into the duplicate or trashed state, carried as a normal outcome.
func (s *server) insertUpload(r *http.Request, owner uuid.UUID, form uploadForm, photo, video *committedUpload) (dbgen.Asset, apigen.AssetResponseState, error) {
	ctx := r.Context()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return dbgen.Asset{}, "", err
	}
	// The rollback is a no-op after a successful commit; the context
	// is captured while the request is still live.
	defer func() { _ = tx.Rollback(ctx) }()
	queries := dbgen.New(tx)

	takenAt := form.deviceCreatedAt
	if takenAt.IsZero() {
		takenAt = time.Now()
	}

	// The Live Photo video is a component row: no device identity of
	// its own, never in the timeline, linked from the photo row.
	var videoID pgtype.UUID
	if video != nil {
		id, err := uuid.NewV7()
		if err != nil {
			return dbgen.Asset{}, "", err
		}
		videoRow, err := queries.CreateAsset(ctx, assetInsertParams(id, owner, form, *video, takenAt, pgtype.UUID{}, true))
		if err != nil {
			// Release the transaction's connection before the lookup
			// resolves the conflict through the pool; the deferred
			// rollback afterwards is a no-op.
			_ = tx.Rollback(ctx)
			return s.resolveUploadConflict(r, owner, form, photo, video, err)
		}
		videoID = videoRow.ID
		if err := queries.CreateExifPlaceholder(ctx, dbgen.CreateExifPlaceholderParams{
			AssetID: videoRow.ID,
			Exif:    uploadExifPlaceholder(form),
		}); err != nil {
			return dbgen.Asset{}, "", err
		}
	}

	photoID, err := uuid.NewV7()
	if err != nil {
		return dbgen.Asset{}, "", err
	}
	photoRow, err := queries.CreateAsset(ctx, assetInsertParams(photoID, owner, form, *photo, takenAt, videoID, false))
	if err != nil {
		// Same as above: the lookup needs a pool connection.
		_ = tx.Rollback(ctx)
		return s.resolveUploadConflict(r, owner, form, photo, video, err)
	}
	if err := queries.CreateExifPlaceholder(ctx, dbgen.CreateExifPlaceholderParams{
		AssetID: photoRow.ID,
		Exif:    uploadExifPlaceholder(form),
	}); err != nil {
		return dbgen.Asset{}, "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return dbgen.Asset{}, "", err
	}
	return photoRow, apigen.Created, nil
}

// resolveUploadConflict turns a failed CreateAsset into the upload
// idempotency outcome. A violation of one of the upload uniqueness
// indexes is resolved through the lookup -- outside the rolled-back
// transaction, straight from the pool, where the row the insert
// collided with is committed and visible: the loser of a concurrent
// insert waits on the winner's index entry, so it observes the winner
// committed or not at all. The lookup walks the photo's checksum
// first, then (for a Live Photo whose video collided with a
// standalone asset) the video's checksum, then the device identity;
// the first hit decides duplicate (the row is live) or trashed (it is
// not). Any other insert error stays an internal error.
func (s *server) resolveUploadConflict(r *http.Request, owner uuid.UUID, form uploadForm, photo, video *committedUpload, insertErr error) (dbgen.Asset, apigen.AssetResponseState, error) {
	var pgErr *pgconn.PgError
	if !errors.As(insertErr, &pgErr) || pgErr.Code != uniqueViolation ||
		!uploadUniquenessConstraints[pgErr.ConstraintName] {
		return dbgen.Asset{}, "", insertErr
	}

	lookups := []dbgen.GetAssetByOwnerChecksumParams{{
		OwnerID:  pgUUID(owner),
		Checksum: photo.checksum,
	}}
	if video != nil {
		lookups = append(lookups, dbgen.GetAssetByOwnerChecksumParams{
			OwnerID:  pgUUID(owner),
			Checksum: video.checksum,
		})
	}
	for _, lookup := range lookups {
		row, err := s.queries.GetAssetByOwnerChecksum(r.Context(), lookup)
		if err == nil {
			return row, uploadStateOf(row), nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return dbgen.Asset{}, "", err
		}
	}

	// The device identity is the remaining key that can have collided
	// (same deviceAssetId, drifted content).
	row, err := s.queries.GetAssetByDeviceId(r.Context(), dbgen.GetAssetByDeviceIdParams{
		OwnerID:       pgUUID(owner),
		DeviceAssetID: pgtype.Text{String: form.deviceAssetID, Valid: true},
	})
	if err == nil {
		return row, uploadStateOf(row), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return dbgen.Asset{}, "", err
	}

	// A violation whose row cannot be found is a state the schema
	// cannot produce; treat it as an internal error rather than
	// guessing a response.
	return dbgen.Asset{}, "", fmt.Errorf(
		"unique violation on %s resolved to no row (owner %s, checksum %s)",
		pgErr.ConstraintName, owner, photo.checksum)
}

// uploadStateOf maps a looked-up row onto its response state.
func uploadStateOf(row dbgen.Asset) apigen.AssetResponseState {
	if row.DeletedAt.Valid {
		return apigen.Trashed
	}
	return apigen.Duplicate
}

// assetInsertParams builds the insert parameters of one upload row.
// The component row keeps its own file part name as the original
// filename and carries no device identity; the photo row records the
// client-declared fileName and the device identity.
func assetInsertParams(id, owner uuid.UUID, form uploadForm, up committedUpload, takenAt time.Time, videoID pgtype.UUID, isComponent bool) dbgen.CreateAssetParams {
	kind := dbgen.MediaKindImage
	if up.format.Video {
		kind = dbgen.MediaKindVideo
	}
	originalFilename := up.partName
	deviceAssetID := pgtype.Text{}
	if !isComponent {
		originalFilename = form.fileName
		deviceAssetID = pgtype.Text{String: form.deviceAssetID, Valid: true}
	}
	return dbgen.CreateAssetParams{
		ID:               pgUUID(id),
		OwnerID:          pgUUID(owner),
		FilePath:         up.relPath,
		Checksum:         up.checksum,
		OriginalFilename: originalFilename,
		FileSize:         up.size,
		MimeType:         up.format.MIMEType,
		MediaKind:        kind,
		LivePhotoVideoID: videoID,
		IsComponent:      isComponent,
		DeviceAssetID:    deviceAssetID,
		TakenAt:          pgtype.Timestamptz{Time: takenAt, Valid: true},
	}
}

// uploadExifPlaceholder builds the metadata sidecar jsonb of a fresh
// asset: the raw device-side fields the metadata job refines into
// extracted EXIF, captured before the client's exact strings are
// normalized away. Both rows of a Live Photo carry the same device
// fields -- they describe the one device asset.
func uploadExifPlaceholder(form uploadForm) []byte {
	payload := make(map[string]string, 2)
	if form.deviceCreatedAtRaw != "" {
		payload["deviceCreatedAt"] = form.deviceCreatedAtRaw
	}
	if form.deviceTimeZone != "" {
		payload["deviceTimeZone"] = form.deviceTimeZone
	}
	data, err := json.Marshal(payload)
	if err != nil {
		// A map of two strings always marshals.
		return []byte("{}")
	}
	return data
}

// normalizeUploadForm validates the collected text fields against the
// contract and returns the normalized form together with the
// offending field names (empty when the form is valid). It runs
// before the first Receiver exists -- structurally, on every path
// that reaches a file part -- and again at end of body.
func normalizeUploadForm(fields map[string]string) (uploadForm, []string) {
	var form uploadForm
	var invalid []string

	deviceAssetID := strings.TrimSpace(fields[uploadFieldDeviceAssetID])
	if deviceAssetID == "" || utf8.RuneCountInString(deviceAssetID) > maxDeviceAssetIDRunes {
		invalid = append(invalid, uploadFieldDeviceAssetID)
	} else {
		form.deviceAssetID = deviceAssetID
	}

	fileName := strings.TrimSpace(fields[uploadFieldFileName])
	if fileName == "" || utf8.RuneCountInString(fileName) > maxFileNameRunes {
		invalid = append(invalid, uploadFieldFileName)
	} else {
		form.fileName = fileName
	}

	if raw, ok := fields[uploadFieldDeviceCreatedAt]; ok && strings.TrimSpace(raw) != "" {
		raw = strings.TrimSpace(raw)
		takenAt, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			invalid = append(invalid, uploadFieldDeviceCreatedAt)
		} else {
			form.deviceCreatedAtRaw = raw
			form.deviceCreatedAt = takenAt
		}
	}

	if raw, ok := fields[uploadFieldDeviceTimeZone]; ok {
		timeZone := strings.TrimSpace(raw)
		if utf8.RuneCountInString(timeZone) > maxDeviceTimeZoneRuns {
			invalid = append(invalid, uploadFieldDeviceTimeZone)
		} else {
			form.deviceTimeZone = timeZone
		}
	}

	return form, invalid
}

// readUploadField reads one text field, bounded: a field over the
// cap fails instead of streaming into memory.
func readUploadField(part *multipart.Part) (string, error) {
	data, err := io.ReadAll(io.LimitReader(part, maxUploadFieldBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) > maxUploadFieldBytes {
		return "", fmt.Errorf("field %q exceeds %d bytes", part.FormName(), maxUploadFieldBytes)
	}
	return string(data), nil
}

// uploadTextFieldName reports whether name is a text field of the
// upload contract.
func uploadTextFieldName(name string) bool {
	switch name {
	case uploadFieldDeviceAssetID, uploadFieldFileName,
		uploadFieldDeviceCreatedAt, uploadFieldDeviceTimeZone:
		return true
	default:
		return false
	}
}

// uploadFilePartName reports whether name is a file part of the
// upload contract.
func uploadFilePartName(name string) bool {
	return name == uploadPartPhoto || name == uploadPartLiveVideo
}

// normalizeExtension derives the canonical (lowercase, dotless)
// extension of a file name; the whitelist decides the rest.
func normalizeExtension(name string) string {
	return strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
}

// uploadStagingDir is the directory the upload receiver stages into.
func (s *server) uploadStagingDir() string {
	return filepath.Join(s.dataDir, "tmp")
}

// writeValidationError answers the 400 validation envelope with the
// offending field names.
func (s *server) writeValidationError(w http.ResponseWriter, message string, fields []string) {
	var details *map[string]interface{}
	if fields != nil {
		values := make([]interface{}, len(fields))
		for i, field := range fields {
			values[i] = field
		}
		wrapped := map[string]interface{}{"fields": values}
		details = &wrapped
	}
	s.writeError(w, http.StatusBadRequest, apigen.VALIDATIONFAILED, message, details)
}

// publishAssetUploaded reports the upload onto the activity stream:
// the resource id and its media kind, nothing more -- the event
// audience discipline of M2a.
func (s *server) publishAssetUploaded(row dbgen.Asset) {
	if s.events == nil {
		return
	}
	s.events.Publish("asset.uploaded", map[string]string{
		"assetId": uuid.UUID(row.ID.Bytes).String(),
		"kind":    string(row.MediaKind),
	})
}

// assetResponseFrom maps an asset row onto the API shape. The state
// is the caller's determination -- created for a fresh insert,
// duplicate or trashed from the idempotency lookup -- because the row
// alone does not know how the request reached it.
func assetResponseFrom(row dbgen.Asset, state apigen.AssetResponseState) apigen.AssetResponse {
	return apigen.AssetResponse{
		Id:               uuid.UUID(row.ID.Bytes),
		OwnerId:          uuid.UUID(row.OwnerID.Bytes),
		Kind:             apigen.AssetResponseKind(row.MediaKind),
		OriginalFilename: row.OriginalFilename,
		FileSize:         row.FileSize,
		MimeType:         row.MimeType,
		Checksum:         row.Checksum,
		State:            state,
		CreatedAt:        row.CreatedAt.Time.UTC(),
		TakenAt:          row.TakenAt.Time.UTC(),
		IsTrashed:        row.DeletedAt.Valid,
	}
}
