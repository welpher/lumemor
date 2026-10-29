-- User queries for the first-run wizard and login. CreateUser hardcodes
-- role='admin' because the M1 wizard only creates the initial admin;
-- parameterizing the role is M2 work.

-- name: CreateUser :one
INSERT INTO users (id, name, email, password_hash, role, locale)
VALUES ($1, $2, $3, $4, 'admin', 'en') RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: CountAdmins :one
SELECT count(*) FROM users WHERE role = 'admin';
