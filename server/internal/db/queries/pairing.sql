-- Pairing token queries for the device pairing flow. The token is
-- stored only as its SHA-256 hex hash, exactly like a session
-- credential, so a leaked database cannot redeem outstanding QR
-- codes. One-timeness is the conditional UPDATE below: a row that is
-- already used, expired, or unknown simply matches nothing.

-- name: CreatePairingToken :one
-- The caller supplies the expiry so the issued response and the row
-- agree on the five-minute TTL.
INSERT INTO pairing_tokens (id, token_hash, created_by, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ConsumePairingToken :one
-- The atomic burn. The conditional UPDATE is the one-time guarantee:
-- only an unused, unexpired row matches, the first consumer stamps
-- used_at, and every later attempt -- a replay, an expiry, a guess --
-- finds no row and can never learn which it was. Run inside the same
-- transaction as the mobile session's CreateSession so a crash
-- between burn and session creation can never strand a burned token
-- without its session (or, rolled back, mint a session for free).
UPDATE pairing_tokens
SET used_at = now()
WHERE token_hash = $1
  AND used_at IS NULL
  AND expires_at > now()
RETURNING id, created_by;
