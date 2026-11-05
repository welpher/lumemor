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

-- User reads of the sharing face (M3c Task 3): the minimal member
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

-- name: GetUserForShareTarget :one
-- One account as the share grant's target lookup: does the id name
-- a usable account? An unknown id answers no row and a disabled
-- account answers its switch -- the caller folds both into the same
-- 400 so user ids cannot be probed through the sharing face.
SELECT id, email, is_disabled
FROM users
WHERE id = $1;

-- name: GetUserRole :one
-- The role half of the pairing consume's identity answer: the burn
-- answers the token creator's id, and the minted session's response
-- carries the account's role with it, so the paired device knows who
-- it paired as from the consume response alone.
SELECT role FROM users WHERE id = $1;
