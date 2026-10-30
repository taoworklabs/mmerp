-- name: OnHandListBase :many
-- data:base
-- Zero balances stay hidden; "key" is unique per balance row, for stable pages.
SELECT b.warehouse_id || ':' || b.item_id || ':' || coalesce(b.lot_id, 0) || ':' || coalesce(b.serial_id, 0) AS key,
       b.warehouse_id, w.code AS warehouse_code, b.item_id, i.code AS item_code, i.name AS item_name, i.unit,
       l.code AS lot_code, l.expiry AS lot_expiry, s.code AS serial_code, b.quantity::text AS quantity
FROM stock.balances b
JOIN stock.warehouses w ON w.id = b.warehouse_id
JOIN item.items i ON i.id = b.item_id
LEFT JOIN stock.lots l ON l.id = b.lot_id
LEFT JOIN stock.serials s ON s.id = b.serial_id
WHERE b.quantity <> 0;

-- name: StockCardOpening :one
SELECT coalesce(sum(quantity), 0)::text AS quantity
FROM stock.moves
WHERE warehouse_id = @warehouse_id AND item_id = @item_id AND date < @from_date::date;

-- name: StockCardMoves :many
SELECT m.id, m.date, m.doc_type, m.doc_id, coalesce(d.number, '') AS number, l.code AS lot_code, s.code AS serial_code,
       m.quantity::text AS quantity
FROM stock.moves m
LEFT JOIN record.documents d ON d.id = m.doc_id AND d.doc_type = m.doc_type
LEFT JOIN stock.lots l ON l.id = m.lot_id
LEFT JOIN stock.serials s ON s.id = m.serial_id
WHERE m.warehouse_id = @warehouse_id AND m.item_id = @item_id AND m.date BETWEEN @from_date::date AND @to_date::date
ORDER BY m.date, m.id;
