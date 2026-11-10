-- User queries for the first-run wizard, login and the admin user
-- management surface. CreateUser takes the role as a parameter: the
-- wizard pins 'admin' (the one-time first administrator) and the
-- admin surface provisions 'user' or 'admin' accounts. Credentials
-- are stored only as their Argon2id hash; the admin listing selects
-- the safe columns only -- the password hash never leaves the table
-- through that surface.

-- name: CreateUser :one
INSERT INTO users (id, name, email, password_hash, role, locale)
VALUES ($1, $2, $3, $4, $5, 'en') RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: CountAdmins :one
SELECT count(*) FROM users WHERE role = 'admin';

-- name: ListUsers :many
-- The admin user management listing: the safe identity columns of
-- every account, oldest first (the creation order the surface
-- renders). No password_hash, ever.
SELECT id, name, email, role, is_disabled, created_at
FROM users
ORDER BY created_at ASC, id ASC;

-- name: SetUserDisabled :one
-- The admin surface's enable/disable toggle. The WHERE guard makes
-- the write idempotent (re-setting the current value is the plain
-- success); an unknown id matches no row, which the caller answers
-- with its 404. updated_at rides the write like every users-table
-- mutation.
UPDATE users
SET is_disabled = $2, updated_at = now()
WHERE id = $1
RETURNING id, name, email, role, is_disabled, created_at;

-- User reads of the sharing face: the minimal member
-- list and the share-target lookup. Both select the safe identity
-- columns only -- a password hash never leaves the table through
-- any non-auth surface.

-- name: ListInstanceUsersMinimal :many
-- The minimal member list: every account, email ascending. Any
-- authenticated account may read it (the family-instance posture);
-- the share dialog's pick list is the first consumer.
SELECT id, email, role, is_disabled
FROM users
ORDER BY email ASC;

-- name: ListUsableShareTargets :many
-- The audience write's resolution arm: which of the REQUESTED ids
-- name usable accounts (known, not disabled)? An unknown id matches
-- no row and a disabled account is filtered by the switch -- the
-- handler folds both into the same skipped reason so user ids
-- cannot be probed through the audience face, the discipline the
-- one-row share grant kept before the full-set write folded it
-- in. The array bounds the scan (the contract caps the audience
-- at 200 ids).
SELECT id
FROM users
WHERE id = ANY($1::uuid[])
  AND NOT is_disabled;

-- name: GetUserRole :one
-- The role half of the pairing consume's identity answer: the burn
-- answers the token creator's id, and the minted session's response
-- carries the account's role with it, so the paired device knows who
-- it paired as from the consume response alone.
SELECT role FROM users WHERE id = $1;

-- name: GetUserSelf :one
-- The me-scoped profile read (the settings feature): the caller's
-- own row. The password hash rides along for the password change's
-- verify step -- the hash never travels past the handler's mapping.
SELECT id, name, email, password_hash, role, locale, theme,
       download_format, download_include_videos, download_include_sidecars,
       is_disabled, totp_secret, created_at, updated_at
FROM users WHERE id = $1;

-- name: UpdateUserSelf :one
-- The me-scoped profile patch (the settings feature): every column
-- is COALESCE'd with the stored value, so an absent field keeps it
-- and a present one replaces it -- one statement, one round trip,
-- and the email's unique constraint is the 409's race-free guard.
-- The handler reads the returned row for the response, so the
-- answer is the stored state, never the request's echo.
UPDATE users SET
    name = COALESCE(sqlc.narg('name'), name),
    email = COALESCE(sqlc.narg('email'), email),
    locale = COALESCE(sqlc.narg('locale'), locale),
    theme = COALESCE(sqlc.narg('theme'), theme),
    download_format = COALESCE(sqlc.narg('download_format'), download_format),
    download_include_videos = COALESCE(sqlc.narg('download_include_videos'), download_include_videos),
    download_include_sidecars = COALESCE(sqlc.narg('download_include_sidecars'), download_include_sidecars),
    updated_at = now()
WHERE id = sqlc.arg('id')
RETURNING id, name, email, password_hash, role, locale, theme,
          download_format, download_include_videos, download_include_sidecars,
          is_disabled, totp_secret, created_at, updated_at;

-- name: UpdateUserPassword :execrows
-- The password change's credential write: a fresh Argon2id hash
-- over the same row (a rehash with a new salt, never a mutation of
-- the old hash). The session revocation of the change rides its
-- own statement so the two writes share one transaction.
UPDATE users SET password_hash = sqlc.arg('password_hash'), updated_at = now()
WHERE id = sqlc.arg('id');

-- name: RevokeOtherUserSessions :execrows
-- The password change's session severance: every session of the
-- account EXCEPT the one performing the change (the caller stays
-- signed in on the device they just proved they own; every other
-- held credential is stale). Revocation is row deletion, the same
-- rule as the auth surface's single revoke.
DELETE FROM sessions WHERE user_id = sqlc.arg('user_id') AND id <> sqlc.arg('keep_id');
