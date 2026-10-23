-- name: UserRoles :many
SELECT r.id, r.product, r.role, r.org_unit_id, u.name AS org_unit_name
FROM iam.user_roles r LEFT JOIN iam.org_units u ON u.id = r.org_unit_id
WHERE r.user_id = $1 ORDER BY r.product, r.role, u.name NULLS FIRST;

-- name: GrantRole :one
INSERT INTO iam.user_roles (user_id, product, role, org_unit_id) VALUES ($1, $2, $3, $4) RETURNING id;

-- name: RevokeRole :one
DELETE FROM iam.user_roles WHERE id = $1 AND user_id = $2 RETURNING product, role, org_unit_id;

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
