-- name: CreateUser :one
INSERT INTO iam.users (login, name, password_hash) VALUES ($1, $2, $3) RETURNING id;

-- name: GetUserByLogin :one
SELECT id, password_hash FROM iam.users WHERE lower(login) = lower($1);

-- name: GetUser :one
SELECT u.id, u.login, u.name, u.email, u.locale, u.authz_version, (SELECT version FROM iam.authz) AS tenant_version
FROM iam.users u WHERE u.id = $1;

-- name: ListUsers :many
SELECT id, login, name, email FROM iam.users ORDER BY lower(login);

-- name: SetUserEmail :execrows
UPDATE iam.users SET email = $2 WHERE id = $1;

-- name: BumpUserAuthz :exec
UPDATE iam.users SET authz_version = authz_version + 1 WHERE id = $1;

-- name: BumpTenantAuthz :exec
UPDATE iam.authz SET version = version + 1;

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

-- name: UserRoles :many
SELECT r.id, r.product, r.role, r.org_unit_id, u.name AS org_unit_name
FROM iam.user_roles r LEFT JOIN iam.org_units u ON u.id = r.org_unit_id
WHERE r.user_id = $1 ORDER BY r.product, r.role, u.name NULLS FIRST;

-- name: GrantRole :one
INSERT INTO iam.user_roles (user_id, product, role, org_unit_id) VALUES ($1, $2, $3, $4) RETURNING id;

-- name: RevokeRole :one
DELETE FROM iam.user_roles WHERE id = $1 AND user_id = $2 RETURNING product, role, org_unit_id;

-- name: SetLocale :exec
UPDATE iam.users SET locale = $2 WHERE id = $1;

-- name: CreateSession :exec
-- Fixed lifetime from login; activity does not extend it.
INSERT INTO iam.sessions (token_hash, user_id, expires_at) VALUES ($1, $2, now() + interval '14 days');

-- name: DeleteExpiredSessions :exec
DELETE FROM iam.sessions WHERE user_id = $1 AND expires_at <= now();

-- name: GetSession :one
SELECT s.user_id, u.authz_version, (SELECT version FROM iam.authz) AS tenant_version
FROM iam.sessions s JOIN iam.users u ON u.id = s.user_id
WHERE s.token_hash = $1 AND s.expires_at > now();

-- name: DeleteSession :one
DELETE FROM iam.sessions WHERE token_hash = $1 RETURNING user_id;

-- name: LegalEntityOf :one
-- The nearest company at or above the unit.
WITH RECURSIVE up AS (
    SELECT o.id, o.parent_id, o.kind, 0 AS depth FROM iam.org_units o WHERE o.id = $1
    UNION ALL
    SELECT u.id, u.parent_id, u.kind, up.depth + 1 FROM iam.org_units u JOIN up ON u.id = up.parent_id
    WHERE up.kind <> 'company'
)
SELECT up.id FROM up WHERE up.kind = 'company' ORDER BY up.depth LIMIT 1;

-- name: UsersWithRole :many
-- Users holding the role tenant-wide or at the unit or one of its ancestors.
WITH RECURSIVE up AS (
    SELECT o.id, o.parent_id FROM iam.org_units o WHERE o.id = @unit
    UNION
    SELECT u.id, u.parent_id FROM iam.org_units u JOIN up ON u.id = up.parent_id
)
SELECT DISTINCT r.user_id FROM iam.user_roles r
WHERE r.product = @product AND r.role = @role AND (r.org_unit_id IS NULL OR r.org_unit_id IN (SELECT up.id FROM up))
ORDER BY r.user_id;
