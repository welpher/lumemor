// The search handler (M3d Task 3): GET /api/v1/search -- the unified
// search's HTTP face. The parameter validation, the cursor codec and
// the semantic arm's model resolution live here; the paths, the
// merge, the facets and their SQL live in the internal search
// package.
//
// The semantic path's readiness is honest degradation, never an
// error: no active text tower (the instance has not downloaded it, or
// the admin switched it off) answers an empty semantic arm while the
// text path still serves; a text tower whose embedding dimension
// disagrees with the clip_embeddings column's current dimension
// (a cross-dimension vision switch happened in between) does the
// same, with a server-side log line carrying the reason. The
// contract documents both.
package api

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/ml"
	mlclip "github.com/welpher/lumemor/server/internal/ml/clip"
	"github.com/welpher/lumemor/server/internal/search"
)

const (
	// searchDefaultLimit is the page size of a request without a limit
	// parameter, pinned by the contract.
	searchDefaultLimit = 100

	// searchMaxLimit caps the page size however large the request
	// asks: one response stays a bounded JSON document.
	searchMaxLimit = 200

	// searchMaxQueryRunes caps the text query: a search box's honest
	// bound (the contract documents it).
	searchMaxQueryRunes = 256

	// searchMaxFilterEntries caps every id filter list, the batch
	// endpoints' own bound discipline.
	searchMaxFilterEntries = 200
)

// cursorWireLen is the search cursor's wire length: the mode byte,
// the fused score, the taken-at nanoseconds and the uuid.
const cursorWireLen = 1 + 8 + 8 + 16

// SearchAssets implements GET /api/v1/search: one page of the merged
// search (or the filter browse) plus the facet counts.
func (s *server) SearchAssets(w http.ResponseWriter, r *http.Request, params apigen.SearchAssetsParams) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	// The locale parameter: the generated binder binds the raw string,
	// so the membership check lives here (the places endpoint's own
	// discipline).
	locale := "en"
	if params.Locale != nil {
		if !params.Locale.Valid() {
			s.writeValidationError(w, "locale must be one of: en, zh-Hans", []string{"locale"})
			return
		}
		locale = string(*params.Locale)
	}

	limit, ok := s.searchLimit(w, params.Limit)
	if !ok {
		return
	}

	// The text query: trimmed, bounded; fewer than two runes takes no
	// text path (the filter browse).
	query := ""
	if params.Query != nil {
		query = strings.TrimSpace(*params.Query)
	}
	if len([]rune(query)) > searchMaxQueryRunes {
		s.writeValidationError(w, "query is too long", []string{"query"})
		return
	}

	filters, ok := s.searchFilters(w, params, query)
	if !ok {
		return
	}

	// The cursor: absent is the first page (the zero cursor of the
	// request's own mode); anything else must be the exact string
	// this endpoint produced, in that same mode.
	wantMode := search.CursorFilterBrowse
	if filters.IsTextQuery() {
		wantMode = search.CursorFusedRank
	}
	var cursor search.Cursor
	if params.Cursor != nil && *params.Cursor != "" {
		decoded, err := decodeSearchCursor(*params.Cursor)
		if err != nil {
			s.writeValidationError(w, "invalid search cursor", []string{"cursor"})
			return
		}
		if decoded.Mode != wantMode {
			s.writeValidationError(w, "invalid search cursor", []string{"cursor"})
			return
		}
		cursor = decoded
	} else {
		cursor = search.Cursor{Mode: wantMode}
	}

	// The semantic arm's inputs: the query's text-tower embedding and
	// the active vision model whose rows it searches. The honest
	// degradations answer an empty semantic arm.
	if filters.IsTextQuery() {
		embedding, visionModel, err := s.semanticInputs(r.Context(), query)
		if err != nil {
			s.internalError(w, r, "prepare the semantic path", err)
			return
		}
		filters.QueryEmbedding = embedding
		filters.VisionModelID = visionModel
	}

	result, err := search.Run(r.Context(), s.pool, requester.userID, filters, locale, cursor, limit)
	if err != nil {
		s.internalError(w, r, "run the search", err)
		return
	}

	response := apigen.SearchResponse{
		Items:  make([]apigen.TimelineItem, 0, len(result.Items)),
		Facets: searchFacetsFrom(result.Facets),
	}
	for _, item := range result.Items {
		response.Items = append(response.Items, searchItemFrom(item))
	}
	if result.NextCursor != nil {
		next := encodeSearchCursor(*result.NextCursor)
		response.NextCursor = &next
	}
	s.writeJSON(w, http.StatusOK, response)
}

// searchLimit validates the page size: absent means the contract
// default, below 1 is a client bug the 400 names, over the maximum
// clamps to it (the timeline's own clamp discipline).
func (s *server) searchLimit(w http.ResponseWriter, limit *int) (int, bool) {
	pageLimit := searchDefaultLimit
	if limit != nil {
		if *limit < 1 {
			s.writeValidationError(w, "limit must be at least 1", []string{"limit"})
			return 0, false
		}
		if *limit > searchMaxLimit {
			pageLimit = searchMaxLimit
		} else {
			pageLimit = *limit
		}
	}
	return pageLimit, true
}

// searchFilters validates the filter parameters and folds them onto
// the search package's filter set.
func (s *server) searchFilters(w http.ResponseWriter, params apigen.SearchAssetsParams, query string) (search.Filters, bool) {
	filters := search.Filters{Query: query}

	if params.From != nil && params.To != nil && !params.From.Before(*params.To) {
		s.writeValidationError(w, "from must be before to", []string{"from", "to"})
		return filters, false
	}
	filters.From = params.From
	filters.To = params.To

	if ids := params.PersonIds; ids != nil {
		if len(*ids) > searchMaxFilterEntries {
			s.writeValidationError(w, "personIds is too long", []string{"personIds"})
			return filters, false
		}
		for _, id := range *ids {
			filters.PersonIDs = append(filters.PersonIDs, uuid.UUID(id))
		}
	}
	if ids := params.AlbumIds; ids != nil {
		if len(*ids) > searchMaxFilterEntries {
			s.writeValidationError(w, "albumIds is too long", []string{"albumIds"})
			return filters, false
		}
		for _, id := range *ids {
			filters.AlbumIDs = append(filters.AlbumIDs, uuid.UUID(id))
		}
	}
	if ids := params.TagIds; ids != nil {
		if len(*ids) > searchMaxFilterEntries {
			s.writeValidationError(w, "tagIds is too long", []string{"tagIds"})
			return filters, false
		}
		for _, id := range *ids {
			filters.TagIDs = append(filters.TagIDs, uuid.UUID(id))
		}
	}
	if ids := params.PlaceIds; ids != nil {
		if len(*ids) > searchMaxFilterEntries {
			s.writeValidationError(w, "placeIds is too long", []string{"placeIds"})
			return filters, false
		}
		for _, id := range *ids {
			if id < 1 {
				s.writeValidationError(w, "placeIds must carry positive place ids", []string{"placeIds"})
				return filters, false
			}
		}
		filters.PlaceIDs = *ids
	}
	if values := params.Cameras; values != nil {
		if len(*values) > searchMaxFilterEntries {
			s.writeValidationError(w, "cameras is too long", []string{"cameras"})
			return filters, false
		}
		filters.Cameras = *values
	}
	if kinds := params.Kinds; kinds != nil {
		if len(*kinds) > 2 {
			s.writeValidationError(w, "kinds is too long", []string{"kinds"})
			return filters, false
		}
		for _, kind := range *kinds {
			// The generated enum accepts the two members; a binder
			// reject never reaches here, but the membership check
			// stays explicit (the locale discipline).
			if kind != apigen.MediaKindImage && kind != apigen.MediaKindVideo {
				s.writeValidationError(w, "kinds must be image or video", []string{"kinds"})
				return filters, false
			}
			filters.Kinds = append(filters.Kinds, string(kind))
		}
	}
	return filters, true
}

// semanticInputs resolves the semantic arm's inputs: the query's
// text-tower embedding and the active clip-embed model id whose rows
// it searches. Every unready state degrades honestly (nil embedding)
// -- only a BROKEN active state (a model that fails to load) is an
// error.
func (s *server) semanticInputs(ctx context.Context, query string) ([]float32, string, error) {
	if s.models == nil {
		return nil, "", nil
	}

	textModel, err := s.models.Active(ml.SlotClipText)
	if err != nil {
		if errors.Is(err, ml.ErrNoActiveModel) {
			s.logger.Info("search: the semantic path is off: no active clip-text model")
			return nil, "", nil
		}
		return nil, "", fmt.Errorf("resolve the active clip-text model: %w", err)
	}
	visionModel, err := s.models.Active(ml.SlotClipEmbed)
	if err != nil {
		if errors.Is(err, ml.ErrNoActiveModel) {
			// Without an active vision tower nothing writes the
			// rows the arm searches: the same honest degradation.
			s.logger.Info("search: the semantic path is off: no active clip-embed model")
			return nil, "", nil
		}
		return nil, "", fmt.Errorf("resolve the active clip-embed model: %w", err)
	}

	session, tokenizer, err := s.textTower(textModel)
	if err != nil {
		return nil, "", fmt.Errorf("load the clip-text tower: %w", err)
	}
	embedding, err := mlclip.EmbedText(session, tokenizer, query)
	if err != nil {
		return nil, "", fmt.Errorf("embed the query: %w", err)
	}

	// The dimension guard: the tower's feature must match the
	// embeddings column's CURRENT dimension (the catalog's truth) --
	// a cross-dimension vision switch in between degrades the arm
	// instead of erroring at the vector comparison.
	columnDim, err := ml.EmbeddingColumnDim(ctx, s.pool)
	if err != nil {
		return nil, "", fmt.Errorf("read the clip_embeddings column dimension: %w", err)
	}
	if len(embedding) != columnDim {
		s.logger.Warn("search: the semantic path is off: the text tower's dimension disagrees with the clip_embeddings column",
			"text_model", textModel.ID, "embedding_dim", len(embedding), "column_dim", columnDim)
		return nil, "", nil
	}
	return embedding, visionModel.ID, nil
}

// textTowerSession is the process-local text tower: the active
// model's loaded session and tokenizer, kept warm for the search
// path (loading a 400 MB graph per request is not a page-shape cost
// the endpoint can carry). A model switch replaces the tower under
// the mutex and closes the old session.
type textTowerSession struct {
	mu        sync.Mutex
	modelID   string
	session   *ml.Session
	tokenizer *mlclip.Tokenizer
}

// textTower answers the text model's warm session and tokenizer,
// loading (or reloading after a model switch) on demand. The
// tokenizer's vocabulary rides the model's manifest as its second
// file: the params' "vocab" name wins, the conventional vocab.txt
// fallback covers manifests without the param.
func (s *server) textTower(model *ml.Model) (*ml.Session, *mlclip.Tokenizer, error) {
	s.textTowerCache.mu.Lock()
	defer s.textTowerCache.mu.Unlock()

	if s.textTowerCache.modelID == model.ID && s.textTowerCache.session != nil {
		return s.textTowerCache.session, s.textTowerCache.tokenizer, nil
	}

	files, err := ml.ModelFiles(model)
	if err != nil {
		return nil, nil, err
	}
	weights := files[0]
	vocabName := "vocab.txt"
	if name, ok := model.Params["vocab"].(string); ok && name != "" {
		vocabName = name
	}
	var vocab *ml.ModelFile
	for i := range files {
		if files[i].Name == vocabName {
			vocab = &files[i]
			break
		}
	}
	if vocab == nil {
		return nil, nil, fmt.Errorf("%w: the clip-text model %s lists no %s file",
			ml.ErrInvalidModel, model.ID, vocabName)
	}

	session, err := ml.NewSession(weights.Path, weights.SHA256)
	if err != nil {
		return nil, nil, err
	}
	tokenizer, err := mlclip.NewTokenizer(vocab.Path)
	if err != nil {
		_ = session.Close()
		return nil, nil, err
	}

	// The old tower (a switched-away model) closes under the same
	// mutex; an in-flight EmbedText on it completes safely --
	// onnxruntime sessions support concurrent Run.
	if s.textTowerCache.session != nil {
		_ = s.textTowerCache.session.Close()
	}
	s.textTowerCache.modelID = model.ID
	s.textTowerCache.session = session
	s.textTowerCache.tokenizer = tokenizer
	return session, tokenizer, nil
}

// encodeSearchCursor renders the cursor of one page's last row: the
// opaque base64 of the binary mode-tagged key. Each mode carries only
// its own ordering key; the other stays zeroed (the canonical wire
// form the decoder enforces -- the zero TIME itself is outside
// UnixNano's representable range, so the rank mode writes the zero
// NANOSECONDS instead).
func encodeSearchCursor(cursor search.Cursor) string {
	var buf [cursorWireLen]byte
	buf[0] = byte(cursor.Mode)
	if cursor.Mode == search.CursorFusedRank {
		binary.BigEndian.PutUint64(buf[1:9], uint64(cursor.Score))
	} else {
		binary.BigEndian.PutUint64(buf[9:17], uint64(cursor.TakenAt.UnixNano()))
	}
	copy(buf[17:], cursor.ID[:])
	return base64.RawURLEncoding.EncodeToString(buf[:])
}

// decodeSearchCursor parses a cursor this endpoint produced: the
// exact wire form, the mode byte in range, and the inactive mode's
// key zeroed (the canonical form). A fabricated but well-formed
// cursor is harmless: it addresses a point of the ordering.
func decodeSearchCursor(value string) (search.Cursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return search.Cursor{}, err
	}
	if len(raw) != cursorWireLen {
		return search.Cursor{}, fmt.Errorf("cursor holds %d bytes, want %d", len(raw), cursorWireLen)
	}
	mode := search.CursorMode(raw[0])
	if mode != search.CursorFilterBrowse && mode != search.CursorFusedRank {
		return search.Cursor{}, fmt.Errorf("cursor mode %d is unknown", mode)
	}
	cursor := search.Cursor{Mode: mode}
	score := int64(binary.BigEndian.Uint64(raw[1:9]))
	nanos := int64(binary.BigEndian.Uint64(raw[9:17]))
	copy(cursor.ID[:], raw[17:])
	switch mode {
	case search.CursorFusedRank:
		if nanos != 0 {
			return search.Cursor{}, fmt.Errorf("the rank cursor carries a taken-at key")
		}
		cursor.Score = score
	default:
		if score != 0 {
			return search.Cursor{}, fmt.Errorf("the browse cursor carries a fused score")
		}
		cursor.TakenAt = time.Unix(0, nanos).UTC()
	}
	return cursor, nil
}

// searchItemFrom maps one search result row onto the timeline item
// shape (the search page renders the same wall).
func searchItemFrom(item search.Item) apigen.TimelineItem {
	var livePhotoVideoID *openapi_types.UUID
	if item.LivePhotoVideoID != nil {
		id := openapi_types.UUID(*item.LivePhotoVideoID)
		livePhotoVideoID = &id
	}
	var durationMs *int
	if item.DurationMs != nil {
		value := int(*item.DurationMs)
		durationMs = &value
	}
	return apigen.TimelineItem{
		Id:               item.ID,
		OwnerId:          item.OwnerID,
		Kind:             apigen.TimelineItemKind(item.Kind),
		OriginalFilename: item.OriginalFilename,
		FileSize:         item.FileSize,
		MimeType:         item.MimeType,
		Checksum:         item.Checksum,
		CreatedAt:        item.CreatedAt.UTC(),
		TakenAt:          item.TakenAt.UTC(),
		TakenAtSource:    apigen.TimelineItemTakenAtSource(item.TakenAtSource),
		IsTrashed:        false,
		IsFavorite:       item.IsFavorite,
		IsReady:          item.IsReady,
		Width:            int(item.Width),
		Height:           int(item.Height),
		DurationMs:       durationMs,
		Blurhash:         item.Blurhash,
		LivePhotoVideoId: livePhotoVideoID,
	}
}

// searchFacetsFrom maps the search package's facet counts onto the
// contract shape (empty arrays, never null).
func searchFacetsFrom(facets search.Facets) apigen.SearchFacets {
	out := apigen.SearchFacets{
		People:  make([]apigen.PersonFacetCount, 0, len(facets.People)),
		Places:  make([]apigen.PlaceFacetCount, 0, len(facets.Places)),
		Years:   make([]apigen.YearFacetCount, 0, len(facets.Years)),
		Cameras: make([]apigen.CameraFacetCount, 0, len(facets.Cameras)),
		Kinds:   make([]apigen.KindFacetCount, 0, len(facets.Kinds)),
		Albums:  make([]apigen.AlbumFacetCount, 0, len(facets.Albums)),
		Tags:    make([]apigen.TagFacetCount, 0, len(facets.Tags)),
	}
	for _, person := range facets.People {
		out.People = append(out.People, apigen.PersonFacetCount{
			Id: person.ID, Name: person.Name, AssetCount: int(person.AssetCount),
		})
	}
	for _, place := range facets.Places {
		out.Places = append(out.Places, apigen.PlaceFacetCount{
			PlaceId: place.PlaceID, Name: place.Name, Country: place.Country, AssetCount: int(place.AssetCount),
		})
	}
	for _, year := range facets.Years {
		out.Years = append(out.Years, apigen.YearFacetCount{
			Year: year.Year, AssetCount: int(year.AssetCount),
		})
	}
	for _, camera := range facets.Cameras {
		out.Cameras = append(out.Cameras, apigen.CameraFacetCount{
			Id: camera.ID, Make: camera.Make, Model: camera.Model, AssetCount: int(camera.AssetCount),
		})
	}
	for _, kind := range facets.Kinds {
		out.Kinds = append(out.Kinds, apigen.KindFacetCount{
			Kind: apigen.MediaKind(kind.Kind), AssetCount: int(kind.AssetCount),
		})
	}
	for _, album := range facets.Albums {
		out.Albums = append(out.Albums, apigen.AlbumFacetCount{
			Id: album.ID, Title: album.Title, AssetCount: int(album.AssetCount),
		})
	}
	for _, tag := range facets.Tags {
		out.Tags = append(out.Tags, apigen.TagFacetCount{
			Id: tag.ID, Name: tag.Name, AssetCount: int(tag.AssetCount),
		})
	}
	return out
}
