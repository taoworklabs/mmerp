-- name: InsertStocktake :exec
INSERT INTO stock.stocktakes (id, warehouse_id, note) VALUES ($1, $2, $3);

-- name: UpdateStocktake :exec
UPDATE stock.stocktakes SET warehouse_id = $2, note = $3 WHERE id = $1;

-- name: DeleteStocktake :exec
DELETE FROM stock.stocktakes WHERE id = $1;

-- name: DeleteStocktakeLines :exec
DELETE FROM stock.stocktake_lines WHERE stocktake_id = $1;

-- name: InsertStocktakeLines :exec
-- No lot, expiry or serial travels as ''.
INSERT INTO stock.stocktake_lines (stocktake_id, line_no, item_id, lot_code, lot_expiry, serial_code, counted)
SELECT @stocktake_id::bigint, k.n, k.i, nullif(k.l, ''), nullif(k.e, '')::date, nullif(k.s, ''), k.q::numeric
FROM (SELECT unnest(@line_nos::int[]) AS n, unnest(@item_ids::bigint[]) AS i, unnest(@lot_codes::text[]) AS l,
             unnest(@lot_expiries::text[]) AS e, unnest(@serials::text[]) AS s, unnest(@counts::text[]) AS q) AS k;

-- name: SetStocktakeResults :exec
UPDATE stock.stocktake_lines l SET book = k.b::numeric, difference = l.counted - k.b::numeric
FROM (SELECT unnest(@line_nos::int[]) AS n, unnest(@books::text[]) AS b) AS k
WHERE l.stocktake_id = @stocktake_id AND l.line_no = k.n;

-- name: GetStocktake :one
SELECT s.id, s.warehouse_id, s.note, w.code AS warehouse_code, w.name AS warehouse_name,
       d.number, d.status, d.version, d.date, d.legal_entity_id
FROM stock.stocktakes s
JOIN record.documents d ON d.id = s.id
JOIN stock.warehouses w ON w.id = s.warehouse_id
WHERE s.id = $1;

-- name: GetStocktakeLines :many
-- on_hand is the line's current balance in the warehouse; a lot or serial not yet known has none.
SELECT l.line_no, l.item_id, i.code AS item_code, i.name AS item_name, i.unit, l.lot_code, l.lot_expiry, l.serial_code,
       l.counted::text AS counted, coalesce(l.book::text, '')::text AS book, coalesce(l.difference::text, '')::text AS difference,
       coalesce((SELECT sum(b.quantity) FROM stock.balances b
                 WHERE b.warehouse_id = s.warehouse_id AND b.item_id = l.item_id
                   AND (l.lot_code IS NULL OR b.lot_id = (SELECT x.id FROM stock.lots x WHERE x.item_id = l.item_id AND lower(x.code) = lower(l.lot_code)))
                   AND (l.serial_code IS NULL OR b.serial_id = (SELECT x.id FROM stock.serials x WHERE x.item_id = l.item_id AND lower(x.code) = lower(l.serial_code)))), 0)::text AS on_hand
FROM stock.stocktake_lines l
JOIN stock.stocktakes s ON s.id = l.stocktake_id
JOIN item.items i ON i.id = l.item_id
WHERE l.stocktake_id = $1 ORDER BY l.line_no;

-- name: StocktakeListBase :many
-- data:base
SELECT s.id, d.number, d.status, d.date, d.org_unit_id, s.warehouse_id, w.code AS warehouse_code, w.name AS warehouse_name, s.note
FROM stock.stocktakes s
JOIN record.documents d ON d.id = s.id
JOIN stock.warehouses w ON w.id = s.warehouse_id;

-- name: OnHandLines :many
-- Non-zero balances of a warehouse, optionally of one item, as stocktake lines counting what is on hand.
SELECT b.item_id, i.code AS item_code, i.name AS item_name, i.unit, l.code AS lot_code, l.expiry AS lot_expiry, s.code AS serial_code, greatest(b.quantity, 0)::text AS quantity
FROM stock.balances b
JOIN item.items i ON i.id = b.item_id
LEFT JOIN stock.lots l ON l.id = b.lot_id
LEFT JOIN stock.serials s ON s.id = b.serial_id
WHERE b.warehouse_id = @warehouse_id AND b.quantity <> 0 AND (@item_id::bigint = 0 OR b.item_id = @item_id)
ORDER BY i.code, i.id, l.code NULLS FIRST, s.code NULLS FIRST
-- One past what a stocktake takes, so the caller knows there is more.
LIMIT 2001;
