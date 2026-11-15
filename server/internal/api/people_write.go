// The people surface writes: the caller's label beside
// PUT/DELETE /people/{clusterId}/label, the person ENTITY's two
// attributes behind PUT /people/{clusterId}/birthdate and
// PUT /people/{clusterId}/avatar, the cluster merge behind
// POST /people/merge, the single face move behind
// POST /faces/{faceId}/move, the from-face person creation behind
// POST /faces/{faceId}/cluster and the single-face detach behind
// DELETE /faces/{faceId}.
//
// The per-user-state discipline runs through the label writes: they
// are per-caller upserts and deletes on the (user_id, cluster_id)
// pair -- they never read or touch another account's row, and
// duplicating another account's name is fine (no NAME_TAKEN exists;
// families have real duplicates). The birth date and the avatar are
// the OPPOSITE shape on purpose: the person entity's attributes are
// one value per cluster for the whole family, so those writes touch
// the cluster's own columns. The structural writes stick to the
// cluster maintenance discipline for every faces.cluster_id
// change: member moves, representative re-pointing and the
// dissolution of vacated clusters (labels cascading) land in ONE
// transaction, never as bare member moves.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	openapi_types "github.com/oapi-codegen/runtime/types"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

const (
	// personNameMinRunes and personNameMaxRunes bound the label
	// name: trimmed, counted in characters (a byte bound would
	// shorten non-ASCII names), pinned by the contract.
	personNameMinRunes = 1
	personNameMaxRunes = 64

	// personMergeMinSources and personMergeMaxSources bound the
	// merge's source list, counted after duplicates collapse: one
	// source is the pairwise fold the cluster detail's "merge into"
	// action posts (that person merges into the picked one), the
	// batch organize path sends more.
	personMergeMinSources = 1
	personMergeMaxSources = 50
)

// writePersonNotFound emits the people surface's pinned 404: the
// same envelope every read answers for an unknown id, so a write
// cannot probe a cluster's existence either.
func (s *server) writePersonNotFound(w http.ResponseWriter) {
	s.writeError(w, http.StatusNotFound, apigen.PERSONNOTFOUND,
		"no such person for this caller", nil)
}

// writeMessageKey emits a 400 VALIDATION_FAILED whose details carry a
// message key: a code-specific sub-condition the clients translate
// through their i18n tables (album_cover_not_member on the album
// face).
func (s *server) writeMessageKey(w http.ResponseWriter, message string, key string) {
	details := map[string]interface{}{"key": key}
	s.writeError(w, http.StatusBadRequest, apigen.VALIDATIONFAILED, message, &details)
}

// personFKViolation reports whether the error is a foreign-key
// violation: on the people writes it can only mean a cluster
// dissolved between the handler's visibility gate and the structural
// write, so the same 404 the gate answers keeps that race honest
// (the read precedent, applied to writes).
func personFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503" // foreign_key_violation
}

// publishPersonEvent reports one people-surface event onto the
// activity stream. The nil guard matches the asset publisher (a
// router wired without a bus is a wiring-order state, tests only);
// the payload carries the acting user beside the ids, because the
// entity writes are shared state a subscriber filters by ids.
func (s *server) publishPersonEvent(name string, payload map[string]string) {
	if s.events == nil {
		return
	}
	s.events.Publish(name, payload)
}

// LabelPerson implements PUT /people/{clusterId}/label: the caller's
// private upsert of one cluster's name. Duplicate names are allowed
// (no NAME_TAKEN); the person entity's attributes (birth date,
// avatar) live on the cluster and stay untouched; the write
// publishes person.labeled.
func (s *server) LabelPerson(w http.ResponseWriter, r *http.Request, clusterId openapi_types.UUID) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	// The name: trimmed, 1-64 characters, counted in runes.
	var request apigen.LabelPersonRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeValidationError(w, "request body must be valid JSON", []string{"body"})
		return
	}
	name := strings.TrimSpace(request.Name)
	if runes := utf8.RuneCountInString(name); runes < personNameMinRunes || runes > personNameMaxRunes {
		s.writeValidationError(w, "name must span 1-64 characters", []string{"name"})
		return
	}

	ctx := r.Context()
	callerID := pgUUID(requester.userID)
	clusterID := pgUUID(uuid.UUID(clusterId))

	visible, err := s.queries.CountClusterFaces(ctx, dbgen.CountClusterFacesParams{
		ClusterID: clusterID,
		UserID:    callerID,
	})
	if err != nil {
		s.internalError(w, r, "count cluster faces", err)
		return
	}
	if visible.FaceCount == 0 {
		s.writePersonNotFound(w)
		return
	}

	labelRow, err := s.queries.UpsertUserFaceLabel(ctx, dbgen.UpsertUserFaceLabelParams{
		UserID:    callerID,
		ClusterID: clusterID,
		Name:      name,
	})
	if err != nil {
		if personFKViolation(err) {
			// The cluster dissolved between the gate and the write.
			s.writePersonNotFound(w)
			return
		}
		s.internalError(w, r, "upsert the person label", err)
		return
	}

	s.publishPersonEvent("person.labeled", map[string]string{
		"clusterId": uuid.UUID(clusterId).String(),
		"userId":    requester.userID.String(),
	})

	s.writeJSON(w, http.StatusOK, apigen.PersonLabelResponse{
		ClusterId: clusterId,
		Name:      labelRow,
	})
}

// UnlabelPerson implements DELETE /people/{clusterId}/label: the
// caller's label row goes, the person returns to the caller's
// unnamed region, and the cluster and every other account's label
// survive. Deleting without a label is an idempotent no-op once the
// cluster is visible. The write publishes person.unlabeled.
func (s *server) UnlabelPerson(w http.ResponseWriter, r *http.Request, clusterId openapi_types.UUID) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	ctx := r.Context()
	callerID := pgUUID(requester.userID)
	clusterID := pgUUID(uuid.UUID(clusterId))

	visible, err := s.queries.CountClusterFaces(ctx, dbgen.CountClusterFacesParams{
		ClusterID: clusterID,
		UserID:    callerID,
	})
	if err != nil {
		s.internalError(w, r, "count cluster faces", err)
		return
	}
	if visible.FaceCount == 0 {
		s.writePersonNotFound(w)
		return
	}

	// The delete is unconditionally a no-op-or-write: a label that
	// already vanished (the cluster dissolving under it) leaves
	// nothing to remove and still answers 204 -- the person is
	// gone either way.
	if err := s.queries.DeleteUserFaceLabel(ctx, dbgen.DeleteUserFaceLabelParams{
		UserID:    callerID,
		ClusterID: clusterID,
	}); err != nil {
		s.internalError(w, r, "delete the person label", err)
		return
	}

	s.publishPersonEvent("person.unlabeled", map[string]string{
		"clusterId": uuid.UUID(clusterId).String(),
		"userId":    requester.userID.String(),
	})

	w.WriteHeader(http.StatusNoContent)
}

// SetPersonBirthdate implements PUT /people/{clusterId}/birthdate:
// the person ENTITY's birth date, one value per cluster read by
// every account. The body carries the field explicitly (a body
// without it is a client bug the 400 names) and null clears it --
// a clear is an act, not an omission. The write publishes
// person.birthdate_updated.
func (s *server) SetPersonBirthdate(w http.ResponseWriter, r *http.Request, clusterId openapi_types.UUID) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	var body map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.writeValidationError(w, "request body must be valid JSON", []string{"body"})
		return
	}
	raw, present := body["birthDate"]
	if !present {
		s.writeValidationError(w, "birthDate must be a YYYY-MM-DD date or null", []string{"birthDate"})
		return
	}
	var request apigen.SetPersonBirthdateRequest
	if err := json.Unmarshal(raw, &request.BirthDate); err != nil {
		s.writeValidationError(w, "birthDate must be a YYYY-MM-DD date or null", []string{"birthDate"})
		return
	}

	ctx := r.Context()
	callerID := pgUUID(requester.userID)
	clusterID := pgUUID(uuid.UUID(clusterId))

	visible, err := s.queries.CountClusterFaces(ctx, dbgen.CountClusterFacesParams{
		ClusterID: clusterID,
		UserID:    callerID,
	})
	if err != nil {
		s.internalError(w, r, "count cluster faces", err)
		return
	}
	if visible.FaceCount == 0 {
		s.writePersonNotFound(w)
		return
	}

	birthDate := pgtype.Date{}
	if request.BirthDate != nil {
		birthDate = pgtype.Date{Time: request.BirthDate.UTC(), Valid: true}
	}
	if err := s.queries.SetClusterBirthDate(ctx, dbgen.SetClusterBirthDateParams{
		ClusterID: clusterID,
		BirthDate: birthDate,
	}); err != nil {
		s.internalError(w, r, "set the person birth date", err)
		return
	}

	s.publishPersonEvent("person.birthdate_updated", map[string]string{
		"clusterId": uuid.UUID(clusterId).String(),
		"userId":    requester.userID.String(),
	})

	s.writeJSON(w, http.StatusOK, apigen.PersonBirthdateResponse{
		ClusterId: clusterId,
		BirthDate: request.BirthDate,
	})
}

// SetPersonAvatar implements PUT /people/{clusterId}/avatar: the
// person ENTITY's avatar face, one pick per cluster read by every
// account. The face must be one of the cluster's own, on an asset
// the caller can see -- anything else answers the pinned 404, so a
// face id cannot be probed either. The pick is advisory to the
// clustering (representative_face_id is untouched) and the read
// falls back to the best visible member when it stops holding. The
// write publishes person.avatar_updated.
func (s *server) SetPersonAvatar(w http.ResponseWriter, r *http.Request, clusterId openapi_types.UUID) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	var request apigen.SetPersonAvatarRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeValidationError(w, "request body must be valid JSON", []string{"body"})
		return
	}
	if uuid.UUID(request.FaceId) == uuid.Nil {
		s.writeValidationError(w, "faceId must be a UUID", []string{"faceId"})
		return
	}

	ctx := r.Context()
	callerID := pgUUID(requester.userID)
	clusterID := pgUUID(uuid.UUID(clusterId))

	// The face-side gate first: another cluster's face is no avatar
	// of this person, and the one 404 keeps the two misses
	// indistinguishable.
	if _, err := s.queries.GetClusterAvatarFaceForCaller(ctx, dbgen.GetClusterAvatarFaceForCallerParams{
		FaceID:    pgUUID(uuid.UUID(request.FaceId)),
		ClusterID: clusterID,
		UserID:    callerID,
	}); errors.Is(err, pgx.ErrNoRows) {
		s.writePersonNotFound(w)
		return
	} else if err != nil {
		s.internalError(w, r, "load the avatar face", err)
		return
	}

	if err := s.queries.SetClusterAvatarFace(ctx, dbgen.SetClusterAvatarFaceParams{
		ClusterID:    clusterID,
		AvatarFaceID: pgUUID(uuid.UUID(request.FaceId)),
	}); err != nil {
		if personFKViolation(err) {
			// The cluster (or the face) dissolved in the race
			// against the write: no such person to the picker.
			s.writePersonNotFound(w)
			return
		}
		s.internalError(w, r, "set the person avatar", err)
		return
	}

	s.publishPersonEvent("person.avatar_updated", map[string]string{
		"clusterId": uuid.UUID(clusterId).String(),
		"userId":    requester.userID.String(),
	})

	s.writeJSON(w, http.StatusOK, apigen.PersonAvatarResponse{
		ClusterId:    clusterId,
		AvatarFaceId: request.FaceId,
	})
}

// MergePeople implements POST /people/merge: every source cluster's
// faces re-home onto the target, the target's representative
// recomputes to the merged set's strongest member, and every source
// cluster dissolves WITH its label rows -- the honest documented
// cost. The whole structural change lands as one transaction through
// the cluster maintenance primitives, and the write publishes one
// person.merged with the dissolved ids comma-joined.
func (s *server) MergePeople(w http.ResponseWriter, r *http.Request) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	var request apigen.MergePeopleRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeValidationError(w, "request body must be valid JSON", []string{"body"})
		return
	}

	// The source list is a set: repeated ids collapse to one
	// application in first-appearance order (the batch surface's
	// pinned pattern), and the bound counts the collapsed list.
	sources := make([]uuid.UUID, 0, len(request.SourceClusterIds))
	seen := make(map[uuid.UUID]bool, len(request.SourceClusterIds))
	for _, raw := range request.SourceClusterIds {
		id := uuid.UUID(raw)
		if !seen[id] {
			seen[id] = true
			sources = append(sources, id)
		}
	}
	if len(sources) < personMergeMinSources {
		s.writeValidationError(w, "merge needs at least one source cluster", []string{"sourceClusterIds"})
		return
	}
	if len(sources) > personMergeMaxSources {
		s.writeValidationError(w, fmt.Sprintf("merge exceeds the %d source cluster limit", personMergeMaxSources), []string{"sourceClusterIds"})
		return
	}

	target := uuid.UUID(request.TargetClusterId)
	if seen[target] {
		// The pinned rule: a target among the sources is no such
		// merge -- the same anti-enumeration 404, byte for byte.
		s.writePersonNotFound(w)
		return
	}

	ctx := r.Context()
	callerID := pgUUID(requester.userID)
	targetID := pgUUID(target)

	// The visibility gate over every merged id in ONE read (the
	// unnamed region's caller-visible predicate, applied to the
	// set): the count must cover every source plus the target, and
	// a shortfall names no id.
	mergedClusterIDs := append([]uuid.UUID{}, sources...)
	visible, err := s.queries.CountVisibleClusters(ctx, dbgen.CountVisibleClustersParams{
		ClusterIds: pgUUIDs(append(mergedClusterIDs, target)),
		UserID:     callerID,
	})
	if err != nil {
		s.internalError(w, r, "count visible clusters", err)
		return
	}
	if visible != int64(len(sources)+1) {
		s.writePersonNotFound(w)
		return
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		s.internalError(w, r, "begin merge transaction", err)
		return
	}
	// The rollback is a no-op after a successful commit; arguments
	// are captured while the request context is still live.
	defer func() { _ = tx.Rollback(ctx) }()
	queries := dbgen.New(tx)

	// The discipline in order: the members move first (the sources'
	// faces land on the target), the target's representative
	// recomputes over its new membership, then the vacated source
	// clusters dissolve with their labels. A cluster is only ever
	// dissolved AFTER its faces left and the NOT EXISTS guard
	// confirms the vacatur.
	if err := queries.MoveFacesToCluster(ctx, dbgen.MoveFacesToClusterParams{
		SourceClusterIds: pgUUIDs(sources),
		TargetClusterID:  targetID,
	}); err != nil {
		if personFKViolation(err) {
			// A gate-passed cluster vanished in the race against
			// the statement: no such person to the merger.
			s.writePersonNotFound(w)
			return
		}
		s.internalError(w, r, "move the merged faces", err)
		return
	}
	if err := queries.RepointClusterRepresentative(ctx, targetID); err != nil {
		s.internalError(w, r, "repoint the surviving cluster's representative", err)
		return
	}
	// The move path repoints the vacated side; the merge owes its
	// sources the same care when a concurrent insert keeps one alive
	// past the NOT EXISTS guard.
	if err := queries.RepointVacatedClusterRepresentatives(ctx, pgUUIDs(sources)); err != nil {
		s.internalError(w, r, "repoint the merged clusters' vacated representatives", err)
		return
	}
	if err := queries.DissolveVacatedClustersByID(ctx, pgUUIDs(sources)); err != nil {
		s.internalError(w, r, "dissolve the merged clusters", err)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		if personFKViolation(err) {
			// A gate-passed cluster dissolved under the
			// transaction; the merge resolves to no such person.
			s.writePersonNotFound(w)
			return
		}
		s.internalError(w, r, "commit merge transaction", err)
		return
	}

	mergedIDs := make([]string, len(sources))
	for i, source := range sources {
		mergedIDs[i] = source.String()
	}
	s.publishPersonEvent("person.merged", map[string]string{
		"survivingClusterId": target.String(),
		"mergedClusterIds":   strings.Join(mergedIDs, ","),
		"userId":             requester.userID.String(),
	})

	response := apigen.MergePeopleResponse{
		SurvivingClusterId: target,
		// openapi_types.UUID is an alias of uuid.UUID, so the
		// merged ids hand over as one slice.
		MergedClusterIds: sources,
	}
	s.writeJSON(w, http.StatusOK, response)
}

// MoveFace implements POST /faces/{faceId}/move: one face's move
// onto an existing target cluster inside the same one-transaction
// discipline as the merge -- the target's representative recomputes,
// the vacated source re-homes its representative to its best
// survivor and dissolves with its label when nothing survives. A
// clusterless face is valid source too (the creation's own shape):
// it simply joins the target, with no source to maintain. The write
// publishes person.face_moved.
func (s *server) MoveFace(w http.ResponseWriter, r *http.Request, faceId openapi_types.UUID) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	var request apigen.MoveFaceRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeValidationError(w, "request body must be valid JSON", []string{"body"})
		return
	}
	target := uuid.UUID(request.TargetClusterId)

	ctx := r.Context()
	callerID := pgUUID(requester.userID)
	faceID := pgUUID(uuid.UUID(faceId))

	// The face-side gate and source read in one: no row means an
	// unknown face or one on an asset the caller cannot see, and
	// both miss shapes answer the one 404. A NULL cluster is NOT a
	// refusal here -- the move's object is the face's membership,
	// and a face the clustering has not grouped yet simply joins
	// the target.
	face, err := s.queries.GetFaceForCaller(ctx, dbgen.GetFaceForCallerParams{
		FaceID: faceID,
		UserID: callerID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		s.writePersonNotFound(w)
		return
	}
	if err != nil {
		s.internalError(w, r, "load the face", err)
		return
	}
	source, hasSource := uuid.UUID{}, face.ClusterID.Valid
	if hasSource {
		source = uuid.UUID(face.ClusterID.Bytes)
	}

	// The no-op: the face already occupies the target. Accepted and
	// silent -- the batch surface's report-changes-not-repeats
	// discipline applies to a repeat of the same structural state.
	if hasSource && source == target {
		s.writeJSON(w, http.StatusOK, apigen.MoveFaceResponse{
			FaceId:    faceId,
			ClusterId: target,
		})
		return
	}

	// The target's structural gate: the same count the detail read
	// answers the double-meaning 404 with.
	visible, err := s.queries.CountClusterFaces(ctx, dbgen.CountClusterFacesParams{
		ClusterID: pgUUID(target),
		UserID:    callerID,
	})
	if err != nil {
		s.internalError(w, r, "count target faces", err)
		return
	}
	if visible.FaceCount == 0 {
		s.writePersonNotFound(w)
		return
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		s.internalError(w, r, "begin face move transaction", err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := dbgen.New(tx)

	// The discipline in order: the face moves, the target's
	// representative recomputes over its new membership, and when
	// the face had a source cluster the vacated source follows its
	// best survivor -- and dissolves with its label when nothing
	// survived. A clusterless face has nothing to maintain.
	if err := queries.MoveFaceToCluster(ctx, dbgen.MoveFaceToClusterParams{
		FaceID:          faceID,
		TargetClusterID: pgUUID(target),
	}); err != nil {
		if personFKViolation(err) {
			// The target dissolved in the race against the
			// statement: no such person to the mover.
			s.writePersonNotFound(w)
			return
		}
		s.internalError(w, r, "move the face", err)
		return
	}
	if err := queries.RepointClusterRepresentative(ctx, pgUUID(target)); err != nil {
		s.internalError(w, r, "repoint the target cluster's representative", err)
		return
	}
	if hasSource {
		if err := queries.RepointVacatedClusterRepresentatives(ctx, pgUUIDs([]uuid.UUID{source})); err != nil {
			s.internalError(w, r, "repoint the vacated cluster's representative", err)
			return
		}
		if err := queries.DissolveVacatedClustersByID(ctx, pgUUIDs([]uuid.UUID{source})); err != nil {
			s.internalError(w, r, "dissolve the vacated cluster", err)
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		if personFKViolation(err) {
			// The target dissolved under the transaction.
			s.writePersonNotFound(w)
			return
		}
		s.internalError(w, r, "commit face move transaction", err)
		return
	}

	s.publishPersonEvent("person.face_moved", map[string]string{
		"faceId":    uuid.UUID(faceId).String(),
		"clusterId": target.String(),
		"userId":    requester.userID.String(),
	})

	s.writeJSON(w, http.StatusOK, apigen.MoveFaceResponse{
		FaceId:    faceId,
		ClusterId: target,
	})
}

// ClusterFace implements POST /faces/{faceId}/cluster: the
// from-face person creation. The face -- clustered or not -- becomes
// the sole member AND the representative of a brand-new cluster; the
// vacated source follows the same maintenance discipline as the move
// (representative re-home, then dissolution with the labels when
// nothing survived). A clusterless face is valid seed: there is
// simply no source to maintain. The write publishes
// person.face_clustered with the face id, the new cluster id and the
// acting user.
func (s *server) ClusterFace(w http.ResponseWriter, r *http.Request, faceId openapi_types.UUID) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	ctx := r.Context()
	callerID := pgUUID(requester.userID)
	faceID := pgUUID(uuid.UUID(faceId))

	// The face-side gate, the move's own: no row means an unknown
	// face or one on an asset the caller cannot curate, and both
	// miss shapes answer the one 404. A NULL cluster is NOT a
	// refusal here -- the creation is defined by its result, and a
	// clusterless face seeds the new cluster exactly like a
	// re-homed one.
	face, err := s.queries.GetFaceForCaller(ctx, dbgen.GetFaceForCallerParams{
		FaceID: faceID,
		UserID: callerID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		s.writePersonNotFound(w)
		return
	}
	if err != nil {
		s.internalError(w, r, "load the face", err)
		return
	}
	source, hasSource := uuid.UUID{}, face.ClusterID.Valid
	if hasSource {
		source = uuid.UUID(face.ClusterID.Bytes)
	}

	// The new cluster's identity is minted here, inside the
	// request: every caller naming this person later holds the id
	// this write returns.
	cluster := uuid.New()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		s.internalError(w, r, "begin face cluster transaction", err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := dbgen.New(tx)

	// The discipline in order: the cluster is BORN first, its
	// representative fixed to the seeding face (a sole member is
	// the strongest member by definition) -- the circular
	// faces/face_clusters foreign keys require the row before the
	// face can point at it. The face then re-homes onto it, and
	// the vacated source follows the move's own maintenance:
	// representative re-home, then dissolution with the labels
	// when nothing survived.
	if err := queries.SeedFaceCluster(ctx, dbgen.SeedFaceClusterParams{
		ID:                   pgUUID(cluster),
		RepresentativeFaceID: faceID,
	}); err != nil {
		if personFKViolation(err) {
			// The face vanished in the race against the
			// insert: no such person to create from.
			s.writePersonNotFound(w)
			return
		}
		s.internalError(w, r, "seed the new cluster", err)
		return
	}
	if err := queries.MoveFaceToCluster(ctx, dbgen.MoveFaceToClusterParams{
		FaceID:          faceID,
		TargetClusterID: pgUUID(cluster),
	}); err != nil {
		if personFKViolation(err) {
			s.writePersonNotFound(w)
			return
		}
		s.internalError(w, r, "move the face onto the new cluster", err)
		return
	}
	if hasSource {
		if err := queries.RepointVacatedClusterRepresentatives(ctx, pgUUIDs([]uuid.UUID{source})); err != nil {
			s.internalError(w, r, "repoint the vacated cluster's representative", err)
			return
		}
		if err := queries.DissolveVacatedClustersByID(ctx, pgUUIDs([]uuid.UUID{source})); err != nil {
			s.internalError(w, r, "dissolve the vacated cluster", err)
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		if personFKViolation(err) {
			// The face vanished under the transaction: no
			// such person to create from.
			s.writePersonNotFound(w)
			return
		}
		s.internalError(w, r, "commit face cluster transaction", err)
		return
	}

	s.publishPersonEvent("person.face_clustered", map[string]string{
		"faceId":    uuid.UUID(faceId).String(),
		"clusterId": cluster.String(),
		"userId":    requester.userID.String(),
	})

	s.writeJSON(w, http.StatusOK, apigen.ClusterFaceResponse{
		FaceId:    faceId,
		ClusterId: cluster,
	})
}

// DetachFace implements DELETE /faces/{faceId}: the single-face
// detach. The face's cluster assignment goes NULL -- back to the
// not-yet-grouped pool a freshly detected face holds -- and the
// source does NOT dissolve unless this emptied it (then the same
// dissolution and label cascade as the merge). The write publishes
// person.face_detached with the face id, the cluster the face left
// and the acting user.
func (s *server) DetachFace(w http.ResponseWriter, r *http.Request, faceId openapi_types.UUID) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	ctx := r.Context()
	callerID := pgUUID(requester.userID)
	faceID := pgUUID(uuid.UUID(faceId))

	// The face-side gate, the move's own -- including its
	// NULL-cluster refusal: the detach's object is the face's
	// person membership, and a face part of no person is no such
	// detachable face, the same 404 byte for byte.
	face, err := s.queries.GetFaceForCaller(ctx, dbgen.GetFaceForCallerParams{
		FaceID: faceID,
		UserID: callerID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		s.writePersonNotFound(w)
		return
	}
	if err != nil {
		s.internalError(w, r, "load the face", err)
		return
	}
	if !face.ClusterID.Valid {
		s.writePersonNotFound(w)
		return
	}
	source := uuid.UUID(face.ClusterID.Bytes)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		s.internalError(w, r, "begin face detach transaction", err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := dbgen.New(tx)

	// The discipline in order: the face's assignment goes NULL,
	// then the vacated source follows the move's own maintenance
	// -- representative re-home, then dissolution with the labels
	// when nothing survived.
	if err := queries.DetachFaceFromCluster(ctx, faceID); err != nil {
		s.internalError(w, r, "detach the face", err)
		return
	}
	if err := queries.RepointVacatedClusterRepresentatives(ctx, pgUUIDs([]uuid.UUID{source})); err != nil {
		s.internalError(w, r, "repoint the vacated cluster's representative", err)
		return
	}
	if err := queries.DissolveVacatedClustersByID(ctx, pgUUIDs([]uuid.UUID{source})); err != nil {
		s.internalError(w, r, "dissolve the vacated cluster", err)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		s.internalError(w, r, "commit face detach transaction", err)
		return
	}

	s.publishPersonEvent("person.face_detached", map[string]string{
		"faceId":    uuid.UUID(faceId).String(),
		"clusterId": source.String(),
		"userId":    requester.userID.String(),
	})

	w.WriteHeader(http.StatusNoContent)
}

// CreateFace implements POST /faces: one hand-drawn face. The owner
// draws a box around a face the detector missed and the drawn box
// becomes a face row of the asset -- the same rows detection writes,
// minus a detector (the row's model is NULL) and minus an embedding
// (the recognition never saw it, so the face joins no cluster until
// the assign flow moves or seeds one). The insert answers the row's
// identity and crop URL; the caller's own session converges through
// the response, so the write publishes no event (a face with no
// cluster announces no person change -- the same silence a fresh
// detection's rows keep).
func (s *server) CreateFace(w http.ResponseWriter, r *http.Request) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	var request apigen.FaceCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeValidationError(w, "request body must be valid JSON", []string{"body"})
		return
	}
	if !bboxInBounds(request.Bbox) {
		s.writeValidationError(w,
			"the drawn box must carry four in-bounds per-mille halves and a positive width and height",
			[]string{"bbox"})
		return
	}

	// The asset-side gate, the face writes' own in reverse: the face
	// must sit on an asset the caller OWNS and that is live -- the
	// same refusal discipline the move and the from-face creation
	// apply to the face's asset. An unknown id and a foreign one
	// resolve to no row and answer the byte-identical 404 (asset
	// ids cannot be enumerated); the live-state checks below fold
	// into the same shape.
	asset, err := s.queries.GetAsset(r.Context(), dbgen.GetAssetParams{
		ID:      pgUUID(uuid.UUID(request.AssetId)),
		OwnerID: pgUUID(requester.userID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		s.writePersonNotFound(w)
		return
	}
	if err != nil {
		s.internalError(w, r, "load the drawn face's asset", err)
		return
	}
	if asset.DeletedAt.Valid || asset.IsHidden || asset.IsComponent {
		s.writePersonNotFound(w)
		return
	}

	id := uuid.New()
	face, err := s.queries.InsertManualFace(r.Context(), dbgen.InsertManualFaceParams{
		ID:      pgUUID(id),
		AssetID: pgUUID(uuid.UUID(request.AssetId)),
		BboxX:   int32(request.Bbox.X),
		BboxY:   int32(request.Bbox.Y),
		BboxW:   int32(request.Bbox.W),
		BboxH:   int32(request.Bbox.H),
	})
	if err != nil {
		s.internalError(w, r, "store the drawn face", err)
		return
	}

	s.writeJSON(w, http.StatusOK, apigen.FaceCreated{
		Id:        uuid.UUID(face.ID.Bytes),
		Thumbnail: faceThumbnailPath(id),
	})
}

// bboxInBounds validates the drawn box: every half inside the
// per-mille range, and both extents positive (a zero-extent box is
// nothing drawn).
func bboxInBounds(bbox apigen.FaceBbox) bool {
	if bbox.X < 0 || bbox.X > 1000 || bbox.Y < 0 || bbox.Y > 1000 ||
		bbox.W < 1 || bbox.W > 1000 || bbox.H < 1 || bbox.H > 1000 {
		return false
	}
	return true
}
