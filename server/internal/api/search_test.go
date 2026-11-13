// Search endpoint tests: the unified search's two paths,
// their merge and the facets, exercised through the HTTP surface the
// contract pins. The text arm runs over seeded filename/description/
// OCR/tag/album-title/person-name fixtures through the unaccented
// substring matching of the search engine; the semantic arm runs over
// embeddings crafted from the fixture text tower's own output (the
// test embeds the query text through the same wrapper the handler
// uses, so the cosine ordering is exact); visibility is exercised
// with dual accounts (the sharee never sees an invisible asset from
// either arm); the facets are pinned against the honest counts of the
// current filtered set minus each dimension.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/ml"
	mlclip "github.com/welpher/lumemor/server/internal/ml/clip"
)

// searchAssetID and friends build the deterministic fixture ids of
// the search tests from two-hex-character suffixes, each namespace
// disjoint (assets, albums, tags, clusters, faces).
func searchAssetID(suffix string) uuid.UUID {
	return uuid.MustParse("00000000-0000-0000-0002-0000000000" + suffix)
}

func searchAlbumID(suffix string) uuid.UUID {
	return uuid.MustParse("20000000-0000-0000-0000-0000000000" + suffix)
}

func searchTagID(suffix string) uuid.UUID {
	return uuid.MustParse("30000000-0000-0000-0000-0000000000" + suffix)
}

func searchClusterID(suffix string) uuid.UUID {
	return uuid.MustParse("40000000-0000-0000-0000-0000000000" + suffix)
}

func searchFaceID(suffix string) uuid.UUID {
	return uuid.MustParse("50000000-0000-0000-0000-0000000000" + suffix)
}

// searchEmbedFixtureSHA256 pins the self-made text-tower fixture
// graph the fixture model directory carries (the same bytes as
// ../ml/testdata/text_embed.onnx).
const searchEmbedFixtureSHA256 = "0acd1246d21ac8dda64a7de1a6e830161554a68bd6b1380e6c239f7ebc588499"

// searchVocabSHA256 pins the committed tokenizer vocabulary: the
// canonical Chinese-CLIP vocab.txt bytes (byte-identical to the
// upstream table the default manifest's text tower downloads).
const searchVocabSHA256 = "45bbac6b341c319adc98a532532882e91a9cefc0329aa57bac9ae761c27b291c"

// searchEnv provisions the search tests: the asset environment's
// helpers (owned assets, albums, shares, faces, tags seeding) plus a
// registry over a throwaway models dir carrying the ACTIVE fixture
// text tower (clip_text) and vision tower (clip_embed), and a second
// signed-in account for the dual-account visibility arms.
type searchEnv struct {
	*assetEnv

	registry    *ml.Registry
	modelsDir   string
	graceID     uuid.UUID
	graceCookie *http.Cookie
}

// newSearchEnv provisions the environment with both fixture models
// registered and activated (the text tower and the vision tower), the
// administrator signed in as the owner and a second plain account for
// the sharee arms.
func newSearchEnv(t *testing.T) *searchEnv {
	t.Helper()

	env := newAssetEnv(t, testUploadLimit)
	modelsDir := t.TempDir()
	writeSearchModel(t, modelsDir, "clip-text-fixture", "clip-text",
		"text_embed.onnx", searchEmbedFixtureSHA256, "cn_clip_vocab.txt", searchVocabSHA256, 512)
	writeSearchModel(t, modelsDir, "clip-vision-fixture", "clip-embed",
		"const512.onnx", adminConst512SHA256, "", "", 512)

	registry := ml.NewRegistry(env.pool, dbgen.New(env.pool), modelsDir)
	if _, err := registry.Scan(modelsDir); err != nil {
		t.Fatalf("scan the fixture models in: %v", err)
	}
	if _, err := registry.Activate(ml.SlotClipText, "clip-text-fixture"); err != nil {
		t.Fatalf("activate the fixture text tower: %v", err)
	}
	if _, err := registry.Activate(ml.SlotClipEmbed, "clip-vision-fixture"); err != nil {
		t.Fatalf("activate the fixture vision tower: %v", err)
	}

	// The search router: the asset environment's wiring plus the
	// registry the semantic arm resolves its text tower through.
	env.router = NewRouter(&Deps{
		Pool:             env.pool,
		Version:          testVersion,
		Events:           env.bus,
		DataDir:          env.dataDir,
		UploadLimitBytes: testUploadLimit,
		Models:           registry,
		ModelsDir:        modelsDir,
	})

	graceID, graceCookie := env.createSecondUser(t, "search-grace@example.com")
	return &searchEnv{
		assetEnv:    env,
		registry:    registry,
		modelsDir:   modelsDir,
		graceID:     graceID,
		graceCookie: graceCookie,
	}
}

// writeSearchModel lays out one fixture model directory under
// modelsDir exactly the way a landed download does: manifest.json
// plus every pinned file. The vocabulary lands under the
// conventional vocab.txt name (the real text tower's manifest does
// the same).
func writeSearchModel(t *testing.T, modelsDir, id, slot, weights, weightsSHA, vocab, vocabSHA string, dim int) {
	t.Helper()

	dir := filepath.Join(modelsDir, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create model dir %s: %v", dir, err)
	}
	copyModelFixture(t, filepath.Join("..", "ml", "testdata", weights), filepath.Join(dir, weights))
	files := []ml.ManifestFile{pinnedModelFile(t, filepath.Join(dir, weights), weights, weightsSHA)}
	if vocab != "" {
		copyModelFixture(t, filepath.Join("..", "ml", "testdata", vocab), filepath.Join(dir, "vocab.txt"))
		files = append(files, pinnedModelFile(t, filepath.Join(dir, "vocab.txt"), "vocab.txt", vocabSHA))
	}
	model := ml.ManifestModel{
		ID:        id,
		Slot:      slot,
		Version:   "fixture",
		License:   "fixture",
		InputSize: ml.ManifestSize{Width: 52, Height: 1},
		Dim:       dim,
		Files:     files,
	}
	blob, err := json.Marshal(model)
	if err != nil {
		t.Fatalf("marshal fixture manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), blob, 0o644); err != nil {
		t.Fatalf("write fixture manifest: %v", err)
	}
}

// copyModelFixture copies one fixture file into a model directory.
func copyModelFixture(t *testing.T, from, to string) {
	t.Helper()
	data, err := os.ReadFile(from)
	if err != nil {
		t.Fatalf("read fixture %s: %v", from, err)
	}
	if err := os.WriteFile(to, data, 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", to, err)
	}
}

// pinnedModelFile describes one copied fixture file as the manifest
// pins it: name, digest and byte count.
func pinnedModelFile(t *testing.T, path, name, sha string) ml.ManifestFile {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat fixture %s: %v", path, err)
	}
	return ml.ManifestFile{Name: name, SHA256: sha, Bytes: info.Size()}
}

// searchAssetOpts carries the optional fixture values of a search
// asset; the zero value is a plain live, visible, ready image.
type searchAssetOpts struct {
	isComponent bool
	isHidden    bool
	deleted     bool
	mediaKind   string // "image" (default) or "video"
	placeID     int32
}

// insertSearchAsset inserts one fixture asset with the given
// filename and description AND lands it in the owner's sync album
// (the production invariant the caller-visible predicate builds on).
func (e *assetEnv) insertSearchAsset(t *testing.T, id, owner uuid.UUID, taken time.Time, filename, description string, opts searchAssetOpts) {
	t.Helper()

	kind := opts.mediaKind
	if kind == "" {
		kind = "image"
	}
	var deletedAt any
	if opts.deleted {
		deletedAt = time.Now()
	}
	var placeID any
	if opts.placeID != 0 {
		placeID = opts.placeID
	}
	e.exec(t, `
		INSERT INTO assets (
			id, owner_id, storage_kind, file_path, checksum, original_filename,
			file_size, mime_type, media_kind, width, height, taken_at,
			taken_at_source, is_component, is_hidden, deleted_at, place_id,
			ready_at, pending_jobs, description
		) VALUES (
			$1, $2, 'blob', $3, $4, $5, 1024, 'image/jpeg', $6, 100, 100, $7,
			'import', $8, $9, $10, $11, now(), 0, $12
		)`,
		pgUUID(id), pgUUID(owner), "originals/fixtures/"+id.String(),
		"fixture-checksum-"+id.String(), filename, kind, taken,
		opts.isComponent, opts.isHidden, deletedAt, placeID, description)
	e.exec(t, `
		INSERT INTO albums (id, owner_id, title, type)
		VALUES ($1, $2, 'Sync', 'sync')
		ON CONFLICT (owner_id) WHERE type = 'sync' DO NOTHING`,
		pgUUID(uuid.New()), pgUUID(owner))
	e.addTimelineItem(t, e.syncAlbumIDOf(t, owner), id, owner)
}

// insertSearchAlbum inserts one fixture album with an explicit title.
func (e *assetEnv) insertSearchAlbum(t *testing.T, id, owner uuid.UUID, albumType, title string) {
	t.Helper()
	e.exec(t, "INSERT INTO albums (id, owner_id, title, type) VALUES ($1, $2, $3, $4)",
		pgUUID(id), pgUUID(owner), title, albumType)
}

// insertOcrText seeds one OCR row for the asset (with the fixture
// OCR model row the schema's foreign key demands).
func (e *assetEnv) insertOcrText(t *testing.T, assetID uuid.UUID, text string) {
	t.Helper()
	e.exec(t, `
		INSERT INTO models (id, slot, version, source, file_path)
		VALUES ('fixture-ocr', 'ocr', '1.0.0', 'builtin', '/models/fixture-ocr.onnx')
		ON CONFLICT (id) DO NOTHING`)
	e.exec(t, `
		INSERT INTO ocr_texts (asset_id, model_id, text)
		VALUES ($1, 'fixture-ocr', $2)`,
		pgUUID(assetID), text)
}

// seedEmbedding stores one crafted embedding under the ACTIVE vision
// tower's model id (the rows the semantic arm filters by).
func (e *searchEnv) seedEmbedding(t *testing.T, assetID uuid.UUID, vec []float32) {
	t.Helper()
	e.exec(t, `
		INSERT INTO clip_embeddings (asset_id, model_id, embedding)
		VALUES ($1, 'clip-vision-fixture', $2)`,
		pgUUID(assetID), pgvector.NewVector(vec))
}

// seedTag creates the owner's tag with the given fixture id and
// assigns it to the assets.
func (e *assetEnv) seedTag(t *testing.T, id, owner uuid.UUID, name string, assets ...uuid.UUID) uuid.UUID {
	t.Helper()
	e.exec(t, "INSERT INTO tags (id, owner_id, name) VALUES ($1, $2, $3)",
		pgUUID(id), pgUUID(owner), name)
	for _, assetID := range assets {
		e.exec(t, "INSERT INTO asset_tags (asset_id, tag_id) VALUES ($1, $2)",
			pgUUID(assetID), pgUUID(id))
	}
	return id
}

// seedPlace inserts one place row with an optional zh-Hans name.
func (e *assetEnv) seedPlace(t *testing.T, id int32, name, zhName string) {
	t.Helper()
	e.exec(t, `
		INSERT INTO places (id, name, country_code, admin1_code, timezone, lat, lon)
		VALUES ($1, $2, 'FR', 'A', 'Europe/Paris', 0, 0)`, id, name)
	if zhName != "" {
		e.exec(t, `
			INSERT INTO place_names (geoname_id, locale, name)
			VALUES ($1, 'zh-Hans', $2)`, id, zhName)
	}
}

// embedFixtureText embeds text through the fixture text tower -- the
// same deterministic wrapper the handler runs, so crafted embeddings
// order the semantic arm exactly.
func (e *searchEnv) embedFixtureText(t *testing.T, text string) []float32 {
	t.Helper()
	if !ml.RuntimeAvailable() {
		t.Skip("onnxruntime shared library not present")
	}
	sess, err := ml.NewSession(filepath.Join(e.modelsDir, "clip-text-fixture", "text_embed.onnx"), searchEmbedFixtureSHA256)
	if err != nil {
		t.Fatalf("load the fixture text tower: %v", err)
	}
	t.Cleanup(func() { _ = sess.Close() })
	tok, err := mlclip.NewTokenizer(filepath.Join(e.modelsDir, "clip-text-fixture", "vocab.txt"))
	if err != nil {
		t.Fatalf("load the fixture vocabulary: %v", err)
	}
	vec, err := mlclip.EmbedText(sess, tok, text)
	if err != nil {
		t.Fatalf("embed %q through the fixture tower: %v", text, err)
	}
	return vec
}

// getSearch fires one search request and decodes the response body.
func (e *searchEnv) getSearch(t *testing.T, cookie *http.Cookie, rawQuery string) apigen.SearchResponse {
	t.Helper()
	rec := e.getSearchRaw(t, cookie, rawQuery)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /search?%s status = %d, want %d (body %q)", rawQuery, rec.Code, http.StatusOK, rec.Body.String())
	}
	var page apigen.SearchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode search response %q: %v", rec.Body.String(), err)
	}
	return page
}

// getSearchRaw fires one search request and returns the recorder.
func (e *searchEnv) getSearchRaw(t *testing.T, cookie *http.Cookie, rawQuery string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?"+rawQuery, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// searchItemsIDs maps the page's items onto their asset ids.
func searchItemsIDs(page apigen.SearchResponse) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(page.Items))
	for _, item := range page.Items {
		ids = append(ids, uuid.UUID(item.Id))
	}
	return ids
}

// requireSearchError asserts the 400 envelope, optionally requiring
// the named fields.
func requireSearchError(t *testing.T, rec *httptest.ResponseRecorder, fields ...string) {
	t.Helper()
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	apiErr, got := decodeError(t, rec)
	if apiErr.Code != apigen.VALIDATIONFAILED {
		t.Errorf("code = %q, want %q", apiErr.Code, apigen.VALIDATIONFAILED)
	}
	for _, want := range fields {
		if !containsField(got, want) {
			t.Errorf("details.fields = %v, want it to contain %q", got, want)
		}
	}
}

func containsField(fields []string, want string) bool {
	for _, field := range fields {
		if field == want {
			return true
		}
	}
	return false
}

func containsID(ids []uuid.UUID, want uuid.UUID) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}

func idOccurrences(ids []uuid.UUID, want uuid.UUID) int {
	count := 0
	for _, id := range ids {
		if id == want {
			count++
		}
	}
	return count
}

// requireSameSearchPage compares two search responses at the JSON
// level (items, cursor, facets) and reports any drift.
func requireSameSearchPage(t *testing.T, a, b apigen.SearchResponse) string {
	t.Helper()
	aj, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal first page: %v", err)
	}
	bj, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("marshal second page: %v", err)
	}
	if string(aj) != string(bj) {
		return fmt.Sprintf("the same query answered two different pages:\n%s\n%s", aj, bj)
	}
	return ""
}

// TestSearchTextPath walks the text arm's matching surfaces: every
// owned text source (filename, description, OCR) and every per-user
// relational source (tags, album titles, person names) hits; the
// query is an unaccented case-insensitive LITERAL substring
// (wildcards carry no meaning); an asset matching through several
// sources appears once; the same query answers the same page twice.
func TestSearchTextPath(t *testing.T) {
	env := newSearchEnv(t)
	owner := env.sessionUserID(t)

	taken := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)

	// The text-path fixtures: every asset matches the query 海边
	// through exactly one source except the multi-source one, whose
	// filename is the exact query (the unbeatable trigram
	// similarity).
	filename := searchAssetID("01")    // filename source
	description := searchAssetID("02") // description source
	ocr := searchAssetID("03")         // OCR source
	multi := searchAssetID("04")       // filename + description + OCR + tag + album + person
	none := searchAssetID("05")        // no source at all

	env.insertSearchAsset(t, filename, owner, taken, "海边2024.jpg", "", searchAssetOpts{})
	env.insertSearchAsset(t, description, owner, taken, "DSC_0001.jpg", "在海边度假", searchAssetOpts{})
	env.insertSearchAsset(t, ocr, owner, taken, "DSC_0002.jpg", "", searchAssetOpts{})
	env.insertOcrText(t, ocr, "杭州西湖景区 海边门票")
	env.insertSearchAsset(t, multi, owner, taken, "海边", "海边的一天", searchAssetOpts{})
	env.insertOcrText(t, multi, "海边浴场")
	env.insertSearchAsset(t, none, owner, taken, "DSC_0003.jpg", "山区徒步", searchAssetOpts{})

	// The relational sources: the caller's tag, a caller-visible
	// album's title, the caller's person label.
	env.seedTag(t, searchTagID("01"), owner, "海边合集", multi)
	album := searchAlbumID("01")
	env.insertSearchAlbum(t, album, owner, "memory", "海边之旅")
	env.addTimelineItem(t, album, multi, owner)
	env.insertPeopleModel(t)
	face := searchFaceID("01")
	cluster := searchClusterID("01")
	env.insertFace(t, face, multi, 0.9)
	env.seedPersonCluster(t, cluster, face)
	env.labelPersonCluster(t, owner, cluster, "海边的人")

	t.Run("SyncAlbumTitleIsNotATextSource", func(t *testing.T) {
		// Every fixture asset lands in the owner's auto-created "Sync"
		// album (the production invariant); an unguarded album-title
		// arm would answer the WHOLE library for any substring of
		// "Sync" (sync, ync, nc, ...). Only genuine text hits may
		// answer: no fixture text contains "sync", so the honest page
		// is empty and the years facet domain is empty with it.
		page := env.getSearch(t, env.cookie, "query="+url.QueryEscape("sync"))
		if got := searchItemsIDs(page); len(got) != 0 {
			t.Errorf("the sync query answered %d assets, want 0 (the sync album's title is not a text source)", len(got))
		}
		requireFacetYears(t, page.Facets.Years)
	})

	t.Run("EverySourceHitsAndDedups", func(t *testing.T) {
		page := env.getSearch(t, env.cookie, "query="+url.QueryEscape("海边"))
		got := searchItemsIDs(page)
		// The multi-source asset appears exactly once (dedup by asset
		// id), every single-source asset hits, the no-source asset
		// does not.
		if count := idOccurrences(got, multi); count != 1 {
			t.Errorf("the multi-source asset occurs %d times, want exactly 1 (dedup by asset id)", count)
		}
		for _, want := range []uuid.UUID{filename, description, ocr, multi} {
			if !containsID(got, want) {
				t.Errorf("items = %v, want the hit %s", got, want)
			}
		}
		if containsID(got, none) {
			t.Errorf("items = %v, want %s absent (no source matches)", got, none)
		}
	})

	t.Run("ChineseSubstringIsLiteral", func(t *testing.T) {
		// 西湖 appears inside the OCR text as a substring: the match
		// is containment, not word segmentation.
		page := env.getSearch(t, env.cookie, "query="+url.QueryEscape("西湖"))
		if got := searchItemsIDs(page); !containsID(got, ocr) {
			t.Errorf("items = %v, want the OCR-substring hit %s", got, ocr)
		}
	})

	t.Run("ExactFilenameLeads", func(t *testing.T) {
		// The rank is the trigram similarity of the best matched
		// source: the exact filename 海边 is the unbeatable match, so
		// the multi-source asset leads the page.
		page := env.getSearch(t, env.cookie, "query="+url.QueryEscape("海边"))
		got := searchItemsIDs(page)
		if len(got) < 2 || got[0] != multi {
			t.Errorf("items = %v, want the exact-filename asset first", got)
		}
	})

	t.Run("CaseAndAccentsFold", func(t *testing.T) {
		accented := searchAssetID("06")
		env.insertSearchAsset(t, accented, owner, taken, "café morning.jpg", "", searchAssetOpts{})
		lower := searchAssetID("07")
		env.insertSearchAsset(t, lower, owner, taken, "DSC_0004.jpg", "BEACH DAY", searchAssetOpts{})
		beaches := searchAssetID("08")
		env.insertSearchAsset(t, beaches, owner, taken, "DSC_0005.jpg", "the beaches of Normandy", searchAssetOpts{})

		for _, query := range []string{"CAFE", "cafe"} {
			page := env.getSearch(t, env.cookie, "query="+url.QueryEscape(query))
			if got := searchItemsIDs(page); !containsID(got, accented) {
				t.Errorf("query %q: items = %v, want the accented hit %s (unaccented matching)", query, got, accented)
			}
		}
		page := env.getSearch(t, env.cookie, "query="+url.QueryEscape("beach"))
		if got := searchItemsIDs(page); !containsID(got, lower) || !containsID(got, beaches) {
			t.Errorf("items = %v, want the case-folded and substring hits", got)
		}
	})

	t.Run("OtherLanguagesMatch", func(t *testing.T) {
		// The substring matching is language-agnostic: Japanese and
		// French text hit by their own words.
		japanese := searchAssetID("09")
		env.insertSearchAsset(t, japanese, owner, taken, "DSC_0006.jpg", "", searchAssetOpts{})
		env.insertOcrText(t, japanese, "東京タワーに行った")
		french := searchAssetID("0a")
		env.insertSearchAsset(t, french, owner, taken, "DSC_0007.jpg", "la plage de Nice", searchAssetOpts{})

		page := env.getSearch(t, env.cookie, "query="+url.QueryEscape("東京"))
		if got := searchItemsIDs(page); !containsID(got, japanese) {
			t.Errorf("items = %v, want the Japanese OCR hit %s", got, japanese)
		}
		page = env.getSearch(t, env.cookie, "query="+url.QueryEscape("plage"))
		if got := searchItemsIDs(page); !containsID(got, french) {
			t.Errorf("items = %v, want the French description hit %s", got, french)
		}
	})

	t.Run("WildcardsAreLiteral", func(t *testing.T) {
		// ILIKE's % and _ carry no meaning: the query is a literal
		// substring, so "100%" matches the literal "100%" text and
		// never globs.
		percent := searchAssetID("0b")
		env.insertSearchAsset(t, percent, owner, taken, "DSC_0008.jpg", "discount 100% off", searchAssetOpts{})
		underscore := searchAssetID("0c")
		env.insertSearchAsset(t, underscore, owner, taken, "DSC_0009.jpg", "code a_b token", searchAssetOpts{})

		page := env.getSearch(t, env.cookie, "query="+url.QueryEscape("100%"))
		if got := searchItemsIDs(page); !containsID(got, percent) || containsID(got, underscore) {
			t.Errorf("items = %v, want the literal-percent hit only", got)
		}
		// "1%0" would glob if the wildcard were live; the literal
		// reading finds nothing.
		page = env.getSearch(t, env.cookie, "query="+url.QueryEscape("1%0"))
		if got := searchItemsIDs(page); len(got) != 0 {
			t.Errorf("items = %v, want none (no literal 1%%0 substring exists)", got)
		}
		page = env.getSearch(t, env.cookie, "query="+url.QueryEscape("a_b"))
		if got := searchItemsIDs(page); !containsID(got, underscore) || containsID(got, percent) {
			t.Errorf("items = %v, want the literal-underscore hit only", got)
		}
	})

	t.Run("SameQuerySamePage", func(t *testing.T) {
		first := env.getSearch(t, env.cookie, "query="+url.QueryEscape("海边"))
		second := env.getSearch(t, env.cookie, "query="+url.QueryEscape("海边"))
		if rec := requireSameSearchPage(t, first, second); rec != "" {
			t.Error(rec)
		}
	})

	t.Run("EmptyAndShortQueriesAreFilterBrowse", func(t *testing.T) {
		// Empty, whitespace and single-character queries take no
		// text path: the response is the pure filter browse over the
		// caller's whole visible set (every fixture asset), exactly
		// like no query at all. 12 assets exist by this subtest (the
		// case/accents, language and wildcard fixtures included).
		for _, query := range []string{"", "%20%20%20", "%E6%B5%B7"} {
			page := env.getSearch(t, env.cookie, "query="+query)
			if got := len(page.Items); got != 12 {
				t.Errorf("query %q returned %d items, want all 12 visible fixture assets", query, got)
			}
		}
	})

	t.Run("QueryTooLongIsRefused", func(t *testing.T) {
		long := strings.Repeat("海", 257)
		rec := env.getSearchRaw(t, env.cookie, "query="+url.QueryEscape(long))
		requireSearchError(t, rec, "query")
	})
}

// TestSearchSemanticPath pins the semantic arm: the query embeds
// through the ACTIVE text tower, the cosine ordering over the
// caller-visible embeddings decides the arm's ranking, both arms fuse
// deterministically with dedup, and a missing text tower (or a
// dimension mismatch with the embeddings column) degrades the
// semantic path to empty while the text arm still serves.
func TestSearchSemanticPath(t *testing.T) {
	env := newSearchEnv(t)
	owner := env.sessionUserID(t)
	taken := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)

	beach := searchAssetID("11")    // embedding of 海边
	mountain := searchAssetID("12") // embedding of 山脉
	noEmbed := searchAssetID("13")  // no embedding row
	env.insertSearchAsset(t, beach, owner, taken, "DSC_1000.jpg", "", searchAssetOpts{})
	env.insertSearchAsset(t, mountain, owner, taken, "DSC_1001.jpg", "", searchAssetOpts{})
	env.insertSearchAsset(t, noEmbed, owner, taken, "DSC_1002.jpg", "", searchAssetOpts{})

	env.seedEmbedding(t, beach, env.embedFixtureText(t, "海边"))
	env.seedEmbedding(t, mountain, env.embedFixtureText(t, "山脉"))

	t.Run("VectorArmOrdersByCosine", func(t *testing.T) {
		// No text source matches; the hits come from the vector arm
		// alone, ordered by cosine similarity to the embedded query.
		page := env.getSearch(t, env.cookie, "query="+url.QueryEscape("海边"))
		got := searchItemsIDs(page)
		if len(got) != 2 || got[0] != beach || got[1] != mountain {
			t.Errorf("items = %v, want [%s %s] (cosine order, no embedding row absent)", got, beach, mountain)
		}
		if containsID(got, noEmbed) {
			t.Errorf("items = %v, want %s absent (no embedding)", got, noEmbed)
		}
	})

	t.Run("BothArmsFuseWithDedup", func(t *testing.T) {
		// Both arms: the exact filename hit carries the best trigram
		// similarity AND the closest embedding; ftsOnly matches the
		// text only; vecOnly carries the second-closest embedding.
		// The earlier subtest's embeddings leave the scope so the
		// fused set is exactly these three.
		env.exec(t, "DELETE FROM clip_embeddings WHERE asset_id IN ($1, $2)", pgUUID(beach), pgUUID(mountain))

		ftsSem := searchAssetID("14")
		ftsOnly := searchAssetID("15")
		vecOnly := searchAssetID("16")
		env.insertSearchAsset(t, ftsSem, owner, taken, "海边", "", searchAssetOpts{})
		env.insertSearchAsset(t, ftsOnly, owner, taken, "海边照片.jpg", "", searchAssetOpts{})
		env.insertSearchAsset(t, vecOnly, owner, taken, "DSC_1003.jpg", "", searchAssetOpts{})
		env.seedEmbedding(t, ftsSem, env.embedFixtureText(t, "海边"))
		env.seedEmbedding(t, vecOnly, env.embedFixtureText(t, "山脉"))

		page := env.getSearch(t, env.cookie, "query="+url.QueryEscape("海边"))
		got := searchItemsIDs(page)
		if len(got) != 3 {
			t.Fatalf("items = %v, want 3 fused hits", got)
		}
		if got[0] != ftsSem {
			t.Errorf("items[0] = %s, want the both-arms asset %s (rank-sum fusion)", got[0], ftsSem)
		}
		if count := idOccurrences(got, ftsSem); count != 1 {
			t.Errorf("the both-arms asset occurs %d times, want 1 (dedup by asset id)", count)
		}
		if !containsID(got, ftsOnly) || !containsID(got, vecOnly) {
			t.Errorf("items = %v, want both single-arm hits present", got)
		}

		// Same query, same page: the merge rule is deterministic.
		again := env.getSearch(t, env.cookie, "query="+url.QueryEscape("海边"))
		if rec := requireSameSearchPage(t, page, again); rec != "" {
			t.Error(rec)
		}
	})

	t.Run("NoActiveTextTowerDegradesToEmpty", func(t *testing.T) {
		// A fresh environment with NO text tower activated: the
		// semantic path answers empty, the text arm still serves.
		fresh := newAssetEnv(t, testUploadLimit)
		modelsDir := t.TempDir()
		writeSearchModel(t, modelsDir, "clip-vision-fixture", "clip-embed",
			"const512.onnx", adminConst512SHA256, "", "", 512)
		registry := ml.NewRegistry(fresh.pool, dbgen.New(fresh.pool), modelsDir)
		if _, err := registry.Scan(modelsDir); err != nil {
			t.Fatalf("scan the fixture vision tower in: %v", err)
		}
		if _, err := registry.Activate(ml.SlotClipEmbed, "clip-vision-fixture"); err != nil {
			t.Fatalf("activate the fixture vision tower: %v", err)
		}
		fresh.router = NewRouter(&Deps{
			Pool:             fresh.pool,
			Version:          testVersion,
			Events:           fresh.bus,
			DataDir:          fresh.dataDir,
			UploadLimitBytes: testUploadLimit,
			Models:           registry,
			ModelsDir:        modelsDir,
		})
		other := fresh.sessionUserID(t)
		textHit := searchAssetID("17")
		vecOnly := searchAssetID("18")
		fresh.insertSearchAsset(t, textHit, other, taken, "海边.jpg", "", searchAssetOpts{})
		fresh.insertSearchAsset(t, vecOnly, other, taken, "DSC_1004.jpg", "", searchAssetOpts{})
		if err := seedEmbeddingFor(t, fresh.pool, vecOnly, "clip-vision-fixture", env.embedFixtureText(t, "海边")); err != nil {
			t.Fatalf("seed the embedding: %v", err)
		}

		rec := fresh.getSearchRaw2(t, "query="+url.QueryEscape("海边"))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
		}
		var page apigen.SearchResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
			t.Fatalf("decode search response: %v", err)
		}
		got := searchItemsIDs(page)
		if len(got) != 1 || got[0] != textHit {
			t.Errorf("items = %v, want only the text hit %s (the semantic path degrades to empty)", got, textHit)
		}
	})

	t.Run("ColumnDimensionMismatchDegradesSemantic", func(t *testing.T) {
		// The embeddings column re-dimensioned to 768 (a 768-dim
		// vision model active) while the text tower emits 512: the
		// semantic path refuses honestly instead of erroring, and the
		// text arm still serves. The 768-dim embedding sits on an
		// asset with NO text match: if the semantic path wrongly ran,
		// that asset would surface (or the dimension comparison would
		// error the whole request).
		writeSearchModel(t, env.modelsDir, "clip-vision-768-fixture", "clip-embed",
			"const768.onnx", adminConst768SHA256, "", "", 768)
		if _, err := env.registry.Scan(env.modelsDir); err != nil {
			t.Fatalf("scan the 768-dim vision fixture in: %v", err)
		}
		// The activation's cross-dimension switch wipes the embedding
		// rows and re-types the column to vector(768).
		if _, err := env.registry.Activate(ml.SlotClipEmbed, "clip-vision-768-fixture"); err != nil {
			t.Fatalf("activate the 768-dim vision fixture: %v", err)
		}

		textHit := searchAssetID("19")
		env.insertSearchAsset(t, textHit, owner, taken, "海边.jpg", "", searchAssetOpts{})
		vecOnly768 := searchAssetID("1a")
		env.insertSearchAsset(t, vecOnly768, owner, taken, "DSC_1005.jpg", "", searchAssetOpts{})
		big := make([]float32, 768)
		for i := range big {
			big[i] = float32(i%7 + 1)
		}
		if err := seedEmbeddingFor(t, env.pool, vecOnly768, "clip-vision-768-fixture", big); err != nil {
			t.Fatalf("seed the 768-dim embedding: %v", err)
		}

		page := env.getSearch(t, env.cookie, "query="+url.QueryEscape("海边"))
		got := searchItemsIDs(page)
		// The 768-dim embedding's asset (no text match) must stay
		// absent -- the degraded semantic path cannot surface it --
		// while every text hit (the earlier subtests' fixtures
		// included) still serves.
		if containsID(got, vecOnly768) {
			t.Errorf("items = %v, want the 768-dim embedding asset absent (the 512-dim text tower cannot search the 768-dim column)", got)
		}
		if !containsID(got, textHit) {
			t.Errorf("items = %v, want the text hit %s present (the text arm still serves)", got, textHit)
		}
	})
}

// seedEmbeddingFor stores one embedding row over a raw pool.
func seedEmbeddingFor(t *testing.T, pool *pgxpool.Pool, assetID uuid.UUID, modelID string, vec []float32) error {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		"INSERT INTO clip_embeddings (asset_id, model_id, embedding) VALUES ($1, $2, $3)",
		pgUUID(assetID), modelID, pgvector.NewVector(vec))
	return err
}

// getSearchRaw2 fires one search request over a bare asset
// environment's router.
func (e *assetEnv) getSearchRaw2(t *testing.T, rawQuery string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?"+rawQuery, nil)
	req.AddCookie(e.cookie)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// TestSearchVisibility pins the union predicate's penetration through
// BOTH arms and the facets (the dual-account scene): a sharee
// never sees an invisible asset from the text arm or the vector arm,
// the facet counts share the scoping, and a revoked share retracts
// everything.
func TestSearchVisibility(t *testing.T) {
	env := newSearchEnv(t)
	owner := env.sessionUserID(t)
	taken := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)

	// The owner's three assets: private (never shared), shared (in
	// the album granted to Grace), and Grace's own. All three match
	// the query through filename AND carry embeddings -- the private
	// one carries the CLOSEST embedding, so an unguarded vector arm
	// would surface it first for the sharee.
	private := searchAssetID("21")
	shared := searchAssetID("22")
	graceOwn := searchAssetID("23")
	env.insertSearchAsset(t, private, owner, taken, "海边.jpg", "", searchAssetOpts{})
	env.insertSearchAsset(t, shared, owner, taken, "海边的合影.jpg", "", searchAssetOpts{})
	env.insertSearchAsset(t, graceOwn, env.graceID, taken, "我的海边.jpg", "", searchAssetOpts{})

	sharedAlbum := searchAlbumID("21")
	env.insertSearchAlbum(t, sharedAlbum, owner, "memory", "Family")
	env.addTimelineItem(t, sharedAlbum, shared, owner)
	env.shareTimelineAlbum(t, sharedAlbum, env.graceID, owner)

	// The private asset's embedding is the closest to the query
	// (cosine 1.0); the others carry a strictly different direction.
	env.seedEmbedding(t, private, env.embedFixtureText(t, "海边"))
	env.seedEmbedding(t, shared, env.embedFixtureText(t, "山脉"))
	env.seedEmbedding(t, graceOwn, env.embedFixtureText(t, "山脉"))

	// The owner's tag on the shared asset: HIS vocabulary, never
	// Grace's.
	env.seedTag(t, searchTagID("21"), owner, "海边合集", shared)

	t.Run("ShareeSeesOnlyVisibleFromBothArms", func(t *testing.T) {
		page := env.getSearch(t, env.graceCookie, "query="+url.QueryEscape("海边"))
		got := searchItemsIDs(page)
		if containsID(got, private) {
			t.Errorf("the sharee's search returned the private asset %s -- the union predicate must gate BOTH arms", private)
		}
		if !containsID(got, shared) || !containsID(got, graceOwn) {
			t.Errorf("items = %v, want the shared asset %s and the own asset %s", got, shared, graceOwn)
		}

		// The owner sees his own two, never Grace's.
		ownerPage := env.getSearch(t, env.cookie, "query="+url.QueryEscape("海边"))
		ownerGot := searchItemsIDs(ownerPage)
		if containsID(ownerGot, graceOwn) {
			t.Errorf("the owner's search returned Grace's asset %s", graceOwn)
		}
		if !containsID(ownerGot, private) || !containsID(ownerGot, shared) {
			t.Errorf("items = %v, want the private and shared assets", ownerGot)
		}
	})

	t.Run("FacetCountsShareTheScoping", func(t *testing.T) {
		// Grace's tag facet is empty (her vocabulary has no tags);
		// the owner's carries his tag over his filtered set.
		gracePage := env.getSearch(t, env.graceCookie, "query="+url.QueryEscape("海边"))
		if len(gracePage.Facets.Tags) != 0 {
			t.Errorf("grace's tag facet = %v, want empty (another account's tags never surface)", gracePage.Facets.Tags)
		}
		ownerPage := env.getSearch(t, env.cookie, "query="+url.QueryEscape("海边"))
		if len(ownerPage.Facets.Tags) != 1 ||
			ownerPage.Facets.Tags[0].Name != "海边合集" ||
			ownerPage.Facets.Tags[0].AssetCount != 1 {
			t.Errorf("owner's tag facet = %v, want the one tag over his filtered set", ownerPage.Facets.Tags)
		}
	})

	t.Run("RevokedShareRetracts", func(t *testing.T) {
		env.exec(t, "DELETE FROM album_shares WHERE album_id = $1", pgUUID(sharedAlbum))

		page := env.getSearch(t, env.graceCookie, "query="+url.QueryEscape("海边"))
		got := searchItemsIDs(page)
		if containsID(got, shared) {
			t.Errorf("items = %v after the revoke, want the shared asset retracted", got)
		}
		if !containsID(got, graceOwn) {
			t.Errorf("items = %v, want Grace's own asset to survive the revoke", got)
		}
	})
}

// TestSearchFiltersAndFacets pins the filter bar and the honest facet
// semantics: each dimension filters the result set, every facet
// counts the CURRENT filtered set MINUS its own dimension, and ids
// the caller cannot resolve (invisible albums, foreign tags) drop
// out of the filter instead of leaking.
func TestSearchFiltersAndFacets(t *testing.T) {
	env := newSearchEnv(t)
	owner := env.sessionUserID(t)

	env.seedPlace(t, 7, "Paris", "巴黎")
	env.seedPlace(t, 8, "Tokyo", "东京")

	paris2024 := searchAssetID("31") // image, Paris, iPhone, beach, Trip album, 2024-06
	paris2023 := searchAssetID("32") // video, Paris, iPhone, mountain, Trip album, 2023-06
	tokyo2024 := searchAssetID("33") // image, Tokyo, Sony, beach, 2024-07
	plain2025 := searchAssetID("34") // image, no place, no camera, 2025-01

	env.insertSearchAsset(t, paris2024, owner, time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC),
		"IMG_0001.jpg", "", searchAssetOpts{placeID: 7})
	env.insertSearchAsset(t, paris2023, owner, time.Date(2023, 6, 1, 12, 0, 0, 0, time.UTC),
		"IMG_0002.jpg", "", searchAssetOpts{placeID: 7, mediaKind: "video"})
	env.insertSearchAsset(t, tokyo2024, owner, time.Date(2024, 7, 1, 12, 0, 0, 0, time.UTC),
		"IMG_0003.jpg", "", searchAssetOpts{placeID: 8})
	env.insertSearchAsset(t, plain2025, owner, time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		"IMG_0004.jpg", "", searchAssetOpts{})

	env.exec(t, "INSERT INTO exif (asset_id, exif, camera_make, camera_model) VALUES ($1, '{}', 'Apple', 'iPhone 15')", pgUUID(paris2024))
	env.exec(t, "INSERT INTO exif (asset_id, exif, camera_make, camera_model) VALUES ($1, '{}', 'Apple', 'iPhone 15')", pgUUID(paris2023))
	env.exec(t, "INSERT INTO exif (asset_id, exif, camera_make, camera_model) VALUES ($1, '{}', 'Sony', 'A7')", pgUUID(tokyo2024))

	beachTag := env.seedTag(t, searchTagID("31"), owner, "beach", paris2024, tokyo2024)
	env.seedTag(t, searchTagID("32"), owner, "mountain", paris2023)

	tripAlbum := searchAlbumID("31")
	env.insertSearchAlbum(t, tripAlbum, owner, "memory", "Trip")
	env.addTimelineItem(t, tripAlbum, paris2024, owner)
	env.addTimelineItem(t, tripAlbum, paris2023, owner)

	t.Run("FilterBrowseAndFullFacets", func(t *testing.T) {
		page := env.getSearch(t, env.cookie, "")
		if got := len(page.Items); got != 4 {
			t.Fatalf("items = %d, want all 4 fixture assets", got)
		}
		facets := page.Facets
		requireFacetYears(t, facets.Years, yearCount{2025, 1}, yearCount{2024, 2}, yearCount{2023, 1})
		requireFacetPlaces(t, facets.Places, placeCount{"Paris", 2}, placeCount{"Tokyo", 1})
		requireFacetCameras(t, facets.Cameras, cameraCount{"Apple", "iPhone 15", 2}, cameraCount{"Sony", "A7", 1})
		requireFacetKinds(t, facets.Kinds, kindCount{"image", 3}, kindCount{"video", 1})
		requireFacetAlbums(t, facets.Albums, albumCount{"Trip", 2})
		requireFacetTags(t, facets.Tags, tagCount{"beach", 2}, tagCount{"mountain", 1})
		if len(facets.People) != 0 {
			t.Errorf("people facet = %v, want empty (no labeled faces)", facets.People)
		}
	})

	t.Run("TimeRangeFiltersButYearFacetStaysHonest", func(t *testing.T) {
		// from=2024-01-01 narrows the items to the two 2024 assets
		// and the 2025 one; the YEARS facet counts the set WITHOUT
		// the time-range dimension (all three years), every other
		// facet counts the narrowed set.
		page := env.getSearch(t, env.cookie, "from=2024-01-01T00:00:00Z")
		if got := len(page.Items); got != 3 {
			t.Fatalf("items = %d, want 3 (the 2024 and 2025 assets)", got)
		}
		facets := page.Facets
		requireFacetYears(t, facets.Years, yearCount{2025, 1}, yearCount{2024, 2}, yearCount{2023, 1})
		requireFacetPlaces(t, facets.Places, placeCount{"Paris", 1}, placeCount{"Tokyo", 1})
		requireFacetCameras(t, facets.Cameras, cameraCount{"Apple", "iPhone 15", 1}, cameraCount{"Sony", "A7", 1})
		// No kind filter is applied, so the kinds facet counts the
		// narrowed set itself: the three images. The mountain tag sits
		// on the 2023 asset the from filter excluded, so it does not
		// list.
		requireFacetKinds(t, facets.Kinds, kindCount{"image", 3})
		requireFacetTags(t, facets.Tags, tagCount{"beach", 2})
	})

	t.Run("KindAndPlaceAndCameraFilters", func(t *testing.T) {
		page := env.getSearch(t, env.cookie, "kinds=video")
		if got := searchItemsIDs(page); len(got) != 1 || got[0] != paris2023 {
			t.Errorf("items = %v, want only the video asset", got)
		}
		// The kinds facet counts the set without the kind filter.
		requireFacetKinds(t, page.Facets.Kinds, kindCount{"image", 3}, kindCount{"video", 1})

		page = env.getSearch(t, env.cookie, "placeIds=7")
		if got := searchItemsIDs(page); len(got) != 2 {
			t.Errorf("items = %v, want the two Paris assets", got)
		}
		// The places facet counts the set without the place filter.
		requireFacetPlaces(t, page.Facets.Places, placeCount{"Paris", 2}, placeCount{"Tokyo", 1})

		page = env.getSearch(t, env.cookie, "cameras="+url.QueryEscape("Apple|iPhone 15"))
		if got := searchItemsIDs(page); len(got) != 2 {
			t.Errorf("items = %v, want the two iPhone assets", got)
		}
		requireFacetCameras(t, page.Facets.Cameras, cameraCount{"Apple", "iPhone 15", 2}, cameraCount{"Sony", "A7", 1})

		page = env.getSearch(t, env.cookie, "tagIds="+beachTag.String())
		if got := searchItemsIDs(page); len(got) != 2 {
			t.Errorf("items = %v, want the two beach-tagged assets", got)
		}
		requireFacetTags(t, page.Facets.Tags, tagCount{"beach", 2}, tagCount{"mountain", 1})

		page = env.getSearch(t, env.cookie, "albumIds="+tripAlbum.String())
		if got := searchItemsIDs(page); len(got) != 2 {
			t.Errorf("items = %v, want the two Trip-album assets", got)
		}
	})

	t.Run("PersonFilterAndFacet", func(t *testing.T) {
		// A labeled face on paris2024: the person filter narrows to
		// it, the people facet lists the caller's label with the
		// count of the set without the person filter.
		env.insertPeopleModel(t)
		face := searchFaceID("31")
		cluster := searchClusterID("31")
		env.insertFace(t, face, paris2024, 0.9)
		env.seedPersonCluster(t, cluster, face)
		env.labelPersonCluster(t, owner, cluster, "Mom")

		page := env.getSearch(t, env.cookie, "personIds="+cluster.String())
		if got := searchItemsIDs(page); len(got) != 1 || got[0] != paris2024 {
			t.Errorf("items = %v, want only the person's asset", got)
		}

		page = env.getSearch(t, env.cookie, "")
		// The people facet entry counts the current set's assets
		// carrying a face of the cluster: Mom's face sits on one
		// asset, so adding her filter would leave that one asset.
		if len(page.Facets.People) != 1 ||
			page.Facets.People[0].Name != "Mom" ||
			page.Facets.People[0].AssetCount != 1 {
			t.Errorf("people facet = %v, want Mom with the count of her face-bearing assets (1)", page.Facets.People)
		}
	})

	t.Run("UnresolvableFilterIdsDrop", func(t *testing.T) {
		// An album the caller cannot see and a foreign tag id drop
		// out of the filter: they contribute nothing (no error, no
		// empty-by-probe).
		privateAlbum := searchAlbumID("32")
		graceAsset := searchAssetID("35")
		env.insertSearchAsset(t, graceAsset, env.graceID, time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC),
			"DSC_2000.jpg", "", searchAssetOpts{})
		env.insertSearchAlbum(t, privateAlbum, env.graceID, "memory", "Grace Private")
		env.addTimelineItem(t, privateAlbum, graceAsset, env.graceID)

		page := env.getSearch(t, env.cookie, "albumIds="+privateAlbum.String())
		if got := len(page.Items); got != 4 {
			t.Errorf("items = %d with an invisible album filter, want 4 (the filter drops, it does not empty)", got)
		}

		foreignTag := searchTagID("33")
		env.exec(t, "INSERT INTO tags (id, owner_id, name) VALUES ($1, $2, 'grace-private')",
			pgUUID(foreignTag), pgUUID(env.graceID))
		page = env.getSearch(t, env.cookie, "tagIds="+foreignTag.String())
		if got := len(page.Items); got != 4 {
			t.Errorf("items = %d with a foreign tag filter, want 4 (the filter drops)", got)
		}
	})

	t.Run("PlaceNamesFollowLocale", func(t *testing.T) {
		page := env.getSearch(t, env.cookie, "locale=zh-Hans")
		requireFacetPlaces(t, page.Facets.Places, placeCount{"巴黎", 2}, placeCount{"东京", 1})
		rec := env.getSearchRaw(t, env.cookie, "locale=esperanto")
		requireSearchError(t, rec, "locale")
	})
}

// The facet-expectation helpers: each pins one facet list's entries
// and order (count descending, the deterministic tiebreak per
// dimension).
type yearCount struct {
	year  int32
	count int
}
type placeCount struct {
	name  string
	count int
}
type cameraCount struct {
	make, model string
	count       int
}
type kindCount struct {
	kind  string
	count int
}
type albumCount struct {
	title string
	count int
}
type tagCount struct {
	name  string
	count int
}

func requireFacetYears(t *testing.T, got []apigen.YearFacetCount, want ...yearCount) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("years facet = %v, want %v", got, want)
	}
	for i, w := range want {
		if got[i].Year != w.year || got[i].AssetCount != w.count {
			t.Errorf("years facet[%d] = %+v, want year %d count %d", i, got[i], w.year, w.count)
		}
	}
}

func requireFacetPlaces(t *testing.T, got []apigen.PlaceFacetCount, want ...placeCount) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("places facet = %v, want %v", got, want)
	}
	for i, w := range want {
		if got[i].Name != w.name || got[i].AssetCount != w.count {
			t.Errorf("places facet[%d] = %+v, want name %q count %d", i, got[i], w.name, w.count)
		}
	}
}

func requireFacetCameras(t *testing.T, got []apigen.CameraFacetCount, want ...cameraCount) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("cameras facet = %v, want %v", got, want)
	}
	for i, w := range want {
		make := ""
		if got[i].Make != nil {
			make = *got[i].Make
		}
		model := ""
		if got[i].Model != nil {
			model = *got[i].Model
		}
		if make != w.make || model != w.model || got[i].AssetCount != w.count {
			t.Errorf("cameras facet[%d] = %+v, want %q %q count %d", i, got[i], w.make, w.model, w.count)
		}
	}
}

func requireFacetKinds(t *testing.T, got []apigen.KindFacetCount, want ...kindCount) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("kinds facet = %v, want %v", got, want)
	}
	for i, w := range want {
		if string(got[i].Kind) != w.kind || got[i].AssetCount != w.count {
			t.Errorf("kinds facet[%d] = %+v, want kind %q count %d", i, got[i], w.kind, w.count)
		}
	}
}

func requireFacetAlbums(t *testing.T, got []apigen.AlbumFacetCount, want ...albumCount) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("albums facet = %v, want the %d pinned album entries", got, len(want))
	}
	for i, w := range want {
		if !strings.Contains(got[i].Title, w.title) || got[i].AssetCount != w.count {
			t.Errorf("albums facet[%d] = %+v, want a %q album with count %d", i, got[i], w.title, w.count)
		}
	}
}

func requireFacetTags(t *testing.T, got []apigen.TagFacetCount, want ...tagCount) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("tags facet = %v, want %v", got, want)
	}
	for i, w := range want {
		if got[i].Name != w.name || got[i].AssetCount != w.count {
			t.Errorf("tags facet[%d] = %+v, want name %q count %d", i, got[i], w.name, w.count)
		}
	}
}

// TestSearchPagination pins the keyset pages of both modes: the
// filter mode pages in the timeline's ordering, the query mode pages
// in the fused-score ordering, cursors are mode-tagged, and a mangled
// cursor is a client error.
func TestSearchPagination(t *testing.T) {
	env := newSearchEnv(t)
	owner := env.sessionUserID(t)
	taken := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)

	a := searchAssetID("41")
	b := searchAssetID("42")
	c := searchAssetID("43")
	d := searchAssetID("44")
	env.insertSearchAsset(t, a, owner, taken.Add(3*time.Hour), "DSC_3001.jpg", "", searchAssetOpts{})
	env.insertSearchAsset(t, b, owner, taken.Add(2*time.Hour), "DSC_3002.jpg", "", searchAssetOpts{})
	env.insertSearchAsset(t, c, owner, taken.Add(1*time.Hour), "DSC_3003.jpg", "", searchAssetOpts{})
	env.insertSearchAsset(t, d, owner, taken, "DSC_3004.jpg", "", searchAssetOpts{})

	t.Run("FilterModeKeyset", func(t *testing.T) {
		first := env.getSearch(t, env.cookie, "limit=3")
		if got := searchItemsIDs(first); len(got) != 3 || got[0] != a || got[2] != c {
			t.Fatalf("first page = %v, want the timeline order a b c", got)
		}
		if first.NextCursor == nil {
			t.Fatal("first page carries no nextCursor, want one (another page follows)")
		}
		second := env.getSearch(t, env.cookie, "limit=3&cursor="+url.QueryEscape(*first.NextCursor))
		if got := searchItemsIDs(second); len(got) != 1 || got[0] != d {
			t.Fatalf("second page = %v, want the trailing asset d", got)
		}
		if second.NextCursor != nil {
			t.Errorf("last page carries nextCursor %q, want none", *second.NextCursor)
		}
	})

	t.Run("QueryModeKeyset", func(t *testing.T) {
		// Four ranked text hits; the pages walk the fused order.
		for i, id := range []uuid.UUID{a, b, c, d} {
			env.insertOcrText(t, id, fmt.Sprintf("rank rank rank rank %d", i+1))
		}
		first := env.getSearch(t, env.cookie, "query=rank&limit=3")
		if got := len(first.Items); got != 3 {
			t.Fatalf("first page = %d items, want 3", got)
		}
		if first.NextCursor == nil {
			t.Fatal("first page carries no nextCursor, want one")
		}
		second := env.getSearch(t, env.cookie, "query=rank&limit=3&cursor="+url.QueryEscape(*first.NextCursor))
		if got := len(second.Items); got != 1 {
			t.Fatalf("second page = %d items, want 1", got)
		}
		if second.NextCursor != nil {
			t.Errorf("last page carries nextCursor %q, want none", *second.NextCursor)
		}
		// The concatenation is the full fused list -- every hit
		// appears exactly once across the pages.
		both := append(searchItemsIDs(first), searchItemsIDs(second)...)
		for _, id := range []uuid.UUID{a, b, c, d} {
			if count := idOccurrences(both, id); count != 1 {
				t.Errorf("asset %s occurs %d times across the pages, want 1", id, count)
			}
		}
		// Determinism: the same first-page request again answers the
		// same page and the same cursor.
		again := env.getSearch(t, env.cookie, "query=rank&limit=3")
		if rec := requireSameSearchPage(t, first, again); rec != "" {
			t.Error(rec)
		}
	})

	t.Run("CursorModeMismatchIsRefused", func(t *testing.T) {
		ranked := env.getSearch(t, env.cookie, "query=rank&limit=3")
		if ranked.NextCursor == nil {
			t.Fatal("the query page carries no cursor to borrow")
		}
		// A rank cursor used in filter mode is a mode mismatch.
		rec := env.getSearchRaw(t, env.cookie, "cursor="+url.QueryEscape(*ranked.NextCursor))
		requireSearchError(t, rec, "cursor")
		// ... and the reverse.
		filtered := env.getSearch(t, env.cookie, "limit=3")
		if filtered.NextCursor == nil {
			t.Fatal("the filter page carries no cursor to borrow")
		}
		rec = env.getSearchRaw(t, env.cookie, "query=rank&cursor="+url.QueryEscape(*filtered.NextCursor))
		requireSearchError(t, rec, "cursor")
	})

	t.Run("MangledCursorIsRefused", func(t *testing.T) {
		for _, cursor := range []string{"garbage", "AAAA", "!!!!"} {
			rec := env.getSearchRaw(t, env.cookie, "cursor="+url.QueryEscape(cursor))
			requireSearchError(t, rec, "cursor")
		}
	})

	t.Run("LimitBounds", func(t *testing.T) {
		rec := env.getSearchRaw(t, env.cookie, "limit=0")
		requireSearchError(t, rec, "limit")
		page := env.getSearch(t, env.cookie, "limit=1")
		if len(page.Items) != 1 {
			t.Errorf("limit=1 returned %d items, want 1", len(page.Items))
		}
		// Values over the maximum clamp to it (no 400).
		page = env.getSearch(t, env.cookie, "limit=500")
		if got := len(page.Items); got != 4 {
			t.Errorf("limit=500 returned %d items, want the 4 fixture assets (clamped page)", got)
		}
	})
}

// TestSearchLimitClampPinsTheCap pins the page-size cap against a
// bulk fixture: a request over the maximum answers exactly the cap,
// and the absent limit answers the default.
func TestSearchLimitClampPinsTheCap(t *testing.T) {
	env := newSearchEnv(t)
	owner := env.sessionUserID(t)
	taken := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)

	// 205 bulk assets through generate_series: ids in a disjoint
	// namespace, all owned and sync-landed.
	env.exec(t, `
		WITH bulk AS (
			SELECT gen_random_uuid() AS id
			FROM generate_series(1, 205)
		)
		INSERT INTO assets (
			id, owner_id, storage_kind, file_path, checksum, original_filename,
			file_size, mime_type, media_kind, width, height, taken_at,
			taken_at_source, is_component, is_hidden, ready_at, pending_jobs, description
		)
		SELECT id, $1, 'blob', 'originals/fixtures/' || id, 'bulk-' || id,
		       'DSC_' || id || '.jpg', 1024, 'image/jpeg', 'image', 100, 100,
		       $2, 'import', false, false, now(), 0, ''
		FROM bulk`,
		pgUUID(owner), taken)
	syncAlbum := env.syncAlbumIDOf(t, owner)
	env.exec(t, `
		INSERT INTO album_items (album_id, asset_id, added_by)
		SELECT $1, a.id, $2 FROM assets a WHERE a.owner_id = $2`,
		pgUUID(syncAlbum), pgUUID(owner))

	page := env.getSearch(t, env.cookie, "limit=500")
	if got := len(page.Items); got != 200 {
		t.Errorf("limit=500 returned %d items, want the 200 cap", got)
	}
	if page.NextCursor == nil {
		t.Error("the capped page carries no nextCursor, want one (assets remain)")
	}
	page = env.getSearch(t, env.cookie, "")
	if got := len(page.Items); got != 100 {
		t.Errorf("the default limit returned %d items, want 100", got)
	}
}

// TestSearchParamsValidation pins the parameter surface's refusals:
// the time-range inversion, the garbage ids and enums, the array
// bounds and the unauthenticated arm.
func TestSearchParamsValidation(t *testing.T) {
	env := newSearchEnv(t)

	t.Run("Unauthenticated", func(t *testing.T) {
		rec := env.getSearchRaw(t, nil, "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("FromAfterTo", func(t *testing.T) {
		rec := env.getSearchRaw(t, env.cookie, "from=2025-01-01T00:00:00Z&to=2024-01-01T00:00:00Z")
		requireSearchError(t, rec, "from", "to")
	})

	t.Run("GarbageIdsAndEnums", func(t *testing.T) {
		for _, query := range []string{
			"personIds=not-a-uuid",
			"albumIds=not-a-uuid",
			"tagIds=not-a-uuid",
			"kinds=hologram",
			"placeIds=-5",
			"from=not-a-date",
		} {
			t.Run(query, func(t *testing.T) {
				rec := env.getSearchRaw(t, env.cookie, query)
				requireSearchError(t, rec)
			})
		}
	})

	t.Run("ArrayBounds", func(t *testing.T) {
		many := make([]string, 201)
		for i := range many {
			many[i] = uuid.New().String()
		}
		rec := env.getSearchRaw(t, env.cookie, "personIds="+strings.Join(many, ","))
		requireSearchError(t, rec, "personIds")
	})
}
