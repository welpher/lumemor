package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	oapitypes "github.com/oapi-codegen/runtime/types"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/auth"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
	"github.com/welpher/lumemor/server/internal/events"
)

// testPassword is the shared credential of the test accounts.
const testPassword = "correct horse battery staple"

// sessionCookieName mirrors the auth middleware constant in assertions.
const wantSessionCookieName = "lumemor_session"

// webSessionTTL mirrors the middleware constant in expiry assertions.
const wantWebSessionTTL = 30 * 24 * time.Hour

// authEnv is a fully wired test environment: a migrated throwaway
// database, a router over it, and the event bus backing the SSE stream.
type authEnv struct {
	router  http.Handler
	pool    *pgxpool.Pool
	queries *dbgen.Queries
	bus     *events.Bus
}

// newAuthEnv provisions the environment; a nil bus gets a live one so
// session events are observable.
func newAuthEnv(t *testing.T, bus *events.Bus) *authEnv {
	t.Helper()
	pool := newSetupEnv(t)
	if bus == nil {
		bus = events.NewBus(5*time.Minute, time.Now)
	}
	return &authEnv{
		router:  NewRouter(&Deps{Pool: pool, Version: testVersion, Events: bus}),
		pool:    pool,
		queries: dbgen.New(pool),
		bus:     bus,
	}
}

// createAdmin seeds the first administrator through the wizard.
func (e *authEnv) createAdmin(t *testing.T, email, password string) {
	t.Helper()
	rec := postAdmin(t, e.router, createAdminBody(t, email, password, nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create admin %s: status = %d, want %d (body %q)", email, rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// loginBody marshals a login payload.
func loginBody(t *testing.T, email, password string) string {
	t.Helper()
	body, err := json.Marshal(apigen.LoginRequest{
		Email:    oapitypes.Email(email),
		Password: password,
	})
	if err != nil {
		t.Fatalf("marshal login body: %v", err)
	}
	return string(body)
}

// postLoginRaw fires a login request with an arbitrary body.
func postLoginRaw(router http.Handler, body, userAgent string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// postLogin fires a login request with the given credentials.
func postLogin(t *testing.T, router http.Handler, email, password, userAgent string) *httptest.ResponseRecorder {
	t.Helper()
	return postLoginRaw(router, loginBody(t, email, password), userAgent)
}

// sessionCookie extracts the lumemor_session cookie from a response.
func sessionCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == wantSessionCookieName {
			return cookie
		}
	}
	t.Fatalf("response carries no %s cookie (Set-Cookie: %v)", wantSessionCookieName, rec.Header()["Set-Cookie"])
	return nil
}

// loginCookie performs a successful login and returns the session
// cookie together with the SessionInfo body.
func loginCookie(t *testing.T, router http.Handler, email, password, userAgent string) (*http.Cookie, apigen.SessionInfo) {
	t.Helper()
	rec := postLogin(t, router, email, password, userAgent)
	if rec.Code != http.StatusOK {
		t.Fatalf("login %s: status = %d, want %d (body %q)", email, rec.Code, http.StatusOK, rec.Body.String())
	}
	var info apigen.SessionInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &info); err != nil {
		t.Fatalf("decode session info %q: %v", rec.Body.String(), err)
	}
	return sessionCookie(t, rec), info
}

// authedRequest fires a request with the session cookie attached.
func authedRequest(router http.Handler, method, target string, cookie *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// bearerRequest fires a request with a bearer credential.
func bearerRequest(router http.Handler, method, target, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// lookupSessionByToken returns the session row behind a raw credential.
func (e *authEnv) lookupSessionByToken(t *testing.T, raw string) (dbgen.GetSessionByTokenHashRow, error) {
	t.Helper()
	return e.queries.GetSessionByTokenHash(t.Context(), auth.HashSessionToken(raw))
}

// exec runs a maintenance statement against the test database. It uses
// a background context deliberately: it also backs t.Cleanup calls,
// which run after the test context is already canceled.
func (e *authEnv) exec(t *testing.T, statement string, args ...any) {
	t.Helper()
	if _, err := e.pool.Exec(context.Background(), statement, args...); err != nil {
		t.Fatalf("exec %q: %v", statement, err)
	}
}

// awaitSessionEvent scans the bus ring for the session lifecycle event
// of one session. The ring replays in publish order, so the scan walks
// the older events and stops at the match; the context bound only
// matters when the expected event was never published.
func awaitSessionEvent(t *testing.T, bus *events.Bus, name, sessionID string) events.Event {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	subscription := bus.Subscribe(ctx, 0)
	for {
		event, ok := subscription.Next()
		if !ok {
			t.Fatalf("no %s event for session %s reached the bus", name, sessionID)
		}
		if event.Name != name {
			continue
		}
		if payload := eventPayload(t, event); payload["id"] == sessionID {
			return event
		}
	}
}

// eventPayload decodes the JSON payload of an event into a map.
func eventPayload(t *testing.T, event events.Event) map[string]string {
	t.Helper()
	var payload map[string]string
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("decode event payload %q: %v", event.Payload, err)
	}
	return payload
}

// TestLogin walks POST /auth/login end to end: the happy path with its
// cookie contract, the anti-enumeration 401s, the disabled account, the
// rate-limit lockout, payload validation, and the session.started event.
func TestLogin(t *testing.T) {
	env := newAuthEnv(t, nil)
	env.createAdmin(t, "ada@example.com", testPassword)

	t.Run("CreatesSessionWithCookieAndHashedToken", func(t *testing.T) {
		before := time.Now()
		cookie, info := loginCookie(t, env.router, "ada@example.com", testPassword, "Firefox on macOS")
		after := time.Now()

		// The cookie attributes over plain http.
		if cookie.HttpOnly != true {
			t.Error("cookie HttpOnly = false, want true")
		}
		if cookie.SameSite != http.SameSiteStrictMode {
			t.Errorf("cookie SameSite = %v, want Strict", cookie.SameSite)
		}
		if cookie.Secure {
			t.Error("cookie Secure = true over http, want false")
		}
		if cookie.Path != "/" {
			t.Errorf("cookie Path = %q, want %q", cookie.Path, "/")
		}
		if want := int(wantWebSessionTTL / time.Second); cookie.MaxAge != want {
			t.Errorf("cookie MaxAge = %d, want %d (30 days)", cookie.MaxAge, want)
		}

		// The body describes the session.
		if info.Id.Version() != 7 {
			t.Errorf("session id = %s, want a UUIDv7", info.Id)
		}
		if info.Kind != apigen.SessionInfoKindWeb {
			t.Errorf("session kind = %q, want web", info.Kind)
		}
		if info.DeviceName == nil || *info.DeviceName != "Firefox on macOS" {
			t.Errorf("session deviceName = %v, want the login user agent", info.DeviceName)
		}
		if info.LastSeenAt.Before(before) || info.LastSeenAt.After(after) {
			t.Errorf("session lastSeenAt = %v, want the login time", info.LastSeenAt)
		}
		if info.ExpiresAt == nil {
			t.Fatal("web session expiresAt = null, want the slid expiry")
		}
		if want := before.Add(wantWebSessionTTL); info.ExpiresAt.Before(want.Add(-time.Minute)) || info.ExpiresAt.After(want.Add(time.Minute)) {
			t.Errorf("session expiresAt = %v, want ~%v (30 days out)", info.ExpiresAt, want)
		}

		// The credential is stored only as its hash.
		raw := cookie.Value
		if raw == "" {
			t.Fatal("cookie carries an empty credential")
		}
		row, err := env.lookupSessionByToken(t, raw)
		if err != nil {
			t.Fatalf("lookup by the presented credential: %v", err)
		}
		if row.TokenHash == raw {
			t.Error("sessions.token_hash stores the raw token: the credential would be in the clear")
		}
		if row.TokenHash != auth.HashSessionToken(raw) {
			t.Errorf("sessions.token_hash = %q, want the SHA-256 hex of the credential", row.TokenHash)
		}
		if row.ID != pgUUID(info.Id) {
			t.Errorf("session row id = %v, want %v", row.ID, info.Id)
		}
	})

	t.Run("UppercaseEmailHits", func(t *testing.T) {
		// The stored address is lower-case; a login with
		// mixed case (and stray spaces) must still hit it.
		rec := postLogin(t, env.router, "  ADA@EXAMPLE.COM ", testPassword, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("login with upper-case email: status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
	})

	t.Run("AntiEnumeration", func(t *testing.T) {
		wrongPassword := postLogin(t, env.router, "ada@example.com", "wrong horse battery staple", "")
		noSuchUser := postLogin(t, env.router, "nobody@example.com", "wrong horse battery staple", "")
		for name, rec := range map[string]*httptest.ResponseRecorder{
			"wrong password": wrongPassword,
			"no such user":   noSuchUser,
		} {
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("%s: status = %d, want %d", name, rec.Code, http.StatusUnauthorized)
			}
		}
		// Identical envelopes: no way to tell a wrong password from an
		// unknown account.
		if wrongPassword.Body.String() != noSuchUser.Body.String() {
			t.Errorf("wrong-password body %q differs from no-such-user body %q", wrongPassword.Body.String(), noSuchUser.Body.String())
		}
		apiErr, _ := decodeError(t, wrongPassword)
		if apiErr.Code != apigen.AUTHINVALIDCREDENTIALS {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.AUTHINVALIDCREDENTIALS)
		}
	})

	t.Run("DisabledAccountSame401", func(t *testing.T) {
		env.exec(t, "UPDATE users SET is_disabled = true WHERE email = 'ada@example.com'")
		t.Cleanup(func() {
			env.exec(t, "UPDATE users SET is_disabled = false WHERE email = 'ada@example.com'")
		})

		rec := postLogin(t, env.router, "ada@example.com", testPassword, "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("disabled account login: status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
		apiErr, _ := decodeError(t, rec)
		if apiErr.Code != apigen.AUTHINVALIDCREDENTIALS {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.AUTHINVALIDCREDENTIALS)
		}
	})

	t.Run("RateLimitLocksSixthAttempt", func(t *testing.T) {
		// Five failures lock the pair; the sixth attempt
		// is rejected with 429 even though its password is correct.
		// The wizard is one-time, so the second account is seeded
		// directly with a real password hash.
		grace, err := env.queries.CreateUser(context.Background(), dbgen.CreateUserParams{
			ID:           pgUUID(mustUUIDv7(t)),
			Name:         "Grace Hopper",
			Email:        "grace@example.com",
			PasswordHash: mustHash(t, testPassword),
			Role:         dbgen.UserRoleUser,
		})
		if err != nil {
			t.Fatalf("seed grace: %v", err)
		}
		for i := 0; i < 5; i++ {
			rec := postLogin(t, env.router, "grace@example.com", "wrong horse battery staple", "")
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("failed attempt %d: status = %d, want %d", i+1, rec.Code, http.StatusUnauthorized)
			}
		}

		rec := postLogin(t, env.router, "grace@example.com", testPassword, "")
		if rec.Code != http.StatusTooManyRequests {
			t.Fatalf("sixth attempt with the correct password: status = %d, want %d (body %q)", rec.Code, http.StatusTooManyRequests, rec.Body.String())
		}
		apiErr, _ := decodeError(t, rec)
		if apiErr.Code != apigen.AUTHRATELIMITED {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.AUTHRATELIMITED)
		}

		retryAfter := rec.Header().Get("Retry-After")
		if retryAfter == "" {
			t.Fatal("429 response carries no Retry-After header")
		}
		seconds, err := strconv.Atoi(retryAfter)
		if err != nil {
			t.Fatalf("Retry-After = %q, want integer seconds: %v", retryAfter, err)
		}
		if seconds <= 0 || seconds > 900 {
			t.Errorf("Retry-After = %d seconds, want (0, 900] for a 15-minute lockout", seconds)
		}

		// No session may exist for the locked account.
		sessions, err := env.queries.ListUserSessions(t.Context(), grace.ID)
		if err != nil {
			t.Fatalf("ListUserSessions(grace): %v", err)
		}
		if len(sessions) != 0 {
			t.Errorf("grace owns %d sessions after a locked login, want 0", len(sessions))
		}
	})

	t.Run("RejectsInvalidPayloads", func(t *testing.T) {
		cases := []struct {
			name   string
			body   string
			fields []string
		}{
			{name: "BodyNotJSON", body: `not json at all`, fields: nil},
			{name: "EmailInvalid", body: `{"email":"not-an-email","password":"x"}`, fields: []string{"email"}},
			{name: "EmailMissing", body: `{"password":"x"}`, fields: []string{"email"}},
			{name: "PasswordMissing", body: `{"email":"ada@example.com"}`, fields: []string{"password"}},
			{
				name:   "BodyTooLarge",
				body:   `{"email":"ada@example.com","password":"` + strings.Repeat("a", 1<<20) + `"}`,
				fields: []string{"body"},
			},
		}
		for _, testCase := range cases {
			t.Run(testCase.name, func(t *testing.T) {
				rec := postLoginRaw(env.router, testCase.body, "")

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
			})
		}
	})

	t.Run("PublishesSessionStarted", func(t *testing.T) {
		cookie, info := loginCookie(t, env.router, "ada@example.com", testPassword, "")

		event := awaitSessionEvent(t, env.bus, "session.started", info.Id.String())
		payload := eventPayload(t, event)
		if payload["kind"] != "web" {
			t.Errorf("event payload kind = %q, want web", payload["kind"])
		}
		if payload["id"] == cookie.Value {
			t.Error("event payload leaks the raw credential")
		}
	})
}

// awaitEventNamed drains an open subscription until an event with the
// given name arrives; the test fails when the subscription's context
// ends first (the expected event was never published).
func awaitEventNamed(t *testing.T, sub *events.Subscription, name string) events.Event {
	t.Helper()
	for {
		event, ok := sub.Next()
		if !ok {
			t.Fatalf("subscription ended before a %s event arrived", name)
		}
		if event.Name == name {
			return event
		}
	}
}

// subscribeAfterCursor opens a subscription that only sees events
// published after the returned cursor id, so assertions are independent
// of whatever the bus already holds from earlier setup.
func subscribeAfterCursor(t *testing.T, bus *events.Bus) (int64, *events.Subscription) {
	t.Helper()
	cursor := bus.Publish("test.cursor", map[string]string{})
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	t.Cleanup(cancel)
	return cursor, bus.Subscribe(ctx, cursor)
}

// TestLoginAuditEvents pins the audit half of the login contract: a
// successful login reports only session.started onto the activity
// stream, while every 401 rejection -- wrong password, unknown
// address, disabled account -- reports auth.login_failed carrying the
// attempted address.
// TestSessionIdentity pins the session's identity face: every
// SessionInfo answer -- the login body, the session list the web's
// boot probe reads to restore a session, and (in the pairing suite)
// the consume response's embedded session -- carries the account's
// userId and role. A client that restores a session never re-logs in;
// the list it probes must answer who is calling, not just that
// someone is.
func TestSessionIdentity(t *testing.T) {
	env := newAuthEnv(t, nil)
	env.createAdmin(t, "ada@example.com", testPassword)
	ada, err := env.queries.GetUserByEmail(t.Context(), "ada@example.com")
	if err != nil {
		t.Fatalf("look up ada: %v", err)
	}

	// A plain member account with a working password hash: the role
	// the second login arm must answer.
	graceHash, err := auth.Hash(testPassword)
	if err != nil {
		t.Fatalf("hash grace's password: %v", err)
	}
	graceID := mustUUIDv7(t)
	env.exec(t,
		"INSERT INTO users (id, name, email, password_hash, role) VALUES ($1, $2, $3, $4, 'user')",
		pgUUID(graceID), "Grace Hopper", "grace@example.com", graceHash)

	t.Run("LoginCarriesIdentity", func(t *testing.T) {
		_, adminInfo := loginCookie(t, env.router, "ada@example.com", testPassword, "")
		if uuid.UUID(adminInfo.UserId) != uuid.UUID(ada.ID.Bytes) {
			t.Errorf("login userId = %s, want ada's account id %s", adminInfo.UserId, ada.ID)
		}
		if adminInfo.Role != apigen.UserRoleAdmin {
			t.Errorf("login role = %q, want admin (the wizard's account)", adminInfo.Role)
		}

		_, userInfo := loginCookie(t, env.router, "grace@example.com", testPassword, "")
		if uuid.UUID(userInfo.UserId) != graceID {
			t.Errorf("member login userId = %s, want grace's account id %s", userInfo.UserId, graceID)
		}
		if userInfo.Role != apigen.UserRoleUser {
			t.Errorf("member login role = %q, want user (the default role)", userInfo.Role)
		}
	})

	t.Run("SessionListCarriesIdentity", func(t *testing.T) {
		// The web restores a signed-in session by probing this list:
		// the identity must ride the restore path, not just the login.
		adminCookie, _ := loginCookie(t, env.router, "ada@example.com", testPassword, "")
		rec := authedRequest(env.router, http.MethodGet, "/api/v1/auth/sessions", adminCookie)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET /auth/sessions: status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		var sessions []apigen.SessionInfo
		if err := json.Unmarshal(rec.Body.Bytes(), &sessions); err != nil {
			t.Fatalf("decode sessions %q: %v", rec.Body.String(), err)
		}
		if len(sessions) == 0 {
			t.Fatal("session list is empty, want the just-logged-in session")
		}
		for _, session := range sessions {
			if uuid.UUID(session.UserId) != uuid.UUID(ada.ID.Bytes) {
				t.Errorf("session list userId = %s, want ada's account id %s", session.UserId, ada.ID)
			}
			if session.Role != apigen.UserRoleAdmin {
				t.Errorf("session list role = %q, want admin", session.Role)
			}
		}

		graceCookie, _ := loginCookie(t, env.router, "grace@example.com", testPassword, "")
		rec = authedRequest(env.router, http.MethodGet, "/api/v1/auth/sessions", graceCookie)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET /auth/sessions (grace): status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		sessions = nil
		if err := json.Unmarshal(rec.Body.Bytes(), &sessions); err != nil {
			t.Fatalf("decode grace's sessions %q: %v", rec.Body.String(), err)
		}
		if len(sessions) == 0 {
			t.Fatal("grace's session list is empty, want her just-logged-in session")
		}
		for _, session := range sessions {
			if uuid.UUID(session.UserId) != graceID {
				t.Errorf("grace's session list userId = %s, want %s", session.UserId, graceID)
			}
			if session.Role != apigen.UserRoleUser {
				t.Errorf("grace's session list role = %q, want user", session.Role)
			}
		}
	})
}

func TestLoginAuditEvents(t *testing.T) {
	env := newAuthEnv(t, nil)
	env.createAdmin(t, "ada@example.com", testPassword)

	t.Run("SuccessPublishesOnlySessionStarted", func(t *testing.T) {
		_, sub := subscribeAfterCursor(t, env.bus)

		cookie, info := loginCookie(t, env.router, "ada@example.com", testPassword, "")

		// The sentinel marks the end of the login's own publishes: the
		// drain below stops there, having seen everything the login
		// put on the bus.
		sentinel := env.bus.Publish("test.sentinel", map[string]string{})
		started := 0
		for {
			event, ok := sub.Next()
			if !ok {
				t.Fatal("subscription ended before the sentinel arrived")
			}
			if event.ID == sentinel {
				break
			}
			if event.Name == "auth.login_failed" {
				t.Errorf("successful login also published auth.login_failed (payload %s)", event.Payload)
			}
			if event.Name == "session.started" {
				started++
				if payload := eventPayload(t, event); payload["id"] != info.Id.String() {
					t.Errorf("session.started payload id = %q, want %q", payload["id"], info.Id)
				}
			}
		}
		if started != 1 {
			t.Errorf("successful login published %d session.started events, want 1", started)
		}
		if cookie.Value == "" {
			t.Error("login cookie carries no credential; the success path is not what this subtest thinks")
		}
	})

	t.Run("FailurePublishesLoginFailed", func(t *testing.T) {
		cases := []struct {
			name     string
			email    string
			disabled bool
		}{
			{name: "WrongPassword", email: "ada@example.com"},
			{name: "UnknownEmail", email: "nobody@example.com"},
			{name: "DisabledAccount", email: "ada@example.com", disabled: true},
		}
		for _, testCase := range cases {
			t.Run(testCase.name, func(t *testing.T) {
				if testCase.disabled {
					env.exec(t, "UPDATE users SET is_disabled = true WHERE email = 'ada@example.com'")
					t.Cleanup(func() {
						env.exec(t, "UPDATE users SET is_disabled = false WHERE email = 'ada@example.com'")
					})
				}

				// Subscribe before the attempt: the assertion must
				// observe the live publish, not only a ring replay.
				cursor, sub := subscribeAfterCursor(t, env.bus)

				rec := postLogin(t, env.router, testCase.email, "wrong horse battery staple", "")
				if rec.Code != http.StatusUnauthorized {
					t.Fatalf("failed login (%s): status = %d, want %d (body %q)", testCase.name, rec.Code, http.StatusUnauthorized, rec.Body.String())
				}

				event := awaitEventNamed(t, sub, "auth.login_failed")
				payload := eventPayload(t, event)
				if payload["email"] != testCase.email {
					t.Errorf("auth.login_failed payload email = %q, want the attempted address %q", payload["email"], testCase.email)
				}
				if len(payload) != 1 {
					t.Errorf("auth.login_failed payload = %v, want exactly the email field (no credentials ever)", payload)
				}
				if event.ID <= cursor {
					t.Errorf("auth.login_failed id = %d, want one published after the cursor %d", event.ID, cursor)
				}
			})
		}
	})
}

// TestLogoutAndRevoke walks the session management surface: logout
// revokes the current session and expires the cookie; DELETE
// /auth/sessions/{id} follows set semantics without existence leaks,
// and revoking the current session behaves like logout.
func TestLogoutAndRevoke(t *testing.T) {
	env := newAuthEnv(t, nil)
	env.createAdmin(t, "ada@example.com", testPassword)
	first, firstInfo := loginCookie(t, env.router, "ada@example.com", testPassword, "Firefox on macOS")

	t.Run("LogoutRevokesAndExpires", func(t *testing.T) {
		rec := authedRequest(env.router, http.MethodPost, "/api/v1/auth/logout", first)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("logout: status = %d, want %d (body %q)", rec.Code, http.StatusNoContent, rec.Body.String())
		}

		// The cookie is expired alongside the revocation.
		expired := sessionCookie(t, rec)
		if expired.MaxAge >= 0 {
			t.Errorf("logout cookie MaxAge = %d, want a negative value that expires it", expired.MaxAge)
		}
		if expired.Value != "" && expired.Value == first.Value {
			t.Error("logout cookie re-issues the revoked credential")
		}

		// The session is gone: the old credential no longer authenticates.
		if _, err := env.lookupSessionByToken(t, first.Value); err == nil {
			t.Error("session row survived logout")
		} else if !isNoRows(err) {
			t.Fatalf("lookup after logout: %v", err)
		}
		if followUp := authedRequest(env.router, http.MethodGet, "/api/v1/auth/sessions", first); followUp.Code != http.StatusUnauthorized {
			t.Errorf("request with the revoked cookie: status = %d, want %d", followUp.Code, http.StatusUnauthorized)
		}

		// The revocation reaches the activity stream.
		event := awaitSessionEvent(t, env.bus, "session.revoked", firstInfo.Id.String())
		if got := eventPayload(t, event)["id"]; got != firstInfo.Id.String() {
			t.Errorf("event payload id = %q, want %q", got, firstInfo.Id)
		}
	})

	t.Run("LogoutWithoutSession", func(t *testing.T) {
		rec := httptest.NewRecorder()
		env.router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated logout: status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
		apiErr, _ := decodeError(t, rec)
		if apiErr.Code != apigen.AUTHINVALIDCREDENTIALS {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.AUTHINVALIDCREDENTIALS)
		}

		garbage := &http.Cookie{Name: wantSessionCookieName, Value: "not-a-real-token"}
		rec = authedRequest(env.router, http.MethodPost, "/api/v1/auth/logout", garbage)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("logout with an unknown cookie: status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("ListSessionsOwnsOnly", func(t *testing.T) {
		current, _ := loginCookie(t, env.router, "ada@example.com", testPassword, "Chrome on Windows")

		// A second user with a live session: it must not appear in
		// ada's list.
		other, err := env.queries.CreateUser(t.Context(), dbgen.CreateUserParams{
			ID:           pgUUID(mustUUIDv7(t)),
			Name:         "Grace Hopper",
			Email:        "grace@example.com",
			PasswordHash: "$argon2id$v=19$m=65536,t=3,p=1$placeholder$placeholder",
			Role:         dbgen.UserRoleUser,
		})
		if err != nil {
			t.Fatalf("create grace: %v", err)
		}
		otherRaw, otherHash := auth.NewSessionToken()
		if _, err := env.queries.CreateSession(t.Context(), dbgen.CreateSessionParams{
			ID:         pgUUID(mustUUIDv7(t)),
			UserID:     other.ID,
			TokenHash:  otherHash,
			Kind:       dbgen.SessionKindWeb,
			DeviceName: "Grace's browser",
			LastSeenAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
			ExpiresAt:  pgtype.Timestamptz{Time: time.Now().Add(wantWebSessionTTL), Valid: true},
		}); err != nil {
			t.Fatalf("create grace's session: %v", err)
		}

		rec := authedRequest(env.router, http.MethodGet, "/api/v1/auth/sessions", current)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET /auth/sessions: status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		var sessions []apigen.SessionInfo
		if err := json.Unmarshal(rec.Body.Bytes(), &sessions); err != nil {
			t.Fatalf("decode sessions %q: %v", rec.Body.String(), err)
		}
		if len(sessions) == 0 {
			t.Fatal("session list is empty, want at least the current session")
		}
		for _, session := range sessions {
			if session.DeviceName != nil && *session.DeviceName == "Grace's browser" {
				t.Error("session list leaks another user's session")
			}
		}
		if sessions[0].Id != mustSessionIDFor(t, env, current) {
			// Most recently used first: the login that just happened
			// heads the list.
			t.Errorf("sessions[0].id = %s, want the most recently used session", sessions[0].Id)
		}

		// The other user's credential works on its own account.
		if rec := bearerRequest(env.router, http.MethodGet, "/api/v1/auth/sessions", otherRaw); rec.Code != http.StatusOK {
			t.Errorf("grace's own session lookup: status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("RevokeSetSemantics", func(t *testing.T) {
		current, _ := loginCookie(t, env.router, "ada@example.com", testPassword, "")
		victim, victimInfo := loginCookie(t, env.router, "ada@example.com", testPassword, "")

		// Revoking another own session: 204, the row is gone, the
		// current session keeps working.
		rec := authedRequest(env.router, http.MethodDelete, "/api/v1/auth/sessions/"+victimInfo.Id.String(), current)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("revoke other session: status = %d, want %d", rec.Code, http.StatusNoContent)
		}
		if _, err := env.lookupSessionByToken(t, victim.Value); !isNoRows(err) {
			t.Errorf("revoked victim lookup: err = %v, want no rows", err)
		}
		if rec := authedRequest(env.router, http.MethodGet, "/api/v1/auth/sessions", current); rec.Code != http.StatusOK {
			t.Errorf("current session after revoking another: status = %d, want %d", rec.Code, http.StatusOK)
		}

		// Unknown id: 204, indistinguishable from a success.
		unknown := mustUUIDv7(t)
		rec = authedRequest(env.router, http.MethodDelete, "/api/v1/auth/sessions/"+unknown.String(), current)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("revoke unknown id: status = %d, want %d (no existence leak)", rec.Code, http.StatusNoContent)
		}

		// Foreign id: 204 and the foreign session survives.
		foreignUser, err := env.queries.GetUserByEmail(t.Context(), "grace@example.com")
		if err != nil {
			t.Fatalf("GetUserByEmail(grace): %v", err)
		}
		foreignRaw, foreignHash := auth.NewSessionToken()
		foreignID := mustUUIDv7(t)
		if _, err := env.queries.CreateSession(t.Context(), dbgen.CreateSessionParams{
			ID:         pgUUID(foreignID),
			UserID:     foreignUser.ID,
			TokenHash:  foreignHash,
			Kind:       dbgen.SessionKindMobile,
			DeviceName: "Grace's phone",
			LastSeenAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
			ExpiresAt:  pgtype.Timestamptz{Time: time.Now().Add(wantWebSessionTTL), Valid: true},
		}); err != nil {
			t.Fatalf("create grace's mobile session: %v", err)
		}
		// The foreign credential keeps authenticating its own user --
		// ada's revoke must not have touched it.
		if rec := bearerRequest(env.router, http.MethodGet, "/api/v1/auth/sessions", foreignRaw); rec.Code != http.StatusOK {
			t.Errorf("grace's own request after ada's foreign revoke: status = %d, want %d", rec.Code, http.StatusOK)
		}
		rec = authedRequest(env.router, http.MethodDelete, "/api/v1/auth/sessions/"+foreignID.String(), current)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("revoke foreign id: status = %d, want %d (no existence leak)", rec.Code, http.StatusNoContent)
		}
		if _, err := env.queries.GetSession(t.Context(), pgUUID(foreignID)); err != nil {
			t.Errorf("foreign session vanished after another user's revoke: %v", err)
		}

		// Revoking the current session behaves like logout: the cookie
		// is expired too.
		currentID := mustSessionIDFor(t, env, current)
		rec = authedRequest(env.router, http.MethodDelete, "/api/v1/auth/sessions/"+currentID.String(), current)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("revoke current session: status = %d, want %d", rec.Code, http.StatusNoContent)
		}
		expired := sessionCookie(t, rec)
		if expired.MaxAge >= 0 {
			t.Errorf("revoke-current cookie MaxAge = %d, want a negative value", expired.MaxAge)
		}
		if followUp := authedRequest(env.router, http.MethodGet, "/api/v1/auth/sessions", current); followUp.Code != http.StatusUnauthorized {
			t.Errorf("request with the revoked credential: status = %d, want %d", followUp.Code, http.StatusUnauthorized)
		}
	})

	t.Run("RevokeRejectsInvalidUUID", func(t *testing.T) {
		current, _ := loginCookie(t, env.router, "ada@example.com", testPassword, "")
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/sessions/not-a-uuid", nil)
		req.AddCookie(current)
		rec := httptest.NewRecorder()
		env.router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("revoke with a malformed uuid: status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		apiErr, fields := decodeError(t, rec)
		if apiErr.Code != apigen.VALIDATIONFAILED {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.VALIDATIONFAILED)
		}
		if !contains(fields, "id") {
			t.Errorf("details.fields = %v, want it to contain %q", fields, "id")
		}
	})

	t.Run("RevokeAndListRequireSession", func(t *testing.T) {
		rec := httptest.NewRecorder()
		env.router.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/api/v1/auth/sessions/"+mustUUIDv7(t).String(), nil))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated revoke: status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
		rec = httptest.NewRecorder()
		env.router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/auth/sessions", nil))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated list: status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})
}

// TestRequireSession walks the auth middleware: cookie and bearer
// credentials, unknown and expired credentials, disabled accounts, and
// the throttled sliding expiry.
func TestRequireSession(t *testing.T) {
	env := newAuthEnv(t, nil)
	env.createAdmin(t, "ada@example.com", testPassword)

	t.Run("RejectsMissingAndUnknownCredentials", func(t *testing.T) {
		for name, request := range map[string]func() *httptest.ResponseRecorder{
			"no credential": func() *httptest.ResponseRecorder {
				rec := httptest.NewRecorder()
				env.router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/auth/sessions", nil))
				return rec
			},
			"unknown bearer": func() *httptest.ResponseRecorder {
				raw, _ := auth.NewSessionToken()
				return bearerRequest(env.router, http.MethodGet, "/api/v1/auth/sessions", raw)
			},
			"malformed authorization": func() *httptest.ResponseRecorder {
				req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/sessions", nil)
				req.Header.Set("Authorization", "Token abc")
				rec := httptest.NewRecorder()
				env.router.ServeHTTP(rec, req)
				return rec
			},
			"garbage cookie": func() *httptest.ResponseRecorder {
				return authedRequest(env.router, http.MethodGet, "/api/v1/auth/sessions",
					&http.Cookie{Name: wantSessionCookieName, Value: "garbage"})
			},
		} {
			rec := request()
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("%s: status = %d, want %d", name, rec.Code, http.StatusUnauthorized)
			}
			apiErr, _ := decodeError(t, rec)
			if apiErr.Code != apigen.AUTHINVALIDCREDENTIALS {
				t.Errorf("%s: code = %q, want %q", name, apiErr.Code, apigen.AUTHINVALIDCREDENTIALS)
			}
		}
	})

	t.Run("RejectsExpiredSession", func(t *testing.T) {
		cookie, _ := loginCookie(t, env.router, "ada@example.com", testPassword, "")
		env.exec(t, "UPDATE sessions SET expires_at = now() - interval '1 minute' WHERE token_hash = $1",
			auth.HashSessionToken(cookie.Value))

		rec := authedRequest(env.router, http.MethodGet, "/api/v1/auth/sessions", cookie)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expired session: status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("RejectsDisabledUser", func(t *testing.T) {
		cookie, _ := loginCookie(t, env.router, "ada@example.com", testPassword, "")
		env.exec(t, "UPDATE users SET is_disabled = true WHERE email = 'ada@example.com'")
		t.Cleanup(func() {
			env.exec(t, "UPDATE users SET is_disabled = false WHERE email = 'ada@example.com'")
		})

		rec := authedRequest(env.router, http.MethodGet, "/api/v1/auth/sessions", cookie)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("disabled user with a live session: status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("BearerSessionFromTheSessionsTable", func(t *testing.T) {
		// The mobile form of the same session records: a bearer
		// credential against the sessions table, never expiring.
		user, err := env.queries.GetUserByEmail(t.Context(), "ada@example.com")
		if err != nil {
			t.Fatalf("GetUserByEmail(ada): %v", err)
		}
		raw, hash := auth.NewSessionToken()
		if _, err := env.queries.CreateSession(t.Context(), dbgen.CreateSessionParams{
			ID:         pgUUID(mustUUIDv7(t)),
			UserID:     user.ID,
			TokenHash:  hash,
			Kind:       dbgen.SessionKindMobile,
			DeviceName: "Test phone",
			LastSeenAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
			// A past sentinel: bearer sessions are revoked, not
			// expired, so the lookup must still serve the row.
			ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(-time.Hour), Valid: true},
		}); err != nil {
			t.Fatalf("CreateSession(mobile): %v", err)
		}

		rec := bearerRequest(env.router, http.MethodGet, "/api/v1/auth/sessions", raw)
		if rec.Code != http.StatusOK {
			t.Fatalf("bearer request: status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		var sessions []apigen.SessionInfo
		if err := json.Unmarshal(rec.Body.Bytes(), &sessions); err != nil {
			t.Fatalf("decode sessions %q: %v", rec.Body.String(), err)
		}
		var mobile *apigen.SessionInfo
		for i := range sessions {
			if sessions[i].Kind == apigen.SessionInfoKindMobile {
				mobile = &sessions[i]
			}
		}
		if mobile == nil {
			t.Fatal("bearer session is missing from the session list")
		}
		if mobile.ExpiresAt != nil {
			t.Errorf("mobile session expiresAt = %v, want null (bearer sessions do not expire)", mobile.ExpiresAt)
		}
	})

	t.Run("SlidingExpiryThrottledToOneMinute", func(t *testing.T) {
		cookie, info := loginCookie(t, env.router, "ada@example.com", testPassword, "")

		// Within the throttle window the row is not rewritten.
		before, err := env.queries.GetSession(t.Context(), pgUUID(info.Id))
		if err != nil {
			t.Fatalf("GetSession(): %v", err)
		}
		authedRequest(env.router, http.MethodGet, "/api/v1/auth/sessions", cookie)
		after, err := env.queries.GetSession(t.Context(), pgUUID(info.Id))
		if err != nil {
			t.Fatalf("GetSession(): %v", err)
		}
		if !after.LastSeenAt.Time.Equal(before.LastSeenAt.Time) {
			t.Errorf("last_seen_at moved from %v to %v within the throttle window, want no write",
				before.LastSeenAt.Time, after.LastSeenAt.Time)
		}

		// A minute of idleness: the next authenticated request slides
		// both last_seen_at and expires_at.
		env.exec(t, "UPDATE sessions SET last_seen_at = now() - interval '2 minutes', expires_at = now() + interval '29 days' WHERE token_hash = $1",
			auth.HashSessionToken(cookie.Value))
		slid := time.Now()
		if rec := authedRequest(env.router, http.MethodGet, "/api/v1/auth/sessions", cookie); rec.Code != http.StatusOK {
			t.Fatalf("request after idle minute: status = %d, want %d", rec.Code, http.StatusOK)
		}
		moved, err := env.queries.GetSession(t.Context(), pgUUID(info.Id))
		if err != nil {
			t.Fatalf("GetSession(): %v", err)
		}
		if !moved.LastSeenAt.Time.After(before.LastSeenAt.Time) {
			t.Errorf("last_seen_at = %v after the idle window, want it moved to now", moved.LastSeenAt.Time)
		}
		if !moved.LastSeenAt.Time.Before(slid.Add(time.Minute)) {
			t.Errorf("last_seen_at = %v, want a stamp near %v", moved.LastSeenAt.Time, slid)
		}
		if want := slid.Add(wantWebSessionTTL); moved.ExpiresAt.Time.Before(want.Add(-time.Minute)) || moved.ExpiresAt.Time.After(want.Add(time.Minute)) {
			t.Errorf("expires_at = %v after the slide, want ~%v (30 days from the touch)", moved.ExpiresAt.Time, want)
		}
	})

	t.Run("EventsRouteRequiresSession", func(t *testing.T) {
		// The SSE stream moved into the authed group: no session, no
		// stream. The request context is bounded so that, were the
		// route still unauthenticated, the stream would end instead of
		// wedging the test.
		ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
		defer cancel()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil).WithContext(ctx)
		rec := httptest.NewRecorder()
		env.router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("GET /api/v1/events without a session: status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
		apiErr, _ := decodeError(t, rec)
		if apiErr.Code != apigen.AUTHINVALIDCREDENTIALS {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.AUTHINVALIDCREDENTIALS)
		}
	})

	t.Run("EventsWithoutBusAnswers500", func(t *testing.T) {
		// The nil-bus guard still answers its documented 500 once the
		// request is authenticated.
		noBus := NewRouter(&Deps{Pool: env.pool, Version: testVersion})
		cookie, _ := loginCookie(t, noBus, "ada@example.com", testPassword, "")

		rec := authedRequest(noBus, http.MethodGet, "/api/v1/events", cookie)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("GET /api/v1/events without a bus: status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
		apiErr, _ := decodeError(t, rec)
		if apiErr.Code != apigen.INTERNAL {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.INTERNAL)
		}
	})
}

// contains reports whether the fields slice holds name.
func contains(fields []string, name string) bool {
	for _, field := range fields {
		if field == name {
			return true
		}
	}
	return false
}

// isNoRows reports whether err is the pgx no-rows result.
func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

// mustUUIDv7 generates a UUIDv7 for test fixtures.
func mustUUIDv7(t *testing.T) uuid.UUID {
	t.Helper()
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("generate uuid v7: %v", err)
	}
	return id
}

// mustHash derives an Argon2id hash of a test password.
func mustHash(t *testing.T, password string) string {
	t.Helper()
	hash, err := auth.Hash(password)
	if err != nil {
		t.Fatalf("hash test password: %v", err)
	}
	return hash
}

// mustSessionIDFor resolves the session id behind a login cookie.
func mustSessionIDFor(t *testing.T, env *authEnv, cookie *http.Cookie) uuid.UUID {
	t.Helper()
	row, err := env.lookupSessionByToken(t, cookie.Value)
	if err != nil {
		t.Fatalf("lookup session of the login cookie: %v", err)
	}
	return uuid.UUID(row.ID.Bytes)
}
