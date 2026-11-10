// The admin user management tests: the admin-role
// guard of the surface (403 for a signed-in plain user, 401 without
// a session), the listing's shape and ordering, the transactional
// account creation (user row and sync album together, the wizard's
// wiring reused) with its atomicity pin (a sync-album failure inside
// the transaction rolls the account row back), the duplicate-email
// 409, the payload validation, and the disabled toggle's pinned
// semantics -- next-request 401s of the account's sessions, sign-in
// refused, session rows surviving, re-enable reviving them -- plus
// the self-disable footgun guard and the PATCH's required-field
// refusal.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	oapitypes "github.com/oapi-codegen/runtime/types"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/auth"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// adminUserEnv provisions the user management test environment: a
// migrated throwaway database with the first administrator (created
// through the wizard, then signed in) as the caller.
type adminUserEnv struct {
	router  http.Handler
	pool    *pgxpool.Pool
	queries *dbgen.Queries
	cookie  *http.Cookie
	// adminID resolves the caller's account for the self-disable
	// guard's tests.
	adminID uuid.UUID
}

// newAdminUserEnv provisions the environment and signs the
// administrator in.
func newAdminUserEnv(t *testing.T) *adminUserEnv {
	t.Helper()
	pool := newSetupEnv(t)
	router := NewRouter(&Deps{Pool: pool, Version: testVersion})

	rec := postAdmin(t, router, createAdminBody(t, "ada@example.com", testPassword, nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create admin: status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	cookie, _ := loginCookie(t, router, "ada@example.com", testPassword, "admin-users-tests")

	admin, err := dbgen.New(pool).GetUserByEmail(context.Background(), "ada@example.com")
	if err != nil {
		t.Fatalf("resolve admin account: %v", err)
	}
	return &adminUserEnv{
		router:  router,
		pool:    pool,
		queries: dbgen.New(pool),
		cookie:  cookie,
		adminID: uuid.UUID(admin.ID.Bytes),
	}
}

// seedPlainUser inserts a plain (non-admin) account directly in the
// database together with a live mobile session, mirroring the asset
// tests' fixture style: the admin surface under test provisions
// accounts itself, so its own create endpoint never builds its
// fixtures.
func seedPlainUser(t *testing.T, pool *pgxpool.Pool, email string) (uuid.UUID, *http.Cookie) {
	t.Helper()
	hash, err := auth.Hash(testPassword)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	userID, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("generate user id: %v", err)
	}
	if _, err := pool.Exec(context.Background(),
		"INSERT INTO users (id, name, email, password_hash) VALUES ($1, $2, $3, $4)",
		pgUUID(userID), email, email, hash); err != nil {
		t.Fatalf("insert plain user: %v", err)
	}
	sessionID, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("generate session id: %v", err)
	}
	raw := "admin-users-session-" + email
	if _, err := pool.Exec(context.Background(),
		"INSERT INTO sessions (id, user_id, token_hash, kind, expires_at) VALUES ($1, $2, $3, 'mobile', $4)",
		pgUUID(sessionID), pgUUID(userID), auth.HashSessionToken(raw),
		time.Now().Add(24*time.Hour)); err != nil {
		t.Fatalf("insert plain user session: %v", err)
	}
	return userID, &http.Cookie{Name: sessionCookieName, Value: raw}
}

// adminCreateUserBody renders a create-user payload; nil name and
// nil role omit the fields entirely.
func adminCreateUserBody(t *testing.T, email, password string, name *string, role *apigen.UserRole) string {
	t.Helper()
	body, err := json.Marshal(apigen.AdminCreateUserRequest{
		Email:    oapitypes.Email(email),
		Password: password,
		Name:     name,
		Role:     role,
	})
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}
	return string(body)
}

// postUser fires the create-user request at the router.
func (e *adminUserEnv) postUser(t *testing.T, cookie *http.Cookie, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// patchUser fires the disabled-toggle request at the router.
func (e *adminUserEnv) patchUser(t *testing.T, cookie *http.Cookie, id uuid.UUID, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/"+id.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// listUsers fires the listing request at the router.
func (e *adminUserEnv) listUsers(t *testing.T, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// countSyncAlbums answers how many sync albums an account owns (the
// transactional-create assertion: exactly one).
func (e *adminUserEnv) countSyncAlbums(t *testing.T, ownerID uuid.UUID) int {
	t.Helper()
	var count int
	if err := e.pool.QueryRow(context.Background(),
		"SELECT count(*) FROM albums WHERE owner_id = $1 AND type = 'sync'",
		pgUUID(ownerID)).Scan(&count); err != nil {
		t.Fatalf("count sync albums: %v", err)
	}
	return count
}

// TestListUsers pins the listing: the admin-role guard, the safe
// columns, and the oldest-first order.
func TestListUsers(t *testing.T) {
	env := newAdminUserEnv(t)
	graceID, graceCookie := seedPlainUser(t, env.pool, "grace@example.com")

	t.Run("RejectsUnauthenticated", func(t *testing.T) {
		rec := env.listUsers(t, nil)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
		apiErr, _ := decodeError(t, rec)
		if apiErr.Code != apigen.AUTHINVALIDCREDENTIALS {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.AUTHINVALIDCREDENTIALS)
		}
	})

	t.Run("RejectsNonAdmin", func(t *testing.T) {
		rec := env.listUsers(t, graceCookie)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusForbidden, rec.Body.String())
		}
		apiErr, _ := decodeError(t, rec)
		if apiErr.Code != apigen.AUTHFORBIDDEN {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.AUTHFORBIDDEN)
		}
	})

	t.Run("ListsEveryAccountOldestFirst", func(t *testing.T) {
		rec := env.listUsers(t, env.cookie)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		var response apigen.AdminUserListResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode users body %q: %v", rec.Body.String(), err)
		}
		if len(response.Items) != 2 {
			t.Fatalf("items = %d, want 2 (the admin and grace)", len(response.Items))
		}
		// Oldest first: the wizard's admin leads the list.
		if string(response.Items[0].Email) != "ada@example.com" {
			t.Errorf("first item email = %q, want %q", response.Items[0].Email, "ada@example.com")
		}
		if string(response.Items[1].Email) != "grace@example.com" {
			t.Errorf("second item email = %q, want %q", response.Items[1].Email, "grace@example.com")
		}
		if response.Items[1].Id != graceID {
			t.Errorf("second item id = %s, want %s", response.Items[1].Id, graceID)
		}

		admin := response.Items[0]
		if admin.Id != env.adminID {
			t.Errorf("admin id = %s, want %s", admin.Id, env.adminID)
		}
		if admin.Role != apigen.UserRoleAdmin {
			t.Errorf("admin role = %q, want %q", admin.Role, apigen.UserRoleAdmin)
		}
		if admin.IsDisabled {
			t.Error("admin isDisabled = true, want false")
		}
		if admin.Name != "ada@example.com" {
			t.Errorf("admin name = %q, want the email default %q", admin.Name, "ada@example.com")
		}
		// The listing must never leak the password hash: the body has
		// no such key at all.
		if strings.Contains(rec.Body.String(), "password") {
			t.Errorf("body leaks a password field: %q", rec.Body.String())
		}
		// The creation time is the stored row's, at the database's
		// microsecond resolution.
		stored, err := env.queries.GetUserByEmail(t.Context(), "ada@example.com")
		if err != nil {
			t.Fatalf("load stored admin: %v", err)
		}
		if !stored.CreatedAt.Time.UTC().Truncate(time.Microsecond).Equal(admin.CreatedAt.Truncate(time.Microsecond)) {
			t.Errorf("createdAt = %v, want the stored %v", admin.CreatedAt, stored.CreatedAt.Time.UTC())
		}
	})
}

// TestCreateUser pins the account creation: the guard, the payload
// validation, the transactional sync album, the role handling, and
// the duplicate-email conflict.
func TestCreateUser(t *testing.T) {
	env := newAdminUserEnv(t)
	_, graceCookie := seedPlainUser(t, env.pool, "grace@example.com")

	t.Run("RejectsNonAdmin", func(t *testing.T) {
		rec := env.postUser(t, graceCookie, adminCreateUserBody(t, "kid@example.com", testPassword, nil, nil))
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusForbidden, rec.Body.String())
		}
		apiErr, _ := decodeError(t, rec)
		if apiErr.Code != apigen.AUTHFORBIDDEN {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.AUTHFORBIDDEN)
		}
	})

	t.Run("RejectsInvalidPayloads", func(t *testing.T) {
		longName := strings.Repeat("A", 101)
		bogusRole := apigen.UserRole("root")
		cases := []struct {
			name   string
			body   string
			fields []string
		}{
			{
				name:   "PasswordTooShort",
				body:   adminCreateUserBody(t, "kid@example.com", "seven77", nil, nil),
				fields: []string{"password"},
			},
			{
				name:   "PasswordTooLong",
				body:   adminCreateUserBody(t, "kid@example.com", strings.Repeat("x", 129), nil, nil),
				fields: []string{"password"},
			},
			{
				// The generated Email type refuses to marshal a
				// malformed address, so this payload is spelled by
				// hand -- exactly what a broken client would send.
				name:   "EmailInvalid",
				body:   `{"email":"not-an-email","password":"` + testPassword + `"}`,
				fields: []string{"email"},
			},
			{
				// The generated Email type refuses to marshal an empty
				// address, so this malformed payload is spelled by
				// hand -- exactly what a broken client would send.
				name:   "EmailEmpty",
				body:   `{"email":"","password":"` + testPassword + `"}`,
				fields: []string{"email"},
			},
			{
				name:   "NameTooLong",
				body:   adminCreateUserBody(t, "kid@example.com", testPassword, &longName, nil),
				fields: []string{"name"},
			},
			{
				name:   "RoleOutsideEnum",
				body:   adminCreateUserBody(t, "kid@example.com", testPassword, nil, &bogusRole),
				fields: []string{"role"},
			},
			{
				name:   "BodyNotJSON",
				body:   `not json at all`,
				fields: nil,
			},
		}
		for _, testCase := range cases {
			t.Run(testCase.name, func(t *testing.T) {
				rec := env.postUser(t, env.cookie, testCase.body)
				if rec.Code != http.StatusBadRequest {
					t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
				}
				apiErr, fields := decodeError(t, rec)
				if apiErr.Code != apigen.VALIDATIONFAILED {
					t.Errorf("code = %q, want %q", apiErr.Code, apigen.VALIDATIONFAILED)
				}
				for _, want := range testCase.fields {
					if !contains(fields, want) {
						t.Errorf("details.fields = %v, want it to contain %q", fields, want)
					}
				}
				// None of the rejected payloads may have created an
				// account or an album.
				users, err := env.queries.ListUsers(t.Context())
				if err != nil {
					t.Fatalf("ListUsers() error = %v", err)
				}
				if len(users) != 2 {
					t.Errorf("accounts = %d after a rejected payload, want 2", len(users))
				}
				if env.countSyncAlbums(t, env.adminID) != 1 {
					t.Errorf("admin sync albums = %d, want 1 (untouched)", env.countSyncAlbums(t, env.adminID))
				}
			})
		}
	})

	t.Run("CreatesUserWithSyncAlbum", func(t *testing.T) {
		name := "  Kid Hopper  "
		rec := env.postUser(t, env.cookie, adminCreateUserBody(t, "Kid@Example.com", testPassword, &name, nil))
		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
		}
		var user apigen.AdminUserResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &user); err != nil {
			t.Fatalf("decode user %q: %v", rec.Body.String(), err)
		}
		// The normalized identity, the trimmed name, the default
		// role, and the fresh-row disabled switch.
		if string(user.Email) != "kid@example.com" {
			t.Errorf("email = %q, want the normalized %q", user.Email, "kid@example.com")
		}
		if user.Name != "Kid Hopper" {
			t.Errorf("name = %q, want the trimmed %q", user.Name, "Kid Hopper")
		}
		if user.Role != apigen.UserRoleUser {
			t.Errorf("role = %q, want the default %q", user.Role, apigen.UserRoleUser)
		}
		if user.IsDisabled {
			t.Error("isDisabled = true, want false")
		}
		if user.Id.Version() != 7 {
			t.Errorf("id = %s, want a UUIDv7", user.Id)
		}
		if user.CreatedAt.IsZero() {
			t.Error("createdAt = zero, want the stored creation time")
		}
		if strings.Contains(rec.Body.String(), "password") {
			t.Errorf("body leaks a password field: %q", rec.Body.String())
		}

		// The transactional wiring: the account's sync album exists,
		// exactly one row.
		if count := env.countSyncAlbums(t, user.Id); count != 1 {
			t.Errorf("sync albums of the new account = %d, want 1", count)
		}

		// The account is real: the stored row verifies the password,
		// and the address is stored lower-cased so the login lookup
		// hits.
		stored, err := env.queries.GetUserByEmail(t.Context(), "kid@example.com")
		if err != nil {
			t.Fatalf("GetUserByEmail(kid@example.com) error = %v, want nil", err)
		}
		if !auth.Verify(stored.PasswordHash, testPassword) {
			t.Error("auth.Verify(stored hash, password) = false, want true")
		}
		loginRec := postLogin(t, env.router, "kid@example.com", testPassword, "admin-users-tests")
		if loginRec.Code != http.StatusOK {
			t.Errorf("login of the created account: status = %d, want %d (body %q)", loginRec.Code, http.StatusOK, loginRec.Body.String())
		}
	})

	t.Run("CreatesSecondAdminWithRole", func(t *testing.T) {
		adminRole := apigen.UserRoleAdmin
		rec := env.postUser(t, env.cookie, adminCreateUserBody(t, "root@example.com", testPassword, nil, &adminRole))
		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
		}
		var user apigen.AdminUserResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &user); err != nil {
			t.Fatalf("decode user %q: %v", rec.Body.String(), err)
		}
		if user.Role != apigen.UserRoleAdmin {
			t.Errorf("role = %q, want %q", user.Role, apigen.UserRoleAdmin)
		}
	})

	t.Run("NameDefaultsToEmail", func(t *testing.T) {
		rec := env.postUser(t, env.cookie, adminCreateUserBody(t, "noname@example.com", testPassword, nil, nil))
		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
		}
		var user apigen.AdminUserResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &user); err != nil {
			t.Fatalf("decode user %q: %v", rec.Body.String(), err)
		}
		if user.Name != "noname@example.com" {
			t.Errorf("name = %q, want the email default %q", user.Name, "noname@example.com")
		}
	})

	t.Run("DuplicateEmailConflicts", func(t *testing.T) {
		// The wizard admin's own address: the identity guard fires
		// for any duplicate, not just same-surface ones.
		rec := env.postUser(t, env.cookie, adminCreateUserBody(t, "ada@example.com", testPassword, nil, nil))
		if rec.Code != http.StatusConflict {
			t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusConflict, rec.Body.String())
		}
		apiErr, _ := decodeError(t, rec)
		if apiErr.Code != apigen.USEREMAILTAKEN {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.USEREMAILTAKEN)
		}

		// A second POST of an address this endpoint itself created.
		created := env.postUser(t, env.cookie, adminCreateUserBody(t, "twin@example.com", testPassword, nil, nil))
		if created.Code != http.StatusCreated {
			t.Fatalf("first create: status = %d, want %d", created.Code, http.StatusCreated)
		}
		duplicate := env.postUser(t, env.cookie, adminCreateUserBody(t, "twin@example.com", testPassword, nil, nil))
		if duplicate.Code != http.StatusConflict {
			t.Fatalf("second create: status = %d, want %d (body %q)", duplicate.Code, http.StatusConflict, duplicate.Body.String())
		}
		apiErr, _ = decodeError(t, duplicate)
		if apiErr.Code != apigen.USEREMAILTAKEN {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.USEREMAILTAKEN)
		}

		// The conflict created nothing: one twin row, one album.
		twin, err := env.queries.GetUserByEmail(t.Context(), "twin@example.com")
		if err != nil {
			t.Fatalf("load twin: %v", err)
		}
		if count := env.countSyncAlbums(t, uuid.UUID(twin.ID.Bytes)); count != 1 {
			t.Errorf("twin sync albums = %d, want 1", count)
		}
	})
}

// TestCreateUserSyncAlbumFailureRollsBack pins the creation's
// atomicity: the account row and its sync album land in ONE
// transaction, so a failure between the two writes must roll the
// account back -- a two-transaction regression (the user row
// committed before EnsureSyncAlbum) leaves the row behind and fails
// this test. The sync album's id is generated inside the handler (a
// fresh UUIDv7, not the user id), so a pre-inserted conflicting row
// cannot target the primary key; the injection is instead a
// test-scoped unique constraint on albums (title): in this fresh
// environment the wizard's own sync album is the only row, so the
// constraint installs cleanly and the new account's EnsureSyncAlbum
// insert (the same fixed title 'Sync') raises a unique violation
// inside the transaction. The statement's ON CONFLICT arbiter is the
// owner's partial index, so this constraint's violation propagates
// as a plain error.
func TestCreateUserSyncAlbumFailureRollsBack(t *testing.T) {
	env := newAdminUserEnv(t)

	if _, err := env.pool.Exec(context.Background(),
		"ALTER TABLE albums ADD CONSTRAINT pin_albums_title_uq UNIQUE (title)"); err != nil {
		t.Fatalf("inject albums title constraint: %v", err)
	}

	rec := env.postUser(t, env.cookie,
		adminCreateUserBody(t, "rollback@example.com", testPassword, nil, nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d (the sync album insert must fail; body %q)",
			rec.Code, http.StatusInternalServerError, rec.Body.String())
	}

	// The account row was rolled back: no row with the email...
	if _, err := env.queries.GetUserByEmail(t.Context(), "rollback@example.com"); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("GetUserByEmail(rollback@example.com) error = %v, want no row (the user row rolled back)", err)
	}
	// ...and the listing count is unchanged (the admin alone).
	users, err := env.queries.ListUsers(t.Context())
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if len(users) != 1 {
		t.Errorf("accounts = %d after the rolled-back create, want 1 (the admin alone)", len(users))
	}
}

// TestUpdateUser pins the disabled toggle: the guard, the footgun
// refusal, the unknown id, and the middleware's observable
// semantics of a disabled account.
func TestUpdateUser(t *testing.T) {
	env := newAdminUserEnv(t)
	graceID, graceCookie := seedPlainUser(t, env.pool, "grace@example.com")

	t.Run("RejectsUnauthenticated", func(t *testing.T) {
		rec := env.patchUser(t, nil, graceID, `{"isDisabled":true}`)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("RejectsNonAdmin", func(t *testing.T) {
		rec := env.patchUser(t, graceCookie, graceID, `{"isDisabled":true}`)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusForbidden, rec.Body.String())
		}
		apiErr, _ := decodeError(t, rec)
		if apiErr.Code != apigen.AUTHFORBIDDEN {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.AUTHFORBIDDEN)
		}
	})

	t.Run("RejectsSelfDisable", func(t *testing.T) {
		rec := env.patchUser(t, env.cookie, env.adminID, `{"isDisabled":true}`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		apiErr, fields := decodeError(t, rec)
		if apiErr.Code != apigen.VALIDATIONFAILED {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.VALIDATIONFAILED)
		}
		if !contains(fields, "id") {
			t.Errorf("details.fields = %v, want it to contain %q", fields, "id")
		}

		// The refusal wrote nothing: the account stays enabled.
		stored, err := env.queries.GetUserByEmail(t.Context(), "ada@example.com")
		if err != nil {
			t.Fatalf("load admin: %v", err)
		}
		if stored.IsDisabled {
			t.Error("admin isDisabled = true after the refused self-disable, want false")
		}

		// The guard aims at the footgun only: a self-enable is a
		// plain idempotent write, not a lockout.
		selfEnable := env.patchUser(t, env.cookie, env.adminID, `{"isDisabled":false}`)
		if selfEnable.Code != http.StatusOK {
			t.Fatalf("self-enable: status = %d, want %d (body %q)", selfEnable.Code, http.StatusOK, selfEnable.Body.String())
		}
	})

	t.Run("RejectsMalformedBody", func(t *testing.T) {
		rec := env.patchUser(t, env.cookie, graceID, `not json`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		apiErr, _ := decodeError(t, rec)
		if apiErr.Code != apigen.VALIDATIONFAILED {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.VALIDATIONFAILED)
		}
	})

	t.Run("RejectsMissingIsDisabled", func(t *testing.T) {
		// The contract marks isDisabled required: the generated
		// struct's plain bool decodes an absent field as a silent
		// false (an enable nobody asked for), so the handler refuses
		// the payload itself. A JSON null counts as absent -- the
		// pointer decode reads both shapes the same way.
		cases := []struct {
			name string
			body string
		}{
			{name: "FieldAbsent", body: `{}`},
			{name: "FieldNull", body: `{"isDisabled":null}`},
		}
		for _, testCase := range cases {
			t.Run(testCase.name, func(t *testing.T) {
				rec := env.patchUser(t, env.cookie, graceID, testCase.body)
				if rec.Code != http.StatusBadRequest {
					t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
				}
				apiErr, fields := decodeError(t, rec)
				if apiErr.Code != apigen.VALIDATIONFAILED {
					t.Errorf("code = %q, want %q", apiErr.Code, apigen.VALIDATIONFAILED)
				}
				if !contains(fields, "isDisabled") {
					t.Errorf("details.fields = %v, want it to contain %q", fields, "isDisabled")
				}
			})
		}

		// The refusal wrote nothing: Grace stays enabled.
		stored, err := env.queries.GetUserByEmail(t.Context(), "grace@example.com")
		if err != nil {
			t.Fatalf("load grace: %v", err)
		}
		if stored.IsDisabled {
			t.Error("grace isDisabled = true after the refused payloads, want false")
		}
	})

	t.Run("UnknownIdAnswersNotFound", func(t *testing.T) {
		unknown, err := uuid.NewV7()
		if err != nil {
			t.Fatalf("generate unknown id: %v", err)
		}
		rec := env.patchUser(t, env.cookie, unknown, `{"isDisabled":true}`)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusNotFound, rec.Body.String())
		}
		apiErr, _ := decodeError(t, rec)
		if apiErr.Code != apigen.USERNOTFOUND {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.USERNOTFOUND)
		}
	})

	t.Run("DisableStripsSessionsReEnableRevivesThem", func(t *testing.T) {
		// Grace's session works before the toggle.
		before := authedRequest(env.router, http.MethodGet, "/api/v1/auth/sessions", graceCookie)
		if before.Code != http.StatusOK {
			t.Fatalf("grace's session before: status = %d, want %d", before.Code, http.StatusOK)
		}

		// The disable write answers 200 with the account after the
		// write.
		rec := env.patchUser(t, env.cookie, graceID, `{"isDisabled":true}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("disable: status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		var user apigen.AdminUserResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &user); err != nil {
			t.Fatalf("decode user %q: %v", rec.Body.String(), err)
		}
		if !user.IsDisabled {
			t.Error("isDisabled = false after the disable, want true")
		}
		if user.Id != graceID {
			t.Errorf("id = %s, want %s", user.Id, graceID)
		}

		// The pinned semantics: the account's next request 401s
		// (the session row survives, its authority does not), and
		// sign-in is refused the same way.
		after := authedRequest(env.router, http.MethodGet, "/api/v1/auth/sessions", graceCookie)
		if after.Code != http.StatusUnauthorized {
			t.Fatalf("grace's session after disable: status = %d, want %d", after.Code, http.StatusUnauthorized)
		}
		apiErr, _ := decodeError(t, after)
		if apiErr.Code != apigen.AUTHINVALIDCREDENTIALS {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.AUTHINVALIDCREDENTIALS)
		}
		loginRec := postLogin(t, env.router, "grace@example.com", testPassword, "admin-users-tests")
		if loginRec.Code != http.StatusUnauthorized {
			t.Fatalf("grace's login while disabled: status = %d, want %d", loginRec.Code, http.StatusUnauthorized)
		}

		// The session row itself survived the disable.
		var sessionCount int
		if err := env.pool.QueryRow(context.Background(),
			"SELECT count(*) FROM sessions WHERE user_id = $1", pgUUID(graceID)).Scan(&sessionCount); err != nil {
			t.Fatalf("count grace's sessions: %v", err)
		}
		if sessionCount != 1 {
			t.Errorf("grace's session rows = %d, want 1 (disabling revokes authority, not existence)", sessionCount)
		}

		// Re-enabling revives the unexpired session without a new
		// sign-in.
		reenable := env.patchUser(t, env.cookie, graceID, `{"isDisabled":false}`)
		if reenable.Code != http.StatusOK {
			t.Fatalf("re-enable: status = %d, want %d (body %q)", reenable.Code, http.StatusOK, reenable.Body.String())
		}
		var revived apigen.AdminUserResponse
		if err := json.Unmarshal(reenable.Body.Bytes(), &revived); err != nil {
			t.Fatalf("decode user %q: %v", reenable.Body.String(), err)
		}
		if revived.IsDisabled {
			t.Error("isDisabled = true after the re-enable, want false")
		}
		revivedSession := authedRequest(env.router, http.MethodGet, "/api/v1/auth/sessions", graceCookie)
		if revivedSession.Code != http.StatusOK {
			t.Fatalf("grace's session after re-enable: status = %d, want %d", revivedSession.Code, http.StatusOK)
		}
	})

	t.Run("IdempotentRewriteAnswersUnchangedAccount", func(t *testing.T) {
		// Enable the already-enabled account: the plain 200, the
		// same row, no error.
		rec := env.patchUser(t, env.cookie, graceID, `{"isDisabled":false}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		var user apigen.AdminUserResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &user); err != nil {
			t.Fatalf("decode user %q: %v", rec.Body.String(), err)
		}
		if user.IsDisabled {
			t.Error("isDisabled = true, want false (the idempotent no-op)")
		}
	})
}
