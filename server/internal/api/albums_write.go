// The album surface writes (reshaped by the albums-as-sources
// and album-autonomy reworks): the directory album
// creator behind POST /albums (the write face's only creatable type
// -- the request names an existing path the server validates and
// references in place; the face is the administrator's, and the
// creation-time write probe lands the source-writable fact; the
// create dialog's two switches ride the same request -- the
// deletability switch onto the row, the creator's timeline
// exclusion into their own preference), the property patch behind
// PATCH /albums/{albumId} (with the cover's live-member validation
// and the deletability switch of the directory albums), the album
// removal behind DELETE /albums/{albumId} (the virtual reference
// set's plain cascade and the directory album's un-ingest -- see
// albumremoval.go) and the membership removal behind
// DELETE /albums/{albumId}/items (batch
// semantics: the honest {removed, skipped} report, never per-item
// errors; the membership ADD is gone -- albums are sources, not
// curation targets).
//
// Every write shares the two-tier authorization answer: an album the
// caller cannot see at all (unknown, foreign-unshared, every sync
// album) answers the reads' byte-identical single-shape 404, so
// album ids cannot be probed; an album the caller CAN see but does
// not own (a share row exists) answers the honest 403 -- the share is
// view-only. The owner writes, and the admin role bypasses the owner
// check (the permanent delete's discipline: the refusal, not the
// read, shapes the response). The live album events
// (album.created/updated/deleted/items-changed, plus the audience
// pair of albumaudience.go) publish after the writes commit,
// ids-only payloads -- no album state rides the shared stream.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	openapi_types "github.com/oapi-codegen/runtime/types"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

const (
	// albumTitleMinRunes and albumTitleMaxRunes bound the album
	// title: trimmed, counted in characters (a byte bound would
	// shorten non-ASCII titles), pinned by the contract.
	albumTitleMinRunes = 1
	albumTitleMaxRunes = 64

	// albumItemsMaxIds caps the membership write's id list, pinned
	// by the contract: 200 ids keep one request a bounded statement
	// and a bounded report, the batch surface's own discipline.
	albumItemsMaxIds = 200
)

// publishAlbumEvent reports one album write onto the activity
// stream. The payload carries ONLY the album's id and the acting
// user's id: titles, covers, the deletability switch and the
// membership itself are album state the subscribers refetch, never
// bus payload -- the person.* events' ids-only discipline, applied
// to the album face. The nil guard matches the people publisher (a
// router wired without a bus is a wiring-order state, tests only).
func (s *server) publishAlbumEvent(name string, albumID, userID uuid.UUID) {
	if s.events == nil {
		return
	}
	s.events.Publish(name, map[string]string{
		"albumId": albumID.String(),
		"userId":  userID.String(),
	})
}

// publishAlbumMembershipEvents reports the album walls' convergence
// signal for every album whose LIVE membership the trash lifecycle
// just changed (the deletability chain): the changed ids'
// membership rows name the albums, and each album carries ONE
// album.items-changed event, deduplicated across the ids -- the
// rescan's own restart signal, published for the curation arms that
// move the live membership without touching its rows. The requester
// is the write's owner (the lifecycle arms are owner-scoped), so the
// payload's acting user is theirs. A read failure is logged and
// skipped: the asset events already carried the state change, and a
// wall that misses its restart signal converges on the next one.
func (s *server) publishAlbumMembershipEvents(ctx context.Context, requester uuid.UUID, ids []uuid.UUID) {
	if s.events == nil || len(ids) == 0 {
		return
	}
	seen := make(map[uuid.UUID]bool)
	for _, id := range ids {
		albums, err := s.queries.ListAlbumsOfAsset(ctx, pgUUID(id))
		if err != nil {
			s.logger.Warn("list the albums of a trash-lifecycle write", "error", err, "asset_id", id)
			continue
		}
		for _, albumID := range albums {
			album := uuid.UUID(albumID.Bytes)
			if seen[album] {
				continue
			}
			seen[album] = true
			s.publishAlbumEvent("album.items-changed", album, requester)
		}
	}
}

// albumTitle trims the title and bounds its rune count; it answers
// ok=false once it has written the 400.
func (s *server) albumTitle(w http.ResponseWriter, title string) (string, bool) {
	trimmed := strings.TrimSpace(title)
	if runes := utf8.RuneCountInString(trimmed); runes < albumTitleMinRunes || runes > albumTitleMaxRunes {
		s.writeValidationError(w, "title must span 1-64 characters", []string{"title"})
		return "", false
	}
	return trimmed, true
}

// albumWriteGate resolves the two-tier authorization answer every
// album write shares. The read is deliberately unfiltered by
// ownership: the row plus the caller's share flag come back for
// every non-sync album, and the refusal below, not the read, shapes
// the response -- the owner or an administrator proceeds (the admin
// role bypasses the owner check), a sharee answers the honest 403
// because the album is visible to them, and everyone else folds
// into the reads' byte-identical 404. It answers ok=false once it
// has written the response.
func (s *server) albumWriteGate(w http.ResponseWriter, r *http.Request, requester authContext, albumID uuid.UUID) (dbgen.GetAlbumForWriteRow, bool) {
	row, err := s.queries.GetAlbumForWrite(r.Context(), dbgen.GetAlbumForWriteParams{
		AlbumID: pgUUID(albumID),
		UserID:  pgUUID(requester.userID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		s.writeAlbumNotFound(w)
		return dbgen.GetAlbumForWriteRow{}, false
	}
	if err != nil {
		s.internalError(w, r, "load album for write", err)
		return dbgen.GetAlbumForWriteRow{}, false
	}
	if uuid.UUID(row.Album.OwnerID.Bytes) != requester.userID && requester.role != dbgen.UserRoleAdmin {
		if row.SharedToCaller {
			// The caller sees the album -- their share row exists --
			// so the write is an honest authorization failure, not a
			// missing resource: the share is view-only.
			s.writeError(w, http.StatusForbidden, apigen.AUTHFORBIDDEN,
				"only the album's owner may write it", nil)
			return dbgen.GetAlbumForWriteRow{}, false
		}
		s.writeAlbumNotFound(w)
		return dbgen.GetAlbumForWriteRow{}, false
	}
	return row, true
}

// albumCurationGate resolves the membership writes' gate: the write
// authorization and the virtual-only rule. Directory
// albums manage their own membership (the rescan flow of a later
// milestone) and sync albums are no album resources, so both fold
// into the single-shape 404. It answers ok=false once it has
// written the response.
func (s *server) albumCurationGate(w http.ResponseWriter, r *http.Request, requester authContext, albumID uuid.UUID) bool {
	gate, ok := s.albumWriteGate(w, r, requester, albumID)
	if !ok {
		return false
	}
	if gate.Album.Type != dbgen.AlbumTypeVirtual {
		s.writeAlbumNotFound(w)
		return false
	}
	return true
}

// albumSummaryCardForID reads one album's wall card for a write
// response: the row with its honest companions (the share badge, the
// live member count, the caller's timeline preference) plus the
// derived cover candidates when no pin exists -- the wall's own
// derivation, so the write's answer is the card GET /albums lists.
// The card's callerRole is owner: only an owner-or-admin write
// reaches this point, and the role is the write-affordance
// discriminator of the card shape. It answers ok=false once it has
// written the response.
func (s *server) albumSummaryCardForID(w http.ResponseWriter, r *http.Request, callerID, albumID uuid.UUID) (apigen.AlbumSummary, bool) {
	row, err := s.queries.GetAlbumSummary(r.Context(), dbgen.GetAlbumSummaryParams{
		AlbumID: pgUUID(albumID),
		UserID:  pgUUID(callerID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		s.writeAlbumNotFound(w)
		return apigen.AlbumSummary{}, false
	}
	if err != nil {
		s.internalError(w, r, "load album summary", err)
		return apigen.AlbumSummary{}, false
	}

	// The derived cover: the wall's own candidates query over the one
	// album, bounded at four rows by construction.
	var derived []uuid.UUID
	if !row.Album.CoverAssetID.Valid {
		candidates, err := s.queries.ListAlbumCoverCandidates(r.Context(), []pgtype.UUID{row.Album.ID})
		if err != nil {
			s.internalError(w, r, "load album cover candidates", err)
			return apigen.AlbumSummary{}, false
		}
		for _, candidate := range candidates {
			derived = append(derived, uuid.UUID(candidate.AssetID.Bytes))
		}
	}
	return albumSummaryCard(row.Album, row.IsShared, row.ItemCount,
		apigen.AlbumSummaryCallerRoleOwner, derived,
		timelinePrefField(row.Album.Type, "owner", row.InTimeline)), true
}

// albumItemsIDs validates the membership write's id list and
// collapses it to the distinct ids in first-appearance order (the
// batch surface's pinned pattern: the actions are set operations,
// and a duplicated id addresses one row). It answers ok=false once
// it has written the 400.
func (s *server) albumItemsIDs(w http.ResponseWriter, assetIds []openapi_types.UUID) ([]uuid.UUID, bool) {
	if len(assetIds) == 0 {
		s.writeValidationError(w, "assetIds must carry at least one asset id", []string{"assetIds"})
		return nil, false
	}
	if len(assetIds) > albumItemsMaxIds {
		s.writeValidationError(w,
			fmt.Sprintf("assetIds exceed the limit of %d", albumItemsMaxIds),
			[]string{"assetIds"})
		return nil, false
	}
	ids := make([]uuid.UUID, 0, len(assetIds))
	seen := make(map[uuid.UUID]bool, len(assetIds))
	for _, raw := range assetIds {
		id := uuid.UUID(raw)
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return ids, true
}

// CreateAlbum implements POST /api/v1/albums: the directory album
// creator, the albums-as-sources core. The request carries no type
// field -- this endpoint IS directory creation, the one creatable
// album type (the sync album is born per account, the virtual album
// is a legacy shape nothing creates anymore) -- and the source path
// runs the full validation matrix before anything is written: each
// failing branch answers its own 400 error code, and the album row
// stores the RESOLVED path. The face is the ADMINISTRATOR's (the
// boundary ruling: mounting an existing photo directory is a
// configuration act -- a plain member answers the admin surfaces'
// own 403 before any part of the request is considered, and the
// members' contribution path stays the upload stream). After the
// matrix passes, the write probe discovers the source's writability
// once; the fact lands on the row and rides the response -- a
// read-only directory is a legitimate source, and "photos of this
// directory cannot be deleted through the application" is honest
// creation feedback, never a failure. The request's audience lands
// in the same write through the full-set semantics of the audience
// face: unusable ids are skipped and reported, never errors. The
// create dialog's two switches ride the same request: the
// deletability switch lands on the row (on by default; a read-only
// source keeps the stored value governing nothing), and an
// `inTimeline: false` lands the CREATOR's own timeline-exclusion
// preference row in the creation itself -- the album is born out of
// its creator's aggregated timeline, the audience's own feeds
// governed by their own preference rows. The response is the fresh
// row's wall card -- the owner role, no members, no cover -- widened
// with the probe's fact and the honest audience report, and the
// write publishes album.created plus one album.shared per audience
// row that landed.
func (s *server) CreateAlbum(w http.ResponseWriter, r *http.Request) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}
	if requester.role != dbgen.UserRoleAdmin {
		// The configuration act's own refusal, the admin surfaces'
		// semantics: the member is signed in, just not entitled to
		// point the application at a filesystem location.
		s.writeError(w, http.StatusForbidden, apigen.AUTHFORBIDDEN,
			"administrator role required", nil)
		return
	}

	// The body is a small JSON document (the route group's 1 MiB cap
	// bounds it before this decode); anything that does not decode
	// is a client bug.
	var request apigen.AlbumCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeValidationError(w, "request body must be valid JSON", []string{"body"})
		return
	}
	title, ok := s.albumTitle(w, request.Title)
	if !ok {
		return
	}

	// The source path's shape arms: present, absolute, within the
	// path bound. The matrix's filesystem branches run after these,
	// in the domain function.
	if request.RootPath == "" {
		s.writeValidationError(w, "rootPath is required", []string{"rootPath"})
		return
	}
	if !filepath.IsAbs(request.RootPath) {
		s.writeValidationError(w, "rootPath must be an absolute path", []string{"rootPath"})
		return
	}
	if len(request.RootPath) > albumPathMaxBytes {
		s.writeValidationError(w, "rootPath exceeds the limit of 4096 bytes", []string{"rootPath"})
		return
	}
	if request.Audience != nil && len(*request.Audience) > albumAudienceMaxIds {
		s.writeValidationError(w, "audience exceeds the limit of 200", []string{"audience"})
		return
	}
	var audience []openapi_types.UUID
	if request.Audience != nil {
		audience = *request.Audience
	}

	ctx := r.Context()

	// The path matrix: existence, directory-ness, readability, the
	// managed tree on the resolved path. The answer is the resolved
	// path the row will store.
	resolved, failure, err := validateAlbumRootPath(s.dataDir, request.RootPath)
	if err != nil {
		s.internalError(w, r, "validate the source path", err)
		return
	}
	if failure != "" {
		if failure == apigen.VALIDATIONFAILED {
			s.writeValidationError(w, "rootPath is invalid", []string{"rootPath"})
		} else {
			s.writeAlbumPathError(w, failure)
		}
		return
	}

	// The creation-time write probe: the ONE moment the source's
	// writability is discovered. The answer is a fact, not a gate --
	// a read-only directory is a legitimate source whose photos
	// cannot be deleted through the application, and the response
	// says so instead of the creation failing.
	writable := probeDirectoryWritable(resolved)

	id, err := uuid.NewV7()
	if err != nil {
		s.internalError(w, r, "generate album id", err)
		return
	}
	if _, err := s.queries.CreateDirectoryAlbum(ctx, dbgen.CreateDirectoryAlbumParams{
		ID:      pgUUID(id),
		OwnerID: pgUUID(requester.userID),
		Title:   title,
		RootPath: pgtype.Text{
			String: resolved,
			Valid:  true,
		},
		SourceWritable:   writable,
		AllowAppDeletion: request.AllowAppDeletion == nil || *request.AllowAppDeletion,
	}); err != nil {
		s.internalError(w, r, "create the album", err)
		return
	}

	// The creator's timeline exclusion, carried from the dialog: the
	// per-user preference row is written HERE, in the creation, when
	// the request excludes the album -- absent or true writes
	// nothing (no row IS the feeding-the-timeline default, the
	// reset face's own semantics). The exclusion is the creator's
	// own state: the audience members' feeds are governed by their
	// own preference rows, never by this write.
	if request.InTimeline != nil && !*request.InTimeline {
		if _, err := s.queries.UpsertAlbumTimelinePreference(ctx, dbgen.UpsertAlbumTimelinePreferenceParams{
			UserID:     pgUUID(requester.userID),
			AlbumID:    pgUUID(id),
			InTimeline: false,
		}); err != nil {
			s.internalError(w, r, "record the timeline exclusion", err)
			return
		}
	}

	s.publishAlbumEvent("album.created", id, requester.userID)

	// The creation's audience lands through the same convergence the
	// audience face runs: the grants that land fire their events,
	// the unusable ids ride the response's skipped report.
	_, _, skipped, ok := s.applyAlbumAudience(w, r, id, requester.userID, requester.userID, distinctUserIDs(audience))
	if !ok {
		return
	}

	summary, ok := s.albumSummaryCardForID(w, r, requester.userID, id)
	if !ok {
		return
	}
	s.writeJSON(w, http.StatusCreated, albumCreateResponse(summary, writable, skipped))
}

// albumCreateResponse widens one wall card with the creation's two
// honest companions: the probe's source-writable fact (the
// "this directory is read-only" feedback the dialog renders -- a
// fact, never a failure) and the audience report. The summary
// fields copy verbatim (the generated allOf flattening makes the
// widening a field copy), and the skipped list rides along -- never
// nil, so the JSON carries the empty array.
func albumCreateResponse(card apigen.AlbumSummary, sourceWritable bool, skipped []apigen.AlbumAudienceSkip) apigen.AlbumCreateResponse {
	response := apigen.AlbumCreateResponse{
		Id:             card.Id,
		Title:          card.Title,
		Type:           apigen.AlbumCreateResponseType(card.Type),
		IsShared:       card.IsShared,
		CoverAssetIds:  card.CoverAssetIds,
		ItemCount:      card.ItemCount,
		UpdatedAt:      card.UpdatedAt,
		CallerRole:     apigen.AlbumCreateResponseCallerRole(card.CallerRole),
		SourceWritable: sourceWritable,
		Skipped:        skipped,
	}
	if response.Skipped == nil {
		response.Skipped = []apigen.AlbumAudienceSkip{}
	}
	if card.Owner != nil {
		response.Owner = card.Owner
	}
	if card.InTimeline != nil {
		response.InTimeline = card.InTimeline
	}
	return response
}

// UpdateAlbum implements PATCH /api/v1/albums/{albumId}: the owner's
// partial edit of the properties -- any subset, at least one
// field, JSON null counting as absent. The cover validation refuses
// every id that is not a live member of the album itself, and the
// deletability switch is the directory album's own (a virtual album
// answers the field's own 400, not a generic refusal). The write
// publishes album.updated unconditionally on success -- the
// convergent refetch makes the unconditional fire harmless -- and
// the response is the album's wall card after the edit.
func (s *server) UpdateAlbum(w http.ResponseWriter, r *http.Request, albumId openapi_types.UUID) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	var request apigen.AlbumPatch
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeValidationError(w, "request body must be valid JSON", []string{"body"})
		return
	}
	if request.Title == nil &&
		request.CoverAssetId == nil && request.AllowAppDeletion == nil {
		s.writeValidationError(w, "the patch must carry a field", []string{"body"})
		return
	}

	ctx := r.Context()

	gate, ok := s.albumWriteGate(w, r, requester, uuid.UUID(albumId))
	if !ok {
		return
	}

	// The deletability switch is the directory album's own field: a
	// virtual album's deletions never move files, so the switch has
	// no home there -- the refusal names the field's own condition.
	if request.AllowAppDeletion != nil && gate.Album.Type != dbgen.AlbumTypeDirectory {
		s.writeMessageKey(w, "the deletability switch is a directory album's own setting", "album_deletable_not_directory")
		return
	}

	// The cover validation, the application-layer check: a
	// present cover must be a LIVE member of THIS album. The check
	// runs before the write so an invalid patch lands nothing.
	if request.CoverAssetId != nil {
		matches, err := s.queries.CountLiveAlbumMember(ctx, dbgen.CountLiveAlbumMemberParams{
			AlbumID: pgUUID(albumId),
			AssetID: pgUUID(uuid.UUID(*request.CoverAssetId)),
		})
		if err != nil {
			s.internalError(w, r, "validate the album cover", err)
			return
		}
		if matches == 0 {
			s.writeMessageKey(w, "the cover must be a live member of this album", "album_cover_not_member")
			return
		}
	}

	// The tri-state patch: an absent field passes NULL and the
	// statement's COALESCE keeps the stored value.
	params := dbgen.UpdateAlbumFieldsParams{AlbumID: pgUUID(albumId)}
	if request.Title != nil {
		title, ok := s.albumTitle(w, *request.Title)
		if !ok {
			return
		}
		params.Title = pgtype.Text{String: title, Valid: true}
	}
	if request.CoverAssetId != nil {
		params.CoverAssetID = pgUUID(uuid.UUID(*request.CoverAssetId))
	}
	if request.AllowAppDeletion != nil {
		// The owner's deletability posture, the edit dialog's switch:
		// repeating the stored value converges through the COALESCE
		// write like every property field.
		params.AllowAppDeletion = pgtype.Bool{Bool: *request.AllowAppDeletion, Valid: true}
	}

	if _, err := s.queries.UpdateAlbumFields(ctx, params); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// The album vanished between the gate and the write.
			s.writeAlbumNotFound(w)
			return
		}
		s.internalError(w, r, "patch the album", err)
		return
	}

	s.publishAlbumEvent("album.updated", uuid.UUID(albumId), requester.userID)

	summary, ok := s.albumSummaryCardForID(w, r, requester.userID, uuid.UUID(albumId))
	if !ok {
		return
	}
	s.writeJSON(w, http.StatusOK, summary)
}

// DeleteAlbum implements DELETE /api/v1/albums/{albumId}: the album
// removal, two arms behind one gate. The VIRTUAL arm is the
// reference set's plain delete: the row goes and every membership
// and share row follows through the schema's cascades, the members'
// asset rows untouched. The DIRECTORY arm is the un-ingest (see
// albumremoval.go): the album's own rows go in the same single
// transaction as every member asset no other album references --
// the guard's survivors keep their rows, and the source
// files of the purged stay on the mount whatever happens to the
// library data. Both arms answer 204 with no body and publish
// album.deleted; the directory arm additionally publishes
// asset.deleted for every purged non-component member after the
// commit, so the timeline, the search and the people surfaces drop
// their tiles while the file half is still running. The sync album
// folds into the single-shape 404 at the gate (it is no album
// resource), and a concurrent removal that won the race answers the
// same 404 like an unknown id.
func (s *server) DeleteAlbum(w http.ResponseWriter, r *http.Request, albumId openapi_types.UUID) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	gate, ok := s.albumWriteGate(w, r, requester, uuid.UUID(albumId))
	if !ok {
		return
	}

	if gate.Album.Type == dbgen.AlbumTypeVirtual {
		deleted, err := s.queries.DeleteVirtualAlbum(r.Context(), pgUUID(albumId))
		if err != nil {
			s.internalError(w, r, "delete the album", err)
			return
		}
		if deleted == 0 {
			// A concurrent delete won the race: the album is gone
			// exactly like an unknown id.
			s.writeAlbumNotFound(w)
			return
		}

		s.publishAlbumEvent("album.deleted", uuid.UUID(albumId), requester.userID)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// The directory arm: the un-ingest. The database half is one
	// transaction (the album's rows plus every guarded member's
	// purge); the events and the file half follow the commit --
	// the events first, because they announce rows that are already
	// committed and the file half may be slow; the files best-effort,
	// because a missed file is inert garbage, never a row that names
	// it.
	purged, ok, err := s.removeDirectoryAlbum(r.Context(), uuid.UUID(albumId))
	if err != nil {
		s.internalError(w, r, "remove the album", err)
		return
	}
	if !ok {
		// A concurrent removal won the race: the album is gone
		// exactly like an unknown id.
		s.writeAlbumNotFound(w)
		return
	}

	s.publishAlbumEvent("album.deleted", uuid.UUID(albumId), requester.userID)
	for _, member := range purged {
		if !member.component {
			// The component rows stay silent (the purge events'
			// standing discipline): no client view ever listed one,
			// so an announcement for it is noise no client can act
			// on -- the row still went.
			s.publishAssetEventByID("asset.deleted", member.id, member.kind)
		}
		s.removePurgedMemberFiles(r, member)
	}
	w.WriteHeader(http.StatusNoContent)
}

// RemoveAlbumItems implements DELETE /api/v1/albums/{albumId}/items:
// the membership remove. The rows that exist leave; the ids that are
// no members are the skipped of the honest report. The removal
// carries no live predicate -- it tidies the set itself, so a
// trashed member's row still leaves -- and the write publishes
// album.items-changed exactly when at least one row left.
func (s *server) RemoveAlbumItems(w http.ResponseWriter, r *http.Request, albumId openapi_types.UUID) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	var request apigen.AlbumItemsRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeValidationError(w, "request body must be valid JSON", []string{"body"})
		return
	}
	ids, ok := s.albumItemsIDs(w, request.AssetIds)
	if !ok {
		return
	}

	if !s.albumCurationGate(w, r, requester, uuid.UUID(albumId)) {
		return
	}

	removed, err := s.queries.RemoveAlbumItems(r.Context(), dbgen.RemoveAlbumItemsParams{
		AlbumID:  pgUUID(albumId),
		AssetIds: pgUUIDs(ids),
	})
	if err != nil {
		s.internalError(w, r, "remove the album items", err)
		return
	}

	if removed > 0 {
		s.publishAlbumEvent("album.items-changed", uuid.UUID(albumId), requester.userID)
	}

	s.writeJSON(w, http.StatusOK, apigen.RemoveAlbumItemsResponse{
		Removed: int(removed),
		Skipped: len(ids) - int(removed),
	})
}
