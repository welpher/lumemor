// The me-scoped user settings tests (the settings feature): the
// profile read and patch (each section's fields, the 409 of a taken
// email, the 400s of invalid fields), the password change's
// revocation contract (THE PIN: every OTHER session dies, the
// current one stays -- pinned with a dual-session fixture -- and a
// wrong current password is a 400, never a 401, so the caller's
// session survives the mistake), and the device list with its
// isCurrent marker, its creation-time address and its revoke (the
// current session refused, the other device signed out).
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	oapitypes "github.com/oapi-codegen/runtime/types"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
)

// userSettingsEnv provisions the me-scoped settings test
// environment: a migrated throwaway database with the first
// administrator (created through the wizard, then signed in) as the
// caller.
type userSettingsEnv struct {
	router http.Handler
	pool   *pgxpool.Pool
	cookie *http.Cookie
	// sessionID is the caller's own session, from the login body
	// that minted the cookie (the device list's isCurrent marker is
	// asserted against it).
	sessionID uuid.UUID
}

// newUserSettingsEnv provisions the environment and signs the
// administrator in.
func newUserSettingsEnv(t *testing.T) *userSettingsEnv {
	t.Helper()
	pool := newSetupEnv(t)
	router := NewRouter(&Deps{Pool: pool, Version: testVersion})

	rec := postAdmin(t, router, createAdminBody(t, "ada@example.com", testPassword, nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create admin: status = %d, want %d (body %q)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	cookie, info := loginCookie(t, router, "ada@example.com", testPassword, "user-settings-tests")
	return &userSettingsEnv{router: router, pool: pool, cookie: cookie, sessionID: uuid.UUID(info.Id)}
}

// request fires one authed request with a JSON body at the router.
func (e *userSettingsEnv) request(t *testing.T, method, target, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// getSelf fires GET /users/me.
func (e *userSettingsEnv) getSelf(t *testing.T, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	return e.request(t, http.MethodGet, "/api/v1/users/me", "", cookie)
}

// patchSelf fires PATCH /users/me with one JSON body.
func (e *userSettingsEnv) patchSelf(t *testing.T, cookie *http.Cookie, body string) *httptest.ResponseRecorder {
	t.Helper()
	return e.request(t, http.MethodPatch, "/api/v1/users/me", body, cookie)
}

// changePassword fires PUT /users/me/password with one JSON body.
func (e *userSettingsEnv) changePassword(t *testing.T, cookie *http.Cookie, current, next string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(apigen.PasswordChangeRequest{Current: current, Next: next})
	if err != nil {
		t.Fatalf("marshal password change body: %v", err)
	}
	return e.request(t, http.MethodPut, "/api/v1/users/me/password", string(body), cookie)
}

// decodeSelf decodes a 200 self body.
func decodeSelf(t *testing.T, rec *httptest.ResponseRecorder) apigen.UserSelfResponse {
	t.Helper()
	var response apigen.UserSelfResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode self body %q: %v", rec.Body.String(), err)
	}
	return response
}

// tryLogin performs a login and reports whether it succeeded (the
// expected-failure arm of the password tests: a 401 is an outcome to
// assert, not a fatal wiring state).
func (e *userSettingsEnv) tryLogin(t *testing.T, email, password string) (*http.Cookie, bool) {
	t.Helper()
	body, err := json.Marshal(apigen.LoginRequest{Email: oapitypes.Email(email), Password: password})
	if err != nil {
		t.Fatalf("marshal login body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		return nil, false
	}
	return sessionCookie(t, rec), true
}

// TestUserSelfGetAnswersDefaults pins the read of a fresh account:
// the wizard's identity fields plus the column defaults of the
// appearance pair and the download preferences.
func TestUserSelfGetAnswersDefaults(t *testing.T) {
	env := newUserSettingsEnv(t)

	rec := env.getSelf(t, env.cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	self := decodeSelf(t, rec)
	if self.Email != "ada@example.com" {
		t.Errorf("email = %q, want the wizard's address", self.Email)
	}
	if self.Role != apigen.UserRoleAdmin {
		t.Errorf("role = %q, want admin", self.Role)
	}
	if self.Theme != apigen.UserThemeAuto {
		t.Errorf("theme = %q, want auto", self.Theme)
	}
	if self.Locale != apigen.UserLocaleEn {
		t.Errorf("locale = %q, want en", self.Locale)
	}
	if self.DownloadFormat != apigen.Zip {
		t.Errorf("downloadFormat = %q, want zip", self.DownloadFormat)
	}
	if !self.DownloadIncludeVideos {
		t.Error("downloadIncludeVideos = false, want the true default")
	}
	if self.DownloadIncludeSidecars {
		t.Error("downloadIncludeSidecars = true, want the false default")
	}
	if self.DownloadIncludeEmbeddedVideos {
		t.Error("downloadIncludeEmbeddedVideos = true, want the false default")
	}
	if self.Name == "" {
		t.Error("name is empty, want the wizard's default (the address)")
	}
}

// TestUserSelfPatchSavesEachSection pins the patch's three faces:
// the profile (name and email), the appearance (theme and locale)
// and the download preferences. Every face answers the complete
// saved state, an absent field keeps its stored value, and the users
// row persists the values for the next sign-in.
func TestUserSelfPatchSavesEachSection(t *testing.T) {
	env := newUserSettingsEnv(t)

	// The profile section.
	rec := env.patchSelf(t, env.cookie, `{"name":"Ada Lovelace","email":"ada@lovelace.example"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("profile PATCH status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	self := decodeSelf(t, rec)
	if self.Name != "Ada Lovelace" {
		t.Errorf("name = %q, want the patched display name", self.Name)
	}
	if self.Email != "ada@lovelace.example" {
		t.Errorf("email = %q, want the patched address", self.Email)
	}
	if self.Theme != apigen.UserThemeAuto {
		t.Errorf("theme = %q, want the untouched auto", self.Theme)
	}

	// The appearance section: theme and locale ride the same
	// endpoint; the name of the last patch survives untouched.
	rec = env.patchSelf(t, env.cookie, `{"theme":"dark","locale":"zh-Hans"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("appearance PATCH status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	self = decodeSelf(t, rec)
	if self.Theme != apigen.UserThemeDark {
		t.Errorf("theme = %q, want dark", self.Theme)
	}
	if self.Locale != apigen.UserLocaleZhHans {
		t.Errorf("locale = %q, want zh-Hans", self.Locale)
	}
	if self.Name != "Ada Lovelace" {
		t.Errorf("name = %q, want the untouched display name", self.Name)
	}

	// The download preferences section.
	rec = env.patchSelf(t, env.cookie,
		`{"downloadFormat":"tar","downloadIncludeVideos":false,"downloadIncludeSidecars":true,"downloadIncludeEmbeddedVideos":true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("download PATCH status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	self = decodeSelf(t, rec)
	if self.DownloadFormat != apigen.Tar {
		t.Errorf("downloadFormat = %q, want tar", self.DownloadFormat)
	}
	if self.DownloadIncludeVideos {
		t.Error("downloadIncludeVideos = true, want the patched false")
	}
	if !self.DownloadIncludeSidecars {
		t.Error("downloadIncludeSidecars = false, want the patched true")
	}
	if !self.DownloadIncludeEmbeddedVideos {
		t.Error("downloadIncludeEmbeddedVideos = false, want the patched true")
	}

	// The row persists everything for the next sign-in (the read
	// answers from the database, not this handler's memory).
	rec = env.getSelf(t, env.cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET after the patches status = %d, want %d", rec.Code, http.StatusOK)
	}
	self = decodeSelf(t, rec)
	if self.Theme != apigen.UserThemeDark || self.Locale != apigen.UserLocaleZhHans ||
		self.DownloadFormat != apigen.Tar || self.DownloadIncludeVideos || !self.DownloadIncludeSidecars ||
		!self.DownloadIncludeEmbeddedVideos {
		t.Errorf("reread self = %+v, want every patched value persisted", self)
	}
}

// TestUserSelfPatchEmailTakenAnswers409 pins the email's
// cross-account guard: an address another account owns answers 409
// USER_EMAIL_TAKEN and nothing is written.
func TestUserSelfPatchEmailTakenAnswers409(t *testing.T) {
	env := newUserSettingsEnv(t)
	seedPlainUser(t, env.pool, "grace@example.com")

	rec := env.patchSelf(t, env.cookie, `{"email":"GRACE@example.com"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("PATCH status = %d, want %d (body %q)", rec.Code, http.StatusConflict, rec.Body.String())
	}
	apiErr, _ := decodeError(t, rec)
	if apiErr.Code != apigen.USEREMAILTAKEN {
		t.Errorf("code = %q, want %q", apiErr.Code, apigen.USEREMAILTAKEN)
	}

	// Nothing was written: the caller's address is unchanged.
	self := decodeSelf(t, env.getSelf(t, env.cookie))
	if self.Email != "ada@example.com" {
		t.Errorf("email = %q after the refused patch, want the unchanged address", self.Email)
	}
}

// TestUserSelfPatchRejectsInvalidFields pins the validation of every
// patchable field: each offending value answers 400 VALIDATION_FAILED
// naming its field, and nothing is written.
func TestUserSelfPatchRejectsInvalidFields(t *testing.T) {
	env := newUserSettingsEnv(t)

	cases := []struct {
		name   string
		body   string
		fields []string
	}{
		{"EmptyName", `{"name":"   "}`, []string{"name"}},
		{"NameTooLong", `{"name":"` + strings.Repeat("A", 101) + `"}`, []string{"name"}},
		{"MalformedEmail", `{"email":"not-an-email"}`, []string{"email"}},
		{"UnknownTheme", `{"theme":"sepia"}`, []string{"theme"}},
		{"UnknownLocale", `{"locale":"klingon"}`, []string{"locale"}},
		{"UnknownDownloadFormat", `{"downloadFormat":"rar"}`, []string{"downloadFormat"}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			rec := env.patchSelf(t, env.cookie, testCase.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("PATCH status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
			apiErr, fields := decodeError(t, rec)
			if apiErr.Code != apigen.VALIDATIONFAILED {
				t.Errorf("code = %q, want %q", apiErr.Code, apigen.VALIDATIONFAILED)
			}
			for _, want := range testCase.fields {
				if !containsField(fields, want) {
					t.Errorf("details.fields = %v, want it to contain %q", fields, want)
				}
			}
		})
	}

	// Nothing was written: the account is still on its defaults.
	self := decodeSelf(t, env.getSelf(t, env.cookie))
	if self.Theme != apigen.UserThemeAuto || self.Locale != apigen.UserLocaleEn {
		t.Errorf("self = %+v after refused patches, want the untouched defaults", self)
	}
}

// TestChangePasswordRevokesOtherSessionsKeepsCurrent pins THE
// semantic pin of the settings feature, with a dual-session
// fixture: two signed-in sessions of one account; the password
// change performed through the first revokes the second (its
// credential answers 401 from its next request on) while the
// performing session stays signed in; the old password stops
// working at sign-in, the new one works.
func TestChangePasswordRevokesOtherSessionsKeepsCurrent(t *testing.T) {
	env := newUserSettingsEnv(t)

	// The dual-session fixture: a second login of the same account.
	otherCookie, _ := loginCookie(t, env.router, "ada@example.com", testPassword, "another device")

	// Both sessions are live before the change.
	if rec := env.getSelf(t, env.cookie); rec.Code != http.StatusOK {
		t.Fatalf("current session before the change: status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec := env.getSelf(t, otherCookie); rec.Code != http.StatusOK {
		t.Fatalf("other session before the change: status = %d, want %d", rec.Code, http.StatusOK)
	}

	rec := env.changePassword(t, env.cookie, testPassword, "another fine passphrase here")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("PUT password status = %d, want %d (body %q)", rec.Code, http.StatusNoContent, rec.Body.String())
	}

	// THE PIN, first half: the performing session STAYS signed in.
	if rec := env.getSelf(t, env.cookie); rec.Code != http.StatusOK {
		t.Fatalf("current session after the change: status = %d, want %d (the performing session stays signed in)", rec.Code, http.StatusOK)
	}

	// THE PIN, second half: the OTHER session is revoked -- its
	// credential fails authentication like every dead credential.
	other := env.getSelf(t, otherCookie)
	if other.Code != http.StatusUnauthorized {
		t.Fatalf("other session after the change: status = %d, want %d (the changed credential invalidated it)", other.Code, http.StatusUnauthorized)
	}
	apiErr, _ := decodeError(t, other)
	if apiErr.Code != apigen.AUTHINVALIDCREDENTIALS {
		t.Errorf("other session's code = %q, want %q", apiErr.Code, apigen.AUTHINVALIDCREDENTIALS)
	}

	// The old password no longer signs in; the new one does.
	if _, ok := env.tryLogin(t, "ada@example.com", testPassword); ok {
		t.Error("login with the old password succeeded, want the credential to be gone")
	}
	freshCookie, ok := env.tryLogin(t, "ada@example.com", "another fine passphrase here")
	if !ok {
		t.Fatal("login with the new password failed, want the fresh credential to work")
	}
	if rec := env.getSelf(t, freshCookie); rec.Code != http.StatusOK {
		t.Errorf("fresh session status = %d, want %d", rec.Code, http.StatusOK)
	}
}

// TestChangePasswordWrongCurrentIs400Not401 pins the failure shape:
// a wrong current password answers 400 VALIDATION_FAILED naming
// `current` -- NOT 401 -- so the caller's own session survives the
// mistake (a 401 would read as an expiry through clients' shared
// interceptors and sign the caller out of the very session they
// were using).
func TestChangePasswordWrongCurrentIs400Not401(t *testing.T) {
	env := newUserSettingsEnv(t)

	rec := env.changePassword(t, env.cookie, "wrong current password", "another fine passphrase here")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("PUT password status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	apiErr, fields := decodeError(t, rec)
	if apiErr.Code != apigen.VALIDATIONFAILED {
		t.Errorf("code = %q, want %q", apiErr.Code, apigen.VALIDATIONFAILED)
	}
	if !containsField(fields, "current") {
		t.Errorf("details.fields = %v, want it to contain %q", fields, "current")
	}

	// The session stays signed in, and the old password still works.
	if rec := env.getSelf(t, env.cookie); rec.Code != http.StatusOK {
		t.Fatalf("session after the refused change: status = %d, want %d (the session survives the mistake)", rec.Code, http.StatusOK)
	}
	if _, ok := env.tryLogin(t, "ada@example.com", testPassword); !ok {
		t.Error("login with the old password failed, want the refused change to write nothing")
	}
}

// TestChangePasswordRejectsBadNext pins the next password's bounds:
// under the 8-rune minimum (and over the 128-rune maximum) answers
// 400 naming `next`.
func TestChangePasswordRejectsBadNext(t *testing.T) {
	env := newUserSettingsEnv(t)

	for name, next := range map[string]string{
		"TooShort": "seven77",
		"TooLong":  strings.Repeat("x", 129),
	} {
		t.Run(name, func(t *testing.T) {
			rec := env.changePassword(t, env.cookie, testPassword, next)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("PUT password status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
			apiErr, fields := decodeError(t, rec)
			if apiErr.Code != apigen.VALIDATIONFAILED {
				t.Errorf("code = %q, want %q", apiErr.Code, apigen.VALIDATIONFAILED)
			}
			if !containsField(fields, "next") {
				t.Errorf("details.fields = %v, want it to contain %q", fields, "next")
			}
		})
	}
}

// TestUserDevicesListMarksCurrentAndAddress pins the device list:
// every active session of the caller with the isCurrent marker
// derived against the request's own credential, the device label,
// and the creation-time address.
func TestUserDevicesListMarksCurrentAndAddress(t *testing.T) {
	env := newUserSettingsEnv(t)
	_, otherInfo := loginCookie(t, env.router, "ada@example.com", testPassword, "another device")

	rec := env.request(t, http.MethodGet, "/api/v1/users/me/sessions", "", env.cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET devices status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var devices []apigen.DeviceSession
	if err := json.Unmarshal(rec.Body.Bytes(), &devices); err != nil {
		t.Fatalf("decode devices body %q: %v", rec.Body.String(), err)
	}
	if len(devices) != 2 {
		t.Fatalf("devices = %d entries, want 2 (the caller's and the other login's)", len(devices))
	}

	// Exactly one entry is current, and it is the caller's own.
	current := 0
	for _, device := range devices {
		if device.IsCurrent {
			current++
		}
	}
	if current != 1 {
		t.Errorf("isCurrent count = %d, want exactly 1", current)
	}
	// The caller's own row is the marked one; the other login's is
	// not. (Both logins share one account, so the ids decide.)
	byID := make(map[string]apigen.DeviceSession, len(devices))
	for _, device := range devices {
		byID[device.Id.String()] = device
	}
	if marked := byID[env.sessionID.String()]; !marked.IsCurrent {
		t.Errorf("the caller's own session %s is not marked isCurrent", env.sessionID)
	}
	if marked := byID[otherInfo.Id.String()]; marked.IsCurrent {
		t.Errorf("the other login's session %s is marked isCurrent, want not", otherInfo.Id)
	}

	// Every entry carries the creation-time address of its login
	// (the httptest peer address) and its device label.
	for _, device := range devices {
		if device.IpAddress == nil || *device.IpAddress != "192.0.2.1" {
			t.Errorf("device %s ipAddress = %v, want the captured 192.0.2.1", device.Id, device.IpAddress)
		}
		if device.DeviceName == nil || *device.DeviceName == "" {
			t.Errorf("device %s carries no deviceName, want the login's label", device.Id)
		}
	}
}

// TestUserDeviceRevokeSignsOutOtherDeviceAndRefusesCurrent pins the
// revoke: the other device's session dies (its credential answers
// 401), the caller's stays, and the current session itself is
// REFUSED with 400 -- the sign-out flow is that session's exit.
func TestUserDeviceRevokeSignsOutOtherDeviceAndRefusesCurrent(t *testing.T) {
	env := newUserSettingsEnv(t)
	otherCookie, otherInfo := loginCookie(t, env.router, "ada@example.com", testPassword, "another device")

	target := "/api/v1/users/me/sessions/" + otherInfo.Id.String()
	rec := env.request(t, http.MethodDelete, target, "", env.cookie)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE other device status = %d, want %d (body %q)", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if rec := env.getSelf(t, otherCookie); rec.Code != http.StatusUnauthorized {
		t.Errorf("other session after the revoke: status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if rec := env.getSelf(t, env.cookie); rec.Code != http.StatusOK {
		t.Errorf("caller session after the revoke: status = %d, want %d", rec.Code, http.StatusOK)
	}

	// The current session is refused: 400, never a self-sign-out.
	rec = env.request(t, http.MethodDelete, "/api/v1/users/me/sessions/"+env.sessionID.String(), "", env.cookie)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("DELETE current session status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	apiErr, fields := decodeError(t, rec)
	if apiErr.Code != apigen.VALIDATIONFAILED {
		t.Errorf("code = %q, want %q", apiErr.Code, apigen.VALIDATIONFAILED)
	}
	if !containsField(fields, "id") {
		t.Errorf("details.fields = %v, want it to contain %q", fields, "id")
	}
	// And the refusal signed nobody out.
	if rec := env.getSelf(t, env.cookie); rec.Code != http.StatusOK {
		t.Errorf("caller session after the refused self-revoke: status = %d, want %d (still signed in)", rec.Code, http.StatusOK)
	}
}

// TestUserSettingsRequireSession pins the me-scoping's other half:
// without a session there is no identity, and every endpoint of the
// surface answers the uniform 401.
func TestUserSettingsRequireSession(t *testing.T) {
	env := newUserSettingsEnv(t)

	targets := []struct {
		method string
		target string
		body   string
	}{
		{http.MethodGet, "/api/v1/users/me", ""},
		{http.MethodPatch, "/api/v1/users/me", `{"name":"x"}`},
		{http.MethodPut, "/api/v1/users/me/password", `{"current":"x","next":"12345678"}`},
		{http.MethodGet, "/api/v1/users/me/sessions", ""},
		{http.MethodDelete, "/api/v1/users/me/sessions/0192f2a1-3b4c-7d8e-9f0a-1b2c3d4e5f60", ""},
	}
	for _, target := range targets {
		t.Run(target.method+" "+target.target, func(t *testing.T) {
			rec := env.request(t, target.method, target.target, target.body, nil)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusUnauthorized, rec.Body.String())
			}
			apiErr, _ := decodeError(t, rec)
			if apiErr.Code != apigen.AUTHINVALIDCREDENTIALS {
				t.Errorf("code = %q, want %q", apiErr.Code, apigen.AUTHINVALIDCREDENTIALS)
			}
		})
	}
}
