// The admin storage settings tests (the storage-template feature):
// the admin-role guard of the surface (403 for a signed-in plain
// user, 401 without a session), the read's shape (the template, the
// sample render, the sixteen presets), the default's answer on a
// fresh instance, and the save's three-stage validation -- a strict
// compile failure, an empty sample render, and a structural refusal
// each reject with the specific reason and write nothing, while a
// valid template persists and answers the saved state.
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
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
