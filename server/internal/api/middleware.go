// Middleware of the server: the security headers (root-level, every
// response), the /api/v1 request pipeline (origin check, JSON body
// limit for the JSON surface), the per-request handler timeout
// wrapping the non-streaming route groups, the session
// middleware that turns a cookie or bearer credential into
// request-scoped user identity, and the admin-role guard stacked
// behind it for the admin surface. Order matters and is pinned in
// NewRouter: the security headers run on every request of the
// server; inside the /api/v1 tree the origin check runs on every
// request; limitJSONBody wraps the JSON route groups only (uploads
// stream their own multipart bodies and are capped by the receiver's
// size limit instead); every non-streaming group sits inside
// timeoutMiddleware; the SSE stream group and the upload group run
// outside it; the authed groups sit behind requireSession, and the
// admin group behind requireSession plus requireAdmin.
package api

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	apigen "github.com/welpher/lumemor/server/internal/api/gen"
	"github.com/welpher/lumemor/server/internal/auth"
	dbgen "github.com/welpher/lumemor/server/internal/db/gen"
)

const (
	// sessionCookieName is the web credential: the raw session token
	// rides this cookie.
	sessionCookieName = "lumemor_session"

	// maxJSONBodyBytes caps every JSON request body of the API: the
	// endpoints are small structured payloads, and unbounded buffering
	// on unauthenticated routes is an attack surface. The multipart
	// upload route (POST /assets) is mounted in its own group
	// outside this middleware -- the streaming receiver's per-file
	// size limit governs its bodies.
	maxJSONBodyBytes = 1 << 20

	// sessionTouchThreshold throttles the sliding-expiry write: a
	// session active more than a minute since its last recorded use
	// gets one row update; a burst of requests costs at most one write
	// per minute.
	sessionTouchThreshold = time.Minute

	// webSessionTTL is both the fresh login expiry and the sliding
	// window: every touched session expires 30 days from its last
	// use.
	webSessionTTL = 30 * 24 * time.Hour
)

// handlerTimeout bounds every non-streaming /api/v1 handler: half a
// minute is generous for this surface while keeping a wedged handler
// from pinning its connection forever. It is a per-request bound --
// chi has no per-route timeout -- implemented by wrapping the
// non-streaming route groups in http.TimeoutHandler; see
// timeoutMiddleware.
const handlerTimeout = 30 * time.Second

// timeoutMiddleware wraps a handler chain in http.TimeoutHandler: a
// chain that outlives the bound answers 503 (TimeoutHandler's
// default) while the request context handed to the chain is canceled,
// so parked queries wind down on their own. The duration is a
// parameter only so tests can shorten it; production wiring pins
// handlerTimeout.
//
// Streaming handlers must never sit behind this wrapper: the
// timeoutWriter buffers the whole response until the handler returns
// and implements no Flusher, so a stream would first stall and then
// be cut at the deadline. The SSE activity stream therefore lives in
// its own route group outside it.
func timeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, timeout, "")
	}
}

// limitJSONBody wraps every request body in an http.MaxBytesReader
// capped at maxJSONBodyBytes.
func limitJSONBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
		next.ServeHTTP(w, r)
	})
}

// setSecurityHeaders stamps the two response headers every response
// of the server carries: API, pages, static assets and the SPA shell
// alike (NewRouter mounts it at the router root, so every route --
// hand-mounted or catch-all -- passes through it). HSTS and the
// dynamic CSP arrive with the map feature (the CSP must know the
// tile origins first, and the app shell's pre-paint script needs a
// hash); CORS stays closed by design -- the SPA is same-origin
// inside the single binary and external tools authenticate as
// bearers, which CORS never constrains.
func setSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

// checkOrigin implements the CSRF defense of an all-JSON API: a
// state-changing request that declares a foreign Origin is refused
// outright. Requests without an Origin header pass -- they are
// non-browser clients (the mobile app, curl), which are not part of
// the browser CSRF threat model. Reads (GET/HEAD/OPTIONS) are exempt.
func checkOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isStateChanging(r.Method) {
			if origin := r.Header.Get("Origin"); origin != "" && !sameOrigin(r, origin) {
				http.Error(w, "cross-origin request rejected", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// isStateChanging reports whether the method mutates server state.
func isStateChanging(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	default:
		return true
	}
}

// sameOrigin reports whether the declared Origin matches the request's
// own host. Only host and port are compared, not the scheme: behind a
// TLS-terminating reverse proxy the forwarded request is plain HTTP
// while the browser's Origin is HTTPS, and that deployment must keep
// working. A malformed Origin (including the literal "null") never
// matches.
func sameOrigin(r *http.Request, origin string) bool {
	originURL, err := url.Parse(origin)
	if err != nil || originURL.Host == "" {
		return false
	}
	return strings.EqualFold(originURL.Host, r.Host)
}

// authContextKey is the unexported context key for the authenticated
// requester.
type authContextKey struct{}

// authContext is the identity the session middleware injects into the
// request context: who is calling (user id, role, email) and through
// which session.
type authContext struct {
	userID    uuid.UUID
	role      dbgen.UserRole
	email     string
	sessionID uuid.UUID
	kind      dbgen.SessionKind
}

// authFrom extracts the authenticated requester; ok is false outside
// the authed route group.
func authFrom(ctx context.Context) (authContext, bool) {
	user, ok := ctx.Value(authContextKey{}).(authContext)
	return user, ok
}

// requireSession authenticates a request of the authed group: it
// extracts the session credential (the lumemor_session cookie or an
// Authorization: Bearer token -- both front the same sessions table),
// resolves it to a live session, refuses expired sessions and disabled
// accounts, slides the expiry when the session has been idle for more
// than a minute, and injects the identity into the context. Every
// failure answers the 401 envelope; the code deliberately reuses
// AUTH_INVALID_CREDENTIALS so an absent, unknown, expired or revoked
// credential is indistinguishable.
func (s *server) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := sessionCredential(r)
		if raw == "" {
			s.rejectUnauthenticated(w)
			return
		}

		row, err := s.queries.GetSessionByTokenHash(r.Context(), auth.HashSessionToken(raw))
		if errors.Is(err, pgx.ErrNoRows) {
			s.rejectUnauthenticated(w)
			return
		}
		if err != nil {
			s.internalError(w, r, "lookup session", err)
			return
		}
		// A disabled account loses its sessions' authority without
		// deleting them: the check rides the same lookup.
		if row.UserIsDisabled {
			s.rejectUnauthenticated(w)
			return
		}

		// Sliding expiry, throttled: sessions idle for over a minute
		// get their last_seen and expiry moved to now. A failed touch
		// logs and continues -- the session itself is valid, only the
		// renewal write failed, and the next request retries it.
		now := time.Now()
		lastSeen := row.CreatedAt.Time
		if row.LastSeenAt.Valid {
			lastSeen = row.LastSeenAt.Time
		}
		if now.Sub(lastSeen) > sessionTouchThreshold {
			err := s.queries.TouchSession(r.Context(), dbgen.TouchSessionParams{
				ID:         row.ID,
				LastSeenAt: pgtype.Timestamptz{Time: now, Valid: true},
				ExpiresAt:  pgtype.Timestamptz{Time: now.Add(webSessionTTL), Valid: true},
			})
			if err != nil {
				s.logger.Error("slide session expiry", "error", err, "session_id", uuid.UUID(row.ID.Bytes))
			}
		}

		ctx := context.WithValue(r.Context(), authContextKey{}, authContext{
			userID:    uuid.UUID(row.UserID.Bytes),
			role:      row.UserRole,
			email:     row.UserEmail,
			sessionID: uuid.UUID(row.ID.Bytes),
			kind:      row.Kind,
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// sessionCredential extracts the presented session token: the web
// cookie first, then the bearer form. The bearer scheme is matched
// case-insensitively per RFC 7235.
func sessionCredential(r *http.Request) string {
	if cookie, err := r.Cookie(sessionCookieName); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	header := r.Header.Get("Authorization")
	if len(header) > len("bearer ") && strings.EqualFold(header[:len("bearer ")], "bearer ") {
		return strings.TrimSpace(header[len("bearer "):])
	}
	return ""
}

// rejectUnauthenticated answers the uniform 401 envelope.
func (s *server) rejectUnauthenticated(w http.ResponseWriter) {
	s.writeError(w, http.StatusUnauthorized, apigen.AUTHINVALIDCREDENTIALS, "authentication required", nil)
}

// requireAdmin guards the admin route group: it runs behind
// requireSession (the caller is authenticated), and a caller whose
// role is not admin answers 403 AUTH_FORBIDDEN -- a code distinct
// from the 401 family, because the caller is signed in, just not
// entitled to the admin surface.
func (s *server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requester, ok := authFrom(r.Context())
		if !ok {
			// requireSession sits ahead of this middleware in every
			// mounted group; this arm only keeps the guard
			// self-sufficient if a future wiring drops that order.
			s.rejectUnauthenticated(w)
			return
		}
		if requester.role != dbgen.UserRoleAdmin {
			s.writeError(w, http.StatusForbidden, apigen.AUTHFORBIDDEN,
				"administrator role required", nil)
			return
		}
		next.ServeHTTP(w, r.WithContext(r.Context()))
	})
}
