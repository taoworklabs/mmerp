-- name: CreateSupplier :one
INSERT INTO purchase.suppliers (code, partner_id, payment_terms, org_unit_id, active)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;

-- name: UpdateSupplier :exec
UPDATE purchase.suppliers SET payment_terms = $2, org_unit_id = $3, active = $4 WHERE id = $1;

-- name: GetSupplier :one
-- The supplier role with its partner's identity.
SELECT s.id, s.code, s.partner_id, p.code AS partner_code, p.name, p.tax_code, p.address, p.phone, p.email,
       p.contact_name, s.payment_terms, s.org_unit_id, s.active, u.name AS org_unit_name
FROM purchase.suppliers s
JOIN partner.partners p ON p.id = s.partner_id
JOIN iam.org_units u ON u.id = s.org_unit_id
WHERE s.id = $1;

-- name: LockSupplier :exec
-- NO KEY: documents referencing the supplier stay creatable meanwhile. Also orders supplier
-- invoices being checked for duplicates.
SELECT 1 FROM purchase.suppliers WHERE id = $1 FOR NO KEY UPDATE;

-- name: SupplierActive :one
-- Read under a share lock, so the supplier cannot be deactivated while a document posts.
SELECT active FROM purchase.suppliers WHERE id = $1 FOR SHARE;

-- name: SupplierListBase :many
-- data:base
SELECT s.id, s.code, s.partner_id, p.name, p.tax_code, p.phone, s.payment_terms, s.org_unit_id, u.name AS org_unit_name, s.active
FROM purchase.suppliers s
JOIN partner.partners p ON p.id = s.partner_id
JOIN iam.org_units u ON u.id = s.org_unit_id;

-- name: SupplierCodeTaken :one
SELECT EXISTS (SELECT 1 FROM purchase.suppliers WHERE lower(code) = lower($1));

-- name: AnyCatalog :one
SELECT EXISTS (SELECT 1 FROM purchase.suppliers)::bool AS any;
