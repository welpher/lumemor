// Asset handlers of the upload endpoint: the multipart upload
// with its idempotent three-state semantics, the Live Photo pairing,
// and the device idempotency lookup.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/db"
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
// IS the upload idempotency: the blob domain's total owner+checksum
// index (live and trashed rows alike -- a directory album's position
// rows are outside it by storage kind) and the owner+device identity.
// A violation of any of them routes the request into the three-state
// lookup instead of the error path.
var uploadUniquenessConstraints = map[string]bool{
	"assets_owner_checksum_blob_uq":        true,
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

// sealedUpload is one file part that streamed and validated: the
// receiver holding its sealed staged file, plus everything the path
// render and the database row need. Nothing is placed yet -- the
// placement renders from the checksum, the header and the asset
// identity, which exist only after the whole request validated.
type sealedUpload struct {
	// partName is the file part's own file name; the component video
	// row keeps it as its original filename.
	partName string
	// ext is the canonical (lowercase, dotless) extension the part
	// carried; the whitelist validated it and the placement
	// normalizes it (jpeg to jpg, tif to tiff).
	ext string
	// header is the EXIF capture context parsed from the sealed file:
	// the taken-at fallback chain's first link and the camera
	// variables of the path render.
	header storage.MediaHeader
	// receiver holds the sealed staged file until the placement links
	// it into the originals tree.
	receiver *storage.Receiver
	// checksum is the sha256 hex digest of the payload.
	checksum string
	// size is the streamed byte count.
	size int64
	// format is the whitelisted format the extension mapped to.
	format storage.UploadFormat
	// relPath is the placed storage path relative to the data root,
	// the portable form stored in file_path (set by the placement).
	relPath string
}

// UploadAsset implements POST /api/v1/assets: the multipart backup
// upload with idempotent three-state semantics.
//
// The body streams through in one pass: text fields are read as they
// arrive (they precede the file parts per contract), every file part
// is handed to a fresh Receiver that stages, validates and seals it
// -- the EXIF capture context of the sealed file is parsed for the
// path render -- and the owner+checksum dedup runs BEFORE any
// placement: a true duplicate answers through the three-state lookup
// with nothing placed at all. A fresh upload then renders the
// instance's storage template per part and places the sealed files
// into the per-owner originals tree (overwrite-proof links,
// collision-suffixed), and the database insert -- one row for the
// photo, plus a linked component row for the Live Photo video --
// decides the outcome through the unique indexes: a clean insert
// answers 201 created; a violation routes into the lookup that
// answers 200 duplicate (the identical asset is live) or 200 trashed
// (it sits in the trash), discarding the placed files the loser
// never got to reference. The handler validates the fields before
// the first Receiver exists, so a request that fails validation
// reaches its terminal state without staging anything; a request
// rejected only after a file part legitimately sealed discards its
// staged files and leaves no residue at all.
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
	var photo, liveVideo *sealedUpload

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
				s.rejectUpload(w, photo, liveVideo, "invalid upload fields", invalid)
				return
			}
			if part.FileName() == "" {
				s.rejectUpload(w, photo, liveVideo, "file parts must carry a file name", []string{name})
				return
			}
			target := &photo
			if name == uploadPartLiveVideo {
				if photo == nil {
					// The pairing video follows its photo: streaming
					// cannot hold the video until a later photo part
					// arrives, so an early liveVideo is refused before
					// anything is staged.
					s.rejectUpload(w, photo, liveVideo, "liveVideo must follow the photo part",
						[]string{uploadPartLiveVideo})
					return
				}
				target = &liveVideo
			}
			if *target != nil {
				s.rejectUpload(w, photo, liveVideo, "duplicate file part", []string{name})
				return
			}
			sealed, handled := s.receiveFilePart(w, r, part)
			if !handled {
				return
			}
			*target = &sealed
			continue
		}

		if uploadTextFieldName(name) {
			if photo != nil || liveVideo != nil {
				s.rejectUpload(w, photo, liveVideo, "multipart fields must precede file parts", []string{name})
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

		s.rejectUpload(w, photo, liveVideo, "unknown multipart part", []string{name})
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
	// so the pair is refused -- nothing was placed yet, the sealed
	// files are simply discarded.
	if liveVideo != nil && liveVideo.checksum == photo.checksum {
		discardUpload(photo, liveVideo)
		s.writeValidationError(w, "the Live Photo video must differ from the photo",
			[]string{uploadPartLiveVideo})
		return
	}
	// THE PAIRING VIDEO MUST BE A VIDEO. The pair's component row is
	// only ever the motion: an image-family part would create a
	// component no transcode can make playable and no motion can use,
	// so the request is refused outright -- the same loud refusal the
	// link's own checks give (exists, is a video, same owner) rather
	// than a silently dropped part. Nothing was placed yet.
	if liveVideo != nil && !liveVideo.format.Video {
		discardUpload(photo, liveVideo)
		s.writeValidationError(w, "the Live Photo motion must be a video",
			[]string{uploadPartLiveVideo})
		return
	}

	// The (owner+checksum) dedup runs FIRST, before any placement: a
	// re-upload of content the owner already holds -- live or trashed
	// -- answers through the same three-state lookup the insert's
	// conflict path uses, with the sealed files discarded and nothing
	// placed, so a true duplicate never reaches the collision loop.
	row, state, found, err := s.lookupExistingUpload(r, requester.userID, form, photo, liveVideo)
	if err != nil {
		discardUpload(photo, liveVideo)
		s.internalError(w, r, "lookup existing upload", err)
		return
	}
	if found {
		discardUpload(photo, liveVideo)
		s.logger.Info("upload matched an existing asset",
			"owner_id", requester.userID, "state", state,
			"checksum", photo.checksum, "asset_id", uuid.UUID(row.ID.Bytes))
		s.writeJSON(w, http.StatusOK, assetResponseFrom(row, state))
		return
	}

	asset, state, err := s.insertUpload(r, requester.userID, form, photo, liveVideo)
	if err != nil {
		s.internalError(w, r, "insert upload", err)
		return
	}
	if state == apigen.Created {
		// The transaction committed: the asset's rows are durable and
		// visible, so the pipeline may start. Enqueueing earlier (inside
		// the transaction) would race the job's reads against the
		// commit; enqueueing never at all leaves the asset honestly
		// not-ready, so a failure here is logged, not failed -- the
		// upload itself succeeded, and failing the response would not
		// undo the commit.
		s.enqueueAssetJobs(r, asset, liveVideo)
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

// receiveFilePart streams one file part through a fresh Receiver:
// staged, sniffed, hashed and sealed, with the EXIF capture context
// of the sealed file parsed for the path render. No placement happens
// here -- the destination renders from values (the checksum, the
// header, the asset id) that exist only once the whole request
// validated. It reports whether the request was answered (a rejection
// writes the response itself), and the sealed upload on success. The
// part's own file name carries the extension; a payload inconsistent
// with it, or a file over the per-file limit, aborts with the file
// part named in the details.
func (s *server) receiveFilePart(w http.ResponseWriter, r *http.Request, part *multipart.Part) (sealedUpload, bool) {
	ext := normalizeExtension(part.FileName())
	receiver, err := storage.NewReceiver(s.uploadStagingDir(), s.uploadLimit, part)
	if err != nil {
		s.internalError(w, r, "stage upload", err)
		return sealedUpload{}, false
	}
	if err := receiver.Seal(ext); err != nil {
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
			s.internalError(w, r, "seal upload", err)
		}
		return sealedUpload{}, false
	}

	format, ok := storage.LookupUploadFormat(ext)
	if !ok {
		// Seal has already validated the extension against the
		// whitelist, so this cannot happen; the guard keeps the
		// invariant explicit rather than trusting a faraway table.
		s.internalError(w, r, "resolve upload format",
			fmt.Errorf("extension %q left the whitelist", ext))
		return sealedUpload{}, false
	}

	return sealedUpload{
		partName: part.FileName(),
		ext:      ext,
		header:   receiver.Header(),
		receiver: receiver,
		checksum: receiver.Checksum(),
		size:     receiver.Size(),
		format:   format,
	}, true
}

// lookupExistingUpload answers the upload idempotency BEFORE any
// placement: the owner+checksum lookups of both parts over the FULL
// content domain (live and trashed rows of every storage kind, in the
// photo-then-video order the conflict resolution walks), then the
// device identity. A hit means the upload is a duplicate or a trashed
// re-upload -- the caller discards the sealed files and answers
// through the same three-state mapping the insert's conflict path
// produces, so the early answer and the race answer agree -- with one
// exception the position model added: a directory source's ORPHAN
// MARK (the file vanished, the row is dead and no trash copy exists)
// is not content the owner holds any more, so the upload REPLACES it
// -- the dead row goes and the fresh blob row lands in its place,
// carrying the same content back into the library. The device
// identity is the remaining collision the pre-check can see: a fresh
// content under a known deviceAssetId resolves to the existing row
// exactly as the insert's unique violation would.
func (s *server) lookupExistingUpload(r *http.Request, owner uuid.UUID, form uploadForm, photo, video *sealedUpload) (dbgen.Asset, apigen.AssetResponseState, bool, error) {
	created, found, err := s.resolveContentLookups(r, owner, photo, video)
	if err != nil || found {
		return created, uploadStateOf(created), found, err
	}

	row, err := s.queries.GetAssetByDeviceId(r.Context(), dbgen.GetAssetByDeviceIdParams{
		OwnerID:       pgUUID(owner),
		DeviceAssetID: pgtype.Text{String: form.deviceAssetID, Valid: true},
	})
	if err == nil {
		return row, uploadStateOf(row), true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return dbgen.Asset{}, "", false, err
	}
	return dbgen.Asset{}, "", false, nil
}

// resolveContentLookups walks the content lookups of both parts and
// answers the first row that still claims the content. A live row (of
// whichever storage kind) and a trashed row whose bytes the server
// holds (a blob in place, a source file moved into the trash area)
// both claim it: the caller answers duplicate or trashed through the
// same three-state mapping the race path uses. An orphan-marked
// source row does not claim it: the file it referenced is gone from
// its folder, the row is dead, and nothing can restore it -- so the
// upload removes the dead row here and answers "no existing content",
// letting the placement and the fresh insert run. The removal is
// guarded in the statement itself (only an orphan mark of this owner
// matches), so a concurrent upload that already replaced it leaves
// this one to the insert's own conflict resolution.
func (s *server) resolveContentLookups(r *http.Request, owner uuid.UUID, photo, video *sealedUpload) (dbgen.Asset, bool, error) {
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
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return dbgen.Asset{}, false, err
		}
		if row.DeletedAt.Valid && row.StorageKind == dbgen.StorageKindExternal && !row.TrashPath.Valid {
			// The purge path's own transaction: the row's faces
			// re-home or dissolve before the cascade takes them, and
			// its derivatives go with the id. trashedOnly is true: the
			// row is an ORPHAN MARK -- a trashed row -- and the delete
			// proves it still is (a state that cannot be restored
			// through the endpoints, so the guard is defense in depth,
			// and a zero-row answer simply leaves the insert to the
			// conflict resolution).
			removed, err := s.purgeAssetRow(r, row, true)
			if err != nil {
				return dbgen.Asset{}, false, err
			}
			if removed {
				s.logger.Info("upload replaced a vanished source row",
					"owner_id", owner, "checksum", lookup.Checksum,
					"asset_id", uuid.UUID(row.ID.Bytes))
			}
			continue
		}
		return row, true, nil
	}
	return dbgen.Asset{}, false, nil
}

// discardUpload cleans the sealed files of a request that will never
// place them (the pre-check's duplicate, a post-stream validation
// failure), or the placed files of one whose rows never committed
// (the insert's conflict path): the receiver's Discard removes
// whichever artifact its state holds.
func discardUpload(photo, video *sealedUpload) {
	if photo != nil {
		photo.receiver.Discard()
	}
	if video != nil {
		video.receiver.Discard()
	}
}

// rejectUpload answers a validation failure of the multipart walk and
// cleans whatever the request sealed so far: a rejection after a file
// part sealed discards the staged files -- nothing was placed, so
// nothing else can reference them -- leaving no residue at all.
func (s *server) rejectUpload(w http.ResponseWriter, photo, liveVideo *sealedUpload, message string, fields []string) {
	discardUpload(photo, liveVideo)
	s.writeValidationError(w, message, fields)
}

// placeSealedPart renders one part's path segment from the instance
// template and places the sealed file at
// originals/<ownerID>/<rendered>.<ext> -- the placement normalizes the
// extension and appends the collision suffix when the target exists.
func placeSealedPart(receiver *storage.Receiver, template *storage.Template, owner uuid.UUID, vars storage.RenderVars) (string, error) {
	rendered, err := template.Render(vars)
	if err != nil {
		return "", fmt.Errorf("render storage template: %w", err)
	}
	return receiver.Place(owner.String(), rendered, vars.Ext)
}

// uploadTemplate loads the instance's storage template, compiled
// strictly. A setting that fails to load or to compile falls back to
// the default with a log line: the admin surface validates a template
// (compile, sample render, structure) before it saves, so a bad row
// is a corruption to point at, not an upload to fail.
func (s *server) uploadTemplate(ctx context.Context) *storage.Template {
	text, err := db.StorageTemplate(ctx, s.queries)
	if err != nil {
		s.logger.Error("read the storage template setting; using the default", "error", err)
	}
	compiled, err := storage.CompileTemplate(text)
	if err == nil {
		return compiled
	}
	s.logger.Error("compile the storage template setting; using the default", "error", err)
	compiled, err = storage.CompileTemplate(storage.DefaultTemplate)
	if err != nil {
		// The default is a compile-time constant of this package's
		// own grammar; failing to compile it is unreachable.
		panic(fmt.Sprintf("api: the default storage template does not compile: %v", err))
	}
	return compiled
}

// ingestTakenAt resolves the capture instant the PATH renders from,
// per the fallback chain: the EXIF wall clock with its own offset
// (the OffsetTime tag, else the per-tag OffsetTimeOriginal), else the
// same wall clock read in the device-declared zone, else the
// device-reported time, else the upload time. The chain mirrors the
// metadata job's resolveExifTime -- including its refusal to guess a
// zone for a bare wall clock -- so the folder the path lands in
// agrees with the taken_at the job later refines. The date tokens
// render the instant's own zone, so the folder says the wall-clock
// day the capture happened.
func ingestTakenAt(header storage.MediaHeader, form uploadForm) time.Time {
	if wall, ok := parseExifWallClock(header.DateTimeOriginal); ok {
		if seconds, ok := parseExifOffset(header.OffsetTime); ok {
			return wallInZone(wall, time.FixedZone("", seconds))
		}
		if seconds, ok := parseExifOffset(header.OffsetTimeOriginal); ok {
			return wallInZone(wall, time.FixedZone("", seconds))
		}
		if form.deviceTimeZone != "" {
			if loc, err := time.LoadLocation(form.deviceTimeZone); err == nil {
				return wallInZone(wall, loc)
			}
		}
	}
	if !form.deviceCreatedAt.IsZero() {
		return form.deviceCreatedAt
	}
	return time.Now()
}

// parseExifWallClock reads the EXIF "2006:01:02 15:04:05" form.
func parseExifWallClock(raw string) (time.Time, bool) {
	wall, err := time.Parse("2006:01:02 15:04:05", raw)
	if err != nil {
		return time.Time{}, false
	}
	return wall, true
}

// parseExifOffset reads an EXIF zone offset ("+05:30", "-08:00") as
// seconds -- the same reading the metadata job applies.
func parseExifOffset(s string) (int, bool) {
	if len(s) != 6 || (s[0] != '+' && s[0] != '-') || s[3] != ':' {
		return 0, false
	}
	hours, err := strconv.Atoi(s[1:3])
	if err != nil || hours < 0 || hours > 23 {
		return 0, false
	}
	minutes, err := strconv.Atoi(s[4:6])
	if err != nil || minutes < 0 || minutes > 59 {
		return 0, false
	}
	seconds := hours*3600 + minutes*60
	if s[0] == '-' {
		seconds = -seconds
	}
	return seconds, true
}

// wallInZone re-reads the EXIF wall clock in the resolved zone: the
// wall clock read there IS the instant (a plain .In(zone) would
// convert an already-parsed instant instead).
func wallInZone(wall time.Time, zone *time.Location) time.Time {
	return time.Date(wall.Year(), wall.Month(), wall.Day(), wall.Hour(), wall.Minute(), wall.Second(), wall.Nanosecond(), zone)
}

// insertUpload persists the upload: both parts are placed first (the
// path rendered from the instance's storage template and linked into
// the per-owner originals tree, overwrite-proof), then the Live Photo
// video component row, the photo row referencing it, and the EXIF
// placeholders land in one transaction -- all rows or none. The
// taken-at value the ROWS carry is the device-reported capture time
// or, absent one, the upload time; the metadata pipeline refines it
// later. The PATH's taken-at is the ingest chain (ingestTakenAt),
// which mirrors the metadata job's resolution so the folder agrees
// with the refined value. A unique index violation is not a failure
// but the idempotency at work: the transaction rolls back, the placed
// files the loser never got to reference are discarded, and the
// conflict resolves through the lookup into the duplicate or trashed
// state, carried as a normal outcome.
//
// A COMMIT error is the one outcome that does NOT discard: the commit
// may have landed while its acknowledgement was lost, so the placed
// files stay (see settleAmbiguousCommit; the boot orphan sweep
// reclaims a true leak). Every earlier failure -- validation, an
// insert, the album wiring -- is a certain rollback, and those paths
// discard as before.
func (s *server) insertUpload(r *http.Request, owner uuid.UUID, form uploadForm, photo, video *sealedUpload) (dbgen.Asset, apigen.AssetResponseState, error) {
	ctx := r.Context()

	// The template the paths render from: the instance setting,
	// compiled strictly. A setting that fails to load or compile
	// falls back to the default (logged) -- the admin surface
	// validates before saving, so a bad row is a corruption to log,
	// not an upload to fail.
	template := s.uploadTemplate(ctx)

	// The asset ids arrive before the placement: the {{assetId}}
	// family renders from them, and the rows below reuse them.
	photoID, err := uuid.NewV7()
	if err != nil {
		discardUpload(photo, video)
		return dbgen.Asset{}, "", err
	}
	var videoID pgtype.UUID
	if video != nil {
		id, err := uuid.NewV7()
		if err != nil {
			discardUpload(photo, video)
			return dbgen.Asset{}, "", err
		}
		videoID = pgUUID(id)
	}

	// The still's capture context drives BOTH parts' renders: the
	// taken-at fallback chain over the photo's EXIF header, and the
	// camera identity of the photo part. The Live Photo's motion
	// video renders with the still's taken-at (the pair shares its
	// folder) and its own file name, extension and asset id; its
	// camera variables come from its own header, which videos do not
	// carry.
	takenAt := ingestTakenAt(photo.header, form)

	// Render and place each part. The album variables render empty:
	// a direct upload joins only the system sync stream, which is not
	// a naming source; membership joined later never re-paths (the
	// contract documents this).
	photoVars := storage.RenderVars{
		Filename:  form.fileName,
		Ext:       photo.ext,
		Video:     photo.format.Video,
		TakenAt:   takenAt,
		Make:      photo.header.Make,
		Model:     photo.header.Model,
		LensModel: photo.header.LensModel,
		AssetID:   photoID.String(),
	}
	if photo.relPath, err = placeSealedPart(photo.receiver, template, owner, photoVars); err != nil {
		discardUpload(photo, video)
		return dbgen.Asset{}, "", fmt.Errorf("place photo: %w", err)
	}
	if video != nil {
		videoVars := storage.RenderVars{
			Filename: video.partName,
			Ext:      video.ext,
			Video:    video.format.Video,
			TakenAt:  takenAt,
			AssetID:  uuid.UUID(videoID.Bytes).String(),
		}
		if video.relPath, err = placeSealedPart(video.receiver, template, owner, videoVars); err != nil {
			discardUpload(photo, video)
			return dbgen.Asset{}, "", fmt.Errorf("place live photo video: %w", err)
		}
	}

	// The rows' taken-at, unchanged from the pre-template flow: the
	// device-reported time or the upload time; the metadata job
	// refines it from the file's full EXIF.
	rowTakenAt := form.deviceCreatedAt
	if rowTakenAt.IsZero() {
		rowTakenAt = time.Now()
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		discardUpload(photo, video)
		return dbgen.Asset{}, "", err
	}
	// The rollback is a no-op after a successful commit; the context
	// is captured while the request is still live.
	defer func() { _ = tx.Rollback(ctx) }()
	queries := dbgen.New(tx)

	// The Live Photo video is a component row: no device identity of
	// its own, never in the timeline, linked from the photo row.
	if video != nil {
		videoRow, err := queries.CreateAsset(ctx, assetInsertParams(uuid.UUID(videoID.Bytes), owner, form, *video, rowTakenAt, pgtype.UUID{}, true))
		if err != nil {
			// Release the transaction's connection before the lookup
			// resolves the conflict through the pool; the deferred
			// rollback afterwards is a no-op.
			_ = tx.Rollback(ctx)
			// The rows never commit: the files this request placed are
			// orphans nothing else can reference (the placement's link
			// proved each path did not exist before), so they go.
			discardUpload(photo, video)
			return s.resolveUploadConflict(r, owner, form, photo, video, err)
		}
		videoID = videoRow.ID
		if err := queries.CreateExifPlaceholder(ctx, dbgen.CreateExifPlaceholderParams{
			AssetID: videoRow.ID,
			Exif:    uploadExifPlaceholder(form),
		}); err != nil {
			discardUpload(photo, video)
			return dbgen.Asset{}, "", err
		}
	}

	photoRow, err := queries.CreateAsset(ctx, assetInsertParams(photoID, owner, form, *photo, rowTakenAt, videoID, false))
	if err != nil {
		// Same as above: the lookup needs a pool connection.
		_ = tx.Rollback(ctx)
		discardUpload(photo, video)
		return s.resolveUploadConflict(r, owner, form, photo, video, err)
	}
	if err := queries.CreateExifPlaceholder(ctx, dbgen.CreateExifPlaceholderParams{
		AssetID: photoRow.ID,
		Exif:    uploadExifPlaceholder(form),
	}); err != nil {
		discardUpload(photo, video)
		return dbgen.Asset{}, "", err
	}

	// The photo lands in the owner's sync album -- the core stream the
	// timeline aggregates. The insert rides the upload transaction, so
	// the asset and its album membership commit atomically: an upload
	// that committed but stayed album-less would be invisible to every
	// timeline. The component video row lands in the album TOO: the
	// caller-visible union reads membership, so the
	// component stays reachable BY ITS ID through the detail, the
	// original download and the play stream exactly like the contract
	// documents -- while the membership stays invisible to every
	// listing surface, which all filter component rows (the row is
	// dead weight for the timeline alone). The ensure (rather than a
	// plain insert) is the recovery path for an account whose rows
	// predate this wiring; its upsert arm keeps a concurrent recovery
	// from failing this upload.
	syncAlbumID, err := uuid.NewV7()
	if err != nil {
		discardUpload(photo, video)
		return dbgen.Asset{}, "", err
	}
	albumID, err := queries.EnsureSyncAlbum(ctx, dbgen.EnsureSyncAlbumParams{
		ID:      pgUUID(syncAlbumID),
		OwnerID: pgUUID(owner),
	})
	if err != nil {
		discardUpload(photo, video)
		return dbgen.Asset{}, "", err
	}
	if err := queries.AddAssetToAlbum(ctx, dbgen.AddAssetToAlbumParams{
		AlbumID: albumID,
		AssetID: photoRow.ID,
		AddedBy: pgUUID(owner),
	}); err != nil {
		discardUpload(photo, video)
		return dbgen.Asset{}, "", err
	}
	if videoID.Valid {
		if err := queries.AddAssetToAlbum(ctx, dbgen.AddAssetToAlbumParams{
			AlbumID: albumID,
			AssetID: videoID,
			AddedBy: pgUUID(owner),
		}); err != nil {
			discardUpload(photo, video)
			return dbgen.Asset{}, "", err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		// A COMMIT ERROR IS NOT PROOF THE ROWS DID NOT LAND. The commit
		// can be durable server-side while the client sees an error --
		// the connection reset, a proxy cut the reply, the context was
		// cancelled, the database restarted inside the reply window.
		// The placed files therefore STAY: a leak is reclaimable by the
		// boot orphan sweep, a deleted original behind a committed row
		// is not. settleAmbiguousCommit only decides what to ANSWER.
		return s.settleAmbiguousCommit(ctx, owner, photo.checksum, err)
	}
	return photoRow, apigen.Created, nil
}

// commitProbeTimeout bounds the read-only probe that answers an
// ambiguous commit. It is short because it only decides the response;
// a probe that cannot finish answers the original commit error.
const commitProbeTimeout = 5 * time.Second

// settleAmbiguousCommit answers a failed transaction commit without
// touching the placed originals. The only question it settles is what
// the response says: a read-only probe over the owner+checksum, on a
// DETACHED context (the request's cancellation is exactly one of the
// ways a commit fails, so the probe cannot ride it; WithoutCancel keeps
// the values and sheds the cancellation, and a short timeout bounds
// it), finds the row when the commit was durable. Present -> the upload
// DID land: answer the row as created. Absent or a probe error -> the
// original commit error stands.
//
// THE PROBE NEVER AUTHORIZES DELETION. There is a window between the
// probe and the commit's own outcome in which the transaction lands
// after the probe found nothing, so "not found" says nothing about the
// files; they stay on disk for the orphan sweep either way. Adding a
// discard call to the absent branch would reintroduce exactly the loss
// this helper exists to prevent.
func (s *server) settleAmbiguousCommit(ctx context.Context, owner uuid.UUID, checksum string, commitErr error) (dbgen.Asset, apigen.AssetResponseState, error) {
	probeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), commitProbeTimeout)
	defer cancel()

	row, err := s.queries.GetAssetByOwnerChecksum(probeCtx, dbgen.GetAssetByOwnerChecksumParams{
		OwnerID:  pgUUID(owner),
		Checksum: checksum,
	})
	if err == nil {
		s.logger.Warn("the upload commit reported an error but landed; answering created",
			"owner_id", owner, "checksum", checksum,
			"asset_id", uuid.UUID(row.ID.Bytes), "error", commitErr)
		return row, apigen.Created, nil
	}
	return dbgen.Asset{}, "", commitErr
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
func (s *server) resolveUploadConflict(r *http.Request, owner uuid.UUID, form uploadForm, photo, video *sealedUpload, insertErr error) (dbgen.Asset, apigen.AssetResponseState, error) {
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
func assetInsertParams(id, owner uuid.UUID, form uploadForm, up sealedUpload, takenAt time.Time, videoID pgtype.UUID, isComponent bool) dbgen.CreateAssetParams {
	kind := dbgen.MediaKindImage
	if up.format.Video {
		kind = dbgen.MediaKindVideo
	}
	originalFilename := up.partName
	deviceAssetID := pgtype.Text{}
	// The pipeline count the uploader enqueues. A photo row counts
	// its metadata and thumbnail jobs (the main asset gets a
	// thumbnail whatever its kind -- an image's three AVIF sizes, a
	// video's poster frame), plus the transcode job when the photo
	// row itself is a video (a standalone video upload: three jobs).
	// A component video row counts ONE job: the transcode that
	// derives its playable stream -- HEVC Live Photo components need
	// a web-safe copy for browser playback, and the pair's motion
	// plays through the component's own id. The metadata job still
	// runs for it (both rows carry sidecars to refine), but its
	// completion never consumes the transcode's count. A component
	// whose part is image-family counts none: no job of any kind can
	// run for it, so a counted transcode would strand the counter.
	pendingJobs := int32(2)
	if isComponent {
		pendingJobs = 0
		if kind == dbgen.MediaKindVideo {
			pendingJobs = 1
		}
	} else {
		originalFilename = form.fileName
		deviceAssetID = pgtype.Text{String: form.deviceAssetID, Valid: true}
		if kind == dbgen.MediaKindVideo {
			pendingJobs = 3
		}
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
		PendingJobs:      pendingJobs,
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

// enqueueAssetJobs starts the pipeline of a committed upload: one
// metadata job for the photo row and one for the Live Photo's
// component video row (both carry exif sidecars the job refines; only
// the photo's pipeline announces readiness), plus one thumbnail job
// for the photo row itself -- image or video kind alike: an image gets
// its three AVIF sizes, a video its poster frame -- and, when the
// photo row itself is a video (a standalone video upload), the
// transcode job that derives its playable web-safe MP4 (three jobs
// total; the pending_jobs counter was initialized to match). The
// component video gets no thumbnail, but it gets the transcode
// when its own part is video-family: its playable stream is what the
// pair's motion plays through -- HEVC components re-encode to the
// web-safe H.264/AAC pair every browser plays. The component's
// counter counts exactly that one job; the metadata job the component
// also receives never consumes the count, and the component's
// transcode completion never announces (the photo row owns the pair's
// asset.ready). An image-family component part receives no transcode:
// the job would be a permanent no-op against a counter left stranded
// -- the enqueue therefore stays symmetric with the pending_jobs
// initialization in assetInsertParams.
//
// The enqueue runs after the upload transaction committed, so its
// context must not die with the client connection: a disconnect in
// the commit-to-enqueue window would fail the enqueue and strand the
// asset's pipeline counter not-ready forever, with no API path to
// requeue it. WithoutCancel keeps the request's deadline and values
// while shedding its cancellation. A failed enqueue logs and moves
// on: the upload is committed and its idempotent re-upload resolves
// as duplicate without re-enqueueing, so recovery from a lost job
// belongs to the requeue sweep, not to the response of this request.
func (s *server) enqueueAssetJobs(r *http.Request, asset dbgen.Asset, liveVideo *sealedUpload) {
	if s.jobs == nil {
		return
	}
	ctx := context.WithoutCancel(r.Context())
	assetID := uuid.UUID(asset.ID.Bytes)
	assetIDs := []uuid.UUID{assetID}
	if asset.LivePhotoVideoID.Valid {
		assetIDs = append(assetIDs, uuid.UUID(asset.LivePhotoVideoID.Bytes))
	}
	for _, id := range assetIDs {
		if err := s.jobs.EnqueueMetadata(ctx, id); err != nil {
			s.logger.Error("enqueue metadata job", "error", err, "asset_id", id)
		}
	}
	if err := s.jobs.EnqueueThumbnail(ctx, assetID); err != nil {
		s.logger.Error("enqueue thumbnail job", "error", err, "asset_id", assetID)
	}
	if asset.MediaKind == dbgen.MediaKindVideo {
		if err := s.jobs.EnqueueTranscode(ctx, assetID); err != nil {
			s.logger.Error("enqueue transcode job", "error", err, "asset_id", assetID)
		}
	}
	// The Live Photo component's own transcode, when its part belongs
	// to the video family (the format the row's media kind derived
	// from -- the same gate assetInsertParams applied to the
	// component's pending_jobs).
	if liveVideo != nil && liveVideo.format.Video && asset.LivePhotoVideoID.Valid {
		componentID := uuid.UUID(asset.LivePhotoVideoID.Bytes)
		if err := s.jobs.EnqueueTranscode(ctx, componentID); err != nil {
			s.logger.Error("enqueue transcode job", "error", err, "asset_id", componentID)
		}
	}
}

// publishAssetUploaded reports the upload onto the activity stream:
// the resource id and its media kind, nothing more -- the event
// audience discipline.
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
