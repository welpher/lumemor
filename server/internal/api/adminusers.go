// The admin user management handlers (M2d Task 8): the listing, the
// account creation and the disabled toggle of GET/POST
// /admin/users and PATCH /admin/users/{id}. The permission model is
// the admin role itself: every route of this surface sits behind
// requireSession AND requireAdmin (see middleware.go and the router),
// so a signed-in caller without the role answers 403 AUTH_FORBIDDEN
// before any handler runs.
//
// The account creation is the management counterpart of the
// first-run wizard (setup.go): the same validation rules, the same
// Argon2id password hashing and UUIDv7 ids, and -- the M2c wiring
// reuse -- the account's sync album created inside the SAME
// transaction, so a freshly provisioned account can take uploads
// immediately. Unlike the wizard there is no one-time guarantee to
// defend: this endpoint exists exactly because an administrator
// already does, and the email's unique constraint is the only guard
// it needs (409 USER_EMAIL_TAKEN).
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	oapitypes "github.com/oapi-codegen/runtime/types"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/auth"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// ListUsers implements GET /api/v1/admin/users: every account of the
// instance, oldest first, through the listing that selects only the
// safe columns (the password hash never leaves the table here).
func (s *server) ListUsers(w http.ResponseWriter, r *http.Request) {
	// The route group already proved the admin role; the authContext
	// read keeps the handler's shape uniform with every other one.
	if _, ok := authFrom(r.Context()); !ok {
		s.rejectUnauthenticated(w)
		return
	}

	rows, err := s.queries.ListUsers(r.Context())
	if err != nil {
		s.internalError(w, r, "list users", err)
		return
	}

	response := apigen.AdminUserListResponse{Items: make([]apigen.AdminUserResponse, 0, len(rows))}
	for _, row := range rows {
		response.Items = append(response.Items,
			adminUserFrom(row.ID, row.Name, row.Email, row.Role, row.IsDisabled, row.CreatedAt))
	}
	s.writeJSON(w, http.StatusOK, response)
}

// CreateUser implements POST /api/v1/admin/users: the transactional
// account creation (the user row and its sync album land together).
func (s *server) CreateUser(w http.ResponseWriter, r *http.Request) {
	// requireAdmin proved the role; the handler mirrors the wizard's
	// decode-and-validate shape, including the capped-body error
	// mapping (the JSON group's limitJSONBody cap reports through the
	// same MaxBytesError path).
	var req apigen.AdminCreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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

	in, invalid := normalizeCreateUser(req)
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

	// One transaction: the account row and its sync album land
	// together (the wizard's wiring, reused). No advisory lock is
	// needed here -- the one-time guarantee is the wizard's concern;
	// this surface's races end at the email's unique constraint.
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		s.internalError(w, r, "begin create user transaction", err)
		return
	}
	// The rollback is a no-op after a successful commit; arguments
	// are captured while the request context is still live.
	defer func() { _ = tx.Rollback(r.Context()) }()

	queries := dbgen.New(tx)
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
		Role:         in.role,
	})
	if err != nil {
		// The email's unique constraint is the identity guard of this
		// surface: a duplicate address is the expected 409, any other
		// failure stays internal.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation &&
			pgErr.ConstraintName == "users_email_key" {
			s.writeError(w, http.StatusConflict, apigen.USEREMAILTAKEN,
				"an account with this email already exists", nil)
			return
		}
		s.internalError(w, r, "create user", err)
		return
	}

	// The account's sync album, in the same transaction (the wizard's
	// reasoning applies verbatim: an account whose stream is missing
	// would take uploads that never surface anywhere). The upsert arm
	// is the concurrent-creation safety net; the insert arm runs.
	syncAlbumID, err := uuid.NewV7()
	if err != nil {
		s.internalError(w, r, "generate sync album id", err)
		return
	}
	if _, err := queries.EnsureSyncAlbum(r.Context(), dbgen.EnsureSyncAlbumParams{
		ID:      pgtype.UUID{Bytes: syncAlbumID, Valid: true},
		OwnerID: user.ID,
	}); err != nil {
		s.internalError(w, r, "create sync album", err)
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		s.internalError(w, r, "commit create user transaction", err)
		return
	}

	s.writeJSON(w, http.StatusCreated,
		adminUserFrom(user.ID, user.Name, user.Email, user.Role, user.IsDisabled, user.CreatedAt))
}

// UpdateUser implements PATCH /api/v1/admin/users/{id}: the
// enable/disable toggle. The switch's enforcement point is the
// session middleware, so the handler only writes the row -- the
// contract description pins the observable semantics (next-request
// 401s, session rows surviving, re-enable reviving unexpired
// sessions).
func (s *server) UpdateUser(w http.ResponseWriter, r *http.Request, id oapitypes.UUID) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}

	// The contract marks isDisabled required, but the generated
	// struct's plain bool cannot tell an explicit false from an
	// absent field -- a body of {} (or a JSON null) would decode as
	// a silent enable nobody asked for. The body therefore decodes
	// through a pointer shape solely to pin the field's presence;
	// the wire name is the generated type's own.
	var decoded struct {
		IsDisabled *bool `json:"isDisabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&decoded); err != nil {
		message := "request body must be valid JSON"
		var fields []interface{}
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			message = "request body exceeds the size limit"
			fields = []interface{}{"body"}
		}
		var details *map[string]interface{}
		if fields != nil {
			details = &map[string]interface{}{"fields": fields}
		}
		s.writeError(w, http.StatusBadRequest, apigen.VALIDATIONFAILED, message, details)
		return
	}
	if decoded.IsDisabled == nil {
		s.writeValidationError(w, "isDisabled is required", []string{"isDisabled"})
		return
	}
	isDisabled := *decoded.IsDisabled

	// The self-disable footgun: the last admin locking themselves out
	// would strand this surface, so the caller's own id is refused
	// before any write (a self-enable is not the footgun and stays a
	// plain idempotent write). The web client cannot know its own
	// user id through this surface (the listing carries no caller
	// marker), so this guard is the authoritative one.
	if isDisabled && uuid.UUID(id) == requester.userID {
		details := map[string]interface{}{"fields": []interface{}{"id"}}
		s.writeError(w, http.StatusBadRequest, apigen.VALIDATIONFAILED,
			"an administrator cannot disable their own account", &details)
		return
	}

	// The idempotent write; an unknown id matches no row and answers
	// the 404 (account ids are enumerable through this surface's own
	// listing, so there is no anti-enumeration pairing to keep).
	row, err := s.queries.SetUserDisabled(r.Context(), dbgen.SetUserDisabledParams{
		ID:         pgUUID(id),
		IsDisabled: isDisabled,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		s.writeError(w, http.StatusNotFound, apigen.USERNOTFOUND,
			"no account with this identifier", nil)
		return
	}
	if err != nil {
		s.internalError(w, r, "set user disabled", err)
		return
	}

	s.writeJSON(w, http.StatusOK,
		adminUserFrom(row.ID, row.Name, row.Email, row.Role, row.IsDisabled, row.CreatedAt))
}

// createUserInput is the normalized form of a validated account
// creation payload (the wizard's shape plus the role).
type createUserInput struct {
	name     string
	email    string
	password string
	role     dbgen.UserRole
}

// normalizeCreateUser validates req against the account rules the
// wizard pins and returns the normalized input together with the
// offending field names (empty when the payload is valid): the email
// is parsed and lower-cased, the password is 8-128 runes, the
// optional name is trimmed to at most 100 runes (defaulting to the
// email address), and the optional role must be a member of the
// user_role enum (defaulting to 'user' -- the wizard pins 'admin'
// itself).
func normalizeCreateUser(req apigen.AdminCreateUserRequest) (createUserInput, []string) {
	var in createUserInput
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

	in.role = dbgen.UserRoleUser
	if req.Role != nil {
		switch *req.Role {
		case apigen.UserRoleAdmin:
			in.role = dbgen.UserRoleAdmin
		case apigen.UserRoleUser:
			in.role = dbgen.UserRoleUser
		default:
			invalid = append(invalid, "role")
		}
	}

	return in, invalid
}

// adminUserFrom maps one account's safe columns onto the API shape.
func adminUserFrom(id pgtype.UUID, name, email string, role dbgen.UserRole, isDisabled bool, createdAt pgtype.Timestamptz) apigen.AdminUserResponse {
	return apigen.AdminUserResponse{
		Id:         uuid.UUID(id.Bytes),
		Name:       name,
		Email:      oapitypes.Email(email),
		Role:       apigen.UserRole(role),
		IsDisabled: isDisabled,
		CreatedAt:  createdAt.Time.UTC(),
	}
}
