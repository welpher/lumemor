// The account disposal: DELETE /admin/users/{id}, the transaction
// behind the admin users page's two-step delete. The account row goes
// immediately -- there is no soft delete and no recovery window for
// the account itself -- and everything it held is settled before it:
//
//   - The TRANSFER arm hands the departing account's content to
//     another account: the photos (live, trashed, hidden and the Live
//     Photo components alike), the albums and memories, and the tags
//     attached to them. The departing stream folds into the receiver's
//     own stream, so the photos keep their place in the family
//     timeline. Photo content the receiver already holds -- in ANY
//     state, because the storage's uniqueness rule is owner plus
//     checksum over the live and the trashed rows alike -- is a
//     collision the ownership write cannot carry: it folds into the
//     receiver's copy under the survivor rule (see
//     foldDuplicateAsset), which keeps the live copy of a mixed pair
//     and the receiver's own row on a tie, moving the losing copy's
//     memberships, tags, favorites and album cover onto the survivor.
//     The share grants on the departing account's albums ride the
//     container: the receiver carries them and the grantees keep
//     their access.
//
//   - The TRASH arm walks the photos into the CALLING administrator's
//     trash, where the instance's retention window governs them. The
//     departing account's own albums and memories go with it (a
//     container whose owner is gone has no reader), so the photos
//     survive as individual trash items; an external row's source
//     file moves into the server's trash area exactly like any other
//     trash move, and the family's other source files are never
//     touched. The move is the one step the transaction cannot do
//     itself (it needs the operating system's permission), so it runs
//     ahead of the row writes and EVERY failure after the first move
//     compensates the moves already performed before the transaction
//     rolls back -- the re-own's own failures inside the arm as well
//     as the account row's delete and the commit after it. Content the
//     calling administrator already holds collides here exactly as it
//     does on the transfer arm (the same total index), so the caller's
//     own copy is the fold's destination owner.
//
// Both arms settle the rows that would otherwise block the account's
// deletion -- the photos, the albums, the tag rows, the authorship of
// album memberships and the grants on its albums all carry a foreign
// key without a cascade -- and let the schema's cascades take what
// lives with the account alone: the sessions, API keys and pairing
// tokens (it can never authenticate again), the favorites, the
// per-album timeline preferences, the private names it gave shared
// face clusters, and every album share granted TO it.
//
// The transaction is the whole story: it commits whole or rolls back
// whole, so a failure at any step leaves the account and its content
// exactly as they were. The last-administrator guard and the caller's
// own account are refused before the transaction's writes -- an
// instance without an administrator strands this surface, and the
// disposal hands content to a survivor the caller cannot be.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	oapitypes "github.com/oapi-codegen/runtime/types"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/storage"
)

// The disposal's own refusals, mapped onto their envelopes by the
// handler: the last administrator (409), a receiver that is the
// account being deleted (409), an unknown receiver (400 naming the
// field), a vanished account row (404, the stale-list race) and the
// external trash move's own explicit failure.
var (
	errRemovalLastAdmin      = errors.New("the instance's last administrator cannot be deleted")
	errRemovalSelf           = errors.New("an administrator cannot delete their own account")
	errRemovalTransferToSelf = errors.New("the receiving account must be another account")
	errRemovalUnknownTarget  = errors.New("transferTo names no account")
	errRemovalUserGone       = errors.New("no account with this identifier")
	errRemovalTrashMove      = errors.New("an external file could not be moved into the trash")
)

// userDisposal is the validated disposition of one deletion: exactly
// one arm is set.
type userDisposal struct {
	// transferTo names the receiving account (the transfer arm).
	transferTo uuid.UUID
	// trash is the trash arm: the caller's own trash receives the
	// photos.
	trash bool
}

// userRemovalOutcome carries the post-commit work: the events the
// two arms announce, and the file half of the folded duplicates (the
// files follow the commit -- a crash in between leaves inert files
// rather than a row that names a missing file).
type userRemovalOutcome struct {
	// deletedAlbums are the containers the trash arm removed (the
	// events announce them: the account that owned them is gone).
	deletedAlbums []uuid.UUID
	// movedAlbums are the containers the transfer arm handed over.
	movedAlbums []uuid.UUID
	// trashedAssets are the live photos the trash arm moved, with
	// their media kinds (the events' payload).
	trashedAssets map[uuid.UUID]dbgen.MediaKind
	// purged are the collision fold's losing rows, one entry per
	// purged row (a purged photo's Live Photo component is an entry of
	// its own, flagged silent); the events of the non-component ones
	// and the post-commit file half both read this list.
	purged []purgedMember
}

// DeleteUser implements DELETE /api/v1/admin/users/{id}: the account
// disposal. The body names the disposition of the departing account's
// photos -- transferTo another account, or dispose: "trash" into the
// caller's trash. The response is the bare 204: the account is gone.
func (s *server) DeleteUser(w http.ResponseWriter, r *http.Request, id oapitypes.UUID) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	var request apigen.AdminUserDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
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

	disposal, ok := s.userDisposal(w, request)
	if !ok {
		return
	}

	target := uuid.UUID(id)
	if !disposal.trash && disposal.transferTo == target {
		s.writeError(w, http.StatusConflict, apigen.USERTRANSFERTOSELF,
			"the receiving account must be another account", nil)
		return
	}

	outcome, err := s.removeUser(r, requester.userID, target, disposal)
	switch {
	case errors.Is(err, errRemovalUserGone):
		s.writeError(w, http.StatusNotFound, apigen.USERNOTFOUND,
			"no account with this identifier", nil)
		return
	case errors.Is(err, errRemovalLastAdmin):
		s.writeError(w, http.StatusConflict, apigen.USERLASTADMIN,
			"the last administrator cannot be deleted", nil)
		return
	case errors.Is(err, errRemovalTransferToSelf):
		s.writeError(w, http.StatusConflict, apigen.USERTRANSFERTOSELF,
			"the receiving account must be another account", nil)
		return
	case errors.Is(err, errRemovalSelf):
		// The caller's own account: the disposal hands the content to
		// a survivor, and the caller performs the handover. The web
		// client cannot know its own user id through this surface (the
		// listing carries no caller marker), so this guard is the
		// authoritative one and its mapped message is the visible half
		// -- the self-disable guard's own shape. It answers after the
		// last-administrator guard, so the instance's only
		// administrator gets that refusal instead (the narrower truth:
		// the instance, not the caller, is what blocks it).
		details := map[string]interface{}{"fields": []interface{}{"id"}}
		s.writeError(w, http.StatusBadRequest, apigen.VALIDATIONFAILED,
			"an administrator cannot delete their own account", &details)
		return
	case errors.Is(err, errRemovalUnknownTarget):
		details := map[string]interface{}{"fields": []interface{}{"transferTo"}}
		s.writeError(w, http.StatusBadRequest, apigen.VALIDATIONFAILED,
			"transferTo must name another account", &details)
		return
	case errors.Is(err, errRemovalTrashMove):
		s.logger.Error("the disposal's external trash move failed", "error", err,
			"user_id", target, "method", r.Method, "path", r.URL.Path)
		s.writeError(w, http.StatusInternalServerError, apigen.ASSETTRASHMOVEFAILED,
			"the file could not be moved into the trash", nil)
		return
	case err != nil:
		s.internalError(w, r, "remove user", err)
		return
	}

	// The rows are committed gone; the events follow. The transfer arm
	// announces the containers it moved (their walls and audiences
	// converge through their own reads); the trash arm announces the
	// containers it removed and every photo whose state transitioned;
	// both arms announce every row the collision fold purged, because a
	// client holding that id has to let it go (the folded row is gone
	// and no view will serve it again). The component rows stay silent:
	// no client view ever listed a Live Photo's video half.
	for _, album := range outcome.movedAlbums {
		s.publishAlbumEvent("album.updated", album, requester.userID)
	}
	for _, album := range outcome.deletedAlbums {
		s.publishAlbumEvent("album.deleted", album, requester.userID)
	}
	for asset, kind := range outcome.trashedAssets {
		s.publishAssetEventByID("asset.trashed", asset, kind)
	}
	for _, member := range outcome.purged {
		if !member.component {
			s.publishAssetEventByID("asset.deleted", member.id, member.kind)
		}
		s.removePurgedMemberFiles(r, member)
	}
	w.WriteHeader(http.StatusNoContent)
}

// userDisposal validates the request body's disposition: exactly one
// arm, and the trash arm's enum value must be the one the contract
// names. It answers ok=false once it has written the 400.
func (s *server) userDisposal(w http.ResponseWriter, request apigen.AdminUserDeleteRequest) (userDisposal, bool) {
	switch {
	case request.TransferTo != nil && request.Dispose != nil:
		s.writeValidationError(w, "exactly one of transferTo and dispose is required",
			[]string{"transferTo", "dispose"})
		return userDisposal{}, false
	case request.TransferTo != nil:
		return userDisposal{transferTo: uuid.UUID(*request.TransferTo)}, true
	case request.Dispose != nil:
		if !request.Dispose.Valid() {
			s.writeValidationError(w, "dispose must be \"trash\"", []string{"dispose"})
			return userDisposal{}, false
		}
		return userDisposal{trash: true}, true
	default:
		s.writeValidationError(w, "exactly one of transferTo and dispose is required",
			[]string{"transferTo", "dispose"})
		return userDisposal{}, false
	}
}

// removeUser runs the disposal's ONE transaction: the locks (the
// departing row, the receiving one on the transfer arm, and the
// instance's administrator rows), the guards the locks make race-free,
// the arm's settlement, and the account row's deletion. The trash
// arm's external file moves run inside it (ahead of the row writes
// they describe) and are compensated when anything after them fails,
// so a rolled-back disposal leaves no file moved.
func (s *server) removeUser(r *http.Request, actor, target uuid.UUID, disposal userDisposal) (userRemovalOutcome, error) {
	var outcome userRemovalOutcome
	ctx := r.Context()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return outcome, fmt.Errorf("begin the account disposal transaction: %w", err)
	}
	// The rollback is a no-op after a successful commit; the context
	// is captured while the request is still live.
	defer func() { _ = tx.Rollback(ctx) }()
	queries := dbgen.New(tx)

	// The rows this disposal decides on are locked in one canonical id
	// order: the departing account and, on the transfer arm, the
	// receiving one. Two crossed disposals -- each account handing its
	// content to the other -- then queue behind one lock order instead
	// of deadlocking on each other.
	locked := make([]uuid.UUID, 0, 2)
	locked = append(locked, target)
	if !disposal.trash {
		locked = append(locked, disposal.transferTo)
	}
	sort.Slice(locked, func(i, j int) bool { return locked[i].String() < locked[j].String() })
	for _, id := range locked {
		row, err := queries.LockUserForRemoval(ctx, pgUUID(id))
		if errors.Is(err, pgx.ErrNoRows) {
			if id == target {
				return outcome, errRemovalUserGone
			}
			return outcome, errRemovalUnknownTarget
		}
		if err != nil {
			return outcome, fmt.Errorf("lock account %s: %w", id, err)
		}
		if id == target && row.Role == dbgen.UserRoleAdmin {
			// The last-administrator guard reads the administrator rows
			// under the same lock, so two concurrent disposals cannot
			// both see a survivor: the second waits for the first and
			// then counts the reduced set. One administrator is the
			// line.
			admins, err := queries.LockAdministratorIDs(ctx)
			if err != nil {
				return outcome, fmt.Errorf("lock the instance's administrators: %w", err)
			}
			if len(admins) <= 1 {
				return outcome, errRemovalLastAdmin
			}
		}
	}

	// The caller's own account, refused after the last-administrator
	// guard above: the disposal hands the content to a survivor, and
	// the caller is the one performing the handover -- the trash arm
	// would otherwise have no owner to hand its photos to. The only
	// administrator therefore hears the narrower refusal (the
	// instance is what blocks it), and every other self-delete hears
	// this one.
	if target == actor {
		return outcome, errRemovalSelf
	}

	if disposal.trash {
		outcome, err = s.trashUserContent(r, queries, actor, target)
	} else {
		outcome, err = s.transferUserContent(ctx, queries, target, disposal.transferTo)
	}
	if err != nil {
		return userRemovalOutcome{}, err
	}

	deleted, err := queries.DeleteUserRow(ctx, pgUUID(target))
	if err != nil {
		return userRemovalOutcome{}, fmt.Errorf("delete the account row: %w", err)
	}
	if deleted == 0 {
		// A concurrent disposal of the same account won the race and
		// its cascade reached this row first: the account is gone
		// exactly like an unknown id.
		return userRemovalOutcome{}, errRemovalUserGone
	}

	if err := tx.Commit(ctx); err != nil {
		return userRemovalOutcome{}, fmt.Errorf("commit the account disposal: %w", err)
	}
	return outcome, nil
}

// transferUserContent settles the departing account's content onto the
// receiving account: the stream fold, the collision set, the photo
// handover, the container handover, and the authorship and label rows
// that would otherwise block the account's deletion.
func (s *server) transferUserContent(ctx context.Context, queries *dbgen.Queries, from, to uuid.UUID) (userRemovalOutcome, error) {
	var outcome userRemovalOutcome

	albums, err := queries.ListOwnedAlbumIDs(ctx, pgUUID(from))
	if err != nil {
		return outcome, fmt.Errorf("list the departing account's albums: %w", err)
	}

	// The stream fold: the departing account's own stream is where
	// every upload landed, so its memberships are what keep the
	// photos in the family timeline. They move to the receiver's
	// stream and the emptied row goes; an account whose receiver
	// carries no stream of its own hands the whole row over instead.
	if err := s.foldUserStream(ctx, queries, from, to); err != nil {
		return outcome, err
	}

	// The collision set: content the receiver already holds in ANY
	// state folds into a single surviving row -- the losing copy's
	// memberships, tags, favorites and album cover move across and the
	// losing row goes through the permanent delete's own sweep (the
	// face clusters re-home and dissolve before their faces cascade,
	// exactly like the trash expiry's purge). The survivor rule lives
	// in foldDuplicateAsset.
	duplicates, err := queries.DuplicateOwnedAssets(ctx, dbgen.DuplicateOwnedAssetsParams{
		ToOwner:   pgUUID(to),
		FromOwner: pgUUID(from),
	})
	if err != nil {
		return outcome, fmt.Errorf("resolve the content the receiver already holds: %w", err)
	}
	for _, duplicate := range duplicates {
		purged, err := s.foldDuplicateAsset(ctx, queries, duplicate)
		if err != nil {
			return outcome, err
		}
		outcome.purged = append(outcome.purged, purged...)
	}

	// The device identity's own axis, which the checksum fold cannot
	// cover (two rows may carry one device id with different content):
	// the partial (owner, device asset) index allows one LIVE row per
	// pair, so an arriving live row releases an identity the receiver
	// already holds live.
	if err := queries.ReleaseArrivingDeviceIdentity(ctx, dbgen.ReleaseArrivingDeviceIdentityParams{
		FromOwner: pgUUID(from),
		ToOwner:   pgUUID(to),
	}); err != nil {
		return outcome, fmt.Errorf("release the colliding device identities: %w", err)
	}

	if err := queries.TransferAssetOwnership(ctx, dbgen.TransferAssetOwnershipParams{
		ToOwner:   pgUUID(to),
		FromOwner: pgUUID(from),
	}); err != nil {
		return outcome, fmt.Errorf("hand the photos over: %w", err)
	}

	if err := queries.TransferAlbumOwnership(ctx, dbgen.TransferAlbumOwnershipParams{
		ToOwner:   pgUUID(to),
		FromOwner: pgUUID(from),
	}); err != nil {
		return outcome, fmt.Errorf("hand the albums over: %w", err)
	}

	// The rows that carry the departing account's identity elsewhere:
	// a membership it added to another account's album keeps its place
	// (a photo must not leave an album it sits in), and the grants it
	// issued keep their authority under the album's new owner.
	if err := queries.ReassignAlbumItemAuthorship(ctx, dbgen.ReassignAlbumItemAuthorshipParams{
		ToUser:   pgUUID(to),
		FromUser: pgUUID(from),
	}); err != nil {
		return outcome, fmt.Errorf("carry the album authorship over: %w", err)
	}
	if err := queries.ReassignShareGrants(ctx, dbgen.ReassignShareGrantsParams{
		ToUser:   pgUUID(to),
		FromUser: pgUUID(from),
	}); err != nil {
		return outcome, fmt.Errorf("carry the share grants over: %w", err)
	}

	// The labels: the tags are attached to photos the receiver now
	// holds, so they move with them -- a tag the receiver already
	// carries by name folds into the receiver's own.
	if err := queries.MergeDuplicateTags(ctx, dbgen.MergeDuplicateTagsParams{
		ToOwner:   pgUUID(to),
		FromOwner: pgUUID(from),
	}); err != nil {
		return outcome, fmt.Errorf("fold the labels the receiver already carries: %w", err)
	}
	if err := queries.DropMergedTags(ctx, dbgen.DropMergedTagsParams{
		ToOwner:   pgUUID(to),
		FromOwner: pgUUID(from),
	}); err != nil {
		return outcome, fmt.Errorf("drop the folded labels: %w", err)
	}
	if err := queries.TransferTagOwnership(ctx, dbgen.TransferTagOwnershipParams{
		ToOwner:   pgUUID(to),
		FromOwner: pgUUID(from),
	}); err != nil {
		return outcome, fmt.Errorf("hand the labels over: %w", err)
	}

	for _, album := range albums {
		// The stream is not an album resource: it was folded away
		// above (or handed over whole), so an updated event for it
		// would name a row no client can open.
		if album.Type == dbgen.AlbumTypeSync {
			continue
		}
		outcome.movedAlbums = append(outcome.movedAlbums, uuid.UUID(album.ID.Bytes))
	}
	return outcome, nil
}

// foldUserStream settles the departing account's own stream: its
// memberships move into the receiver's stream (the photos keep their
// place in the timeline) and the emptied row goes. An account whose
// receiver carries no stream of its own -- a hand-made account, or an
// instance whose stream was never created -- hands the row over
// whole, which is the same continuity by a shorter path.
func (s *server) foldUserStream(ctx context.Context, queries *dbgen.Queries, from, to uuid.UUID) error {
	source, err := queries.DisposalSyncAlbum(ctx, pgUUID(from))
	if errors.Is(err, pgx.ErrNoRows) {
		// No stream of its own: nothing to fold. Every account this
		// surface creates has one; a hand-made row is the only case.
		return nil
	}
	if err != nil {
		return fmt.Errorf("resolve the departing account's stream: %w", err)
	}

	receiver, err := queries.DisposalSyncAlbum(ctx, pgUUID(to))
	if errors.Is(err, pgx.ErrNoRows) {
		if _, err := queries.ReassignAlbumOwner(ctx, dbgen.ReassignAlbumOwnerParams{
			ID:      source,
			OwnerID: pgUUID(to),
		}); err != nil {
			return fmt.Errorf("hand the departing stream over: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("resolve the receiving account's stream: %w", err)
	}

	if err := queries.FoldSyncAlbumMembership(ctx, dbgen.FoldSyncAlbumMembershipParams{
		AlbumID:       receiver,
		SourceAlbumID: source,
		AddedBy:       pgUUID(to),
	}); err != nil {
		return fmt.Errorf("fold the departing stream into the receiver's: %w", err)
	}
	if _, err := queries.DeleteSyncAlbumOfOwner(ctx, pgUUID(from)); err != nil {
		return fmt.Errorf("delete the folded stream: %w", err)
	}
	return nil
}

// foldDuplicateAsset settles one collision of the disposal's fold: the
// departing row and the destination owner's row hold the same content,
// and the total (owner, checksum) index -- live and trashed rows alike
// -- leaves room for only one of them once the ownership write runs.
// The survivor rule is one law for both arms:
//
//   - The LIVE row wins a mixed pair. Content that arrives live stays
//     visible under the destination owner (an earlier trash decision
//     about the same content does not bind it -- whoever wants it gone
//     can trash it again), so a live departing row survives a trashed
//     destination copy, and a live destination copy survives a trashed
//     departing row.
//   - A tie keeps the DESTINATION's row: both live keeps the receiver's
//     copy (its file, its readiness, its place), both trashed keeps it
//     with the LATER of the two deletion marks (the family reading is
//     the generous one: the content stays recoverable as long as either
//     copy had left).
//
// The loser's context moves onto the survivor first -- album
// memberships, tag links, favorites and any album cover it pinned --
// because that context describes the content, not the copy. Then the
// loser is purged through the permanent delete's own sweep, and a
// purged photo's Live Photo component goes with it: a component row is
// not content of its own, and left behind it would outlive the photo
// that names it as an orphan row no view can reach.
func (s *server) foldDuplicateAsset(ctx context.Context, queries *dbgen.Queries, duplicate dbgen.DuplicateOwnedAssetsRow) ([]purgedMember, error) {
	departing := duplicate.DepartingID
	existing := duplicate.ExistingID
	departingLive := duplicate.DepartingLive
	existingLive := !duplicate.ExistingDeletedAt.Valid
	// The survivor rule: the live row wins a mixed pair, and a tie --
	// both live, or both trashed -- keeps the destination's row.
	survivor, loser := existing, departing
	if departingLive != existingLive && departingLive {
		survivor, loser = departing, existing
	}

	if err := queries.MoveAssetAlbumMemberships(ctx, dbgen.MoveAssetAlbumMembershipsParams{
		ToAsset:   survivor,
		FromAsset: loser,
	}); err != nil {
		return nil, fmt.Errorf("move the folded photo's album memberships: %w", err)
	}
	if err := queries.MoveAssetTagAssignments(ctx, dbgen.MoveAssetTagAssignmentsParams{
		ToAsset:   survivor,
		FromAsset: loser,
	}); err != nil {
		return nil, fmt.Errorf("move the folded photo's tags: %w", err)
	}
	if err := queries.MoveAssetFavorites(ctx, dbgen.MoveAssetFavoritesParams{
		ToAsset:   survivor,
		FromAsset: loser,
	}); err != nil {
		return nil, fmt.Errorf("move the folded photo's favorites: %w", err)
	}
	if err := queries.MoveAlbumCovers(ctx, dbgen.MoveAlbumCoversParams{
		ToAsset:   survivor,
		FromAsset: loser,
	}); err != nil {
		return nil, fmt.Errorf("move the folded photo's album cover: %w", err)
	}

	// The both-trashed tie keeps the later countdown: the survivor's
	// mark reaches forward to the departing row's when that one left
	// later (GREATEST leaves the survivor's own mark alone otherwise).
	if duplicate.DepartingDeletedAt.Valid && duplicate.ExistingDeletedAt.Valid {
		if err := queries.ExtendAssetTrashMark(ctx, dbgen.ExtendAssetTrashMarkParams{
			ID:        survivor,
			DeletedAt: duplicate.DepartingDeletedAt,
		}); err != nil {
			return nil, fmt.Errorf("extend the folded photo's trash mark: %w", err)
		}
	}

	row, err := queries.GetAssetByID(ctx, loser)
	if err != nil {
		return nil, fmt.Errorf("load the folded photo: %w", err)
	}
	members := make([]purgedMember, 0, 2)
	photo, err := s.purgeFoldedRow(ctx, queries, row)
	if err != nil {
		return nil, err
	}
	members = append(members, photo)

	if !row.LivePhotoVideoID.Valid {
		return members, nil
	}
	// The pair discipline, in the permanent delete's own order: the
	// photo row goes first (the component delete would otherwise trip
	// the foreign key the photo still holds on it), then the component
	// row and its files.
	component, err := queries.GetAssetByID(ctx, row.LivePhotoVideoID)
	if errors.Is(err, pgx.ErrNoRows) {
		return members, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load the folded photo's Live Photo component: %w", err)
	}
	purgedComponent, err := s.purgeFoldedRow(ctx, queries, component)
	if err != nil {
		return nil, err
	}
	return append(members, purgedComponent), nil
}

// purgeFoldedRow purges one losing row through the permanent delete's
// own order and answers its post-commit facts: the shared-blob
// reference count first (taken before the row goes; it excludes the row
// itself, so it answers the same either side), the cluster sweep (a
// representative face re-homes before the cascade removes it, a cluster
// left without members dissolves with its per-user labels), then the
// row -- the schema's cascades take the faces, the exif, the embeddings
// and the OCR text. An external row counts no file reference: the only
// file of ours it names is the trash copy keyed by the id, and the
// family's source file is never touched.
func (s *server) purgeFoldedRow(ctx context.Context, queries *dbgen.Queries, row dbgen.Asset) (purgedMember, error) {
	assetID := uuid.UUID(row.ID.Bytes)
	var fileRefs int64
	if row.StorageKind != dbgen.StorageKindExternal {
		var err error
		if fileRefs, err = queries.CountAssetFileReferences(ctx, dbgen.CountAssetFileReferencesParams{
			FilePath: row.FilePath,
			ID:       row.ID,
		}); err != nil {
			return purgedMember{}, fmt.Errorf("count the file references of the folded photo %s: %w", assetID, err)
		}
	}

	if err := queries.RehomeClusterRepresentatives(ctx, dbgen.RehomeClusterRepresentativesParams{
		AssetID: row.ID,
	}); err != nil {
		return purgedMember{}, fmt.Errorf("rehome the clusters of the folded photo's %s faces: %w", assetID, err)
	}
	if err := queries.DissolveVacatedClusters(ctx, dbgen.DissolveVacatedClustersParams{
		AssetID: row.ID,
	}); err != nil {
		return purgedMember{}, fmt.Errorf("dissolve the clusters the folded photo's %s faces held: %w", assetID, err)
	}
	if _, err := queries.DeleteAssetPermanently(ctx, row.ID); err != nil {
		return purgedMember{}, fmt.Errorf("delete the folded photo's row %s: %w", assetID, err)
	}

	return purgedMember{
		id:        assetID,
		kind:      row.MediaKind,
		component: row.IsComponent,
		external:  row.StorageKind == dbgen.StorageKindExternal,
		filePath:  row.FilePath,
		trashPath: row.TrashPath.String,
		fileRefs:  fileRefs,
	}, nil
}

// trashUserContent walks the departing account's photos into the
// caller's trash: content the caller already holds folds first (the
// same collision set and survivor rule as the transfer arm, onto the
// acting administrator), the albums and memories go with the account
// (their memberships, shares and preferences cascade), the photos
// re-own to the caller with their deletion marks, and an external row's
// source file moves into the server's trash area ahead of its write.
func (s *server) trashUserContent(r *http.Request, queries *dbgen.Queries, actor, from uuid.UUID) (userRemovalOutcome, error) {
	outcome := userRemovalOutcome{trashedAssets: make(map[uuid.UUID]dbgen.MediaKind)}
	ctx := r.Context()

	// The collision set first, before any file moves and before the
	// rows it purges are listed for their events: the total (owner,
	// checksum) index refuses a re-own onto an owner who holds the same
	// content in ANY state, so the acting administrator's own copy is
	// the fold's destination. A purged departing row is gone before the
	// external listing below, so its source file is never moved for a
	// row that no longer exists.
	duplicates, err := queries.DuplicateOwnedAssets(ctx, dbgen.DuplicateOwnedAssetsParams{
		ToOwner:   pgUUID(actor),
		FromOwner: pgUUID(from),
	})
	if err != nil {
		return outcome, fmt.Errorf("resolve the content the caller already holds: %w", err)
	}
	for _, duplicate := range duplicates {
		purged, err := s.foldDuplicateAsset(ctx, queries, duplicate)
		if err != nil {
			return outcome, err
		}
		outcome.purged = append(outcome.purged, purged...)
	}

	albums, err := queries.ListOwnedAlbumIDs(ctx, pgUUID(from))
	if err != nil {
		return outcome, fmt.Errorf("list the departing account's albums: %w", err)
	}
	assets, err := queries.ListLiveOwnedAssets(ctx, pgUUID(from))
	if err != nil {
		return outcome, fmt.Errorf("list the departing account's photos: %w", err)
	}
	externals, err := queries.ListLiveExternalAssets(ctx, pgUUID(from))
	if err != nil {
		return outcome, fmt.Errorf("list the departing account's external photos: %w", err)
	}

	// The file half first, in the trash move's own order: a move that
	// fails fails the whole disposal, and the moves already performed
	// are compensated -- the client reads a state where nothing moved.
	var moves []batchFileMove
	for _, external := range externals {
		assetID := uuid.UUID(external.ID.Bytes)
		trashPath, err := storage.MoveToTrash(s.dataDir, external.FilePath, assetID.String())
		if err != nil {
			s.compensateBatchMoves(r, moves)
			return outcome, fmt.Errorf("%w: %v", errRemovalTrashMove, err)
		}
		moves = append(moves, batchFileMove{
			from:      external.FilePath,
			to:        trashPath,
			trashHome: storage.TrashHome(s.dataDir, assetID.String()),
		})
		if _, err := queries.TrashExternalAssetForOwner(ctx, dbgen.TrashExternalAssetForOwnerParams{
			ToOwner:   pgUUID(actor),
			ID:        external.ID,
			FromOwner: pgUUID(from),
			TrashPath: pgtype.Text{String: trashPath, Valid: true},
		}); err != nil {
			s.compensateBatchMoves(r, moves)
			return outcome, fmt.Errorf("trash the external photo %s: %w", assetID, err)
		}
	}

	// The containers go first: the departing account's albums and
	// memories carry its memberships, shares and preferences away with
	// them, so the authorship arm below only has the OTHER accounts'
	// albums left to settle.
	if _, err := queries.DeleteOwnedAlbums(ctx, pgUUID(from)); err != nil {
		s.compensateBatchMoves(r, moves)
		return outcome, fmt.Errorf("delete the departing account's albums: %w", err)
	}

	if err := queries.TrashOwnedAssets(ctx, dbgen.TrashOwnedAssetsParams{
		ToOwner:   pgUUID(actor),
		FromOwner: pgUUID(from),
	}); err != nil {
		s.compensateBatchMoves(r, moves)
		return outcome, fmt.Errorf("trash the departing account's photos: %w", err)
	}

	if err := queries.ReassignAlbumItemAuthorship(ctx, dbgen.ReassignAlbumItemAuthorshipParams{
		ToUser:   pgUUID(actor),
		FromUser: pgUUID(from),
	}); err != nil {
		s.compensateBatchMoves(r, moves)
		return outcome, fmt.Errorf("carry the album authorship over: %w", err)
	}
	// The grants the departing account issued stood on its own albums,
	// which are gone: this arm settles nothing in practice. It runs
	// anyway, so the foreign key is proven satisfied whatever an older
	// deployment's rows say.
	if err := queries.ReassignShareGrants(ctx, dbgen.ReassignShareGrantsParams{
		ToUser:   pgUUID(actor),
		FromUser: pgUUID(from),
	}); err != nil {
		s.compensateBatchMoves(r, moves)
		return outcome, fmt.Errorf("carry the share grants over: %w", err)
	}

	// The labels move with the photos they describe: restoring a photo
	// from the trash finds its labels where the surviving owner keeps
	// them.
	if err := queries.MergeDuplicateTags(ctx, dbgen.MergeDuplicateTagsParams{
		ToOwner:   pgUUID(actor),
		FromOwner: pgUUID(from),
	}); err != nil {
		s.compensateBatchMoves(r, moves)
		return outcome, fmt.Errorf("fold the labels the caller already carries: %w", err)
	}
	if err := queries.DropMergedTags(ctx, dbgen.DropMergedTagsParams{
		ToOwner:   pgUUID(actor),
		FromOwner: pgUUID(from),
	}); err != nil {
		s.compensateBatchMoves(r, moves)
		return outcome, fmt.Errorf("drop the folded labels: %w", err)
	}
	if err := queries.TransferTagOwnership(ctx, dbgen.TransferTagOwnershipParams{
		ToOwner:   pgUUID(actor),
		FromOwner: pgUUID(from),
	}); err != nil {
		s.compensateBatchMoves(r, moves)
		return outcome, fmt.Errorf("hand the labels over: %w", err)
	}

	for _, album := range albums {
		// The stream is not an album resource: no client ever listed
		// it, so its removal announces nothing (the transfer arm's
		// fold keeps the same silence).
		if album.Type == dbgen.AlbumTypeSync {
			continue
		}
		outcome.deletedAlbums = append(outcome.deletedAlbums, uuid.UUID(album.ID.Bytes))
	}
	for _, asset := range assets {
		if asset.IsComponent {
			// The component rows stay silent (the purge events'
			// standing discipline): no client view ever listed one.
			continue
		}
		outcome.trashedAssets[uuid.UUID(asset.ID.Bytes)] = asset.MediaKind
	}
	return outcome, nil
}
