-- name: CreateSession :exec
-- Fixed lifetime from login; activity does not extend it.
INSERT INTO iam.sessions (token_hash, user_id, user_agent, expires_at) VALUES (@token_hash, @user_id, @user_agent, now() + interval '14 days');

-- name: DeleteExpiredSessions :exec
DELETE FROM iam.sessions WHERE user_id = $1 AND expires_at <= now();

-- name: GetSession :one
SELECT s.user_id, u.authz_version, u.must_change_password, (SELECT version FROM iam.authz) AS tenant_version
FROM iam.sessions s JOIN iam.users u ON u.id = s.user_id
WHERE s.token_hash = $1 AND s.expires_at > now() AND u.locked_at IS NULL;

-- name: DeleteSession :one
DELETE FROM iam.sessions WHERE token_hash = $1 RETURNING user_id;

-- name: DeleteUserSessions :exec
DELETE FROM iam.sessions WHERE user_id = $1;

-- name: ListUserSessions :many
SELECT id, token_hash, created_at, expires_at, user_agent FROM iam.sessions
WHERE user_id = $1 AND expires_at > now() ORDER BY created_at DESC, id DESC;

-- name: DeleteUserSession :execrows
DELETE FROM iam.sessions WHERE id = $1 AND user_id = $2;
