// The admin settings tests (the storage-template feature and the
// settings feature): the storage template's read shape, default,
// three-stage validation and preset list; the trash retention's
// default, bounds and the expiry job's reaction to a saved 0 (the
// immediate-permanent-delete pin); the video transcoding profile's
// default, enum validation and corrupt-row default. The admin-role
// guard (403 for a signed-in plain user, 401 without a session) is
// pinned for every face.
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/trash"
)

// storageSettingsEnv provisions the storage settings test
// environment: a migrated throwaway database with the first
// administrator (created through the wizard, then signed in) as the
// caller.
type storageSettingsEnv struct {
	router http.Handler
	pool   *pgxpool.Pool
	cookie *http.Cookie
}

// newStorageSettingsEnv provisions the environment and signs the
// administrator in.
func newStorageSettingsEnv(t *testing.T) *storageSettingsEnv {
	t.Helper()
	pool := newSetupEnv(t)
	router := NewRouter(&Deps{Pool: pool, Version: testVersion})

	rec := postAdmin(t, router, createAdminBody(t, "ada@example.com", testPassword, nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create admin: status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	cookie, _ := loginCookie(t, router, "ada@example.com", testPassword, "storage-settings-tests")
	return &storageSettingsEnv{router: router, pool: pool, cookie: cookie}
}

// getSettings fires the read request at the router.
func (e *storageSettingsEnv) getSettings(t *testing.T, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings/storage", nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// putSettings fires the save request with one template value.
func (e *storageSettingsEnv) putSettings(t *testing.T, cookie *http.Cookie, template string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(apigen.StorageSettingsUpdateRequest{Template: template})
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings/storage", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// decodeSettings decodes a 200 settings body.
func decodeSettings(t *testing.T, rec *httptest.ResponseRecorder) apigen.StorageSettingsResponse {
	t.Helper()
	var response apigen.StorageSettingsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode settings body %q: %v", rec.Body.String(), err)
	}
	return response
}

// requireSavedTemplate asserts the instance_settings row holds the
// given template (a missing row is the empty answer).
func (e *storageSettingsEnv) requireSavedTemplate(t *testing.T, want string) {
	t.Helper()
	var value string
	err := e.pool.QueryRow(context.Background(),
		"SELECT value->>0 FROM instance_settings WHERE key = 'storage_template'").Scan(&value)
	if want == "" {
		if err == nil {
			t.Fatalf("the storage_template row exists holding %q, want no row", value)
		}
		return
	}
	if err != nil {
		t.Fatalf("load the storage_template row: %v", err)
	}
	if value != want {
		t.Errorf("saved template = %q, want %q", value, want)
	}
}

// TestStorageSettingsGetAnswersDefault pins the read of a fresh
// instance: the default template, its sample render and the sixteen
// presets.
func TestStorageSettingsGetAnswersDefault(t *testing.T) {
	env := newStorageSettingsEnv(t)

	rec := env.getSettings(t, env.cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	settings := decodeSettings(t, rec)
	if settings.Template != "{{y}}/{{MM}}/{{filename}}" {
		t.Errorf("template = %q, want the default", settings.Template)
	}
	wantSample := "originals/" + storageSampleOwner + "/2023/07/IMG_1234.jpg"
	if settings.SamplePath != wantSample {
		t.Errorf("samplePath = %q, want %q", settings.SamplePath, wantSample)
	}
	if len(settings.Presets) != 16 {
		t.Errorf("presets = %d entries, want 16", len(settings.Presets))
	}
	if len(settings.Presets) > 0 && settings.Presets[0] != "{{y}}/{{MM}}/{{filename}}" {
		t.Errorf("presets[0] = %q, want the default template first", settings.Presets[0])
	}
}

// TestStorageSettingsGetAnswersSavedTemplate pins the read after a
// save: the stored value and its own sample render (the camera
// preset's sample lands under the camera directories).
func TestStorageSettingsGetAnswersSavedTemplate(t *testing.T) {
	env := newStorageSettingsEnv(t)

	if rec := env.putSettings(t, env.cookie, "{{make}}/{{model}}/{{filename}}"); rec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	rec := env.getSettings(t, env.cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", rec.Code, http.StatusOK)
	}
	settings := decodeSettings(t, rec)
	if settings.Template != "{{make}}/{{model}}/{{filename}}" {
		t.Errorf("template = %q, want the saved camera preset", settings.Template)
	}
	wantSample := "originals/" + storageSampleOwner + "/Acme/Alpha 5/IMG_1234.jpg"
	if settings.SamplePath != wantSample {
		t.Errorf("samplePath = %q, want %q", settings.SamplePath, wantSample)
	}
}

// TestStorageSettingsPutSavesValidTemplate pins the save's happy path:
// a valid template persists and the response carries the saved value,
// its sample render and the presets.
func TestStorageSettingsPutSavesValidTemplate(t *testing.T) {
	env := newStorageSettingsEnv(t)

	rec := env.putSettings(t, env.cookie, "{{y}}-{{MM}}-{{dd}}/{{filename}}")
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	settings := decodeSettings(t, rec)
	if settings.Template != "{{y}}-{{MM}}-{{dd}}/{{filename}}" {
		t.Errorf("template = %q, want the saved value", settings.Template)
	}
	wantSample := "originals/" + storageSampleOwner + "/2023-07-14/IMG_1234.jpg"
	if settings.SamplePath != wantSample {
		t.Errorf("samplePath = %q, want %q", settings.SamplePath, wantSample)
	}
	if len(settings.Presets) != 16 {
		t.Errorf("presets = %d entries, want 16", len(settings.Presets))
	}
	env.requireSavedTemplate(t, "{{y}}-{{MM}}-{{dd}}/{{filename}}")

	// A second save replaces the row: the setting is a single value.
	if rec := env.putSettings(t, env.cookie, "{{assetId}}"); rec.Code != http.StatusOK {
		t.Fatalf("second PUT status = %d, want %d", rec.Code, http.StatusOK)
	}
	env.requireSavedTemplate(t, "{{assetId}}")
}

// TestStorageSettingsPutRejectsInvalidTemplates pins the three-stage
// validation: an unknown variable (the compile stage), an album-only
// template over the album-less sample (the sample-render stage), and
// a traversal or a depth violation (the structural stage) each answer
// 400 VALIDATION_FAILED with the reason in the message and nothing
// written.
func TestStorageSettingsPutRejectsInvalidTemplates(t *testing.T) {
	env := newStorageSettingsEnv(t)

	cases := []struct {
		name     string
		template string
		wantIn   string
	}{
		{"unknown variable", "{{category}}/{{filename}}", "category"},
		{"malformed conditional", "{{#if album}}x", "unterminated"},
		{"album-only render is empty", "{{album}}", "empty"},
		{"traversal in the literals", "../{{filename}}", "traversal"},
		{"depth over the bound", "a/b/c/d/e/f/g/h/i/j/{{filename}}", "depth"},
		{"empty template", "", "template"},
		{"template over the length bound", "{{y}}/" + strings.Repeat("x", 520), "rune"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rec := env.putSettings(t, env.cookie, tt.template)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("PUT status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
			var err apigen.Error
			if err := json.Unmarshal(rec.Body.Bytes(), &err); err != nil {
				t.Fatalf("decode error body %q: %v", rec.Body.String(), err)
			}
			if err.Code != apigen.VALIDATIONFAILED {
				t.Errorf("code = %q, want VALIDATION_FAILED", err.Code)
			}
			if err.Message == nil || !strings.Contains(*err.Message, tt.wantIn) {
				t.Errorf("message = %v, want it to mention %q", err.Message, tt.wantIn)
			}
			// Nothing was written: the setting row stays absent.
			env.requireSavedTemplate(t, "")
		})
	}
}

// TestStorageSettingsPutAcceptsEveryPreset proves the built-in preset
// list is saveable as-is: every preset passes the three stages.
func TestStorageSettingsPutAcceptsEveryPreset(t *testing.T) {
	env := newStorageSettingsEnv(t)

	for _, preset := range []string{
		"{{y}}/{{MM}}/{{filename}}",
		"{{y}}/{{y}}-{{MM}}/{{filename}}",
		"{{y}}/{{y}}-{{MM}}-{{dd}}/{{filename}}",
		"{{y}}/{{MMMM}}/{{filename}}",
		"{{y}}-{{MM}}-{{dd}}/{{filename}}",
		"{{y}}/{{MM}}/{{dd}}/{{filename}}",
		"{{y}}/{{filename}}",
		"{{#if album}}{{album}}/{{else}}{{y}}/{{MM}}/{{/if}}{{filename}}",
		"{{album}}/{{filename}}",
		"{{album-startDate-y}}/{{album}}/{{filename}}",
		"{{make}}/{{model}}/{{filename}}",
		"{{lensModel}}/{{filename}}",
		"{{filetype}}/{{y}}/{{MM}}/{{filename}}",
		"{{y}}/{{y}}-{{MM}}/{{assetId}}",
		"{{assetId}}",
		"{{filename}}",
	} {
		if rec := env.putSettings(t, env.cookie, preset); rec.Code != http.StatusOK {
			t.Errorf("PUT of preset %q status = %d, want %d (body %q)", preset, rec.Code, http.StatusOK, rec.Body.String())
		}
	}
}

// TestStorageSettingsGuardPinsTheAdminRole pins the permission model:
// the plain signed-in user answers 403, the anonymous caller 401, on
// both routes.
func TestStorageSettingsGuardPinsTheAdminRole(t *testing.T) {
	env := newStorageSettingsEnv(t)
	_, plainCookie := seedPlainUser(t, env.pool, "grace@example.com")

	for _, tt := range []struct {
		name   string
		method string
		cookie *http.Cookie
		want   int
	}{
		{"plain user GET", http.MethodGet, plainCookie, http.StatusForbidden},
		{"plain user PUT", http.MethodPut, plainCookie, http.StatusForbidden},
		{"anonymous GET", http.MethodGet, nil, http.StatusUnauthorized},
		{"anonymous PUT", http.MethodPut, nil, http.StatusUnauthorized},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/v1/admin/settings/storage", strings.NewReader(`{"template":"{{y}}"}`))
			req.Header.Set("Content-Type", "application/json")
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}
			rec := httptest.NewRecorder()
			env.router.ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Fatalf("%s status = %d, want %d (body %q)", tt.name, rec.Code, tt.want, rec.Body.String())
			}
		})
	}
}

// TestStorageSettingsCorruptRowAnswersDefault pins the read's
// corruption discipline: a row holding a non-string (or an
// over-long) value answers the default template, not an error -- the
// upload path depends on the read never failing.
func TestStorageSettingsCorruptRowAnswersDefault(t *testing.T) {
	env := newStorageSettingsEnv(t)

	for _, value := range []string{`42`, `null`, `""`, `"` + strings.Repeat("x", 600) + `"`} {
		if _, err := env.pool.Exec(context.Background(),
			`INSERT INTO instance_settings (key, value) VALUES ('storage_template', $1::jsonb)
			ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, value); err != nil {
			t.Fatalf("seed corrupt setting %q: %v", value, err)
		}
		rec := env.getSettings(t, env.cookie)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET over the corrupt row %q status = %d, want %d", value, rec.Code, http.StatusOK)
		}
		if settings := decodeSettings(t, rec); settings.Template != "{{y}}/{{MM}}/{{filename}}" {
			t.Errorf("template over the corrupt row %q = %q, want the default", value, settings.Template)
		}
	}
}

// getSetting fires the read request at one settings face.
func (e *storageSettingsEnv) getSetting(t *testing.T, path string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings/"+path, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// putSetting fires the save request with one JSON body at one
// settings face.
func (e *storageSettingsEnv) putSetting(t *testing.T, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings/"+path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// requireTrashRow asserts the trash_retention_days row's stored
// form: present with the given number, or absent for the zero value.
func (e *storageSettingsEnv) requireTrashRow(t *testing.T, want int) {
	t.Helper()
	var value int
	var kind string
	err := e.pool.QueryRow(context.Background(),
		`SELECT value::int, jsonb_typeof(value) FROM instance_settings WHERE key = 'trash_retention_days'`).
		Scan(&value, &kind)
	if err != nil {
		t.Fatalf("load the trash_retention_days row: %v", err)
	}
	if kind != "number" {
		t.Errorf("stored trash retention's JSON type = %q, want a number (the raw writer's discipline)", kind)
	}
	if value != want {
		t.Errorf("stored trash retention = %d, want %d", value, want)
	}
}

// TestTrashSettingsGetAnswersDefault pins the read of a fresh
// instance: the documented default retention of 30 days.
func TestTrashSettingsGetAnswersDefault(t *testing.T) {
	env := newStorageSettingsEnv(t)

	rec := env.getSetting(t, "trash", env.cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET trash status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var settings apigen.TrashSettingsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &settings); err != nil {
		t.Fatalf("decode trash settings %q: %v", rec.Body.String(), err)
	}
	if settings.RetentionDays != 30 {
		t.Errorf("retentionDays = %d, want the default 30", settings.RetentionDays)
	}
}

// TestTrashSettingsPutSavesBounds pins the save: the bound's both
// ends (0 and 365) persist as a jsonb NUMBER and answer the saved
// value; a value outside the bound answers 400 naming the field and
// writes nothing.
func TestTrashSettingsPutSavesBounds(t *testing.T) {
	env := newStorageSettingsEnv(t)

	for _, days := range []int{0, 365, 14} {
		rec := env.putSetting(t, "trash", `{"retentionDays":`+strconv.Itoa(days)+`}`, env.cookie)
		if rec.Code != http.StatusOK {
			t.Fatalf("PUT trash %d status = %d, want %d (body %q)", days, rec.Code, http.StatusOK, rec.Body.String())
		}
		var settings apigen.TrashSettingsResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &settings); err != nil {
			t.Fatalf("decode trash settings %q: %v", rec.Body.String(), err)
		}
		if settings.RetentionDays != days {
			t.Errorf("answered retentionDays = %d, want %d", settings.RetentionDays, days)
		}
		env.requireTrashRow(t, days)
	}

	for _, body := range []string{`{"retentionDays":-1}`, `{"retentionDays":366}`, `{"retentionDays":30.5}`, `not json`} {
		rec := env.putSetting(t, "trash", body, env.cookie)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("PUT trash %q status = %d, want %d (body %q)", body, rec.Code, http.StatusBadRequest, rec.Body.String())
			continue
		}
		apiErr, fields := decodeError(t, rec)
		if apiErr.Code != apigen.VALIDATIONFAILED {
			t.Errorf("PUT trash %q code = %q, want VALIDATION_FAILED", body, apiErr.Code)
		}
		if body != "not json" && len(fields) > 0 && fields[0] != "retentionDays" {
			t.Errorf("PUT trash %q fields = %v, want [retentionDays]", body, fields)
		}
	}
	// The last successful save's value survived the refused ones.
	env.requireTrashRow(t, 14)
}

// TestTrashSettingsZeroIsImmediateDelete pins THE trash pin: a saved
// retention of 0 makes the NEXT expiry run purge everything trashed
// -- rows and files together -- through the same accessor the
// hourly River job reads at the start of each run.
func TestTrashSettingsZeroIsImmediateDelete(t *testing.T) {
	pool := newSetupEnv(t)
	router := NewRouter(&Deps{Pool: pool, Version: testVersion})
	rec := postAdmin(t, router, createAdminBody(t, "ada@example.com", testPassword, nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create admin: status = %d, want %d", rec.Code, http.StatusCreated)
	}
	cookie, info := loginCookie(t, router, "ada@example.com", testPassword, "trash-settings-tests")

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings/trash", strings.NewReader(`{"retentionDays":0}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	put := httptest.NewRecorder()
	router.ServeHTTP(put, req)
	if put.Code != http.StatusOK {
		t.Fatalf("PUT trash 0 status = %d, want %d (body %q)", put.Code, http.StatusOK, put.Body.String())
	}

	// A trashed asset: a real file on disk, a row soft-deleted an
	// hour ago -- inside any positive retention window, outside 0.
	dataDir := t.TempDir()
	rel := "originals/trash-settings/IMG_0001.jpg"
	abs := filepath.Join(dataDir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("prepare the fixture directory: %v", err)
	}
	if err := os.WriteFile(abs, []byte("trash settings expiry fixture"), 0o644); err != nil {
		t.Fatalf("write the fixture file: %v", err)
	}
	owner := uuid.UUID(info.UserId)
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO assets (id, owner_id, storage_kind, file_path, checksum, original_filename,
			file_size, mime_type, media_kind, width, height, taken_at, taken_at_source, deleted_at)
		VALUES ($1, $2, 'blob', $3, 'trash-settings-checksum', 'IMG_0001.jpg', 32, 'image/jpeg',
			'image', 0, 0, now() - interval '2 hours', 'import', now() - interval '1 hour')`,
		pgUUID(owner), owner, rel); err != nil {
		t.Fatalf("insert the trashed fixture asset: %v", err)
	}

	counts, err := trash.RunExpiry(context.Background(), pool, dataDir, nil)
	if err != nil {
		t.Fatalf("run the expiry after the 0 retention: %v", err)
	}
	if counts.Expired != 1 {
		t.Errorf("expiry counts.Expired = %d, want 1 (retention 0 purges everything trashed)", counts.Expired)
	}
	if counts.FilesRemoved != 1 {
		t.Errorf("expiry counts.FilesRemoved = %d, want 1", counts.FilesRemoved)
	}
	if _, err := os.Stat(abs); !os.IsNotExist(err) {
		t.Errorf("the trashed fixture file still exists, want it purged with the row")
	}
}

// TestTranscodingSettingsGetAnswersDefault pins the read of a fresh
// instance: the documented default profile.
func TestTranscodingSettingsGetAnswersDefault(t *testing.T) {
	env := newStorageSettingsEnv(t)

	rec := env.getSetting(t, "transcoding", env.cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET transcoding status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var settings apigen.TranscodingSettingsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &settings); err != nil {
		t.Fatalf("decode transcoding settings %q: %v", rec.Body.String(), err)
	}
	if settings.Preset != apigen.TranscodePresetAuto {
		t.Errorf("preset = %q, want the default auto", settings.Preset)
	}
	if settings.Concurrency != 2 {
		t.Errorf("concurrency = %d, want the default 2", settings.Concurrency)
	}
	if !settings.ThumbnailFrame {
		t.Error("thumbnailFrame = false, want the default true")
	}
	if !settings.KeepOriginal {
		t.Error("keepOriginal = false, want the default true")
	}
}

// TestTranscodingSettingsPutSavesProfile pins the save: a full
// profile persists (answering the saved value) and the read adopts
// it; a second save replaces the row (the setting is one value).
func TestTranscodingSettingsPutSavesProfile(t *testing.T) {
	env := newStorageSettingsEnv(t)

	body := `{"preset":"720p","concurrency":1,"thumbnailFrame":false,"keepOriginal":true}`
	rec := env.putSetting(t, "transcoding", body, env.cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT transcoding status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var settings apigen.TranscodingSettingsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &settings); err != nil {
		t.Fatalf("decode transcoding settings %q: %v", rec.Body.String(), err)
	}
	if settings.Preset != apigen.TranscodePresetN720p || settings.Concurrency != 1 || settings.ThumbnailFrame || !settings.KeepOriginal {
		t.Errorf("answered settings = %+v, want the saved profile", settings)
	}

	rec = env.getSetting(t, "transcoding", env.cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET after the save status = %d, want %d", rec.Code, http.StatusOK)
	}
	settings = apigen.TranscodingSettingsResponse{}
	if err := json.Unmarshal(rec.Body.Bytes(), &settings); err != nil {
		t.Fatalf("decode transcoding settings %q: %v", rec.Body.String(), err)
	}
	if settings.Preset != apigen.TranscodePresetN720p || settings.Concurrency != 1 || settings.ThumbnailFrame || !settings.KeepOriginal {
		t.Errorf("reread settings = %+v, want the saved profile persisted", settings)
	}

	// A second save replaces the row.
	second := env.putSetting(t, "transcoding", `{"preset":"off","concurrency":4,"thumbnailFrame":true,"keepOriginal":false}`, env.cookie)
	if second.Code != http.StatusOK {
		t.Fatalf("second PUT transcoding status = %d, want %d", second.Code, http.StatusOK)
	}
	settings = apigen.TranscodingSettingsResponse{}
	if err := json.Unmarshal(second.Body.Bytes(), &settings); err != nil {
		t.Fatalf("decode transcoding settings %q: %v", second.Body.String(), err)
	}
	if settings.Preset != apigen.TranscodePresetOff || settings.Concurrency != 4 || !settings.ThumbnailFrame || settings.KeepOriginal {
		t.Errorf("replaced settings = %+v, want the second profile", settings)
	}
}

// TestTranscodingSettingsPutRejectsOutOfEnum pins the validation: a
// preset or concurrency outside its enum answers 400 naming the
// field, and nothing is written.
func TestTranscodingSettingsPutRejectsOutOfEnum(t *testing.T) {
	env := newStorageSettingsEnv(t)

	cases := []struct {
		name  string
		body  string
		field string
	}{
		{"unknown preset", `{"preset":"2160p","concurrency":2,"thumbnailFrame":true,"keepOriginal":true}`, "preset"},
		{"zero concurrency", `{"preset":"auto","concurrency":0,"thumbnailFrame":true,"keepOriginal":true}`, "concurrency"},
		{"odd concurrency", `{"preset":"auto","concurrency":3,"thumbnailFrame":true,"keepOriginal":true}`, "concurrency"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rec := env.putSetting(t, "transcoding", tt.body, env.cookie)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("PUT transcoding status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
			apiErr, fields := decodeError(t, rec)
			if apiErr.Code != apigen.VALIDATIONFAILED {
				t.Errorf("code = %q, want VALIDATION_FAILED", apiErr.Code)
			}
			if len(fields) == 0 || fields[0] != tt.field {
				t.Errorf("details.fields = %v, want [%s]", fields, tt.field)
			}
		})
	}

	// Nothing was written: the row stays absent (the default reads
	// back).
	var count int
	if err := env.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM instance_settings WHERE key = 'transcoding_settings'`).Scan(&count); err != nil {
		t.Fatalf("count the transcoding settings rows: %v", err)
	}
	if count != 0 {
		t.Errorf("transcoding_settings rows = %d after refused saves, want 0", count)
	}
}

// TestTranscodingSettingsCorruptRowAnswersDefault pins the read's
// corruption discipline: garbage rows (a non-object, an unknown
// preset, a wrong-typed field) answer the documented default, never
// an error -- the transcode job depends on the read never failing.
func TestTranscodingSettingsCorruptRowAnswersDefault(t *testing.T) {
	env := newStorageSettingsEnv(t)

	for _, value := range []string{`42`, `"auto"`, `{"preset":"2160p"}`, `{"preset":"auto","concurrency":3}`, `{"preset":"auto","thumbnailFrame":"yes"}`} {
		if _, err := env.pool.Exec(context.Background(),
			`INSERT INTO instance_settings (key, value) VALUES ('transcoding_settings', $1::jsonb)
			ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, value); err != nil {
			t.Fatalf("seed corrupt setting %q: %v", value, err)
		}
		rec := env.getSetting(t, "transcoding", env.cookie)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET over the corrupt row %q status = %d, want %d", value, rec.Code, http.StatusOK)
		}
		var settings apigen.TranscodingSettingsResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &settings); err != nil {
			t.Fatalf("decode transcoding settings %q: %v", rec.Body.String(), err)
		}
		if settings.Preset != apigen.TranscodePresetAuto || settings.Concurrency != 2 ||
			!settings.ThumbnailFrame || !settings.KeepOriginal {
			t.Errorf("settings over the corrupt row %q = %+v, want the documented default", value, settings)
		}
	}
}

// TestNewSettingsFacesGuardPinsTheAdminRole pins the permission
// model of the two new faces: the plain signed-in user answers 403,
// the anonymous caller 401, on every route.
func TestNewSettingsFacesGuardPinsTheAdminRole(t *testing.T) {
	env := newStorageSettingsEnv(t)
	_, plainCookie := seedPlainUser(t, env.pool, "grace@example.com")

	for _, tt := range []struct {
		name   string
		method string
		path   string
		cookie *http.Cookie
		want   int
	}{
		{"plain user GET trash", http.MethodGet, "trash", plainCookie, http.StatusForbidden},
		{"plain user PUT trash", http.MethodPut, "trash", plainCookie, http.StatusForbidden},
		{"anonymous GET trash", http.MethodGet, "trash", nil, http.StatusUnauthorized},
		{"anonymous PUT trash", http.MethodPut, "trash", nil, http.StatusUnauthorized},
		{"plain user GET transcoding", http.MethodGet, "transcoding", plainCookie, http.StatusForbidden},
		{"plain user PUT transcoding", http.MethodPut, "transcoding", plainCookie, http.StatusForbidden},
		{"anonymous GET transcoding", http.MethodGet, "transcoding", nil, http.StatusUnauthorized},
		{"anonymous PUT transcoding", http.MethodPut, "transcoding", nil, http.StatusUnauthorized},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var rec *httptest.ResponseRecorder
			if tt.method == http.MethodGet {
				rec = env.getSetting(t, tt.path, tt.cookie)
			} else {
				rec = env.putSetting(t, tt.path, `{"retentionDays":30}`, tt.cookie)
			}
			if rec.Code != tt.want {
				t.Fatalf("%s status = %d, want %d (body %q)", tt.name, rec.Code, tt.want, rec.Body.String())
			}
		})
	}
}

// The image card's settings face and the run budget's face: the same
// GET/PUT shape over the instance_settings rows the derivative
// settings own (the image card's five knobs; the task panel's three
// dials). Their tests ride this file's env -- the settings faces are
// one surface family.

// TestImageSettingsGetAnswersDefault pins the read of a fresh
// instance: the shipped posture -- the 250 tile, the 1080 preview,
// AVIF, no full-size copy, RAW sources deriving from their embedded
// previews.
func TestImageSettingsGetAnswersDefault(t *testing.T) {
	env := newStorageSettingsEnv(t)

	rec := env.getSetting(t, "image", env.cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET image settings status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var settings apigen.ImageSettingsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &settings); err != nil {
		t.Fatalf("decode image settings %q: %v", rec.Body.String(), err)
	}
	if settings.TileSize != 250 || settings.PreviewSize != 1080 {
		t.Errorf("sizes = %d/%d, want the shipped 250/1080", settings.TileSize, settings.PreviewSize)
	}
	if settings.Format != apigen.Avif {
		t.Errorf("format = %q, want the shipped avif", settings.Format)
	}
	if settings.FullSize {
		t.Error("fullSize = true, want the shipped off")
	}
	if !settings.RawEmbedded {
		t.Error("rawEmbedded = false, want the shipped on")
	}
}

// TestImageSettingsPutSavesDocument pins the save: a full document
// persists (answering the saved value) and the read adopts it; a
// second save replaces the row (the setting is one value).
func TestImageSettingsPutSavesDocument(t *testing.T) {
	env := newStorageSettingsEnv(t)

	body := `{"tileSize":320,"previewSize":1440,"format":"webp","fullSize":false,"rawEmbedded":true}`
	rec := env.putSetting(t, "image", body, env.cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT image settings status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var settings apigen.ImageSettingsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &settings); err != nil {
		t.Fatalf("decode image settings %q: %v", rec.Body.String(), err)
	}
	if settings.TileSize != 320 || settings.PreviewSize != 1440 || settings.Format != apigen.Webp || settings.FullSize || !settings.RawEmbedded {
		t.Errorf("answered settings = %+v, want the saved document", settings)
	}

	rec = env.getSetting(t, "image", env.cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET after the save status = %d, want %d", rec.Code, http.StatusOK)
	}
	settings = apigen.ImageSettingsResponse{}
	if err := json.Unmarshal(rec.Body.Bytes(), &settings); err != nil {
		t.Fatalf("decode reread image settings: %v", err)
	}
	if settings.TileSize != 320 || settings.Format != apigen.Webp {
		t.Errorf("reread settings = %+v, want the saved document persisted", settings)
	}
}

// TestImageSettingsPutRejectsInvalid pins the validation: a missing
// required field or a value outside its enum answers 400 naming the
// field, and nothing is written.
func TestImageSettingsPutRejectsInvalid(t *testing.T) {
	env := newStorageSettingsEnv(t)
	for _, tt := range []struct {
		name   string
		body   string
		fields []string
	}{
		{
			name:   "missing tileSize",
			body:   `{"previewSize":1080,"format":"avif","fullSize":false,"rawEmbedded":true}`,
			fields: []string{"tileSize"},
		},
		{
			name:   "tileSize out of enum",
			body:   `{"tileSize":500,"previewSize":1080,"format":"avif","fullSize":false,"rawEmbedded":true}`,
			fields: []string{"tileSize"},
		},
		{
			name:   "previewSize out of enum",
			body:   `{"tileSize":250,"previewSize":9999,"format":"avif","fullSize":false,"rawEmbedded":true}`,
			fields: []string{"previewSize"},
		},
		{
			name:   "format out of enum",
			body:   `{"tileSize":250,"previewSize":1080,"format":"jpegxl","fullSize":false,"rawEmbedded":true}`,
			fields: []string{"format"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rec := env.putSetting(t, "image", tt.body, env.cookie)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body %q)", rec.Code, rec.Body.String())
			}
			var envelope struct {
				Code    string   `json:"code"`
				Details struct{} `json:"details"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
				t.Fatalf("decode the error envelope: %v", err)
			}
			if envelope.Code != "VALIDATION_FAILED" {
				t.Errorf("code = %q, want VALIDATION_FAILED", envelope.Code)
			}
		})
	}
	// Nothing was written: the read still answers the defaults.
	rec := env.getSetting(t, "image", env.cookie)
	var settings apigen.ImageSettingsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &settings); err != nil {
		t.Fatalf("decode the untouched read: %v", err)
	}
	if settings.TileSize != 250 || settings.Format != apigen.Avif {
		t.Errorf("settings after the rejected saves = %+v, want the untouched defaults", settings)
	}
}

// TestRunBudgetSettingsGetAnswersDefault pins the read of a fresh
// instance: the standard dial, the window off (the shipped times ride
// along), nothing paused.
func TestRunBudgetSettingsGetAnswersDefault(t *testing.T) {
	env := newStorageSettingsEnv(t)

	rec := env.getSetting(t, "run-budget", env.cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET run budget status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var budget apigen.RunBudgetSettingsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &budget); err != nil {
		t.Fatalf("decode run budget %q: %v", rec.Body.String(), err)
	}
	if budget.ConcurrencyPreset != apigen.ConcurrencyPresetStandard {
		t.Errorf("preset = %q, want the standard step", budget.ConcurrencyPreset)
	}
	if budget.NightlyEnabled {
		t.Error("nightlyEnabled = true, want the shipped off")
	}
	if budget.NightlyStart != "02:00" || budget.NightlyEnd != "06:00" {
		t.Errorf("window = %s-%s, want the shipped 02:00-06:00", budget.NightlyStart, budget.NightlyEnd)
	}
	if budget.PausedJobs.Thumbnail || budget.PausedJobs.Metadata || budget.PausedJobs.Transcode || budget.PausedJobs.Ml || budget.PausedJobs.Places {
		t.Errorf("paused jobs = %+v, want nothing paused", budget.PausedJobs)
	}
}

// TestRunBudgetSettingsPutSavesDocument pins the save: a full
// document persists and the read adopts it.
func TestRunBudgetSettingsPutSavesDocument(t *testing.T) {
	env := newStorageSettingsEnv(t)

	body := `{"concurrencyPreset":"light","nightlyEnabled":true,"nightlyStart":"01:30","nightlyEnd":"05:30","pausedJobs":{"thumbnail":true,"metadata":false,"transcode":false,"ml":false,"places":false}}`
	rec := env.putSetting(t, "run-budget", body, env.cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT run budget status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var budget apigen.RunBudgetSettingsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &budget); err != nil {
		t.Fatalf("decode run budget %q: %v", rec.Body.String(), err)
	}
	if budget.ConcurrencyPreset != apigen.ConcurrencyPresetLight || !budget.NightlyEnabled || budget.NightlyStart != "01:30" || !budget.PausedJobs.Thumbnail {
		t.Errorf("answered budget = %+v, want the saved document", budget)
	}

	rec = env.getSetting(t, "run-budget", env.cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET after the save status = %d, want %d", rec.Code, http.StatusOK)
	}
	budget = apigen.RunBudgetSettingsResponse{}
	if err := json.Unmarshal(rec.Body.Bytes(), &budget); err != nil {
		t.Fatalf("decode the reread budget: %v", err)
	}
	if budget.ConcurrencyPreset != apigen.ConcurrencyPresetLight || !budget.NightlyEnabled {
		t.Errorf("reread budget = %+v, want the saved document persisted", budget)
	}
}

// TestRunBudgetSettingsPutRejectsInvalid pins the validation: a
// missing field, a preset outside its enum, a malformed time, an
// equal-time window and a missing pause switch each answer 400, and
// nothing is written.
func TestRunBudgetSettingsPutRejectsInvalid(t *testing.T) {
	env := newStorageSettingsEnv(t)
	for _, tt := range []struct {
		name string
		body string
	}{
		{
			name: "missing nightlyStart",
			body: `{"concurrencyPreset":"light","nightlyEnabled":false,"nightlyEnd":"06:00","pausedJobs":{"thumbnail":false,"metadata":false,"transcode":false,"ml":false,"places":false}}`,
		},
		{
			name: "preset out of enum",
			body: `{"concurrencyPreset":"turbo","nightlyEnabled":false,"nightlyStart":"02:00","nightlyEnd":"06:00","pausedJobs":{"thumbnail":false,"metadata":false,"transcode":false,"ml":false,"places":false}}`,
		},
		{
			name: "malformed start time",
			body: `{"concurrencyPreset":"light","nightlyEnabled":true,"nightlyStart":"2:0","nightlyEnd":"06:00","pausedJobs":{"thumbnail":false,"metadata":false,"transcode":false,"ml":false,"places":false}}`,
		},
		{
			name: "equal times",
			body: `{"concurrencyPreset":"light","nightlyEnabled":true,"nightlyStart":"06:00","nightlyEnd":"06:00","pausedJobs":{"thumbnail":false,"metadata":false,"transcode":false,"ml":false,"places":false}}`,
		},
		{
			name: "missing pause switch",
			body: `{"concurrencyPreset":"light","nightlyEnabled":false,"nightlyStart":"02:00","nightlyEnd":"06:00","pausedJobs":{"thumbnail":false,"metadata":false,"transcode":false,"places":false}}`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rec := env.putSetting(t, "run-budget", tt.body, env.cookie)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body %q)", rec.Code, rec.Body.String())
			}
			var envelope struct {
				Code string `json:"code"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
				t.Fatalf("decode the error envelope: %v", err)
			}
			if envelope.Code != "VALIDATION_FAILED" {
				t.Errorf("code = %q, want VALIDATION_FAILED", envelope.Code)
			}
		})
	}
	rec := env.getSetting(t, "run-budget", env.cookie)
	var budget apigen.RunBudgetSettingsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &budget); err != nil {
		t.Fatalf("decode the untouched read: %v", err)
	}
	if budget.ConcurrencyPreset != apigen.ConcurrencyPresetStandard {
		t.Errorf("budget after the rejected saves = %+v, want the untouched defaults", budget)
	}
}

// The derivative actions' endpoint tests: the summary read, the
// rebuild start (validation, the in-flight join) and the stop (the
// 409 with nothing in flight). The walk's own behavior is the jobs
// package's; these pin the endpoints' contract shapes.

// fakeRebuildEnqueuer records the enqueue's scope: the handler-level
// seam the tests inject instead of a real queue.
type fakeRebuildEnqueuer struct {
	scope string
	calls int
}

func (f *fakeRebuildEnqueuer) EnqueueDerivativesRebuild(_ context.Context, scope string) (uuid.UUID, error) {
	f.calls++
	f.scope = scope
	id, _ := uuid.NewV7()
	return id, nil
}

// newDerivativesEnv is the settings env with a rebuild seam wired.
func newDerivativesEnv(t *testing.T) (*storageSettingsEnv, *fakeRebuildEnqueuer) {
	t.Helper()
	pool := newSetupEnv(t)
	seam := &fakeRebuildEnqueuer{}
	router := NewRouter(&Deps{Pool: pool, Version: testVersion, DerivativesRebuilds: seam})
	rec := postAdmin(t, router, createAdminBody(t, "ada@example.com", testPassword, nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create admin: status = %d, want %d", rec.Code, http.StatusCreated)
	}
	cookie, _ := loginCookie(t, router, "ada@example.com", testPassword, "derivatives-tests")
	return &storageSettingsEnv{router: router, pool: pool, cookie: cookie}, seam
}

// TestDerivativesSummaryAnswersTheDeclaredEstimate pins the summary
// read: an empty library counts zero images and floors the estimate
// at one minute.
func TestDerivativesSummaryAnswersTheDeclaredEstimate(t *testing.T) {
	env, _ := newDerivativesEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/derivatives/summary", nil)
	req.AddCookie(env.cookie)
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET summary status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var summary apigen.DerivativesSummaryResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &summary); err != nil {
		t.Fatalf("decode the summary: %v", err)
	}
	if summary.ImageCount != 0 {
		t.Errorf("imageCount = %d, want 0 on a fresh instance", summary.ImageCount)
	}
	if summary.EstimateMinutes != 1 {
		t.Errorf("estimateMinutes = %d, want the floor of 1", summary.EstimateMinutes)
	}
}

// TestRebuildDerivativesValidationAndStart pins the rebuild start:
// a scope outside the enum answers 400; a valid scope with no queue
// wired answers 500; a valid scope with a wired seam answers 202 and
// hands the scope to the queue.
func TestRebuildDerivativesValidationAndStart(t *testing.T) {
	env, seam := newDerivativesEnv(t)

	post := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/derivatives/rebuild", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(env.cookie)
		rec := httptest.NewRecorder()
		env.router.ServeHTTP(rec, req)
		return rec
	}

	if rec := post(`{"scope":"everything"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("invalid scope status = %d, want 400 (body %q)", rec.Code, rec.Body.String())
	}
	if seam.calls != 0 {
		t.Errorf("enqueue calls after the invalid scope = %d, want 0", seam.calls)
	}

	// No queue wired: the plain settings env's router has no seam.
	plain := newStorageSettingsEnv(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/derivatives/rebuild", strings.NewReader(`{"scope":"all"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(plain.cookie)
	rec := httptest.NewRecorder()
	plain.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("no-seam status = %d, want 500 (body %q)", rec.Code, rec.Body.String())
	}

	if rec := post(`{"scope":"missing"}`); rec.Code != http.StatusAccepted {
		t.Fatalf("valid scope status = %d, want 202 (body %q)", rec.Code, rec.Body.String())
	}
	if seam.calls != 1 || seam.scope != "missing" {
		t.Errorf("seam = %d calls scope %q, want one call with the missing scope", seam.calls, seam.scope)
	}
}

// TestRebuildDerivativesJoinsInFlight pins the idempotence: a rebuild
// run still in flight answers 200 with its identifier and enqueues
// nothing.
func TestRebuildDerivativesJoinsInFlight(t *testing.T) {
	env, seam := newDerivativesEnv(t)

	inFlight, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("new run id: %v", err)
	}
	if _, err := env.pool.Exec(context.Background(),
		`INSERT INTO job_runs (id, kind, total, state) VALUES ($1, 'rebuild_derivatives', 12, 'running')`, pgUUID(inFlight)); err != nil {
		t.Fatalf("seed the in-flight run: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/derivatives/rebuild", strings.NewReader(`{"scope":"all"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(env.cookie)
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("join status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	var response apigen.RebuildDerivativesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode the join response: %v", err)
	}
	if response.JobRunId != inFlight {
		t.Errorf("joined run = %s, want the in-flight run %s", response.JobRunId, inFlight)
	}
	if seam.calls != 0 {
		t.Errorf("enqueue calls after the join = %d, want 0", seam.calls)
	}
}

// TestStopDerivativesRebuild pins the stop: a stop with nothing in
// flight answers 409 naming the code; a stop over an in-flight run
// answers 200 and sets the row's flag (the driver's breakpoint
// check reads it).
func TestStopDerivativesRebuild(t *testing.T) {
	env, _ := newDerivativesEnv(t)

	post := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/derivatives/rebuild/stop", nil)
		req.AddCookie(env.cookie)
		rec := httptest.NewRecorder()
		env.router.ServeHTTP(rec, req)
		return rec
	}

	first := post()
	if first.Code != http.StatusConflict {
		t.Fatalf("stop with nothing in flight status = %d, want 409 (body %q)", first.Code, first.Body.String())
	}
	var envelope struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &envelope); err != nil || envelope.Code != "REBUILD_NOT_RUNNING" {
		t.Errorf("stop's code = %q err %v, want REBUILD_NOT_RUNNING", envelope.Code, err)
	}

	inFlight, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("new run id: %v", err)
	}
	if _, err := env.pool.Exec(context.Background(),
		`INSERT INTO job_runs (id, kind, total, state) VALUES ($1, 'rebuild_derivatives', 12, 'running')`, pgUUID(inFlight)); err != nil {
		t.Fatalf("seed the in-flight run: %v", err)
	}
	rec := post()
	if rec.Code != http.StatusOK {
		t.Fatalf("stop over an in-flight run status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	var flagged bool
	if err := env.pool.QueryRow(context.Background(),
		"SELECT stop_requested FROM job_runs WHERE id = $1", pgUUID(inFlight)).Scan(&flagged); err != nil {
		t.Fatalf("read the stop flag: %v", err)
	}
	if !flagged {
		t.Error("stop_requested = false, want the stop recorded on the row")
	}
}
