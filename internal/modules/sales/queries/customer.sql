-- name: CreateCustomer :one
INSERT INTO sales.customers (code, partner_id, payment_terms, org_unit_id, active)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;

-- name: UpdateCustomer :exec
UPDATE sales.customers SET payment_terms = $2, org_unit_id = $3, active = $4 WHERE id = $1;

-- name: GetCustomer :one
-- The customer role with its partner's identity.
SELECT c.id, c.code, c.partner_id, p.code AS partner_code, p.name, p.tax_code, p.address, p.phone, p.email,
       p.contact_name, c.payment_terms, c.org_unit_id, c.active, u.name AS org_unit_name
FROM sales.customers c
JOIN partner.partners p ON p.id = c.partner_id
JOIN iam.org_units u ON u.id = c.org_unit_id
WHERE c.id = $1;

-- name: LockCustomer :exec
-- NO KEY: documents referencing the customer stay creatable meanwhile.
SELECT 1 FROM sales.customers WHERE id = $1 FOR NO KEY UPDATE;

-- name: CustomerActive :one
-- Read under a share lock, so the customer cannot be deactivated while a document posts.
SELECT active FROM sales.customers WHERE id = $1 FOR SHARE;

-- name: CustomerListBase :many
-- data:base
SELECT c.id, c.code, c.partner_id, p.name, p.tax_code, p.phone, c.payment_terms, c.org_unit_id, u.name AS org_unit_name, c.active
FROM sales.customers c
JOIN partner.partners p ON p.id = c.partner_id
JOIN iam.org_units u ON u.id = c.org_unit_id;

-- name: CustomerCodeTaken :one
SELECT EXISTS (SELECT 1 FROM sales.customers WHERE lower(code) = lower($1));

-- name: AnyCatalog :one
SELECT EXISTS (SELECT 1 FROM sales.customers)::bool AS any;
