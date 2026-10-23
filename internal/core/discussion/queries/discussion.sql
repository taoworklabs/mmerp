-- name: CreateComment :one
INSERT INTO discussion.comments (doc_type, doc_id, author_id, body) VALUES ($1, $2, $3, $4) RETURNING id;

-- name: ListComments :many
SELECT c.id, c.body, c.created_at, u.name AS author_name
FROM discussion.comments c JOIN iam.users u ON u.id = c.author_id
WHERE c.doc_type = $1 AND c.doc_id = $2
ORDER BY c.id;

-- name: DeleteRecordComments :exec
DELETE FROM discussion.comments WHERE doc_type = $1 AND doc_id = $2;

-- name: UsersByLogin :many
SELECT id FROM iam.users WHERE lower(login) = ANY(@logins::text[]);

-- name: Users :many
SELECT id, login, name FROM iam.users ORDER BY name, login;
