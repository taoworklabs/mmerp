-- name: CreateFile :one
INSERT INTO attachment.files (doc_type, doc_id, file_id, name, size, content_type, uploaded_by)
VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id;

-- name: GetFile :one
SELECT * FROM attachment.files WHERE id = $1;

-- name: ListFiles :many
SELECT f.*, u.name AS uploaded_by_name
FROM attachment.files f JOIN iam.users u ON u.id = f.uploaded_by
WHERE f.doc_type = $1 AND f.doc_id = $2
ORDER BY f.id;

-- name: DeleteFile :one
DELETE FROM attachment.files WHERE id = $1 RETURNING *;

-- name: KnownFiles :many
SELECT file_id FROM attachment.files WHERE file_id = ANY(@ids::text[]);

-- name: DeleteRecordFiles :exec
DELETE FROM attachment.files WHERE doc_type = $1 AND doc_id = $2;
