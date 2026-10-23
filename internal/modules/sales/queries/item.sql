-- name: CreateItem :one
INSERT INTO sales.items (code, name, unit, price, vat_rate, active) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id;

-- name: UpdateItem :one
UPDATE sales.items SET name = $2, unit = $3, price = $4, vat_rate = $5, active = $6 WHERE id = $1 RETURNING code;

-- name: ItemCodeTaken :one
SELECT EXISTS (SELECT 1 FROM sales.items WHERE lower(code) = lower($1));

-- name: GetItem :one
SELECT * FROM sales.items WHERE id = $1;

-- name: ItemsByID :many
SELECT * FROM sales.items WHERE id = ANY(@ids::bigint[]);

-- name: ItemListBase :many
-- data:base
SELECT id, code, name, unit, price, vat_rate, active FROM sales.items;

-- name: AnyCatalog :one
-- Items count as Sales data too, though they are no record type.
SELECT (EXISTS (SELECT 1 FROM sales.customers) OR EXISTS (SELECT 1 FROM sales.items))::bool AS any;
