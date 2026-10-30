-- name: DocumentHeader :one
SELECT legal_entity_id, date FROM record.documents WHERE doc_type = $1 AND id = $2;

-- name: HasLiveMoves :one
-- Moves neither reversing nor reversed.
SELECT EXISTS (
    SELECT 1 FROM stock.moves m
    WHERE m.doc_type = $1 AND m.doc_id = $2 AND m.reverses IS NULL
      AND NOT EXISTS (SELECT 1 FROM stock.moves r WHERE r.reverses = m.id)
);

-- name: MoveWarehouses :many
SELECT id, legal_entity_id, active FROM stock.warehouses WHERE id = ANY(@ids::bigint[]);

-- name: MoveItems :many
SELECT id, tracking, allow_negative FROM item.items WHERE id = ANY(@ids::bigint[]);

-- name: InsertLot :exec
INSERT INTO stock.lots (item_id, code, expiry) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING;

-- name: LotByCode :one
SELECT id, expiry FROM stock.lots WHERE item_id = $1 AND lower(code) = lower(@code);

-- name: InsertSerial :exec
INSERT INTO stock.serials (item_id, code) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: SerialByCode :one
SELECT id FROM stock.serials WHERE item_id = $1 AND lower(code) = lower(@code);

-- name: LockSerials :exec
-- A serial is on hand once across warehouses, so its row guards every balance of it.
SELECT id FROM stock.serials WHERE id = ANY(@ids::bigint[]) ORDER BY id FOR UPDATE;

-- Balance keys travel as parallel arrays, a missing lot or serial as 0, in key order.

-- name: InsertBalances :exec
INSERT INTO stock.balances (warehouse_id, item_id, lot_id, serial_id, quantity)
SELECT k.w, k.i, nullif(k.l, 0), nullif(k.s, 0), 0
FROM (SELECT unnest(@warehouse_ids::bigint[]) AS w, unnest(@item_ids::bigint[]) AS i, unnest(@lot_ids::bigint[]) AS l, unnest(@serial_ids::bigint[]) AS s) AS k
ON CONFLICT DO NOTHING;

-- name: LockBalances :exec
SELECT 1 FROM stock.balances b
JOIN (SELECT unnest(@warehouse_ids::bigint[]) AS w, unnest(@item_ids::bigint[]) AS i, unnest(@lot_ids::bigint[]) AS l, unnest(@serial_ids::bigint[]) AS s) AS k
  ON b.warehouse_id = k.w AND b.item_id = k.i AND coalesce(b.lot_id, 0) = k.l AND coalesce(b.serial_id, 0) = k.s
ORDER BY b.warehouse_id, b.item_id, coalesce(b.lot_id, 0), coalesce(b.serial_id, 0)
FOR UPDATE OF b;

-- name: ApplyDeltas :many
UPDATE stock.balances b SET quantity = b.quantity + k.d::numeric
FROM (SELECT unnest(@warehouse_ids::bigint[]) AS w, unnest(@item_ids::bigint[]) AS i, unnest(@lot_ids::bigint[]) AS l, unnest(@serial_ids::bigint[]) AS s,
             unnest(@deltas::text[]) AS d) AS k
WHERE b.warehouse_id = k.w AND b.item_id = k.i AND coalesce(b.lot_id, 0) = k.l AND coalesce(b.serial_id, 0) = k.s
RETURNING b.warehouse_id, b.item_id, coalesce(b.lot_id, 0)::bigint AS lot_id, coalesce(b.serial_id, 0)::bigint AS serial_id, b.quantity::text AS quantity;

-- name: SerialsOverOne :many
SELECT serial_id::bigint FROM stock.balances WHERE serial_id = ANY(@ids::bigint[]) GROUP BY serial_id HAVING sum(quantity) > 1;

-- name: InsertMoves :exec
INSERT INTO stock.moves (doc_type, doc_id, line_no, warehouse_id, item_id, lot_id, serial_id, quantity, date)
SELECT @doc_type::text, @doc_id::bigint, k.n, k.w, k.i, nullif(k.l, 0), nullif(k.s, 0), k.q::numeric, @date::date
FROM (SELECT unnest(@line_nos::int[]) AS n, unnest(@warehouse_ids::bigint[]) AS w, unnest(@item_ids::bigint[]) AS i,
             unnest(@lot_ids::bigint[]) AS l, unnest(@serial_ids::bigint[]) AS s, unnest(@quantities::text[]) AS q) AS k;

-- name: LotsOnHand :many
-- First expiring first; undated lots last, then oldest received.
SELECT l.id, l.code, l.expiry, b.quantity::text AS quantity
FROM stock.balances b JOIN stock.lots l ON l.id = b.lot_id
WHERE b.warehouse_id = $1 AND b.item_id = $2 AND b.serial_id IS NULL AND b.quantity > 0
  AND (l.expiry IS NULL OR l.expiry >= @today::date)
ORDER BY l.expiry NULLS LAST, l.received_at, l.id;
