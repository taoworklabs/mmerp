-- name: InsertGoodsReceipt :exec
INSERT INTO purchase.goods_receipts (id, order_id, warehouse_id, note) VALUES ($1, $2, $3, $4);

-- name: UpdateGoodsReceipt :exec
UPDATE purchase.goods_receipts SET warehouse_id = $2, note = $3 WHERE id = $1;

-- name: DeleteGoodsReceipt :exec
DELETE FROM purchase.goods_receipts WHERE id = $1;

-- name: GoodsReceiptOrder :one
SELECT order_id FROM purchase.goods_receipts WHERE id = $1;

-- name: ReceiptWarehouse :one
SELECT legal_entity_id, active FROM stock.warehouses WHERE id = $1;

-- name: GetGoodsReceipt :one
SELECT r.order_id, r.warehouse_id, r.note, d.number, d.status, d.version, d.date, d.org_unit_id,
       o.number AS order_number, w.code AS warehouse_code, w.name AS warehouse_name
FROM purchase.goods_receipts r
JOIN record.documents d ON d.id = r.id
JOIN record.documents o ON o.id = r.order_id
JOIN stock.warehouses w ON w.id = r.warehouse_id
WHERE r.id = $1;

-- name: GetGoodsReceiptLines :many
SELECT order_position, item_id, item_code, description, unit, lot_code, lot_expiry, serial_code, quantity::text AS quantity
FROM purchase.goods_receipt_lines WHERE receipt_id = $1 ORDER BY position;

-- name: ReceivableLines :many
-- An order's lines with their item's tracking and what goods receipts other than this one,
-- not cancelled, already take of each.
SELECT l.position, l.item_id, l.item_code, l.description, l.unit, l.stocked, l.quantity::text AS quantity, i.tracking,
       coalesce((
           SELECT sum(rl.quantity)
           FROM purchase.goods_receipt_lines rl
           JOIN purchase.goods_receipts r ON r.id = rl.receipt_id
           JOIN record.documents d ON d.id = r.id
           WHERE r.order_id = l.order_id AND rl.order_position = l.position AND r.id <> @receipt_id AND d.status <> 'cancelled'
       ), 0)::text AS taken
FROM purchase.order_lines l
JOIN item.items i ON i.id = l.item_id
WHERE l.order_id = @order_id
ORDER BY l.position;

-- name: DeleteGoodsReceiptLines :exec
DELETE FROM purchase.goods_receipt_lines WHERE receipt_id = $1;

-- name: InsertGoodsReceiptLines :exec
INSERT INTO purchase.goods_receipt_lines (receipt_id, position, order_position, item_id, item_code, description, unit,
    lot_code, lot_expiry, serial_code, quantity)
SELECT @receipt_id::bigint, unnest(@positions::int[]), unnest(@order_positions::int[]), unnest(@item_ids::bigint[]),
    unnest(@item_codes::text[]), unnest(@descriptions::text[]), unnest(@units::text[]),
    nullif(unnest(@lot_codes::text[]), ''), nullif(unnest(@lot_expiries::text[]), '')::date,
    nullif(unnest(@serial_codes::text[]), ''), unnest(@quantities::text[])::numeric;

-- name: LiveReceiptOfOrder :one
-- Any goods receipt of the order that is not cancelled.
SELECT r.id FROM purchase.goods_receipts r JOIN record.documents d ON d.id = r.id
WHERE r.order_id = $1 AND d.status <> 'cancelled' LIMIT 1;

-- name: OrderGoodsReceipts :many
-- The order's goods receipts not cancelled.
SELECT d.id, d.number, d.status
FROM purchase.goods_receipts r JOIN record.documents d ON d.id = r.id
WHERE r.order_id = $1 AND d.status <> 'cancelled'
ORDER BY d.id;

-- name: GoodsReceiptListBase :many
-- data:base
SELECT r.id, d.number, d.status, d.date, d.org_unit_id, r.order_id, o.number AS order_number,
       h.supplier_id, h.supplier_name, r.warehouse_id, w.name AS warehouse_name, u.name AS org_unit_name
FROM purchase.goods_receipts r
JOIN record.documents d ON d.id = r.id
JOIN record.documents o ON o.id = r.order_id
JOIN purchase.orders h ON h.id = r.order_id
JOIN stock.warehouses w ON w.id = r.warehouse_id
JOIN iam.org_units u ON u.id = d.org_unit_id;

-- name: ReceiptWarehouses :many
-- Active warehouses of a legal entity, the default first.
SELECT id, code, name, is_default FROM stock.warehouses
WHERE legal_entity_id = $1 AND active ORDER BY is_default DESC, lower(code), id;
