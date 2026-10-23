-- name: CreatePartner :one
INSERT INTO partner.partners (code, name, tax_code, address, phone, email, contact_name)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id;

-- name: UpdatePartner :exec
UPDATE partner.partners
SET name = $2, tax_code = $3, address = $4, phone = $5, email = $6, contact_name = $7
WHERE id = $1;

-- name: GetPartner :one
SELECT * FROM partner.partners WHERE id = $1;

-- name: LockPartner :one
-- NO KEY: customers referencing the partner stay creatable meanwhile.
SELECT * FROM partner.partners WHERE id = $1 FOR NO KEY UPDATE;

-- name: PartnerByTaxCode :one
SELECT id FROM partner.partners WHERE lower(btrim(tax_code)) = lower(btrim(@tax_code::text));

-- name: PartnerCodeTaken :one
SELECT EXISTS (SELECT 1 FROM partner.partners WHERE lower(code) = lower($1));

-- name: PartnerListBase :many
-- data:base
SELECT id, code, name, tax_code, address, phone, email, contact_name FROM partner.partners;
