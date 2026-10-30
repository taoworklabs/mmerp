-- name: InsertDelivery :exec
INSERT INTO sales.deliveries (id, order_id, warehouse_id, note) VALUES ($1, $2, $3, $4);

-- name: UpdateDelivery :exec
UPDATE sales.deliveries SET warehouse_id = $2, note = $3 WHERE id = $1;

-- name: DeleteDelivery :exec
DELETE FROM sales.deliveries WHERE id = $1;

-- name: DeliveryOrder :one
SELECT order_id FROM sales.deliveries WHERE id = $1;

-- name: DeliveryWarehouse :one
SELECT legal_entity_id, active FROM stock.warehouses WHERE id = $1;

-- name: GetDelivery :one
SELECT v.order_id, v.warehouse_id, v.note, d.number, d.status, d.version, d.date, d.org_unit_id,
       o.number AS order_number, w.code AS warehouse_code, w.name AS warehouse_name
FROM sales.deliveries v
JOIN record.documents d ON d.id = v.id
JOIN record.documents o ON o.id = v.order_id
JOIN stock.warehouses w ON w.id = v.warehouse_id
WHERE v.id = $1;

-- name: GetDeliveryLines :many
SELECT order_position, item_id, item_code, description, unit, lot_code, serial_code, quantity::text AS quantity
FROM sales.delivery_lines WHERE delivery_id = $1 ORDER BY position;

-- name: DeliverableLines :many
-- An order's lines with their item's tracking and what deliveries other than this one,
-- not cancelled, already take of each.
SELECT l.position, l.item_id, l.item_code, l.description, l.unit, l.quantity::text AS quantity, i.tracking,
       coalesce((
           SELECT sum(dl.quantity)
           FROM sales.delivery_lines dl
           JOIN sales.deliveries v ON v.id = dl.delivery_id
           JOIN record.documents d ON d.id = v.id
           WHERE v.order_id = l.doc_id AND dl.order_position = l.position AND v.id <> @delivery_id AND d.status <> 'cancelled'
       ), 0)::text AS taken
FROM sales.lines l
JOIN item.items i ON i.id = l.item_id
WHERE l.doc_id = @order_id
ORDER BY l.position;

-- name: DeleteDeliveryLines :exec
DELETE FROM sales.delivery_lines WHERE delivery_id = $1;

-- name: InsertDeliveryLines :exec
INSERT INTO sales.delivery_lines (delivery_id, position, order_position, item_id, item_code, description, unit,
    lot_code, serial_code, quantity)
SELECT @delivery_id::bigint, unnest(@positions::int[]), unnest(@order_positions::int[]), unnest(@item_ids::bigint[]),
    unnest(@item_codes::text[]), unnest(@descriptions::text[]), unnest(@units::text[]),
    nullif(unnest(@lot_codes::text[]), ''), nullif(unnest(@serial_codes::text[]), ''), unnest(@quantities::text[])::numeric;

-- name: LiveDeliveryOfOrder :one
-- Any delivery of the order that is not cancelled.
SELECT v.id FROM sales.deliveries v JOIN record.documents d ON d.id = v.id
WHERE v.order_id = $1 AND d.status <> 'cancelled' LIMIT 1;
