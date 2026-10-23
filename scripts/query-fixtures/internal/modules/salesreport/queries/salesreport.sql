-- Allowed: own schema, a sibling module of the same product, and that product's
-- declared dependency.
-- name: ReportRow :one
SELECT c.built_at, o.total, i.name
FROM salesreport.cache c
JOIN sales.orders o ON o.id = c.order_id
JOIN inventory.items i ON i.id = o.item_id
WHERE c.id = $1;

-- Planted violation: the dependency belongs to the product, and sales does not
-- declare hrm.
-- name: ReportSeller :one
SELECT e.full_name FROM hrm.employees e WHERE e.id = $1;
