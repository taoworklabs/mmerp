-- name: GetRule :one
SELECT * FROM approval.rules WHERE doc_type = $1 AND action = $2;

-- name: ListRules :many
SELECT * FROM approval.rules ORDER BY doc_type, action;

-- name: SaveRule :exec
INSERT INTO approval.rules (doc_type, action, flow, max_levels, fallback_product, fallback_role) VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (doc_type, action) DO UPDATE SET flow = $3, max_levels = $4, fallback_product = $5, fallback_role = $6;

-- name: DeleteRule :exec
DELETE FROM approval.rules WHERE doc_type = $1 AND action = $2;
