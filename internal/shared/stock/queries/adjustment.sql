-- name: AdjustmentWarehouse :one
SELECT legal_entity_id, active FROM stock.warehouses WHERE id = $1;

-- name: CountItems :one
SELECT count(*) FROM item.items WHERE id = ANY(@ids::bigint[]);

-- name: InsertAdjustment :exec
INSERT INTO stock.adjustments (id, warehouse_id, reason) VALUES ($1, $2, $3);

-- name: UpdateAdjustment :exec
UPDATE stock.adjustments SET warehouse_id = $2, reason = $3 WHERE id = $1;

-- name: DeleteAdjustment :exec
DELETE FROM stock.adjustments WHERE id = $1;

-- name: DeleteAdjustmentLines :exec
DELETE FROM stock.adjustment_lines WHERE adjustment_id = $1;

-- name: InsertAdjustmentLines :exec
-- No lot, expiry or serial travels as ''.
INSERT INTO stock.adjustment_lines (adjustment_id, line_no, item_id, lot_code, lot_expiry, serial_code, quantity)
SELECT @adjustment_id::bigint, k.n, k.i, nullif(k.l, ''), nullif(k.e, '')::date, nullif(k.s, ''), k.q::numeric
FROM (SELECT unnest(@line_nos::int[]) AS n, unnest(@item_ids::bigint[]) AS i, unnest(@lot_codes::text[]) AS l,
             unnest(@lot_expiries::text[]) AS e, unnest(@serials::text[]) AS s, unnest(@quantities::text[]) AS q) AS k;

-- name: GetAdjustment :one
SELECT a.id, a.warehouse_id, a.reason, w.code AS warehouse_code, w.name AS warehouse_name,
       d.number, d.status, d.version, d.date, d.legal_entity_id
FROM stock.adjustments a
JOIN record.documents d ON d.id = a.id
JOIN stock.warehouses w ON w.id = a.warehouse_id
WHERE a.id = $1;

-- name: GetAdjustmentLines :many
SELECT l.line_no, l.item_id, i.code AS item_code, i.name AS item_name, i.unit, l.lot_code, l.lot_expiry, l.serial_code, l.quantity::text AS quantity
FROM stock.adjustment_lines l JOIN item.items i ON i.id = l.item_id
WHERE l.adjustment_id = $1 ORDER BY l.line_no;

-- name: AdjustmentListBase :many
-- data:base
SELECT a.id, d.number, d.status, d.date, d.org_unit_id, a.warehouse_id, w.code AS warehouse_code, w.name AS warehouse_name, a.reason
FROM stock.adjustments a
JOIN record.documents d ON d.id = a.id
JOIN stock.warehouses w ON w.id = a.warehouse_id;
