// The asset attribute PATCH: PATCH /assets/{id}, the
// owner's partial edit of the owner-level attributes -- the
// free-text description the detail panel renders, the archive
// switch that keeps the asset out of every family flow, the capture
// time (instant plus display zone) and the manual location (the
// coordinates plus the place they resolve to). The permission
// model is owner-only, the attribute-editing row of the
// permission matrix: no administrator override here (unlike the permanent
// delete), so the owner-scoped read decides the anti-enumeration 404
// and the writes carry the owner guard themselves. Set semantics
// throughout: the guards inside both writes make a repeat of the
// stored value the zero-row no-op, and the rows-affected counts --
// not the request -- decide whether anything changed; only an actual
// change publishes the single asset.updated event, after the commit.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	openapi_types "github.com/oapi-codegen/runtime/types"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/metawrite"
	"github.com/welpher/lumemor/server/internal/storage"
)

// descriptionLimit is the contract's bound on the asset description:
// 10,000 Unicode characters (runes) -- enough for a long caption,
// small enough that the column stays a bounded text document. The
// handler enforces it because the generated binder does not check
// the schema's maxLength.
const descriptionLimit = 10000

// PatchAsset implements PATCH /api/v1/assets/{id}: the partial edit
// of the owner-level attributes -- the description, the archive
// switch, the capture time (instant plus display zone) and the
// manual location (coordinates plus resolved place). The response is
// the detail after the edit -- the same read and serializer the GET
// endpoint serves, so the client sees the applied values (and the
// idempotent no-op sees the row's current state unchanged).
func (s *server) PatchAsset(w http.ResponseWriter, r *http.Request, id openapi_types.UUID, params apigen.PatchAssetParams) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	// The locale parameter: the GET face's own check, riding the
	// same-shape discipline (the PATCH response serves the detail
	// exactly as the GET serves it, place join included).
	locale := apigen.PatchAssetParamsLocaleEn
	if params.Locale != nil {
		if !params.Locale.Valid() {
			s.writeValidationError(w, "locale must be one of: en, zh-Hans", []string{"locale"})
			return
		}
		locale = *params.Locale
	}

	// The body is a small JSON document (the route group's 1 MiB cap
	// bounds it before this decode); anything that does not decode --
	// malformed JSON, a field of the wrong type -- is a client bug.
	// A field set to JSON null decodes as absent, so the
	// no-field check below also refuses a body of pure nulls. The
	// takenAt decode doubles as the contract's zone check: the RFC
	// 3339 binding refuses a timestamp without an offset designator,
	// which is exactly the ambiguity the contract refuses.
	var request apigen.AssetPatch
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeValidationError(w, "request body must be valid JSON", []string{"body"})
		return
	}
	if request.Description == nil && request.IsArchived == nil && request.TakenAt == nil &&
		request.Latitude == nil && request.Longitude == nil {
		s.writeValidationError(w,
			"the patch must carry description, takenAt, latitude/longitude or isArchived", []string{"body"})
		return
	}
	if request.Description != nil && utf8.RuneCountInString(*request.Description) > descriptionLimit {
		s.writeValidationError(w,
			fmt.Sprintf("description exceeds the limit of %d characters", descriptionLimit),
			[]string{"description"})
		return
	}
	// The location edit rides as the pair: one half alone is half a
	// coordinate, no location.
	if (request.Latitude == nil) != (request.Longitude == nil) {
		s.writeValidationError(w, "the location edit must carry latitude and longitude together", []string{"latitude"})
		return
	}

	// The owner-scoped read decides the 404: unknown and foreign ids
	// resolve to no row and answer the byte-identical envelope, so
	// asset ids cannot be enumerated. The row also feeds the event's
	// media kind. No state filter: a trashed asset is editable like
	// it is readable (the detail answers 200 with isTrashed true),
	// and the edit resurfaces with the restore.
	row, err := s.queries.GetAsset(r.Context(), dbgen.GetAssetParams{
		ID:      pgUUID(id),
		OwnerID: pgUUID(requester.userID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		s.writeAssetNotFound(w)
		return
	}
	if err != nil {
		s.internalError(w, r, "load asset for patch", err)
		return
	}

	// The albums face and the write grant share one read. The
	// capture-time edit writes the original FILE, so it needs the
	// grant: the owner of a managed original, or the owner of a
	// directory album whose source was writable at creation and whose
	// write switch is on. The gate is enforced HERE, not only in the
	// UI; a format with no standard field is a 409.
	ctx := r.Context()
	albums, albumsWritable, err := s.assetAlbumsForCaller(ctx, pgUUID(id), pgUUID(requester.userID))
	if err != nil {
		s.internalError(w, r, "load asset albums", err)
		return
	}

	var stagedWrite *metawrite.Staged
	var written *captureWrite
	if request.TakenAt != nil {
		if !assetMetadataWritable(row.OwnerID, pgUUID(requester.userID), row.StorageKind, row.MimeType, albumsWritable) {
			if !metawrite.Supported(row.MimeType) {
				s.writeError(w, http.StatusConflict, apigen.ASSETMETADATAUNSUPPORTED,
					"the file format carries no standard capture-time field", nil)
				return
			}
			s.writeError(w, http.StatusForbidden, apigen.ASSETMETADATAREADONLY,
				"the app may not modify this file", nil)
			return
		}
		// The file and the row speak seconds: EXIF carries no
		// sub-second field here, and Postgres truncates to
		// microseconds. Both sides truncate so a repeat is the
		// idempotent no-op.
		at := request.TakenAt.Truncate(time.Second)
		zone := at.Format("-07:00")
		alreadySet := row.TakenAt.Valid && row.TakenAt.Time.Truncate(time.Second).Equal(at) &&
			row.TakenAtZone.Valid && row.TakenAtZone.String == zone
		if !alreadySet {
			target, err := s.captureTimeTarget(row)
			if err != nil {
				s.logger.Error("resolve the asset file for the time edit", "asset_id", id, "error", err)
				s.writeError(w, http.StatusInternalServerError, apigen.ASSETMETADATAWRITEFAILED,
					"the capture time could not be written into the file", nil)
				return
			}
			// The subprocess runs BEFORE the transaction: no database
			// connection and no row lock is held across it.
			staged, err := metawrite.Stage(ctx, target, row.MimeType, at)
			if err != nil {
				s.logger.Error("write the capture time into the file", "asset_id", id, "error", err)
				s.writeError(w, http.StatusInternalServerError, apigen.ASSETMETADATAWRITEFAILED,
					"the capture time could not be written into the file", nil)
				return
			}
			sum, size, mtime, err := staged.Fingerprint()
			if err != nil {
				staged.Discard()
				s.logger.Error("read the written file back", "asset_id", id, "error", err)
				s.writeError(w, http.StatusInternalServerError, apigen.ASSETMETADATAWRITEFAILED,
					"the written file could not be read back", nil)
				return
			}
			stagedWrite = staged
			written = &captureWrite{at: at, zone: zone, checksum: sum, size: size, mtime: mtime}
		}
	}
	// A staging file never outlives the request: commit consumes it,
	// every other path discards it.
	defer func() {
		if stagedWrite != nil {
			stagedWrite.Discard()
		}
	}()

	// One transaction for the whole patch: when the request carries
	// both fields they land together or not at all -- a partial patch
	// never surfaces. Each write's set-semantics guard makes the
	// idempotent repeat match zero rows, and the rows-affected count
	// is the change signal of the event discipline below. The
	// isArchived arm is the batch's own SetAssetHidden -- one
	// authority for the switch, whatever the surfaces call it -- and
	// the description arm is its owner-guarded sibling.
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		s.internalError(w, r, "begin patch transaction", err)
		return
	}
	// The rollback is a no-op after a successful commit; the context
	// is captured while the request is still live.
	defer func() { _ = tx.Rollback(ctx) }()
	queries := dbgen.New(tx)

	changed := false
	if request.Description != nil {
		rows, err := queries.UpdateAssetDescription(ctx, dbgen.UpdateAssetDescriptionParams{
			ID:          pgUUID(id),
			OwnerID:     pgUUID(requester.userID),
			Description: *request.Description,
		})
		if err != nil {
			s.internalError(w, r, "update asset description", err)
			return
		}
		if rows > 0 {
			changed = true
		}
	}
	if request.TakenAt != nil && written != nil {
		fileMtime := pgtype.Timestamptz{}
		if row.StorageKind == dbgen.StorageKindExternal {
			fileMtime = pgtype.Timestamptz{Time: written.mtime, Valid: true}
		}
		rows, err := queries.UpdateAssetTakenAtWritten(ctx, dbgen.UpdateAssetTakenAtWrittenParams{
			ID:               pgUUID(id),
			OwnerID:          pgUUID(requester.userID),
			TakenAt:          pgtype.Timestamptz{Time: written.at, Valid: true},
			TakenAtZone:      pgtype.Text{String: written.zone, Valid: true},
			Checksum:         written.checksum,
			FileSize:         written.size,
			FileMtime:        fileMtime,
			ExpectedChecksum: row.Checksum,
		})
		if err != nil {
			s.internalError(w, r, "update asset capture time", err)
			return
		}
		if rows == 0 {
			s.writeError(w, http.StatusConflict, apigen.ASSETMETADATAWRITEFAILED,
				"the asset changed while the edit ran; nothing changed", nil)
			return
		}
		changed = true
	}
	if request.Latitude != nil && request.Longitude != nil {
		// The place the pin resolves to: the grid's nearest city, or
		// the honest absence. Without a grid (a deployment without
		// the baked place data) the resolution keeps whatever the
		// row held -- a data-less edit never destroys a resolution a
		// data-carrying boot produced (the metadata step's own
		// discipline).
		place := s.resolveManualPlace(ctx, *request.Latitude, *request.Longitude)
		if s.places == nil {
			place = row.PlaceID
		}
		rows, err := queries.UpdateAssetLocation(ctx, dbgen.UpdateAssetLocationParams{
			ID:      pgUUID(id),
			OwnerID: pgUUID(requester.userID),
			Lat:     pgtype.Float8{Float64: *request.Latitude, Valid: true},
			Lon:     pgtype.Float8{Float64: *request.Longitude, Valid: true},
			PlaceID: place,
		})
		if err != nil {
			s.internalError(w, r, "update asset location", err)
			return
		}
		if rows > 0 {
			changed = true
		}
	}
	if request.IsArchived != nil {
		rows, err := queries.SetAssetHidden(ctx, dbgen.SetAssetHiddenParams{
			ID:      pgUUID(id),
			OwnerID: pgUUID(requester.userID),
			Hidden:  *request.IsArchived,
		})
		if err != nil {
			s.internalError(w, r, "update asset archive switch", err)
			return
		}
		if rows > 0 {
			changed = true
		}
	}

	// The file lands first, then the transaction: a staging or DB
	// failure leaves the original byte-identical (the rollback path),
	// and a commit failure after the rename is the one residual
	// window, logged rather than silently swallowed.
	if stagedWrite != nil {
		if err := stagedWrite.Commit(); err != nil {
			stagedWrite.Discard()
			s.internalError(w, r, "write the capture time into the file", err)
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		if stagedWrite != nil {
			s.logger.Error("the file was rewritten but the row commit failed",
				"asset_id", id, "error", err)
		}
		s.internalError(w, r, "commit asset patch", err)
		return
	}

	// One event per patch that actually changed something, after the
	// commit: a subscriber that reacts to asset.updated reads rows
	// that already carry the new values. Both fields are owner-level
	// asset state -- the description is detail-panel content with no
	// timeline effect of its own, but it is asset state the same --
	// so one asset.updated covers whichever field changed; the
	// idempotent no-op stays silent, the transition-only discipline
	// of the batch and the trash lifecycle.
	if changed {
		s.publishAssetEventByID("asset.updated", uuid.UUID(id), row.MediaKind)
	}

	// The response: the detail after the edit, fetched fresh so it
	// reflects the applied values. A row deleted between the commit
	// and this read answers the 404 of a gone asset, exactly like
	// the delete endpoint's repeat. The faces ride along through the
	// shared read: the PATCH answer is the detail the GET serves,
	// people section included. The read rides the caller-visible
	// union like the GET's own -- the gate above already proved the
	// caller owns the row, and an owner's sync membership is what
	// the union resolves it through.
	detail, err := s.queries.GetAssetDetail(r.Context(), dbgen.GetAssetDetailParams{
		ID:     pgUUID(id),
		Viewer: pgUUID(requester.userID),
		Locale: string(locale),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		s.writeAssetNotFound(w)
		return
	}
	if err != nil {
		s.internalError(w, r, "load patched asset detail", err)
		return
	}
	faces, err := s.assetFacesForCaller(r.Context(), pgUUID(id), pgUUID(requester.userID))
	if err != nil {
		s.internalError(w, r, "load patched asset faces", err)
		return
	}
	tags, err := s.assetTagsForCaller(r.Context(), pgUUID(id), pgUUID(requester.userID))
	if err != nil {
		s.internalError(w, r, "load patched asset tags", err)
		return
	}
	includePath := requester.role == dbgen.UserRoleAdmin
	metadataWritable := assetMetadataWritable(detail.OwnerID, pgUUID(requester.userID), detail.StorageKind, detail.MimeType, albumsWritable)
	s.writeJSON(w, http.StatusOK, assetDetailFrom(detail, faces, tags, albums, includePath, metadataWritable))
}

// captureTimeTarget resolves the file the write-back edits: the
// original for a live row, the trash copy for a trashed external row
// (the trash move relocated the file; the row stays editable), and
// the storage layout's path for a managed original.
func (s *server) captureTimeTarget(row dbgen.Asset) (string, error) {
	if row.StorageKind == dbgen.StorageKindExternal && row.DeletedAt.Valid && row.TrashPath.Valid {
		return row.TrashPath.String, nil
	}
	return storage.ResolveAssetPath(s.dataDir, row.FilePath, string(row.StorageKind))
}

// captureWrite is one staged file write's facts: the truncated
// capture time, its display zone, and the new file's digest, size and
// mtime -- what the row takes once the write lands.
type captureWrite struct {
	at       time.Time
	zone     string
	checksum string
	size     int64
	mtime    time.Time
}
