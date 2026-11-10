// The people surface reads: the named region behind
// GET /people, the unnamed region's keyset pages behind
// GET /people/unnamed and one cluster's detail behind
// GET /people/{clusterId}. The per-user-state discipline runs through all three:
// clusters are global rows, labels are the caller's private state,
// and every face count is honest for the caller's visible assets --
// the queries compute each caller's view, nothing stores it.
package api

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

const (
	// peopleDefaultLimit is the page size of an unnamed-region
	// request without a limit parameter, pinned by the contract.
	peopleDefaultLimit = 60

	// peopleMaxLimit caps the page size: unlike the timeline, a
	// value over the cap is a client bug the 400 names, not a
	// clamp -- the brief's grid pages at 60 and the cap only bounds
	// honest bulk walkers.
	peopleMaxLimit = 200
)

// peopleCursor is the decoded continuation token of the unnamed
// region: the (face_count, cluster_id) pair of the last row of the
// previous page, the exact ordering key of the keyset predicate
// (face_count DESC, cluster_id DESC -- the row-value pair mirrors the
// timeline's (taken_at, id) precedent). The zero cursor is the first
// page: the handler passes the pair as NULL arguments instead, which
// short-circuits the predicate to TRUE.
type peopleCursor struct {
	faceCount pgtype.Int8
	clusterID pgtype.UUID
}

// peopleCursorWireBytes is the byte length of the wire form: 4 bytes
// of face count plus the 16 bytes of the cluster id.
const peopleCursorWireBytes = 20

// faceThumbnailPath renders the pinned crop URL of one face: the
// endpoint itself lands with the face thumbnails task and answers 404
// until then -- the contract documents the shape now so clients can
// bind it.
func faceThumbnailPath(faceID uuid.UUID) string {
	return "/api/v1/faces/" + faceID.String() + "/thumbnail"
}

// encodePeopleCursor renders the cursor of one page's last row: the
// opaque base64 of the binary pair. Clients treat it as an opaque
// string; the pair layout keeps it compact (28 characters) and stable
// under the page's own values.
func encodePeopleCursor(faceCount int64, clusterID pgtype.UUID) string {
	var buf [peopleCursorWireBytes]byte
	binary.BigEndian.PutUint32(buf[:4], uint32(faceCount))
	copy(buf[4:], clusterID.Bytes[:])
	return base64.RawURLEncoding.EncodeToString(buf[:])
}

// decodePeopleCursor parses a cursor this endpoint produced. The
// decoding fails on anything but the exact wire form -- a mangled or
// truncated value is a client error, not a server one. A
// well-formed but fabricated cursor is harmless: it simply addresses
// some (face_count, cluster_id) point in the ordering, which yields
// the clusters after that point or none.
func decodePeopleCursor(value string) (peopleCursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return peopleCursor{}, err
	}
	if len(raw) != peopleCursorWireBytes {
		return peopleCursor{}, fmt.Errorf("cursor holds %d bytes, want %d", len(raw), peopleCursorWireBytes)
	}
	var clusterID uuid.UUID
	copy(clusterID[:], raw[4:])
	return peopleCursor{
		faceCount: pgtype.Int8{Int64: int64(binary.BigEndian.Uint32(raw[:4])), Valid: true},
		clusterID: pgtype.UUID{Bytes: clusterID, Valid: true},
	}, nil
}

// peoplePageParams validates and normalizes the unnamed region's
// pagination parameters: the limit -- absent means the contract
// default, below 1 or over the maximum is a client bug the 400 names
// -- and the cursor -- absent is the first page, anything else must
// be exactly the string this endpoint produced. It answers ok=false
// once it has written the 400 envelope.
func (s *server) peoplePageParams(w http.ResponseWriter, limit *int, cursor *string) (int, peopleCursor, bool) {
	pageLimit := peopleDefaultLimit
	if limit != nil {
		if *limit < 1 {
			s.writeValidationError(w, "limit must be at least 1", []string{"limit"})
			return 0, peopleCursor{}, false
		}
		if *limit > peopleMaxLimit {
			s.writeValidationError(w, "limit must be at most 200", []string{"limit"})
			return 0, peopleCursor{}, false
		}
		pageLimit = *limit
	}

	var pageCursor peopleCursor
	if cursor != nil && *cursor != "" {
		decoded, err := decodePeopleCursor(*cursor)
		if err != nil {
			s.writeValidationError(w, "invalid people cursor", []string{"cursor"})
			return 0, peopleCursor{}, false
		}
		pageCursor = decoded
	}
	return pageLimit, pageCursor, true
}

// ListPeople implements GET /api/v1/people: the caller's named
// people -- the clusters carrying a label of the caller and at least
// one face on a caller-visible asset, sorted by name
// case-insensitively. Hidden people stay listed carrying isHidden:
// the switch is a badge, not a filter.
func (s *server) ListPeople(w http.ResponseWriter, r *http.Request) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	rows, err := s.queries.ListLabeledPeople(r.Context(), pgUUID(requester.userID))
	if err != nil {
		s.internalError(w, r, "load people", err)
		return
	}

	people := make([]apigen.PersonSummary, 0, len(rows))
	for _, row := range rows {
		// The card face is the query's best caller-visible live face
		// : never the cluster's global
		// representative, which can sit on an asset this caller
		// cannot see -- the listing's join already proved a visible
		// face exists, so the pick is always present here.
		cardFace := uuid.UUID(row.BestFaceID.Bytes)
		people = append(people, apigen.PersonSummary{
			ClusterId:            uuid.UUID(row.ClusterID.Bytes),
			Name:                 row.Name,
			FaceCount:            int(row.FaceCount),
			RepresentativeFaceId: cardFace,
			Thumbnail:            faceThumbnailPath(cardFace),
			IsHidden:             row.IsHidden,
		})
	}

	s.writeJSON(w, http.StatusOK, people)
}

// ListUnnamedPeople implements GET /api/v1/people/unnamed: one
// keyset page of the clusters the caller sees without having labeled
// -- the unnamed region. The page is the look-ahead pattern of the
// timeline: one row beyond the limit decides nextCursor, so the last
// page is the one without the follow-up and no run walks into an
// empty trailing page.
func (s *server) ListUnnamedPeople(w http.ResponseWriter, r *http.Request, params apigen.ListUnnamedPeopleParams) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	limit, cursor, ok := s.peoplePageParams(w, params.Limit, params.Cursor)
	if !ok {
		return
	}

	rows, err := s.queries.ListUnnamedClusters(r.Context(), dbgen.ListUnnamedClustersParams{
		UserID:          pgUUID(requester.userID),
		CursorFaceCount: cursor.faceCount,
		CursorClusterID: cursor.clusterID,
		LimitCount:      int32(limit) + 1,
	})
	if err != nil {
		s.internalError(w, r, "load unnamed people", err)
		return
	}

	response := apigen.UnnamedClustersResponse{Items: make([]apigen.UnnamedClusterSummary, 0, len(rows))}
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[limit-1]
		next := encodePeopleCursor(last.FaceCount, last.ClusterID)
		response.NextCursor = &next
	}
	for _, row := range rows {
		// The tile face is the query's best caller-visible live face
		// -- the same pick the named region's cards apply.
		cardFace := uuid.UUID(row.BestFaceID.Bytes)
		response.Items = append(response.Items, apigen.UnnamedClusterSummary{
			ClusterId:            uuid.UUID(row.ClusterID.Bytes),
			FaceCount:            int(row.FaceCount),
			RepresentativeFaceId: cardFace,
			Thumbnail:            faceThumbnailPath(cardFace),
		})
	}

	s.writeJSON(w, http.StatusOK, response)
}

// GetPersonCluster implements GET /api/v1/people/{clusterId}: one
// cluster's detail as the caller sees it -- the caller's own label
// state (null in the unnamed state) and the cluster's faces on
// caller-visible assets, strongest first. The count gate answers the
// double-meaning 404: a count of zero covers an unknown id and a
// cluster whose faces all sit on other accounts' assets alike, so
// cluster ids cannot be probed for existence.
func (s *server) GetPersonCluster(w http.ResponseWriter, r *http.Request, clusterId uuid.UUID) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	callerID := pgUUID(requester.userID)
	clusterID := pgUUID(clusterId)

	count, err := s.queries.CountClusterFaces(r.Context(), dbgen.CountClusterFacesParams{
		ClusterID: clusterID,
		UserID:    callerID,
	})
	if err != nil {
		s.internalError(w, r, "count cluster faces", err)
		return
	}
	if count == 0 {
		s.writeError(w, http.StatusNotFound, apigen.PERSONNOTFOUND,
			"no such person for this caller", nil)
		return
	}

	labelRow, err := s.queries.GetClusterForCaller(r.Context(), dbgen.GetClusterForCallerParams{
		UserID:    callerID,
		ClusterID: clusterID,
	})
	// The count above proved the cluster visible; no rows here can
	// only mean it vanished between the two reads -- the same 404
	// keeps that race honest.
	if errors.Is(err, pgx.ErrNoRows) {
		s.writeError(w, http.StatusNotFound, apigen.PERSONNOTFOUND,
			"no such person for this caller", nil)
		return
	}
	if err != nil {
		s.internalError(w, r, "load cluster", err)
		return
	}

	faces, err := s.queries.ListClusterFacesForCaller(r.Context(), dbgen.ListClusterFacesForCallerParams{
		ClusterID: clusterID,
		UserID:    callerID,
	})
	if err != nil {
		s.internalError(w, r, "load cluster faces", err)
		return
	}

	detail := apigen.ClusterDetail{
		ClusterId: clusterId,
		Label:     nil,
		Faces:     make([]apigen.FaceDetail, 0, len(faces)),
	}
	if labelRow.LabelName.Valid {
		detail.Label = &apigen.PersonLabel{
			Name:     labelRow.LabelName.String,
			IsHidden: labelRow.LabelHidden.Bool,
		}
	}
	for _, row := range faces {
		faceID := uuid.UUID(row.ID.Bytes)
		face := apigen.FaceDetail{
			Id:        faceID,
			AssetId:   uuid.UUID(row.AssetID.Bytes),
			Score:     row.Score,
			Thumbnail: faceThumbnailPath(faceID),
		}
		// The generated Asset is an inline struct: fill it by field.
		face.Asset.OriginalFilename = row.OriginalFilename
		face.Asset.TakenAt = row.TakenAt.Time.UTC()
		detail.Faces = append(detail.Faces, face)
	}

	s.writeJSON(w, http.StatusOK, detail)
}
