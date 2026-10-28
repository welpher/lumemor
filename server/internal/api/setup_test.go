package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	oapitypes "github.com/oapi-codegen/runtime/types"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/auth"
	"github.com/welpher/lumemor/server/internal/db"
	"github.com/welpher/lumemor/server/internal/db/dbtest"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// testVersion is the build version injected into the test Deps and
// asserted in GET /server/info responses.
const testVersion = "9.9.9-test"

// newSetupEnv provisions a migrated throwaway database for a first-run
// wizard test.
func newSetupEnv(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := dbtest.New(t)
	if err := db.Migrate(t.Context(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	return pool
}

// createAdminBody renders a valid wizard payload; a nil name omits the
// field entirely.
func createAdminBody(t *testing.T, email, password string, name *string) string {
	t.Helper()
	body, err := json.Marshal(apigen.CreateAdminRequest{
		Email:    oapitypes.Email(email),
		Password: password,
		Name:     name,
	})
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}
	return string(body)
}

// postAdmin fires a CreateAdmin request at router and returns the
// recorder.
func postAdmin(t *testing.T, router http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/server/setup/admin", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// decodeError unmarshals the shared error envelope and extracts the
// details.fields list (empty when the envelope carries no fields).
func decodeError(t *testing.T, rec *httptest.ResponseRecorder) (apigen.Error, []string) {
	t.Helper()
	var apiErr apigen.Error
	if err := json.Unmarshal(rec.Body.Bytes(), &apiErr); err != nil {
		t.Fatalf("decode error body %q: %v", rec.Body.String(), err)
	}
	if apiErr.Details == nil {
		return apiErr, nil
	}
	raw, ok := (*apiErr.Details)["fields"]
	if !ok {
		return apiErr, nil
	}
	list, ok := raw.([]interface{})
	if !ok {
		t.Fatalf("details.fields = %#v, want an array of strings", raw)
	}
	fields := make([]string, len(list))
	for i, value := range list {
		field, ok := value.(string)
		if !ok {
			t.Fatalf("details.fields[%d] = %#v, want a string", i, value)
		}
		fields[i] = field
	}
	return apiErr, fields
}

// TestCreateAdmin walks the first-run wizard against a throwaway
// database: rejected payloads before setup, capability discovery, the
// successful creation, and the refusal of a second admin from a
// restarted (freshly wired) server.
func TestCreateAdmin(t *testing.T) {
	pool := newSetupEnv(t)
	router := NewRouter(&Deps{Pool: pool, Version: testVersion})
	queries := dbgen.New(pool)

	t.Run("RejectsInvalidPayloadsBeforeSetup", func(t *testing.T) {
		cases := []struct {
			name   string
			body   string
			fields []string
		}{
			{
				name:   "PasswordTooShort",
				body:   `{"email":"ada@example.com","password":"seven77"}`,
				fields: []string{"password"},
			},
			{
				name:   "PasswordTooLong",
				body:   `{"email":"ada@example.com","password":"` + strings.Repeat("x", 129) + `"}`,
				fields: []string{"password"},
			},
			{
				name:   "EmailMissing",
				body:   `{"password":"correct horse battery staple"}`,
				fields: []string{"email"},
			},
			{
				name:   "EmailInvalid",
				body:   `{"email":"not-an-email","password":"correct horse battery staple"}`,
				fields: []string{"email"},
			},
			{
				name: "NameTooLong",
				body: `{"email":"ada@example.com","password":"correct horse battery staple","name":"` +
					strings.Repeat("A", 101) + `"}`,
				fields: []string{"name"},
			},
			{
				name:   "BodyNotJSON",
				body:   `not json at all`,
				fields: nil,
			},
		}
		for _, testCase := range cases {
			t.Run(testCase.name, func(t *testing.T) {
				rec := postAdmin(t, router, testCase.body)

				if rec.Code != http.StatusBadRequest {
					t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
				}
				apiErr, fields := decodeError(t, rec)
				if apiErr.Code != apigen.VALIDATIONFAILED {
					t.Errorf("code = %q, want %q", apiErr.Code, apigen.VALIDATIONFAILED)
				}
				for _, want := range testCase.fields {
					if !slices.Contains(fields, want) {
						t.Errorf("details.fields = %v, want it to contain %q", fields, want)
					}
				}
			})
		}

		// None of the rejected payloads may have created an account.
		admins, err := queries.CountAdmins(t.Context())
		if err != nil {
			t.Fatalf("CountAdmins() error = %v, want nil", err)
		}
		if admins != 0 {
			t.Fatalf("CountAdmins() = %d after rejected payloads, want 0", admins)
		}
	})

	t.Run("ServerInfoBeforeSetup", func(t *testing.T) {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/server/info", nil))

		if rec.Code != http.StatusOK {
			t.Fatalf("GET /server/info status = %d, want %d", rec.Code, http.StatusOK)
		}
		var info apigen.ServerInfo
		if err := json.Unmarshal(rec.Body.Bytes(), &info); err != nil {
			t.Fatalf("decode server info %q: %v", rec.Body.String(), err)
		}
		if info.Initialized {
			t.Error("initialized = true, want false before the first admin exists")
		}
		if info.Version != testVersion {
			t.Errorf("version = %q, want %q", info.Version, testVersion)
		}
		if info.Features == nil {
			t.Error("features = null, want an object (empty in M1)")
		}
	})

	t.Run("CreatesFirstAdmin", func(t *testing.T) {
		name := "  Ada Lovelace  "
		rec := postAdmin(t, router, createAdminBody(t, "A@X.com", "correct horse battery staple", &name))

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
		}
		var user apigen.User
		if err := json.Unmarshal(rec.Body.Bytes(), &user); err != nil {
			t.Fatalf("decode user %q: %v", rec.Body.String(), err)
		}
		if user.Role != apigen.UserRoleAdmin {
			t.Errorf("role = %q, want %q", user.Role, apigen.UserRoleAdmin)
		}
		if string(user.Email) != "a@x.com" {
			t.Errorf("email = %q, want the normalized %q", user.Email, "a@x.com")
		}
		if user.Name != "Ada Lovelace" {
			t.Errorf("name = %q, want the trimmed %q", user.Name, "Ada Lovelace")
		}
		if user.Id.Version() != 7 {
			t.Errorf("id = %s, want a UUIDv7", user.Id)
		}
		if user.CreatedAt.IsZero() {
			t.Error("createdAt = zero, want the stored creation time")
		}

		// The write path must store the normalized address: lookups by
		// the lower-case form hit, and the column keeps the lower-case
		// value.
		stored, err := queries.GetUserByEmail(t.Context(), "a@x.com")
		if err != nil {
			t.Fatalf("GetUserByEmail(%q) error = %v, want nil", "a@x.com", err)
		}
		if stored.Email != "a@x.com" {
			t.Errorf("stored email = %q, want %q", stored.Email, "a@x.com")
		}
		if stored.Name != "Ada Lovelace" {
			t.Errorf("stored name = %q, want %q", stored.Name, "Ada Lovelace")
		}
		if stored.Role != dbgen.UserRoleAdmin {
			t.Errorf("stored role = %q, want %q", stored.Role, dbgen.UserRoleAdmin)
		}
		if !auth.Verify(stored.PasswordHash, "correct horse battery staple") {
			t.Error("auth.Verify(stored hash, password) = false, want true")
		}
	})

	t.Run("SecondAdminOnRestartedServer", func(t *testing.T) {
		// Simulate a process restart: a brand-new Deps and router, no
		// in-memory state, same database. The wizard must still be
		// refused because `initialized` derives from the users table.
		restarted := NewRouter(&Deps{Pool: pool, Version: testVersion})

		rec := postAdmin(t, restarted, createAdminBody(t, "grace@example.com", "another correct horse", nil))
		if rec.Code != http.StatusConflict {
			t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusConflict, rec.Body.String())
		}
		apiErr, _ := decodeError(t, rec)
		if apiErr.Code != apigen.SETUPALREADYCOMPLETED {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.SETUPALREADYCOMPLETED)
		}

		infoRec := httptest.NewRecorder()
		restarted.ServeHTTP(infoRec, httptest.NewRequest(http.MethodGet, "/api/v1/server/info", nil))
		if infoRec.Code != http.StatusOK {
			t.Fatalf("GET /server/info status = %d, want %d", infoRec.Code, http.StatusOK)
		}
		var info apigen.ServerInfo
		if err := json.Unmarshal(infoRec.Body.Bytes(), &info); err != nil {
			t.Fatalf("decode server info %q: %v", infoRec.Body.String(), err)
		}
		if !info.Initialized {
			t.Error("initialized = false after restart, want true (derived from the database)")
		}
	})
}

// TestCreateAdminNameDefaultsToEmail verifies the optional display name
// falls back to the email address when the payload omits it.
func TestCreateAdminNameDefaultsToEmail(t *testing.T) {
	pool := newSetupEnv(t)
	router := NewRouter(&Deps{Pool: pool, Version: testVersion})

	rec := postAdmin(t, router, `{"email":"ada@example.com","password":"correct horse battery staple"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var user apigen.User
	if err := json.Unmarshal(rec.Body.Bytes(), &user); err != nil {
		t.Fatalf("decode user %q: %v", rec.Body.String(), err)
	}
	if user.Name != "ada@example.com" {
		t.Errorf("name = %q, want it to default to the email address", user.Name)
	}

	stored, err := dbgen.New(pool).GetUserByEmail(t.Context(), "ada@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail() error = %v, want nil", err)
	}
	if stored.Name != "ada@example.com" {
		t.Errorf("stored name = %q, want %q", stored.Name, "ada@example.com")
	}
}
