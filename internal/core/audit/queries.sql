-- name: Insert :exec
INSERT INTO audit.log (actor_id, action, data, doc_type, doc_id) VALUES ($1, $2, $3, $4, $5);
