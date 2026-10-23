-- Allowed: a core module reads its own schema.
-- name: User :one
SELECT * FROM iam.users WHERE id = $1;
