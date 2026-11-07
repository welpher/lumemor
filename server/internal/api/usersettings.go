// The me-scoped user settings handlers (the settings feature): the
// caller's own account through GET/PATCH /users/me, the password
// change PUT /users/me/password, and the device list GET
// /users/me/sessions with its revoke DELETE /users/me/sessions/{id}.
//
// THE ME-SCOPING (the contract's pin): there is no userId parameter
// anywhere on this surface -- the session that authenticates the
// request IS the identity, the middleware's authContext decides the
// row, and one account can never address another's through it. The
// routes sit behind requireSession in the router's authed group.
//
// THE PASSWORD REVOCATION (the contract's pin): a successful change
// writes the fresh Argon2id hash and deletes every OTHER session of
// the account in one transaction; the session performing the change
// stays signed in (its cookie keeps working), every other held
// credential answers 401 from its next request on.
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"
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

// userSelfResponse maps one account row onto the me-scoped profile
// shape. The password hash the row carries never travels past this
// mapping.
func userSelfResponse(row dbgen.GetUserSelfRow) apigen.UserSelfResponse {
	return apigen.UserSelfResponse{
		Id:                      row.ID.Bytes,
		Name:                    row.Name,
		Email:                   oapitypes.Email(row.Email),
		Role:                    apigen.UserRole(row.Role),
		Theme:                   apigen.UserTheme(row.Theme),
		Locale:                  apigen.UserLocale(row.Locale),
		DownloadFormat:          apigen.DownloadFormat(row.DownloadFormat),
		DownloadIncludeVideos:   row.DownloadIncludeVideos,
		DownloadIncludeSidecars: row.DownloadIncludeSidecars,
	}
}

// GetUserSelf implements GET /api/v1/users/me: the caller's own
// account row -- the profile fields, the appearance pair and the
// download preferences.
func (s *server) GetUserSelf(w http.ResponseWriter, r *http.Request) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}
	row, err := s.queries.GetUserSelf(r.Context(), pgUUID(requester.userID))
	if errors.Is(err, pgx.ErrNoRows) {
		// The middleware proved a live session over the users join;
		// a missing row here is a wiring state, not a client one.
		s.internalError(w, r, "load the caller's account", err)
		return
	}
	if err != nil {
		s.internalError(w, r, "load the caller's account", err)
		return
	}
	s.writeJSON(w, http.StatusOK, userSelfResponse(row))
}

// UpdateUserSelf implements PATCH /api/v1/users/me: the self-scoped
// profile, appearance and download-preference patch. Every field is
// optional (an absent field keeps its stored value); the email's
// unique constraint answers 409 USER_EMAIL_TAKEN; the response is
// the stored state the UPDATE returned, never the request's echo.
func (s *server) UpdateUserSelf(w http.ResponseWriter, r *http.Request) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}
	var req apigen.UserSelfPatch
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
	params, invalid := normalizeUserSelfPatch(req)
	if len(invalid) > 0 {
		s.writeValidationError(w, "invalid profile payload", invalid)
		return
	}
	params.ID = pgUUID(requester.userID)

	row, err := s.queries.UpdateUserSelf(r.Context(), params)
	if err != nil {
		// The email's unique constraint is the patch's one
		// cross-account guard: a duplicate address is the expected
		// 409, race-free at the statement itself.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation &&
			pgErr.ConstraintName == "users_email_key" {
			s.writeError(w, http.StatusConflict, apigen.USEREMAILTAKEN,
				"an account with this email already exists", nil)
			return
		}
		s.internalError(w, r, "update the caller's account", err)
		return
	}
	s.writeJSON(w, http.StatusOK, userSelfResponse(dbgen.GetUserSelfRow(row)))
}

// ChangeUserPassword implements PUT /api/v1/users/me/password: the
// current password verified, the next stored as a fresh Argon2id
// hash, and every OTHER session of the account revoked in the same
// transaction -- the current session stays signed in.
func (s *server) ChangeUserPassword(w http.ResponseWriter, r *http.Request) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}
	var req apigen.PasswordChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
	var invalid []string
	if req.Current == "" {
		invalid = append(invalid, "current")
	}
	nextRunes := utf8.RuneCountInString(req.Next)
	if nextRunes < minPasswordRunes || nextRunes > maxPasswordRunes {
		invalid = append(invalid, "next")
	}
	if len(invalid) > 0 {
		s.writeValidationError(w, "invalid password change payload", invalid)
		return
	}

	row, err := s.queries.GetUserSelf(r.Context(), pgUUID(requester.userID))
	if errors.Is(err, pgx.ErrNoRows) {
		s.internalError(w, r, "load the caller's account", err)
		return
	}
	if err != nil {
		s.internalError(w, r, "load the caller's account", err)
		return
	}
	// The current password's verify. The failure answers 400
	// VALIDATION_FAILED naming `current`, deliberately NOT 401: the
	// request's session is valid and healthy, only the request's own
	// field failed to verify, and a 401 would (through the web
	// client's shared interceptor) sign the caller out of the very
	// session they are changing the password from.
	if !auth.Verify(row.PasswordHash, req.Current) {
		s.writeValidationError(w, "current password does not match", []string{"current"})
		return
	}

	hash, err := auth.Hash(req.Next)
	if err != nil {
		s.internalError(w, r, "hash the new password", err)
		return
	}

	// One transaction: the credential write and the session
	// severance land together, so a failure of either leaves the
	// account exactly where it was (old password, all sessions
	// alive) instead of half-changed.
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		s.internalError(w, r, "begin the password change transaction", err)
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	queries := dbgen.New(tx)

	if _, err := queries.UpdateUserPassword(r.Context(), dbgen.UpdateUserPasswordParams{
		ID:           pgUUID(requester.userID),
		PasswordHash: hash,
	}); err != nil {
		s.internalError(w, r, "write the new password", err)
		return
	}
	revoked, err := queries.RevokeOtherUserSessions(r.Context(), dbgen.RevokeOtherUserSessionsParams{
		UserID: pgUUID(requester.userID),
		KeepID: pgUUID(requester.sessionID),
	})
	if err != nil {
		s.internalError(w, r, "revoke the account's other sessions", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		s.internalError(w, r, "commit the password change", err)
		return
	}

	// The audit line: the credential severance the change performed.
	// The count names how many other devices were signed out; the
	// current session's survival is the contract's other half and
	// needs no log line to be true.
	s.logger.Info("password changed; other sessions revoked",
		"user_id", requester.userID, "revoked_sessions", revoked, "method", r.Method)
	s.publishPasswordChanged(pgUUID(requester.userID))
	w.WriteHeader(http.StatusNoContent)
}

// ListUserDevices implements GET /api/v1/users/me/sessions: the
// caller's active sessions with the isCurrent marker derived against
// the request's own session id, and each row's creation-time address.
func (s *server) ListUserDevices(w http.ResponseWriter, r *http.Request) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}
	rows, err := s.queries.ListUserSessions(r.Context(), pgUUID(requester.userID))
	if err != nil {
		s.internalError(w, r, "list the caller's devices", err)
		return
	}
	devices := make([]apigen.DeviceSession, 0, len(rows))
	for _, row := range rows {
		device := apigen.DeviceSession{
			Id:         row.ID.Bytes,
			Kind:       apigen.DeviceSessionKind(row.Kind),
			LastSeenAt: lastSeenOf(row.CreatedAt, row.LastSeenAt),
			IsCurrent:  uuidEqual(row.ID, requester.sessionID),
		}
		if row.DeviceName != "" {
			name := row.DeviceName
			device.DeviceName = &name
		}
		if row.IpAddress.Valid && row.IpAddress.String != "" {
			address := row.IpAddress.String
			device.IpAddress = &address
		}
		devices = append(devices, device)
	}
	s.writeJSON(w, http.StatusOK, devices)
}

// RevokeUserDevice implements DELETE /api/v1/users/me/sessions/{id}:
// the device list's revoke. The current session answers 400 (the
// sign-out flow is that session's exit); every other id follows the
// auth surface's owner-scoped set semantics -- unknown, foreign and
// already-revoked ids answer the same 204.
func (s *server) RevokeUserDevice(w http.ResponseWriter, r *http.Request, id oapitypes.UUID) {
	requester, ok := authFrom(r.Context())
	if !ok {
		s.rejectUnauthenticated(w)
		return
	}
	if id == requester.sessionID {
		s.writeValidationError(w, "the current session is signed out through the sign-out flow", []string{"id"})
		return
	}
	kind, err := s.queries.RevokeSession(r.Context(), dbgen.RevokeSessionParams{
		ID:     pgUUID(id),
		UserID: pgUUID(requester.userID),
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.internalError(w, r, "revoke the device's session", err)
		return
	}
	if err == nil {
		s.publishSessionEvent("session.revoked", pgUUID(id), kind)
	}
	w.WriteHeader(http.StatusNoContent)
}

// normalizeUserSelfPatch validates a self-patch against the account
// rules and maps it onto the query's nullable-parameter form: nil
// keeps the stored value (the COALESCE), a value replaces it. The
// offending field names come back for the 400's details; the email
// follows the creation paths' normalization (parsed, lower-cased).
func normalizeUserSelfPatch(req apigen.UserSelfPatch) (dbgen.UpdateUserSelfParams, []string) {
	var params dbgen.UpdateUserSelfParams
	var invalid []string

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" || utf8.RuneCountInString(name) > maxNameRunes {
			invalid = append(invalid, "name")
		} else {
			params.Name = pgtype.Text{String: name, Valid: true}
		}
	}
	if req.Email != nil {
		email := strings.TrimSpace(string(*req.Email))
		if addr, err := mail.ParseAddress(email); err != nil {
			invalid = append(invalid, "email")
		} else {
			params.Email = pgtype.Text{String: strings.ToLower(addr.Address), Valid: true}
		}
	}
	if req.Theme != nil {
		switch *req.Theme {
		case apigen.UserThemeAuto, apigen.UserThemeLight, apigen.UserThemeDark:
			params.Theme = pgtype.Text{String: string(*req.Theme), Valid: true}
		default:
			invalid = append(invalid, "theme")
		}
	}
	if req.Locale != nil {
		switch *req.Locale {
		case apigen.UserLocaleEn, apigen.UserLocaleZhHans:
			params.Locale = pgtype.Text{String: string(*req.Locale), Valid: true}
		default:
			invalid = append(invalid, "locale")
		}
	}
	if req.DownloadFormat != nil {
		switch *req.DownloadFormat {
		case apigen.Zip, apigen.Tar:
			params.DownloadFormat = pgtype.Text{String: string(*req.DownloadFormat), Valid: true}
		default:
			invalid = append(invalid, "downloadFormat")
		}
	}
	if req.DownloadIncludeVideos != nil {
		params.DownloadIncludeVideos = pgtype.Bool{Bool: *req.DownloadIncludeVideos, Valid: true}
	}
	if req.DownloadIncludeSidecars != nil {
		params.DownloadIncludeSidecars = pgtype.Bool{Bool: *req.DownloadIncludeSidecars, Valid: true}
	}
	return params, invalid
}

// lastSeenOf resolves a session row's last-use timestamp: the
// recorded last_seen_at, else the creation time (the login's own
// answer -- a session used exactly once was used at its creation).
func lastSeenOf(createdAt, lastSeenAt pgtype.Timestamptz) time.Time {
	if lastSeenAt.Valid {
		return lastSeenAt.Time.UTC()
	}
	return createdAt.Time.UTC()
}

// uuidEqual compares a row's pgtype.UUID against the request
// identity's plain UUID.
func uuidEqual(a pgtype.UUID, b uuid.UUID) bool {
	return a.Valid && a.Bytes == b
}

// publishPasswordChanged reports the credential severance onto the
// activity stream: the user whose password changed, never a
// credential fragment. Devices that lost their sessions learn it
// through their own next request's 401, not through this event.
func (s *server) publishPasswordChanged(id pgtype.UUID) {
	if s.events == nil {
		return
	}
	s.events.Publish("auth.password_changed", map[string]string{
		"userId": uuid.UUID(id.Bytes).String(),
	})
}
