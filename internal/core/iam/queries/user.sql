-- name: CreateUser :one
INSERT INTO iam.users (login, name, password_hash) VALUES ($1, $2, $3) RETURNING id;

-- name: GetUserByLogin :one
SELECT id, password_hash FROM iam.users WHERE lower(login) = lower($1);

-- name: GetUser :one
SELECT u.id, u.login, u.name, u.email, u.locale, u.authz_version, (SELECT version FROM iam.authz) AS tenant_version
FROM iam.users u WHERE u.id = $1;

-- name: ListUsers :many
SELECT id, login, name, email FROM iam.users ORDER BY lower(login);

-- name: SetUserEmail :execrows
UPDATE iam.users SET email = $2 WHERE id = $1;

-- name: BumpUserAuthz :exec
UPDATE iam.users SET authz_version = authz_version + 1 WHERE id = $1;

-- name: BumpTenantAuthz :exec
UPDATE iam.authz SET version = version + 1;

-- name: SetLocale :exec
UPDATE iam.users SET locale = $2 WHERE id = $1;

-- name: ListSettings :one
SELECT settings FROM iam.list_settings WHERE user_id = $1 AND list_key = $2;

-- name: SetListSettings :exec
INSERT INTO iam.list_settings (user_id, list_key, settings) VALUES ($1, $2, $3)
ON CONFLICT (user_id, list_key) DO UPDATE SET settings = EXCLUDED.settings;
