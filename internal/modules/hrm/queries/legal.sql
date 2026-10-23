-- name: ListLegalParams :many
SELECT key, effective_from, value FROM hrm.legal_params ORDER BY key, effective_from;

-- name: LegalParamsAt :many
-- Each key's version in force at @at.
SELECT DISTINCT ON (key) key, effective_from, value FROM hrm.legal_params
WHERE effective_from <= @at::date ORDER BY key, effective_from DESC;

-- name: GetLegalParam :one
SELECT value FROM hrm.legal_params WHERE key = $1 AND effective_from = $2;

-- name: SaveLegalParam :exec
INSERT INTO hrm.legal_params (key, effective_from, value) VALUES ($1, $2, $3)
ON CONFLICT (key, effective_from) DO UPDATE SET value = excluded.value;

-- name: LegalEntities :many
SELECT id FROM iam.org_units WHERE kind = 'company' ORDER BY id;
