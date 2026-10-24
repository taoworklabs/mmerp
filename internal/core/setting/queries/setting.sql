-- name: GetValue :one
SELECT value FROM setting.values WHERE key = $1;

-- name: IsLegalEntity :one
SELECT EXISTS (SELECT 1 FROM iam.org_units WHERE id = $1 AND kind = 'company');

-- name: UnitLegalEntityValue :one
-- The value of key at the nearest company at or above the unit; null when none is stored.
WITH RECURSIVE up AS (
    SELECT o.id, o.parent_id, o.kind, 0 AS depth FROM iam.org_units o WHERE o.id = @unit
    UNION ALL
    SELECT u.id, u.parent_id, u.kind, up.depth + 1 FROM iam.org_units u JOIN up ON u.id = up.parent_id
    WHERE up.kind <> 'company'
), le AS (
    SELECT up.id FROM up WHERE up.kind = 'company' ORDER BY up.depth LIMIT 1
)
SELECT v.value FROM le LEFT JOIN setting.legal_entity_values v ON v.legal_entity_id = le.id AND v.key = @key;

-- name: LegalEntityValues :many
SELECT key, value FROM setting.legal_entity_values WHERE legal_entity_id = $1;

-- name: GetLegalEntityValue :one
SELECT value FROM setting.legal_entity_values WHERE legal_entity_id = $1 AND key = $2;

-- name: SetLegalEntityValue :exec
INSERT INTO setting.legal_entity_values (legal_entity_id, key, value) VALUES ($1, $2, $3)
ON CONFLICT (legal_entity_id, key) DO UPDATE SET value = excluded.value;

-- name: Today :one
SELECT (now() AT TIME ZONE @tz::text)::date;
