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
