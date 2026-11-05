-- name: ListStarts :many
SELECT o.id AS legal_entity_id, s.start_date
FROM iam.org_units o LEFT JOIN accounting.starts s ON s.legal_entity_id = o.id
WHERE o.kind = 'company' ORDER BY o.id;

-- name: IsLegalEntity :one
SELECT EXISTS (SELECT 1 FROM iam.org_units WHERE id = $1 AND kind = 'company');

-- name: StartOf :one
SELECT start_date FROM accounting.starts WHERE legal_entity_id = $1;

-- name: SetStart :exec
INSERT INTO accounting.starts (legal_entity_id, start_date) VALUES (@legal_entity_id, @start_date)
ON CONFLICT (legal_entity_id) DO UPDATE SET start_date = EXCLUDED.start_date;

-- name: ClearStart :exec
DELETE FROM accounting.starts WHERE legal_entity_id = $1;
