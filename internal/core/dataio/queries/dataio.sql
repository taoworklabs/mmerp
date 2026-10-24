-- name: CreateFile :exec
INSERT INTO dataio.files (id, name, owner_id, expires_at, export_kind, export_params) VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetFile :one
SELECT * FROM dataio.files WHERE id = $1;

-- name: DeleteExpiredFiles :many
DELETE FROM dataio.files WHERE expires_at < now() RETURNING id;

-- name: KnownFiles :many
SELECT id FROM dataio.files WHERE id = ANY(@ids::text[]);
