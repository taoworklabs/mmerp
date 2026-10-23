-- name: LockOrgUnits :exec
-- Serialises tree writes so concurrent moves cannot build a cycle.
LOCK TABLE iam.org_units IN SHARE ROW EXCLUSIVE MODE;

-- name: ShareOrgUnits :exec
-- Holds tree writes off, not other readers of the tree.
LOCK TABLE iam.org_units IN SHARE MODE;

-- name: ListOrgUnits :many
SELECT id, parent_id, kind, name, tax_code, legal_name, address FROM iam.org_units ORDER BY name, id;

-- name: GetOrgUnit :one
SELECT id, parent_id, kind, name, tax_code, legal_name, address FROM iam.org_units WHERE id = $1;

-- name: CreateOrgUnit :one
INSERT INTO iam.org_units (parent_id, kind, name, tax_code, legal_name, address)
VALUES ($1, $2, $3, $4, $5, $6) RETURNING id;

-- name: UpdateOrgUnit :execrows
UPDATE iam.org_units SET parent_id = $2, kind = $3, name = $4, tax_code = $5, legal_name = $6, address = $7
WHERE id = $1;

-- name: OrgTreeViolations :one
-- Walks down from the roots carrying the nearest company. Nodes not reached sit on a cycle.
WITH RECURSIVE walk AS (
    SELECT id, CASE WHEN kind = 'company' THEN id END AS company, kind NOT IN ('company', 'group') AS bad
    FROM iam.org_units WHERE parent_id IS NULL
    UNION ALL
    SELECT u.id, CASE WHEN u.kind = 'company' THEN u.id ELSE w.company END,
           CASE WHEN u.kind IN ('company', 'group') THEN w.company IS NOT NULL ELSE w.company IS NULL END
    FROM iam.org_units u JOIN walk w ON u.parent_id = w.id
)
SELECT ((SELECT count(*) FROM iam.org_units) - (SELECT count(*) FROM walk))::bigint AS unreachable,
       (SELECT count(*) FROM walk WHERE bad) AS misplaced;

-- name: Subtrees :many
WITH RECURSIVE t AS (
    SELECT id FROM iam.org_units WHERE id = ANY(@roots::bigint[])
    UNION
    SELECT u.id FROM iam.org_units u JOIN t ON u.parent_id = t.id
)
SELECT id FROM t ORDER BY id;

-- name: LegalEntityOf :one
-- The nearest company at or above the unit.
WITH RECURSIVE up AS (
    SELECT o.id, o.parent_id, o.kind, 0 AS depth FROM iam.org_units o WHERE o.id = $1
    UNION ALL
    SELECT u.id, u.parent_id, u.kind, up.depth + 1 FROM iam.org_units u JOIN up ON u.id = up.parent_id
    WHERE up.kind <> 'company'
)
SELECT up.id FROM up WHERE up.kind = 'company' ORDER BY up.depth LIMIT 1;
