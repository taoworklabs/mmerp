-- name: GetValue :one
SELECT value FROM setting.values WHERE key = $1;

-- name: IsLegalEntity :one
SELECT EXISTS (SELECT 1 FROM iam.org_units WHERE id = $1 AND kind = 'company');

-- name: LegalEntityValues :many
SELECT key, value FROM setting.legal_entity_values WHERE legal_entity_id = $1;

-- name: GetLegalEntityValue :one
SELECT value FROM setting.legal_entity_values WHERE legal_entity_id = $1 AND key = $2;

-- name: SetLegalEntityValue :exec
INSERT INTO setting.legal_entity_values (legal_entity_id, key, value) VALUES ($1, $2, $3)
ON CONFLICT (legal_entity_id, key) DO UPDATE SET value = excluded.value;
