package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/auth"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// wantPairingTokenTTL mirrors the handler constant in expiry assertions.
const wantPairingTokenTTL = 5 * time.Minute

// wantPairingDailyLimit mirrors the router's per-user generation budget.
const wantPairingDailyLimit = 10

// postPairingToken fires POST /auth/pairing/token; a nil cookie sends
// no credential.
func postPairingToken(router http.Handler, cookie *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/pairing/token", nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// generatePairingToken asserts a successful generation and returns the
// issued token response.
func generatePairingToken(t *testing.T, router http.Handler, cookie *http.Cookie) apigen.PairingTokenResponse {
	t.Helper()
	rec := postPairingToken(router, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /auth/pairing/token: status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp apigen.PairingTokenResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode pairing token response %q: %v", rec.Body.String(), err)
	}
	return resp
}

// consumeBody marshals a pairing consume payload.
func consumeBody(t *testing.T, token, deviceName, deviceID string) string {
	t.Helper()
	body, err := json.Marshal(apigen.PairingConsumeRequest{
		Token:      token,
		DeviceName: deviceName,
		DeviceId:   deviceID,
	})
	if err != nil {
		t.Fatalf("marshal consume body: %v", err)
	}
	return string(body)
}

// postConsumeRaw fires POST /auth/pairing/consume with an arbitrary body.
func postConsumeRaw(router http.Handler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/pairing/consume", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// consumePairing fires a consume request and returns the recorder.
func consumePairing(t *testing.T, router http.Handler, token, deviceName, deviceID string) *httptest.ResponseRecorder {
	t.Helper()
	return postConsumeRaw(router, consumeBody(t, token, deviceName, deviceID))
}

// pairingRow loads the pairing_tokens row behind a raw token; the
// expected hash is derived exactly the way the store derives it.
func (e *authEnv) pairingRow(t *testing.T, raw string) dbgen.PairingToken {
	t.Helper()
	var row dbgen.PairingToken
	err := e.pool.QueryRow(context.Background(),
		`SELECT id, token_hash, created_by, expires_at, used_at, created_at, updated_at
		 FROM pairing_tokens WHERE token_hash = $1`,
		auth.HashSessionToken(raw),
	).Scan(
		&row.ID, &row.TokenHash, &row.CreatedBy, &row.ExpiresAt,
		&row.UsedAt, &row.CreatedAt, &row.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("load pairing row of token %q: %v", raw, err)
	}
	return row
}

// countPairingTokens reports how many pairing token rows exist.
func (e *authEnv) countPairingTokens(t *testing.T) int {
	t.Helper()
	var count int
	if err := e.pool.QueryRow(context.Background(), "SELECT count(*) FROM pairing_tokens").Scan(&count); err != nil {
		t.Fatalf("count pairing tokens: %v", err)
	}
	return count
}

// TestCreatePairingToken walks the generation side: a signed-in user
// mints a five-minute one-time secret (stored only as its hash), the
// unauthenticated caller is refused, and the eleventh token of a day
// hits the per-user budget.
func TestCreatePairingToken(t *testing.T) {
	env := newAuthEnv(t, nil)
	env.createAdmin(t, "ada@example.com", testPassword)
	cookie, _ := loginCookie(t, env.router, "ada@example.com", testPassword, "Firefox on macOS")

	t.Run("IssuesTokenWithFiveMinuteTTL", func(t *testing.T) {
		before := time.Now()
		resp := generatePairingToken(t, env.router, cookie)
		after := time.Now()

		// Review focus: 16 random bytes, hex-encoded -- 32 lowercase
		// hex characters, decodable, no other shape.
		if len(resp.Token) != 32 {
			t.Errorf("token = %q, want 32 hex characters (16 bytes)", resp.Token)
		}
		if strings.ToLower(resp.Token) != resp.Token {
			t.Errorf("token = %q, want lowercase hex", resp.Token)
		}

		// Review focus: the expiry is five minutes out.
		if want := before.Add(wantPairingTokenTTL); resp.ExpiresAt.Before(want.Add(-time.Minute)) || resp.ExpiresAt.After(after.Add(wantPairingTokenTTL).Add(time.Minute)) {
			t.Errorf("expiresAt = %v, want ~%v (five minutes after issuance)", resp.ExpiresAt, want)
		}

		// Review focus: the store keeps only the hash.
		row := env.pairingRow(t, resp.Token)
		if row.TokenHash == resp.Token {
			t.Error("pairing_tokens.token_hash stores the raw token: the secret would be in the clear")
		}
		if row.TokenHash != auth.HashSessionToken(resp.Token) {
			t.Errorf("pairing_tokens.token_hash = %q, want the SHA-256 hex of the raw token", row.TokenHash)
		}
		if row.UsedAt.Valid {
			t.Error("fresh pairing token has used_at set, want NULL until consumed")
		}
		user, err := env.queries.GetUserByEmail(t.Context(), "ada@example.com")
		if err != nil {
			t.Fatalf("GetUserByEmail(ada): %v", err)
		}
		if row.CreatedBy != user.ID {
			t.Errorf("pairing token created_by = %v, want the requesting user %v", row.CreatedBy, user.ID)
		}
	})

	t.Run("RequiresSession", func(t *testing.T) {
		// Review focus: unauthenticated generation is refused before
		// any token work.
		rec := postPairingToken(env.router, nil)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated generation: status = %d, want %d (body %q)", rec.Code, http.StatusUnauthorized, rec.Body.String())
		}
		apiErr, _ := decodeError(t, rec)
		if apiErr.Code != apigen.AUTHINVALIDCREDENTIALS {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.AUTHINVALIDCREDENTIALS)
		}
		if got := env.countPairingTokens(t); got != 1 {
			t.Errorf("pairing token count after a refused generation = %d, want 1 (only the issued one)", got)
		}
	})
}

// TestPairingTokenDailyLimit pins the generation budget: ten tokens
// per user per day, the eleventh is a 429 with a Retry-After pointing
// at the daily reset, and another user's budget is untouched.
func TestPairingTokenDailyLimit(t *testing.T) {
	env := newAuthEnv(t, nil)
	env.createAdmin(t, "ada@example.com", testPassword)
	cookie, _ := loginCookie(t, env.router, "ada@example.com", testPassword, "")

	// Review focus: the first ten generations of the day succeed.
	for i := 0; i < wantPairingDailyLimit; i++ {
		rec := postPairingToken(env.router, cookie)
		if rec.Code != http.StatusOK {
			t.Fatalf("generation %d: status = %d, want %d (body %q)", i+1, rec.Code, http.StatusOK, rec.Body.String())
		}
	}

	// Review focus: the eleventh is refused with 429 + Retry-After.
	rec := postPairingToken(env.router, cookie)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("generation %d: status = %d, want %d (body %q)", wantPairingDailyLimit+1, rec.Code, http.StatusTooManyRequests, rec.Body.String())
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
	// The daily window resets at UTC midnight: at most the rest of a
	// day, always strictly positive.
	if seconds <= 0 || seconds > 86400 {
		t.Errorf("Retry-After = %d seconds, want (0, 86400] for a daily window", seconds)
	}
	if got := env.countPairingTokens(t); got != wantPairingDailyLimit {
		t.Errorf("pairing token count after the budget hit = %d, want %d", got, wantPairingDailyLimit)
	}

	// The budget is per user: another account still generates.
	_, err = env.queries.CreateUser(context.Background(), dbgen.CreateUserParams{
		ID:           pgUUID(mustUUIDv7(t)),
		Name:         "Grace Hopper",
		Email:        "grace@example.com",
		PasswordHash: mustHash(t, testPassword),
	})
	if err != nil {
		t.Fatalf("seed grace: %v", err)
	}
	graceCookie, _ := loginCookie(t, env.router, "grace@example.com", testPassword, "")
	if rec := postPairingToken(env.router, graceCookie); rec.Code != http.StatusOK {
		t.Errorf("another user's generation: status = %d, want %d (per-user budget)", rec.Code, http.StatusOK)
	}
}

// TestConsumePairingToken walks the exchange: the QR token becomes a
// bearer session of the token's creator, the one-time guarantee
// refuses the replay, expiry is enforced, and payloads are validated
// before anything is burned.
func TestConsumePairingToken(t *testing.T) {
	env := newAuthEnv(t, nil)
	env.createAdmin(t, "ada@example.com", testPassword)
	cookie, _ := loginCookie(t, env.router, "ada@example.com", testPassword, "Firefox on macOS")

	t.Run("ExchangesForBearerSession", func(t *testing.T) {
		pairing := generatePairingToken(t, env.router, cookie)

		rec := consumePairing(t, env.router, pairing.Token, "Pixel 9", "0192f001-6a1b-7cde-8f9a-0123456789ab")
		if rec.Code != http.StatusOK {
			t.Fatalf("consume: status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
		var bearer apigen.BearerSession
		if err := json.Unmarshal(rec.Body.Bytes(), &bearer); err != nil {
			t.Fatalf("decode bearer session %q: %v", rec.Body.String(), err)
		}

		// The response is the only place the credential exists in the
		// clear; the store keeps its hash.
		if bearer.Token == "" {
			t.Fatal("bearer token is empty")
		}
		row, err := env.lookupSessionByToken(t, bearer.Token)
		if err != nil {
			t.Fatalf("lookup by the bearer credential: %v", err)
		}
		if row.TokenHash == bearer.Token {
			t.Error("sessions.token_hash stores the bearer credential in the clear")
		}
		if row.TokenHash != auth.HashSessionToken(bearer.Token) {
			t.Errorf("sessions.token_hash = %q, want the SHA-256 hex of the credential", row.TokenHash)
		}

		// The session belongs to the token's creator, not to whoever
		// walked by with the QR code.
		user, err := env.queries.GetUserByEmail(t.Context(), "ada@example.com")
		if err != nil {
			t.Fatalf("GetUserByEmail(ada): %v", err)
		}
		if row.UserID != user.ID {
			t.Errorf("session user_id = %v, want the token creator %v", row.UserID, user.ID)
		}
		if row.Kind != dbgen.SessionKindMobile {
			t.Errorf("session kind = %q, want mobile", row.Kind)
		}
		if row.DeviceName != "Pixel 9" {
			t.Errorf("session device_name = %q, want the pairing payload's deviceName", row.DeviceName)
		}
		if !row.DeviceID.Valid || row.DeviceID.String != "0192f001-6a1b-7cde-8f9a-0123456789ab" {
			t.Errorf("session device_id = %v, want the pairing payload's deviceId", row.DeviceID)
		}

		// The API shape: mobile kind, no expiry (bearer sessions are
		// revoked, not expired), the device label rides along.
		if row.ID != pgUUID(bearer.Session.Id) {
			t.Errorf("bearer session id = %v, want the created row's id", bearer.Session.Id)
		}
		if bearer.Session.Kind != apigen.Mobile {
			t.Errorf("bearer session kind = %q, want mobile", bearer.Session.Kind)
		}
		if bearer.Session.DeviceName == nil || *bearer.Session.DeviceName != "Pixel 9" {
			t.Errorf("bearer session deviceName = %v, want Pixel 9", bearer.Session.DeviceName)
		}
		if bearer.Session.ExpiresAt != nil {
			t.Errorf("bearer session expiresAt = %v, want null (bearer sessions do not expire)", bearer.Session.ExpiresAt)
		}

		// Review focus: the bearer works on an authed endpoint.
		authed := bearerRequest(env.router, http.MethodGet, "/api/v1/auth/sessions", bearer.Token)
		if authed.Code != http.StatusOK {
			t.Fatalf("authed request with the new bearer: status = %d, want %d (body %q)", authed.Code, http.StatusOK, authed.Body.String())
		}
		var sessions []apigen.SessionInfo
		if err := json.Unmarshal(authed.Body.Bytes(), &sessions); err != nil {
			t.Fatalf("decode sessions %q: %v", authed.Body.String(), err)
		}
		found := false
		for _, session := range sessions {
			if session.Id == bearer.Session.Id {
				found = true
				if session.Kind != apigen.Mobile {
					t.Errorf("listed session kind = %q, want mobile", session.Kind)
				}
			}
		}
		if !found {
			t.Error("the paired mobile session is missing from the session list")
		}

		// The token is burned: used_at is set, once.
		pairingRow := env.pairingRow(t, pairing.Token)
		if !pairingRow.UsedAt.Valid {
			t.Error("consumed pairing token has used_at NULL, want the burn stamp")
		}

		// The pairing reached the activity stream.
		event := awaitSessionEvent(t, env.bus, "session.started", bearer.Session.Id.String())
		if got := eventPayload(t, event)["kind"]; got != "mobile" {
			t.Errorf("session.started payload kind = %q, want mobile", got)
		}
		if payload := eventPayload(t, event); payload["id"] == bearer.Token {
			t.Error("event payload leaks the raw bearer credential")
		}
	})

	t.Run("SecondConsumeRejected", func(t *testing.T) {
		// Review focus: the same token twice -- the first exchange
		// succeeds, the replay answers 401 PAIRING_TOKEN_INVALID.
		pairing := generatePairingToken(t, env.router, cookie)
		creator := pairingCreatedBy(t, env, cookie)
		before := countMobileSessions(t, env, creator)

		first := consumePairing(t, env.router, pairing.Token, "Pixel 9", "device-1")
		if first.Code != http.StatusOK {
			t.Fatalf("first consume: status = %d, want %d (body %q)", first.Code, http.StatusOK, first.Body.String())
		}
		second := consumePairing(t, env.router, pairing.Token, "Pixel 9", "device-1")
		if second.Code != http.StatusUnauthorized {
			t.Fatalf("second consume: status = %d, want %d (body %q)", second.Code, http.StatusUnauthorized, second.Body.String())
		}
		apiErr, _ := decodeError(t, second)
		if apiErr.Code != apigen.PAIRINGTOKENINVALID {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.PAIRINGTOKENINVALID)
		}

		// The replay created no second session: exactly the one the
		// first exchange minted.
		if got, want := countMobileSessions(t, env, creator), before+1; got != want {
			t.Errorf("mobile session count after a replayed consume = %d, want %d", got, want)
		}
	})

	t.Run("ExpiredTokenRejected", func(t *testing.T) {
		// Review focus: a token whose five minutes elapsed (the row is
		// backdated in the store) is refused exactly like an unknown
		// one.
		pairing := generatePairingToken(t, env.router, cookie)
		env.exec(t, "UPDATE pairing_tokens SET expires_at = now() - interval '1 minute' WHERE token_hash = $1",
			auth.HashSessionToken(pairing.Token))

		rec := consumePairing(t, env.router, pairing.Token, "Pixel 9", "device-2")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("consume of an expired token: status = %d, want %d (body %q)", rec.Code, http.StatusUnauthorized, rec.Body.String())
		}
		apiErr, _ := decodeError(t, rec)
		if apiErr.Code != apigen.PAIRINGTOKENINVALID {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.PAIRINGTOKENINVALID)
		}
		// The expired row was not burned: used_at stays NULL (not that
		// anything could redeem it now).
		if row := env.pairingRow(t, pairing.Token); row.UsedAt.Valid {
			t.Error("expired consume burned the token (used_at set), want no write")
		}
	})

	t.Run("UnknownTokenRejected", func(t *testing.T) {
		rec := consumePairing(t, env.router, "deadbeefdeadbeefdeadbeefdeadbeef", "Pixel 9", "device-3")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("consume of an unknown token: status = %d, want %d (body %q)", rec.Code, http.StatusUnauthorized, rec.Body.String())
		}
		apiErr, _ := decodeError(t, rec)
		if apiErr.Code != apigen.PAIRINGTOKENINVALID {
			t.Errorf("code = %q, want %q", apiErr.Code, apigen.PAIRINGTOKENINVALID)
		}
	})

	t.Run("RejectedPayloadDoesNotBurnToken", func(t *testing.T) {
		// Validation happens before the transaction: a payload the
		// endpoint refuses must leave the token redeemable.
		pairing := generatePairingToken(t, env.router, cookie)

		rec := postConsumeRaw(env.router, `{"token":"`+pairing.Token+`","deviceName":"","deviceId":""}`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("consume with an invalid payload: status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if row := env.pairingRow(t, pairing.Token); row.UsedAt.Valid {
			t.Fatal("refused payload burned the token, want it still redeemable")
		}
		if rec := consumePairing(t, env.router, pairing.Token, "Pixel 9", "device-4"); rec.Code != http.StatusOK {
			t.Fatalf("consume after a refused payload: status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
		}
	})

	t.Run("RejectsInvalidPayloads", func(t *testing.T) {
		cases := []struct {
			name   string
			body   string
			fields []string
		}{
			{name: "BodyNotJSON", body: `not json at all`, fields: nil},
			{name: "TokenMissing", body: `{"deviceName":"Pixel 9","deviceId":"d"}`, fields: []string{"token"}},
			{name: "DeviceNameMissing", body: `{"token":"abc","deviceId":"d"}`, fields: []string{"deviceName"}},
			{name: "DeviceIdMissing", body: `{"token":"abc","deviceName":"Pixel 9"}`, fields: []string{"deviceId"}},
			{
				name:   "BodyTooLarge",
				body:   `{"token":"` + strings.Repeat("a", 1<<20) + `","deviceName":"Pixel 9","deviceId":"d"}`,
				fields: []string{"body"},
			},
		}
		for _, testCase := range cases {
			t.Run(testCase.name, func(t *testing.T) {
				rec := postConsumeRaw(env.router, testCase.body)

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
}

// pairingCreatedBy resolves the user id behind a login cookie.
func pairingCreatedBy(t *testing.T, env *authEnv, cookie *http.Cookie) pgtype.UUID {
	t.Helper()
	row, err := env.lookupSessionByToken(t, cookie.Value)
	if err != nil {
		t.Fatalf("lookup session of the login cookie: %v", err)
	}
	return row.UserID
}

// countMobileSessions reports how many mobile sessions the user owns.
func countMobileSessions(t *testing.T, env *authEnv, userID pgtype.UUID) int {
	t.Helper()
	sessions, err := env.queries.ListUserSessions(t.Context(), userID)
	if err != nil {
		t.Fatalf("ListUserSessions: %v", err)
	}
	count := 0
	for _, session := range sessions {
		if session.Kind == dbgen.SessionKindMobile {
			count++
		}
	}
	return count
}
