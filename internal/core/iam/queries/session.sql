-- name: CreateSession :exec
-- Fixed lifetime from login; activity does not extend it.
INSERT INTO iam.sessions (token_hash, user_id, expires_at) VALUES ($1, $2, now() + interval '14 days');

-- name: DeleteExpiredSessions :exec
DELETE FROM iam.sessions WHERE user_id = $1 AND expires_at <= now();

-- name: GetSession :one
SELECT s.user_id, u.authz_version, (SELECT version FROM iam.authz) AS tenant_version
FROM iam.sessions s JOIN iam.users u ON u.id = s.user_id
WHERE s.token_hash = $1 AND s.expires_at > now();

-- name: DeleteSession :one
DELETE FROM iam.sessions WHERE token_hash = $1 RETURNING user_id;
