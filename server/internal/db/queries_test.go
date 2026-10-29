package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/welpher/lumemor/server/internal/auth"
	"github.com/welpher/lumemor/server/internal/db/dbtest"
	"github.com/welpher/lumemor/server/internal/db/gen"
)

// TestUserQueries exercises the sqlc-generated user queries against a fresh
// throwaway database: the first-run wizard's CreateUser seeds the initial
// admin, GetUserByEmail reads it back (and misses unknown emails), and
// CountAdmins reports exactly one admin.
func TestUserQueries(t *testing.T) {
	pool := dbtest.New(t)
	ctx := t.Context()

	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate() error = %v, want nil", err)
	}

	queries := gen.New(pool)

	// Seed the instance the way the first-run wizard will: a UUIDv7 id and
	// an argon2id placeholder hash the query layer never inspects. CreateUser
	// hardcodes role='admin' and locale='en'.
	userID, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("generate uuid v7: %v", err)
	}
	created, err := queries.CreateUser(ctx, gen.CreateUserParams{
		ID:           pgtype.UUID{Bytes: userID, Valid: true},
		Name:         "Ada Lovelace",
		Email:        "ada@example.com",
		PasswordHash: "$argon2id$v=19$m=65536,t=3,p=4$placeholder$placeholder",
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v, want nil", err)
	}
	if created.Role != gen.UserRoleAdmin {
		t.Errorf("CreateUser() role = %q, want %q", created.Role, gen.UserRoleAdmin)
	}
	if created.Locale != "en" {
		t.Errorf("CreateUser() locale = %q, want %q", created.Locale, "en")
	}

	t.Run("GetUserByEmail", func(t *testing.T) {
		got, err := queries.GetUserByEmail(ctx, "ada@example.com")
		if err != nil {
			t.Fatalf("GetUserByEmail() error = %v, want nil", err)
		}
		wantID := pgtype.UUID{Bytes: userID, Valid: true}
		if got.ID != wantID {
			t.Errorf("GetUserByEmail() id = %+v, want %+v", got.ID, wantID)
		}
		if got.Name != "Ada Lovelace" {
			t.Errorf("GetUserByEmail() name = %q, want %q", got.Name, "Ada Lovelace")
		}
		if got.Email != "ada@example.com" {
			t.Errorf("GetUserByEmail() email = %q, want %q", got.Email, "ada@example.com")
		}
		if got.Role != gen.UserRoleAdmin {
			t.Errorf("GetUserByEmail() role = %q, want %q", got.Role, gen.UserRoleAdmin)
		}

		if _, err := queries.GetUserByEmail(ctx, "nobody@example.com"); !errors.Is(err, pgx.ErrNoRows) {
			t.Errorf("GetUserByEmail() unknown email error = %v, want pgx.ErrNoRows", err)
		}
	})

	t.Run("CountAdmins", func(t *testing.T) {
		admins, err := queries.CountAdmins(ctx)
		if err != nil {
			t.Fatalf("CountAdmins() error = %v, want nil", err)
		}
		if admins != 1 {
			t.Errorf("CountAdmins() = %d, want 1", admins)
		}
	})
}

// ts is a helper building a pg timestamptz from a time.
func ts(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// pgUUID wraps a uuid for the generated query parameters.
func pgUUID(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: u, Valid: true}
}

// newSessionFixture inserts a session row with a freshly minted
// credential and returns (id, rawToken, hash).
func newSessionFixture(t *testing.T, ctx context.Context, queries *gen.Queries, userID pgtype.UUID, kind gen.SessionKind, lastSeen, expires time.Time) (uuid.UUID, string, string) {
	t.Helper()
	raw, hash := auth.NewSessionToken()
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("generate session uuid: %v", err)
	}
	_, err = queries.CreateSession(ctx, gen.CreateSessionParams{
		ID:         pgUUID(id),
		UserID:     userID,
		TokenHash:  hash,
		Kind:       kind,
		DeviceName: "test fixture",
		LastSeenAt: ts(lastSeen),
		ExpiresAt:  ts(expires),
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v, want nil", err)
	}
	return id, raw, hash
}

// TestSessionQueries exercises the sqlc-generated session queries
// against a fresh throwaway database: creation, token-hash lookup with
// the joined user columns, the kind-aware expiry rule, the throttled
// touch that slides web expiry, owner-scoped revocation, the
// revoke-all hook, and the session list.
func TestSessionQueries(t *testing.T) {
	pool := dbtest.New(t)
	ctx := t.Context()

	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate() error = %v, want nil", err)
	}

	queries := gen.New(pool)

	// Two users: the sessions under test belong to ada, grace's row
	// proves the owner scoping.
	ada, err := queries.CreateUser(ctx, gen.CreateUserParams{
		ID:           pgUUID(mustUUIDv7(t)),
		Name:         "Ada Lovelace",
		Email:        "ada@example.com",
		PasswordHash: "$argon2id$v=19$m=65536,t=3,p=1$placeholder$placeholder",
	})
	if err != nil {
		t.Fatalf("create ada: %v", err)
	}
	grace, err := queries.CreateUser(ctx, gen.CreateUserParams{
		ID:           pgUUID(mustUUIDv7(t)),
		Name:         "Grace Hopper",
		Email:        "grace@example.com",
		PasswordHash: "$argon2id$v=19$m=65536,t=3,p=1$placeholder$placeholder",
	})
	if err != nil {
		t.Fatalf("create grace: %v", err)
	}

	// All expiry comparisons happen against the database clock inside
	// the queries, so fixtures are anchored to the real now with
	// margins far larger than any clock skew.
	now := time.Now()

	t.Run("CreateAndLookupByTokenHash", func(t *testing.T) {
		id, raw, hash := newSessionFixture(t, ctx, queries, ada.ID, gen.SessionKindWeb, now, now.Add(30*24*time.Hour))

		if hash == raw {
			t.Fatal("stored hash equals the raw token: the credential would be in the clear")
		}
		row, err := queries.GetSessionByTokenHash(ctx, hash)
		if err != nil {
			t.Fatalf("GetSessionByTokenHash() error = %v, want nil", err)
		}
		if row.ID != pgUUID(id) {
			t.Errorf("GetSessionByTokenHash() id = %v, want %v", row.ID, id)
		}
		if row.TokenHash != hash {
			t.Errorf("GetSessionByTokenHash() token_hash = %q, want %q", row.TokenHash, hash)
		}
		if row.Kind != gen.SessionKindWeb {
			t.Errorf("GetSessionByTokenHash() kind = %q, want web", row.Kind)
		}
		if row.UserEmail != "ada@example.com" {
			t.Errorf("GetSessionByTokenHash() user_email = %q, want ada@example.com", row.UserEmail)
		}
		if row.UserRole != gen.UserRoleAdmin {
			t.Errorf("GetSessionByTokenHash() user_role = %q, want admin", row.UserRole)
		}
		if row.UserIsDisabled {
			t.Error("GetSessionByTokenHash() user_is_disabled = true, want false")
		}

		// The presented credential, not the stored hash, must also hit.
		if row2, err := queries.GetSessionByTokenHash(ctx, auth.HashSessionToken(raw)); err != nil {
			t.Fatalf("GetSessionByTokenHash(hash of raw) error = %v, want nil", err)
		} else if row2.ID != pgUUID(id) {
			t.Errorf("GetSessionByTokenHash(hash of raw) id = %v, want %v", row2.ID, id)
		}

		if _, err := queries.GetSessionByTokenHash(ctx, auth.HashSessionToken("no such token")); !errors.Is(err, pgx.ErrNoRows) {
			t.Errorf("GetSessionByTokenHash(unknown) error = %v, want pgx.ErrNoRows", err)
		}
	})

	t.Run("ExpiredWebSessionIsNotServed", func(t *testing.T) {
		_, _, hash := newSessionFixture(t, ctx, queries, ada.ID, gen.SessionKindWeb, now.Add(-time.Hour), now.Add(-time.Minute))

		if _, err := queries.GetSessionByTokenHash(ctx, hash); !errors.Is(err, pgx.ErrNoRows) {
			t.Errorf("GetSessionByTokenHash(expired web) error = %v, want pgx.ErrNoRows", err)
		}
	})

	t.Run("MobileSessionIgnoresExpiry", func(t *testing.T) {
		// Bearer sessions are revoked, not expired: even a row whose
		// expires_at sentinel lies in the past authenticates.
		_, _, hash := newSessionFixture(t, ctx, queries, ada.ID, gen.SessionKindMobile, now.Add(-48*time.Hour), now.Add(-time.Hour))

		if _, err := queries.GetSessionByTokenHash(ctx, hash); !errors.Is(err, nil) {
			t.Errorf("GetSessionByTokenHash(mobile, past expiry) error = %v, want nil", err)
		}
	})

	t.Run("DisabledUserSessionIsStillServedByQuery", func(t *testing.T) {
		// The middleware, not the lookup, rejects disabled users: the
		// join column exists so it can.
		_, _, hash := newSessionFixture(t, ctx, queries, ada.ID, gen.SessionKindWeb, now, now.Add(time.Hour))
		if _, err := pool.Exec(ctx, "UPDATE users SET is_disabled = true WHERE id = $1", ada.ID); err != nil {
			t.Fatalf("disable user: %v", err)
		}
		row, err := queries.GetSessionByTokenHash(ctx, hash)
		if err != nil {
			t.Fatalf("GetSessionByTokenHash() error = %v, want nil", err)
		}
		if !row.UserIsDisabled {
			t.Error("user_is_disabled = false after the update, want true")
		}
	})

	t.Run("TouchSlidesWebExpiryOnly", func(t *testing.T) {
		webID, _, webHash := newSessionFixture(t, ctx, queries, ada.ID, gen.SessionKindWeb, now.Add(-2*time.Minute), now.Add(29*24*time.Hour))
		mobileID, _, _ := newSessionFixture(t, ctx, queries, ada.ID, gen.SessionKindMobile, now.Add(-2*time.Minute), now.Add(100*24*time.Hour))

		touched := now
		if err := queries.TouchSession(ctx, gen.TouchSessionParams{
			ID:         pgUUID(webID),
			LastSeenAt: ts(touched),
			ExpiresAt:  ts(touched.Add(30 * 24 * time.Hour)),
		}); err != nil {
			t.Fatalf("TouchSession() error = %v, want nil", err)
		}
		web, err := queries.GetSessionByTokenHash(ctx, webHash)
		if err != nil {
			t.Fatalf("GetSessionByTokenHash() after touch: %v", err)
		}
		if !web.LastSeenAt.Time.Equal(touched) {
			t.Errorf("last_seen_at = %v, want %v", web.LastSeenAt.Time, touched)
		}
		if want := touched.Add(30 * 24 * time.Hour); !web.ExpiresAt.Time.Equal(want) {
			t.Errorf("expires_at = %v, want the slid %v", web.ExpiresAt.Time, want)
		}

		// The mobile row keeps its expiry: bearer sessions are revoked,
		// not slid.
		if err := queries.TouchSession(ctx, gen.TouchSessionParams{
			ID:         pgUUID(mobileID),
			LastSeenAt: ts(touched),
			ExpiresAt:  ts(touched.Add(30 * 24 * time.Hour)),
		}); err != nil {
			t.Fatalf("TouchSession(mobile) error = %v, want nil", err)
		}
		mobile, err := queries.GetSession(ctx, pgUUID(mobileID))
		if err != nil {
			t.Fatalf("GetSession() error = %v, want nil", err)
		}
		if want := now.Add(100 * 24 * time.Hour); !mobile.ExpiresAt.Time.Equal(want) {
			t.Errorf("mobile expires_at = %v after touch, want the untouched %v", mobile.ExpiresAt.Time, want)
		}
	})

	t.Run("ListUserSessions", func(t *testing.T) {
		// One fresh row per user plus a stale row for ada; the list
		// shows only the owner's active sessions, most recently used
		// first. The fresh row is stamped now -- strictly newer than
		// every fixture above -- so the ordering assertion cannot hit
		// a timestamp tie.
		freshNow := time.Now()
		freshID, _, _ := newSessionFixture(t, ctx, queries, ada.ID, gen.SessionKindWeb, freshNow, freshNow.Add(30*24*time.Hour))
		newSessionFixture(t, ctx, queries, ada.ID, gen.SessionKindWeb, now.Add(-24*time.Hour), now.Add(-time.Hour)) // expired
		_, _, _ = newSessionFixture(t, ctx, queries, grace.ID, gen.SessionKindWeb, now, now.Add(30*24*time.Hour))

		// ada also still owns the earlier subtests' rows; the
		// freshest is the one just inserted.
		rows, err := queries.ListUserSessions(ctx, ada.ID)
		if err != nil {
			t.Fatalf("ListUserSessions() error = %v, want nil", err)
		}
		if len(rows) == 0 {
			t.Fatal("ListUserSessions() = no rows, want ada's sessions")
		}
		if rows[0].ID != pgUUID(freshID) {
			t.Errorf("ListUserSessions()[0].id = %v, want the most recently used session %v", rows[0].ID, freshID)
		}
		for _, row := range rows {
			if row.UserID != ada.ID {
				t.Errorf("ListUserSessions() returned session of user %v, want only ada's", row.UserID)
			}
			if row.Kind == gen.SessionKindWeb && !row.ExpiresAt.Time.After(now.Add(-2*time.Minute)) {
				t.Errorf("ListUserSessions() returned expired web session %v", row.ID)
			}
		}
	})

	t.Run("RevokeSessionIsOwnerScoped", func(t *testing.T) {
		id, _, hash := newSessionFixture(t, ctx, queries, ada.ID, gen.SessionKindWeb, now, now.Add(time.Hour))

		// A foreign user's revoke is a no-op.
		kind, err := queries.RevokeSession(ctx, gen.RevokeSessionParams{ID: pgUUID(id), UserID: grace.ID})
		if !errors.Is(err, pgx.ErrNoRows) {
			t.Fatalf("RevokeSession(foreign) = (%q, %v), want a no-row result", kind, err)
		}
		if _, err := queries.GetSessionByTokenHash(ctx, hash); err != nil {
			t.Fatalf("session vanished after a foreign revoke: %v", err)
		}

		// The owner's revoke deletes the row and reports its kind.
		kind, err = queries.RevokeSession(ctx, gen.RevokeSessionParams{ID: pgUUID(id), UserID: ada.ID})
		if err != nil {
			t.Fatalf("RevokeSession() error = %v, want nil", err)
		}
		if kind != gen.SessionKindWeb {
			t.Errorf("RevokeSession() kind = %q, want the revoked session's kind web", kind)
		}
		if _, err := queries.GetSessionByTokenHash(ctx, hash); !errors.Is(err, pgx.ErrNoRows) {
			t.Errorf("GetSessionByTokenHash() after revoke error = %v, want pgx.ErrNoRows", err)
		}

		// Unknown and already-revoked ids stay silent no-ops.
		kind, err = queries.RevokeSession(ctx, gen.RevokeSessionParams{ID: pgUUID(mustUUIDv7(t)), UserID: ada.ID})
		if kind != "" || !errors.Is(err, pgx.ErrNoRows) {
			t.Errorf("RevokeSession(unknown) = (%q, %v), want an empty no-row result", kind, err)
		}
	})

	t.Run("RevokeAllUserSessions", func(t *testing.T) {
		// The password-change hook: every session of the user goes,
		// other users keep theirs.
		newSessionFixture(t, ctx, queries, ada.ID, gen.SessionKindWeb, now, now.Add(time.Hour))
		newSessionFixture(t, ctx, queries, ada.ID, gen.SessionKindMobile, now, now.Add(100*24*time.Hour))
		graceID, _, graceHash := newSessionFixture(t, ctx, queries, grace.ID, gen.SessionKindWeb, now, now.Add(time.Hour))

		affected, err := queries.RevokeAllUserSessions(ctx, ada.ID)
		if err != nil {
			t.Fatalf("RevokeAllUserSessions() error = %v, want nil", err)
		}
		if affected < 2 {
			t.Fatalf("RevokeAllUserSessions() affected %d rows, want at least 2", affected)
		}
		if rows, err := queries.ListUserSessions(ctx, ada.ID); err != nil || len(rows) != 0 {
			t.Errorf("ListUserSessions(ada) after revoke-all = (%d rows, %v), want (0, nil)", len(rows), err)
		}
		if _, err := queries.GetSessionByTokenHash(ctx, graceHash); err != nil {
			t.Errorf("grace's session %v vanished after ada's revoke-all", graceID)
		}
	})
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

// newPairingFixture inserts a pairing token row with a freshly minted
// credential and returns (rawToken, hash).
func newPairingFixture(t *testing.T, ctx context.Context, queries *gen.Queries, userID pgtype.UUID, expires time.Time) (string, string) {
	t.Helper()
	raw, hash := auth.NewPairingToken()
	_, err := queries.CreatePairingToken(ctx, gen.CreatePairingTokenParams{
		ID:        pgUUID(mustUUIDv7(t)),
		TokenHash: hash,
		CreatedBy: userID,
		ExpiresAt: ts(expires),
	})
	if err != nil {
		t.Fatalf("CreatePairingToken() error = %v, want nil", err)
	}
	return raw, hash
}

// TestPairingQueries exercises the sqlc-generated pairing queries: the
// row stores only the token's hash, and the conditional consume burn
// answers exactly once -- used, expired and unknown hashes all miss.
func TestPairingQueries(t *testing.T) {
	pool := dbtest.New(t)
	ctx := t.Context()

	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate() error = %v, want nil", err)
	}

	queries := gen.New(pool)

	ada, err := queries.CreateUser(ctx, gen.CreateUserParams{
		ID:           pgUUID(mustUUIDv7(t)),
		Name:         "Ada Lovelace",
		Email:        "ada@example.com",
		PasswordHash: "$argon2id$v=19$m=65536,t=3,p=1$placeholder$placeholder",
	})
	if err != nil {
		t.Fatalf("create ada: %v", err)
	}

	now := time.Now()

	t.Run("CreateStoresHashAndCreator", func(t *testing.T) {
		raw, hash := newPairingFixture(t, ctx, queries, ada.ID, now.Add(5*time.Minute))

		if hash == raw {
			t.Fatal("stored hash equals the raw token: the secret would be in the clear")
		}
		if hash != auth.HashPairingToken(raw) {
			t.Errorf("stored hash = %q, want HashPairingToken(raw) = %q", hash, auth.HashPairingToken(raw))
		}
		// The row keeps the five-minute expiry and starts unused.
		var (
			expiresAt pgtype.Timestamptz
			usedAt    pgtype.Timestamptz
			createdBy pgtype.UUID
		)
		if err := pool.QueryRow(ctx,
			"SELECT expires_at, used_at, created_by FROM pairing_tokens WHERE token_hash = $1", hash,
		).Scan(&expiresAt, &usedAt, &createdBy); err != nil {
			t.Fatalf("load pairing row: %v", err)
		}
		if usedAt.Valid {
			t.Error("fresh pairing row has used_at set, want NULL")
		}
		if createdBy != ada.ID {
			t.Errorf("pairing row created_by = %v, want %v", createdBy, ada.ID)
		}
		if want := now.Add(5 * time.Minute); expiresAt.Time.Before(want.Add(-time.Minute)) || expiresAt.Time.After(want.Add(time.Minute)) {
			t.Errorf("pairing row expires_at = %v, want ~%v", expiresAt.Time, want)
		}
	})

	t.Run("ConsumeBurnsExactlyOnce", func(t *testing.T) {
		_, hash := newPairingFixture(t, ctx, queries, ada.ID, now.Add(5*time.Minute))

		// The first consumer wins the burn and learns whose token it
		// was.
		consumed, err := queries.ConsumePairingToken(ctx, hash)
		if err != nil {
			t.Fatalf("ConsumePairingToken() error = %v, want nil", err)
		}
		if consumed.CreatedBy != ada.ID {
			t.Errorf("ConsumePairingToken() created_by = %v, want %v", consumed.CreatedBy, ada.ID)
		}

		// used_at is stamped: the burn is visible in the row.
		var usedAt pgtype.Timestamptz
		if err := pool.QueryRow(ctx, "SELECT used_at FROM pairing_tokens WHERE token_hash = $1", hash).Scan(&usedAt); err != nil {
			t.Fatalf("load burned row: %v", err)
		}
		if !usedAt.Valid {
			t.Error("burned pairing row has used_at NULL, want the stamp")
		}

		// The replay finds no row -- one-timeness is the conditional
		// UPDATE itself.
		if _, err := queries.ConsumePairingToken(ctx, hash); !errors.Is(err, pgx.ErrNoRows) {
			t.Errorf("ConsumePairingToken(replay) error = %v, want pgx.ErrNoRows", err)
		}
	})

	t.Run("ConsumeRejectsExpired", func(t *testing.T) {
		_, hash := newPairingFixture(t, ctx, queries, ada.ID, now.Add(-time.Minute))

		if _, err := queries.ConsumePairingToken(ctx, hash); !errors.Is(err, pgx.ErrNoRows) {
			t.Errorf("ConsumePairingToken(expired) error = %v, want pgx.ErrNoRows", err)
		}
		// The miss did not burn the row.
		var usedAt pgtype.Timestamptz
		if err := pool.QueryRow(ctx, "SELECT used_at FROM pairing_tokens WHERE token_hash = $1", hash).Scan(&usedAt); err != nil {
			t.Fatalf("load expired row: %v", err)
		}
		if usedAt.Valid {
			t.Error("expired consume stamped used_at, want no write")
		}
	})

	t.Run("ConsumeRejectsUnknown", func(t *testing.T) {
		if _, err := queries.ConsumePairingToken(ctx, auth.HashPairingToken("no such token")); !errors.Is(err, pgx.ErrNoRows) {
			t.Errorf("ConsumePairingToken(unknown) error = %v, want pgx.ErrNoRows", err)
		}
	})
}
