-- name: CreateUser :one
INSERT INTO iam.users (login, name, password_hash) VALUES ($1, $2, $3) RETURNING id;

-- name: CreateRoot :one
INSERT INTO iam.users (login, name, password_hash, is_root, must_change_password) VALUES ($1, $1, $2, true, true) RETURNING id;

-- name: GetUserByLogin :one
SELECT id, password_hash, locked_at, is_root FROM iam.users WHERE lower(login) = lower($1);

-- name: LockUserCredentials :one
-- Waits for a password change or lock in flight, so a sign-in sees what it committed.
SELECT password_hash, locked_at IS NOT NULL AS locked FROM iam.users WHERE id = $1 FOR SHARE;

-- name: GetUserPassword :one
SELECT password_hash FROM iam.users WHERE id = $1;

-- name: SetUserPassword :exec
UPDATE iam.users SET password_hash = @password_hash, must_change_password = @must_change, authz_version = authz_version + 1 WHERE id = @id;

-- name: DeleteOtherUserSessions :execrows
DELETE FROM iam.sessions WHERE user_id = $1 AND token_hash <> $2;

-- name: GetUser :one
SELECT u.id, u.login, u.name, u.email, u.locale, u.authz_version, u.must_change_password, u.locked_at, u.is_root, (SELECT version FROM iam.authz) AS tenant_version
FROM iam.users u WHERE u.id = $1;

-- name: ListUsers :many
SELECT id, login, name, email, locked_at, is_root FROM iam.users ORDER BY lower(login);

-- name: SetUserEmail :execrows
UPDATE iam.users SET email = $2 WHERE id = $1;

-- name: SetUserLocked :exec
UPDATE iam.users SET locked_at = CASE WHEN @locked::boolean THEN now() END, authz_version = authz_version + 1 WHERE id = @id;

-- name: SetUserName :exec
UPDATE iam.users SET name = $2, authz_version = authz_version + 1 WHERE id = $1;

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
