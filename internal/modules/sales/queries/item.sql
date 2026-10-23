-- name: ItemsByID :many
SELECT * FROM item.items WHERE id = ANY(@ids::bigint[]);

-- name: ItemOptionsBase :many
-- data:base
SELECT id, code, name FROM item.items;
