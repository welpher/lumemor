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

	// uniqueViolation is the PostgreSQL error code for a unique
	// constraint violation (23505).
	uniqueViolation = "23505"
)

// CreateAdmin implements POST /api/v1/server/setup/admin: the one-time
// first-run wizard that seeds the initial administrator account.
func (s *server) CreateAdmin(w http.ResponseWriter, r *http.Request) {
	// Refuse completed instances before touching the payload so the
	// endpoint never leaks validation feedback once setup is done.
	admins, err := s.queries.CountAdmins(r.Context())
	if err != nil {
		s.internalError(w, r, "count admins", err)
		return
	}
	if admins > 0 {
		s.writeError(w, http.StatusConflict, apigen.SETUPALREADYCOMPLETED,
			"an administrator account already exists", nil)
		return
	}

	var req apigen.CreateAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// The generated Email type validates during unmarshalling, so a
		// syntactically invalid address fails here; name the field so the
		// 400 still points at the right input.
		message := "request body must be valid JSON"
		var details *map[string]interface{}
		if errors.Is(err, oapitypes.ErrValidationEmail) {
			message = "email must be a valid address"
			fields := []interface{}{"email"}
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

	user, err := s.queries.CreateUser(r.Context(), dbgen.CreateUserParams{
		ID:           pgtype.UUID{Bytes: id, Valid: true},
		Name:         in.name,
		Email:        in.email,
		PasswordHash: hash,
	})
	if err != nil {
		// Two concurrent wizards can both pass the admin-less check; the
		// loser loses the race on the users email unique constraint and
		// reports the same conflict as any request after setup completed.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			s.writeError(w, http.StatusConflict, apigen.SETUPALREADYCOMPLETED,
				"an administrator account already exists", nil)
			return
		}
		s.internalError(w, r, "create admin user", err)
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
