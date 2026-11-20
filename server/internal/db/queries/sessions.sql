-- Session queries for the auth middleware and the session management
-- handlers. Credentials are stored only as their SHA-256 hex hash;
-- revocation is row deletion, which is what gives DELETE
-- /auth/sessions/{id} its set semantics and keeps a revoked (deleted)
-- credential failing lookup like an unknown one.

-- name: CreateSession :one
-- kind is web (cookie, sliding 30-day expiry) or mobile (bearer,
-- revoked explicitly); the caller supplies both timestamps so the
-- login and pairing flows pin them to the same clock as the response.
-- ip_address is the creation-time socket peer the device list
-- renders (the login's and the pairing exchange's own callers
-- capture it); NULL is the honest unknown for callers that cannot.
INSERT INTO sessions (id, user_id, token_hash, kind, device_name, device_id, ip_address, last_seen_at, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetSessionByTokenHash :one
-- The auth lookup. Web sessions must still be unexpired; mobile
-- (bearer) sessions are revoked rather than expired, so their
-- expires_at sentinel never rejects them. The user columns ride along
-- so the middleware can refuse disabled accounts without a second
-- round trip.
SELECT s.id, s.user_id, s.token_hash, s.kind, s.device_name, s.device_id,
       s.last_seen_at, s.expires_at, s.created_at, s.updated_at,
       u.email AS user_email, u.role AS user_role, u.is_disabled AS user_is_disabled
FROM sessions s
JOIN users u ON u.id = s.user_id
WHERE s.token_hash = $1
  AND (s.kind = 'mobile' OR s.expires_at > now());

-- name: GetSession :one
SELECT * FROM sessions WHERE id = $1;

-- name: TouchSession :exec
-- The throttled sliding renewal: one write per active minute moves
-- last_seen_at to now and slides a web session's expiry by its TTL.
-- Mobile rows keep their expiry -- bearer sessions never slide.
UPDATE sessions
SET last_seen_at = $2,
    expires_at = CASE WHEN kind = 'mobile' THEN expires_at ELSE $3 END,
    updated_at = $2
WHERE id = $1;

-- name: HasActiveSession :one
-- The folder live sync's activity signal: whether anybody was seen
-- after the cutoff. Revoked sessions are deleted rows, so there is
-- nothing to exclude; an expired web row is still a row, and a lapsed
-- cookie is not activity -- mobile (bearer) rows never expire, they
-- are revoked. The cutoff arrives from the caller's clock, the same
-- clock the touch middleware writes last_seen_at with, so the two
-- sides of the comparison never sit in different time frames.
SELECT EXISTS (
    SELECT 1 FROM sessions
    WHERE last_seen_at > sqlc.arg('cutoff')
      AND (kind = 'mobile' OR expires_at > now())
);

-- name: RevokeSession :one
-- Owner-scoped deletion: a foreign or unknown id matches nothing, so
-- the answer of the DELETE /auth/sessions/{id} handler stays 204
-- without revealing whether the session existed. The returned kind is
-- the revoked session's own -- the activity event reports what was
-- actually revoked.
DELETE FROM sessions WHERE id = $1 AND user_id = $2 RETURNING kind;

-- name: RevokeAllUserSessions :execrows
-- The password-change / TOTP-change hook: every session of the user
-- across all devices goes at once. No handler calls this yet;
-- the query is the interface the v1.x account-security flow builds on.
DELETE FROM sessions WHERE user_id = $1;

-- name: ListUserSessions :many
-- The device list: active sessions of one user, most recently used
-- first. Expired web rows (never deleted, just lapsed) are filtered
-- out; revoked rows are already gone.
SELECT * FROM sessions
WHERE user_id = $1
  AND (kind = 'mobile' OR expires_at > now())
ORDER BY last_seen_at DESC NULLS LAST, created_at DESC;
