package db

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

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
