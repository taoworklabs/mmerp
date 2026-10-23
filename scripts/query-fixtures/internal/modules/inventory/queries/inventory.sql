-- Allowed: own schema and the tiers below.
-- name: ItemOfDocument :one
SELECT i.name, d.status FROM inventory.items i JOIN record.documents d ON d.id = i.document_id WHERE i.id = $1;

-- Planted violation: sales declares inventory, not the other way round; a dependency
-- never reads back into the product that declared it.
-- name: ItemOrders :many
SELECT o.total FROM sales.orders o WHERE o.item_id = $1;
