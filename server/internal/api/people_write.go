// The people surface writes (M3b Task 2): the caller's label and
// hidden switches behind PUT/DELETE /people/{clusterId}/label and
// PUT /people/{clusterId}/hidden, the cluster merge behind
// POST /people/merge and the single face move behind
// POST /faces/{faceId}/move.
//
// The B4 discipline runs through every label write: the writes are
// per-caller upserts on the (user_id, cluster_id) pair -- they never
// read or touch another account's row, and duplicating another
// account's name is fine (no NAME_TAKEN exists; families have real
// duplicates). The structural writes stick to the cluster
// maintenance discipline M3a pinned for every faces.cluster_id
// change: member moves, representative re-pointing and the
// dissolution of vacated clusters (labels cascading) land in ONE
// transaction, never as bare member moves.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
// through their i18n tables (today only
// person_hide_requires_label).
func (s *server) writeMessageKey(w http.ResponseWriter, message string, key string) {
	details := map[string]interface{}{"key": key}
	s.writeError(w, http.StatusBadRequest, apigen.VALIDATIONFAILED, message, &details)
}

// personFKViolation reports whether the error is a foreign-key
// violation: on the people writes it can only mean a cluster
// dissolved between the handler's visibility gate and the structural
// write, so the same 404 the gate answers keeps that race honest
// (the Task 1 read precedent, applied to writes).
func personFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503" // foreign_key_violation
}

// publishPersonEvent reports one people-surface event onto the
// activity stream. The nil guard matches the asset publisher (a
// router wired without a bus is a wiring-order state, tests only);
// the payload carries the acting user beside the ids, because label
// and hidden state are private per user.
func (s *server) publishPersonEvent(name string, payload map[string]string) {
	if s.events == nil {
		return
	}
	s.events.Publish(name, payload)
}

// LabelPerson implements PUT /people/{clusterId}/label: the caller's
// private upsert of one cluster's name. Duplicate names are allowed
// (no NAME_TAKEN); a rename keeps the hidden switch; the write
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
	if visible == 0 {
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
		"name":      labelRow.Name,
		"userId":    requester.userID.String(),
	})

	s.writeJSON(w, http.StatusOK, apigen.PersonLabelResponse{
		ClusterId: clusterId,
		Name:      labelRow.Name,
		IsHidden:  labelRow.IsHidden,
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
	if visible == 0 {
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

// SetPersonHidden implements PUT /people/{clusterId}/hidden: the
// hidden switch flips on the CALLER'S label row, so hiding an
// unlabeled cluster answers the pinned person_hide_requires_label
// message key. Hidden people stay listed (badge semantics). The
// write publishes person.hidden in both directions.
func (s *server) SetPersonHidden(w http.ResponseWriter, r *http.Request, clusterId openapi_types.UUID) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	var request apigen.SetPersonHiddenRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeValidationError(w, "request body must be valid JSON", []string{"body"})
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
	if visible == 0 {
		s.writePersonNotFound(w)
		return
	}

	// The update IS the label existence check, atomically: zero
	// rows mean the caller has not named the cluster (the switch's
	// home row is missing), whatever raced.
	labelRow, err := s.queries.SetUserFaceLabelHidden(ctx, dbgen.SetUserFaceLabelHiddenParams{
		UserID:    callerID,
		ClusterID: clusterID,
		IsHidden:  request.IsHidden,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		s.writeMessageKey(w, "the person carries no label to hide", "person_hide_requires_label")
		return
	}
	if err != nil {
		s.internalError(w, r, "set the person hidden switch", err)
		return
	}

	s.publishPersonEvent("person.hidden", map[string]string{
		"clusterId": uuid.UUID(clusterId).String(),
		"isHidden":  strconv.FormatBool(labelRow.IsHidden),
		"userId":    requester.userID.String(),
	})

	s.writeJSON(w, http.StatusOK, apigen.PersonLabelResponse{
		ClusterId: clusterId,
		Name:      labelRow.Name,
		IsHidden:  labelRow.IsHidden,
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
		s.internalError(w, r, "move the merged faces", err)
		return
	}
	if err := queries.RepointClusterRepresentative(ctx, targetID); err != nil {
		s.internalError(w, r, "repoint the surviving cluster's representative", err)
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
// survivor and dissolves with its label when nothing survives. The
// write publishes person.face_moved.
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
	// unknown face or one on an asset the caller cannot see, and a
	// NULL cluster means the clustering has not made the face part
	// of any person yet -- every miss is the same 404.
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

	// The no-op: the face already occupies the target. Accepted and
	// silent -- the batch surface's report-changes-not-repeats
	// discipline applies to a repeat of the same structural state.
	if source == target {
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
	if visible == 0 {
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
	// representative recomputes over its new membership, the
	// vacated source follows its best survivor when its own
	// representative moved away, and the source dissolves with its
	// label when nothing survived.
	if err := queries.MoveFaceToCluster(ctx, dbgen.MoveFaceToClusterParams{
		FaceID:          faceID,
		TargetClusterID: pgUUID(target),
	}); err != nil {
		s.internalError(w, r, "move the face", err)
		return
	}
	if err := queries.RepointClusterRepresentative(ctx, pgUUID(target)); err != nil {
		s.internalError(w, r, "repoint the target cluster's representative", err)
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
