// Handlers of the auth surface: POST /auth/login, POST /auth/logout,
// GET /auth/sessions and DELETE /auth/sessions/{id}. Web sessions
// authenticate through the lumemor_session cookie, mobile sessions
// through a bearer token; both are the same session records and both
// appear in the session list.
package api

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	oapitypes "github.com/oapi-codegen/runtime/types"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/auth"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

// maxDeviceNameRunes bounds the user agent stored as a web session's
// device label: it is free-form client input, displayed later in the
// device list, so it gets a sane ceiling.
const maxDeviceNameRunes = 200

// Login implements POST /auth/login. The order of the checks is
// pinned -- do not reshuffle:
//
//	email lower-cased and trimmed -> rate-limit gate -> user lookup ->
//	verification -> failure counter -> session -> cookie -> event.
//
// A wrong password, an unknown address and a disabled account answer
// the byte-identical 401 AUTH_INVALID_CREDENTIALS: the endpoint must
// not let anyone enumerate accounts.
func (s *server) Login(w http.ResponseWriter, r *http.Request) {
	var req apigen.LoginRequest
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
	email := strings.ToLower(strings.TrimSpace(string(req.Email)))
	var invalid []string
	if email == "" {
		invalid = append(invalid, "email")
	}
	if req.Password == "" {
		invalid = append(invalid, "password")
	}
	if len(invalid) > 0 {
		fields := make([]interface{}, len(invalid))
		for i, field := range invalid {
			fields[i] = field
		}
		details := map[string]interface{}{"fields": fields}
		s.writeError(w, http.StatusBadRequest, apigen.VALIDATIONFAILED,
			"invalid login payload", &details)
		return
	}

	// The rate-limit gate runs before any credential work so a locked
	// pair burns no Argon2 and, crucially, cannot succeed its way out
	// of the lock.
	key := s.transport.remoteIP(r) + "|" + email
	if !s.limiter.Allow(key) {
		seconds := int64(math.Ceil(s.limiter.RetryAfter(key).Seconds()))
		if seconds < 1 {
			seconds = 1
		}
		w.Header().Set("Retry-After", strconv.FormatInt(seconds, 10))
		s.writeError(w, http.StatusTooManyRequests, apigen.AUTHRATELIMITED,
			"too many failed attempts, retry later", nil)
		return
	}

	user, err := s.queries.GetUserByEmail(r.Context(), email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Unknown accounts count as failures: without this, an
			// attacker could probe addresses without ever hitting the
			// limit.
			s.limiter.Fail(key)
			s.rejectLogin(w, email)
			return
		}
		s.internalError(w, r, "lookup login user", err)
		return
	}
	// The disabled check runs before the (expensive) verification; the
	// response is the same either way, so nothing is leaked and the
	// attempt still counts as a failure.
	if user.IsDisabled {
		s.limiter.Fail(key)
		s.rejectLogin(w, email)
		return
	}
	if !auth.Verify(user.PasswordHash, req.Password) {
		s.limiter.Fail(key)
		s.rejectLogin(w, email)
		return
	}
	s.limiter.Reset(key)

	raw, hash := auth.NewSessionToken()
	id, err := uuid.NewV7()
	if err != nil {
		s.internalError(w, r, "generate session id", err)
		return
	}
	now := time.Now()
	session, err := s.queries.CreateSession(r.Context(), dbgen.CreateSessionParams{
		ID:         pgtype.UUID{Bytes: id, Valid: true},
		UserID:     user.ID,
		TokenHash:  hash,
		Kind:       dbgen.SessionKindWeb,
		DeviceName: deviceLabel(r),
		// The creation-time address the device list renders; the
		// derived client address (the socket peer unless a trusted
		// proxy forwards it -- the same rule as the rate-limit key).
		IpAddress:  pgtype.Text{String: s.transport.remoteIP(r), Valid: true},
		LastSeenAt: pgtype.Timestamptz{Time: now, Valid: true},
		ExpiresAt:  pgtype.Timestamptz{Time: now.Add(webSessionTTL), Valid: true},
	})
	if err != nil {
		s.internalError(w, r, "create session", err)
		return
	}

	// The credential exists exactly once, here: the response carries
	// it as the cookie, the database keeps only its hash.
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    raw,
		Path:     "/",
		MaxAge:   int(webSessionTTL / time.Second),
		HttpOnly: true,
		// Secure follows the deployment's transport policy: the
		// request's own TLS state, the https public origin, or a
		// trusted proxy's https report.
		Secure:   s.transport.requestIsHTTPS(r),
		SameSite: http.SameSiteStrictMode,
	})

	s.publishSessionEvent("session.started", session.ID, session.Kind)
	s.writeJSON(w, http.StatusOK, sessionInfoFrom(session, uuid.UUID(user.ID.Bytes), user.Role))
}

// Logout implements POST /auth/logout: revoke the session that
// authenticates the request and expire the cookie on web clients. A
// request with an already-revoked session never reaches this handler
// (the middleware answers 401 first).
func (s *server) Logout(w http.ResponseWriter, r *http.Request) {
	user, ok := authFrom(r.Context())
	if !ok {
		// Unreachable through the authed group; a routing regression
		// would land here.
		s.internalError(w, r, "logout without session context", errors.New("authed route ran without auth context"))
		return
	}
	// A no-row outcome means a concurrent revocation won the race; the
	// session is revoked either way and the logout still completes.
	if _, err := s.queries.RevokeSession(r.Context(), dbgen.RevokeSessionParams{
		ID:     pgUUID(user.sessionID),
		UserID: pgUUID(user.userID),
	}); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.internalError(w, r, "revoke session", err)
		return
	}
	s.expireSessionCookie(w, r)
	s.publishSessionEvent("session.revoked", pgUUID(user.sessionID), user.kind)
	w.WriteHeader(http.StatusNoContent)
}

// ListSessions implements GET /auth/sessions: the caller's active
// sessions across devices, most recently used first.
func (s *server) ListSessions(w http.ResponseWriter, r *http.Request) {
	user, ok := authFrom(r.Context())
	if !ok {
		s.internalError(w, r, "list sessions without session context", errors.New("authed route ran without auth context"))
		return
	}
	rows, err := s.queries.ListUserSessions(r.Context(), pgUUID(user.userID))
	if err != nil {
		s.internalError(w, r, "list sessions", err)
		return
	}
	sessions := make([]apigen.SessionInfo, 0, len(rows))
	for _, row := range rows {
		sessions = append(sessions, sessionInfoFrom(row, user.userID, user.role))
	}
	s.writeJSON(w, http.StatusOK, sessions)
}

// RevokeSession implements DELETE /auth/sessions/{id} with set
// semantics: unknown, foreign and already-revoked ids answer the same
// 204 as a successful revoke, without revealing whether such a session
// exists. Revoking the session of the current request behaves like
// logout -- the cookie is expired too.
func (s *server) RevokeSession(w http.ResponseWriter, r *http.Request, id oapitypes.UUID) {
	user, ok := authFrom(r.Context())
	if !ok {
		s.internalError(w, r, "revoke session without session context", errors.New("authed route ran without auth context"))
		return
	}
	kind, err := s.queries.RevokeSession(r.Context(), dbgen.RevokeSessionParams{
		ID:     pgUUID(id),
		UserID: pgUUID(user.userID),
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.internalError(w, r, "revoke session", err)
		return
	}
	// The event reflects reality, not the request: a revoke that hit
	// nothing revokes nothing, and the kind is the revoked session's
	// own.
	if err == nil {
		s.publishSessionEvent("session.revoked", pgUUID(id), kind)
	}
	if id == user.sessionID {
		s.expireSessionCookie(w, r)
	}
	w.WriteHeader(http.StatusNoContent)
}

// rejectLogin answers the anti-enumeration 401 -- byte-identical for a
// wrong password, an unknown account and a disabled account -- and
// reports the failed attempt onto the activity stream. The rate-limit
// 429 is not a credential outcome and deliberately publishes nothing.
// The publish precedes the response write and is a cheap synchronous
// bus append, so it never meaningfully delays the response.
func (s *server) rejectLogin(w http.ResponseWriter, email string) {
	s.publishLoginFailed(email)
	s.writeError(w, http.StatusUnauthorized, apigen.AUTHINVALIDCREDENTIALS,
		"invalid email or password", nil)
}

// publishLoginFailed reports a failed login attempt onto the activity
// stream: the audit minimum names login success and
// failure, and this is the failure half. The payload carries the
// attempted address only -- no password, no token, nothing about which
// check failed -- because the stream is the same admin-audience SSE
// surface that carries the session events.
func (s *server) publishLoginFailed(email string) {
	if s.events == nil {
		return
	}
	s.events.Publish("auth.login_failed", map[string]string{"email": email})
}

// expireSessionCookie instructs web clients to drop the session cookie
// by re-issuing it with a past expiry. The refreshed cookie repeats the
// login's Secure determination, so a browser drops the https cookie it
// holds even though the proxied request arrives as plain HTTP.
func (s *server) expireSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.transport.requestIsHTTPS(r),
		SameSite: http.SameSiteStrictMode,
	})
}

// sessionInfoFrom maps a session row onto the API shape, carrying the
// account identity the session belongs to (the identity seam: the login,
// the session list and the pairing response all answer who is
// calling, so a client restoring a session never needs a second
// endpoint). Only the whitelisted fields travel: the token hash never
// leaves the store.
func sessionInfoFrom(row dbgen.Session, userID uuid.UUID, role dbgen.UserRole) apigen.SessionInfo {
	lastSeen := row.CreatedAt.Time.UTC()
	if row.LastSeenAt.Valid {
		lastSeen = row.LastSeenAt.Time.UTC()
	}
	info := apigen.SessionInfo{
		Id:         uuid.UUID(row.ID.Bytes),
		Kind:       apigen.SessionInfoKind(row.Kind),
		UserId:     userID,
		Role:       apigen.UserRole(role),
		CreatedAt:  row.CreatedAt.Time.UTC(),
		LastSeenAt: lastSeen,
	}
	if row.DeviceName != "" {
		name := row.DeviceName
		info.DeviceName = &name
	}
	// Bearer sessions do not expire: their stored sentinel is never
	// reported.
	if row.Kind == dbgen.SessionKindWeb && row.ExpiresAt.Valid {
		expires := row.ExpiresAt.Time.UTC()
		info.ExpiresAt = &expires
	}
	return info
}

// deviceLabel derives the web session's device name from the browser's
// user agent, bounded to maxDeviceNameRunes.
func deviceLabel(r *http.Request) string {
	agent := strings.TrimSpace(r.UserAgent())
	if utf8.RuneCountInString(agent) > maxDeviceNameRunes {
		agent = string([]rune(agent)[:maxDeviceNameRunes])
	}
	return agent
}

// pgUUID wraps a uuid.UUID as the pgtype the generated queries take.
func pgUUID(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: u, Valid: true}
}

// publishSessionEvent reports a session lifecycle event onto the
// activity stream; the payload names the session and its kind, never
// the credential.
func (s *server) publishSessionEvent(name string, id pgtype.UUID, kind dbgen.SessionKind) {
	if s.events == nil {
		return
	}
	s.events.Publish(name, map[string]string{
		"id":   uuid.UUID(id.Bytes).String(),
		"kind": string(kind),
	})
}
