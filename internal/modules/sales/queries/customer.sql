-- name: CreateCustomer :one
INSERT INTO sales.customers (code, name, tax_code, address, phone, email, contact_name, payment_terms, org_unit_id, active)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id;

-- name: UpdateCustomer :exec
UPDATE sales.customers
SET name = $2, tax_code = $3, address = $4, phone = $5, email = $6, contact_name = $7,
    payment_terms = $8, org_unit_id = $9, active = $10
WHERE id = $1;

-- name: GetCustomer :one
SELECT c.*, u.name AS org_unit_name
FROM sales.customers c JOIN iam.org_units u ON u.id = c.org_unit_id
WHERE c.id = $1;

-- name: LockCustomer :exec
-- NO KEY: documents referencing the customer stay creatable meanwhile.
SELECT 1 FROM sales.customers WHERE id = $1 FOR NO KEY UPDATE;

-- name: CustomerActive :one
-- Read under a share lock, so the customer cannot be deactivated while a document posts.
SELECT active FROM sales.customers WHERE id = $1 FOR SHARE;

-- name: CustomerListBase :many
-- data:base
SELECT c.id, c.code, c.name, c.tax_code, c.phone, c.payment_terms, c.org_unit_id, u.name AS org_unit_name, c.active
FROM sales.customers c JOIN iam.org_units u ON u.id = c.org_unit_id;

-- name: CustomerCodeTaken :one
SELECT EXISTS (SELECT 1 FROM sales.customers WHERE lower(code) = lower($1));
