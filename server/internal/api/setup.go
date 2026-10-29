package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	oapitypes "github.com/oapi-codegen/runtime/types"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/auth"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

const (
	// minPasswordRunes and maxPasswordRunes bound wizard passwords.
	minPasswordRunes = 8
	maxPasswordRunes = 128

	// maxNameRunes bounds the optional display name after trimming.
	maxNameRunes = 100

	// maxAdminBodyBytes caps the wizard request body: the endpoint is
	// unauthenticated, so unbounded bodies must never be buffered.
	maxAdminBodyBytes = 1 << 20

	// uniqueViolation is the PostgreSQL error code for a unique
	// constraint violation (23505).
	uniqueViolation = "23505"

	// firstRunSetupLockKey is the advisory lock key serializing first-run
	// setup across every connection and process, making the admin check
	// and the insert atomic. 0x6C756D65 spells "lume" in ASCII; the value
	// is arbitrary but fixed, and any other advisory-lock user in this
	// codebase must pick a different key.
	firstRunSetupLockKey = 0x6C756D65
)

// CreateAdmin implements POST /api/v1/server/setup/admin: the one-time
// first-run wizard that seeds the initial administrator account.
func (s *server) CreateAdmin(w http.ResponseWriter, r *http.Request) {
	// Cap the request body before decoding anything: unbounded payloads
	// on this unauthenticated endpoint would otherwise be fully buffered.
	r.Body = http.MaxBytesReader(w, r.Body, maxAdminBodyBytes)

	var req apigen.CreateAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// The generated Email type validates during unmarshalling, and
		// the capped reader reports its own limit violation; both are
		// named so the 400 still points at the offending input.
		message := "request body must be valid JSON"
		var fields []interface{}
		var maxBytesErr *http.MaxBytesError
		switch {
		case errors.As(err, &maxBytesErr):
			message = "request body exceeds the size limit"
			fields = []interface{}{"body"}
		case errors.Is(err, oapitypes.ErrValidationEmail):
			message = "email must be a valid address"
			fields = []interface{}{"email"}
		}
		var details *map[string]interface{}
		if fields != nil {
			details = &map[string]interface{}{"fields": fields}
		}
		s.writeError(w, http.StatusBadRequest, apigen.VALIDATIONFAILED, message, details)
		return
	}

	in, invalid := normalizeCreateAdmin(req)
	if len(invalid) > 0 {
		fields := make([]interface{}, len(invalid))
		for i, field := range invalid {
			fields[i] = field
		}
		details := map[string]interface{}{"fields": fields}
		s.writeError(w, http.StatusBadRequest, apigen.VALIDATIONFAILED,
			"invalid account payload", &details)
		return
	}

	// The one-time guarantee needs the admin check and the insert to be
	// one atomic step: count-then-insert alone lets two concurrent
	// wizards with distinct emails both observe an admin-less instance
	// and both insert. A transaction-scoped advisory lock serializes
	// first-run setup, and the Argon2id derivation runs inside the lock
	// so the serialization cannot be outrun by slow hashing.
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		s.internalError(w, r, "begin setup transaction", err)
		return
	}
	// The rollback is a no-op after a successful commit; arguments are
	// captured while the request context is still live.
	defer func() { _ = tx.Rollback(r.Context()) }()

	if _, err := tx.Exec(r.Context(), "SELECT pg_advisory_xact_lock($1)", firstRunSetupLockKey); err != nil {
		s.internalError(w, r, "acquire setup lock", err)
		return
	}

	queries := dbgen.New(tx)
	admins, err := queries.CountAdmins(r.Context())
	if err != nil {
		s.internalError(w, r, "count admins", err)
		return
	}
	if admins > 0 {
		s.writeError(w, http.StatusConflict, apigen.SETUPALREADYCOMPLETED,
			"an administrator account already exists", nil)
		return
	}

	hash, err := auth.Hash(in.password)
	if err != nil {
		s.internalError(w, r, "hash password", err)
		return
	}
	id, err := uuid.NewV7()
	if err != nil {
		s.internalError(w, r, "generate user id", err)
		return
	}

	user, err := queries.CreateUser(r.Context(), dbgen.CreateUserParams{
		ID:           pgtype.UUID{Bytes: id, Valid: true},
		Name:         in.name,
		Email:        in.email,
		PasswordHash: hash,
	})
	if err != nil {
		// Belt and suspenders for writers that bypass the setup lock
		// (future admin management paths): only a violation of the
		// account's own unique keys means setup already completed, any
		// other constraint stays an internal error.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation &&
			(pgErr.ConstraintName == "users_email_key" || pgErr.ConstraintName == "users_pkey") {
			s.writeError(w, http.StatusConflict, apigen.SETUPALREADYCOMPLETED,
				"an administrator account already exists", nil)
			return
		}
		s.internalError(w, r, "create admin user", err)
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		s.internalError(w, r, "commit setup transaction", err)
		return
	}

	s.writeJSON(w, http.StatusCreated, apigen.User{
		Id:        uuid.UUID(user.ID.Bytes),
		Email:     oapitypes.Email(user.Email),
		Name:      user.Name,
		Role:      apigen.UserRole(user.Role),
		CreatedAt: user.CreatedAt.Time.UTC(),
	})
}

// createAdminInput is the normalized form of a validated wizard payload.
type createAdminInput struct {
	email    string
	name     string
	password string
}

// normalizeCreateAdmin validates req against the first-run rules and
// returns the normalized input together with the offending field names
// (empty when the payload is valid): the email is parsed and lower-cased,
// the password is 8-128 runes, and the optional name is trimmed to at most
// 100 runes, defaulting to the email address when absent.
func normalizeCreateAdmin(req apigen.CreateAdminRequest) (createAdminInput, []string) {
	var in createAdminInput
	var invalid []string

	email := strings.TrimSpace(string(req.Email))
	if addr, err := mail.ParseAddress(email); err != nil {
		invalid = append(invalid, "email")
	} else {
		in.email = strings.ToLower(addr.Address)
	}

	passwordRunes := utf8.RuneCountInString(req.Password)
	if passwordRunes < minPasswordRunes || passwordRunes > maxPasswordRunes {
		invalid = append(invalid, "password")
	} else {
		in.password = req.Password
	}

	var name string
	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
	}
	if utf8.RuneCountInString(name) > maxNameRunes {
		invalid = append(invalid, "name")
	} else {
		in.name = name
	}
	if in.name == "" && in.email != "" {
		in.name = in.email
	}

	return in, invalid
}
