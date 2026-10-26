CREATE SCHEMA item;

-- The catalogue moves as it is: same ids, so document lines keep pointing at their items.
ALTER TABLE sales.items SET SCHEMA item;

UPDATE numbering.series SET code = 'item.item' WHERE code = 'sales.item';

-- Every Sales role that read the catalogue now holds item viewer at the same scope;
-- catalogue admins become item admins.
INSERT INTO iam.user_roles (user_id, product, role, org_unit_id)
SELECT user_id, 'item', 'viewer', org_unit_id FROM iam.user_roles
WHERE product = 'sales' AND role IN ('staff', 'manager', 'viewer', 'catalog_admin')
ON CONFLICT DO NOTHING;

INSERT INTO iam.user_roles (user_id, product, role, org_unit_id)
SELECT user_id, 'item', 'admin', NULL FROM iam.user_roles
WHERE product = 'sales' AND role = 'catalog_admin'
ON CONFLICT DO NOTHING;

DELETE FROM iam.user_roles WHERE product = 'sales' AND role = 'catalog_admin';

-- Signed-in users pick up the moved grants; a fresh install has none to move.
UPDATE iam.authz SET version = version + 1 WHERE EXISTS (SELECT 1 FROM iam.user_roles WHERE product = 'item');
