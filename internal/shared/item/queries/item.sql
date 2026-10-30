-- name: CreateItem :one
INSERT INTO item.items (code, name, unit, price, vat_rate, active, tracking, allow_negative) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id;

-- name: UpdateItem :one
UPDATE item.items SET name = $2, unit = $3, price = $4, vat_rate = $5, active = $6,
    tracking = coalesce(sqlc.narg(tracking), tracking), allow_negative = coalesce(sqlc.narg(allow_negative), allow_negative)
WHERE id = $1 RETURNING code;

-- name: ItemCodeTaken :one
SELECT EXISTS (SELECT 1 FROM item.items WHERE lower(code) = lower($1));

-- name: ItemListBase :many
-- data:base
SELECT id, code, name, unit, price, vat_rate, active, tracking, allow_negative FROM item.items;
