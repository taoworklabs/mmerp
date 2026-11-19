-- name: InsertTransfer :exec
INSERT INTO stock.transfers (id, from_warehouse_id, to_warehouse_id, note) VALUES ($1, $2, $3, $4);

-- name: UpdateTransfer :exec
UPDATE stock.transfers SET from_warehouse_id = $2, to_warehouse_id = $3, note = $4 WHERE id = $1;

-- name: DeleteTransfer :exec
DELETE FROM stock.transfers WHERE id = $1;

-- name: DeleteTransferLines :exec
DELETE FROM stock.transfer_lines WHERE transfer_id = $1;

-- name: InsertTransferLines :exec
-- No lot or serial travels as ''.
INSERT INTO stock.transfer_lines (transfer_id, line_no, item_id, lot_code, serial_code, quantity)
SELECT @transfer_id::bigint, k.n, k.i, nullif(k.l, ''), nullif(k.s, ''), k.q::numeric
FROM (SELECT unnest(@line_nos::int[]) AS n, unnest(@item_ids::bigint[]) AS i, unnest(@lot_codes::text[]) AS l,
             unnest(@serials::text[]) AS s, unnest(@quantities::text[]) AS q) AS k;

-- name: GetTransfer :one
SELECT t.id, t.from_warehouse_id, t.to_warehouse_id, t.note,
       f.code AS from_warehouse_code, f.name AS from_warehouse_name, w.code AS to_warehouse_code, w.name AS to_warehouse_name,
       d.number, d.status, d.version, d.date, d.legal_entity_id
FROM stock.transfers t
JOIN record.documents d ON d.id = t.id
JOIN stock.warehouses f ON f.id = t.from_warehouse_id
JOIN stock.warehouses w ON w.id = t.to_warehouse_id
WHERE t.id = $1;

-- name: GetTransferLines :many
SELECT l.line_no, l.item_id, i.code AS item_code, i.name AS item_name, i.unit, l.lot_code, l.serial_code, l.quantity::text AS quantity
FROM stock.transfer_lines l JOIN item.items i ON i.id = l.item_id
WHERE l.transfer_id = $1 ORDER BY l.line_no;

-- name: TransferListBase :many
-- data:base
SELECT t.id, d.number, d.status, d.date, d.org_unit_id, t.from_warehouse_id, f.code AS from_warehouse_code, f.name AS from_warehouse_name,
       t.to_warehouse_id, w.code AS to_warehouse_code, w.name AS to_warehouse_name, t.note
FROM stock.transfers t
JOIN record.documents d ON d.id = t.id
JOIN stock.warehouses f ON f.id = t.from_warehouse_id
JOIN stock.warehouses w ON w.id = t.to_warehouse_id;
