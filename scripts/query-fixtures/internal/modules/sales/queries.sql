-- Allowed: own schema, a declared dependency product, and the tiers below.
-- name: OrderWithItem :one
SELECT o.total, i.name, d.status, l.amount
FROM sales.orders o
JOIN inventory.items i ON i.id = o.item_id
JOIN record.documents d ON d.id = o.document_id
JOIN posting.lines l ON l.document_id = d.id
WHERE o.id = $1;

-- Planted violation: hrm is not a declared dependency of sales.
-- name: OrderSeller :one
SELECT e.full_name FROM hrm.employees e WHERE e.id = $1;

-- Planted violation: a module writes only the schema it owns, dependency or not.
-- name: ReserveItem :exec
UPDATE inventory.items SET reserved = reserved + $2 WHERE id = $1;
