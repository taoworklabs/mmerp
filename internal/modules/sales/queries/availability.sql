-- name: Availability :many
-- Per stock item: on hand in the legal entity's warehouses, and what its posted orders
-- still have to deliver (posted deliveries only, never below 0 per line).
SELECT i.id AS item_id,
       coalesce((
           SELECT sum(b.quantity)
           FROM stock.balances b JOIN stock.warehouses w ON w.id = b.warehouse_id
           WHERE w.legal_entity_id = @legal_entity_id AND b.item_id = i.id
       ), 0)::numeric(15, 3)::text AS on_hand,
       coalesce((
           SELECT sum(greatest(l.quantity - coalesce((
               SELECT sum(dl.quantity)
               FROM sales.delivery_lines dl
               JOIN sales.deliveries v ON v.id = dl.delivery_id
               JOIN record.documents vd ON vd.id = v.id
               WHERE v.order_id = l.doc_id AND dl.order_position = l.position AND vd.status = 'posted'
           ), 0), 0))
           FROM sales.lines l
           JOIN sales.headers h ON h.id = l.doc_id
           JOIN record.documents d ON d.id = l.doc_id
           WHERE h.kind = 'order' AND d.status = 'posted' AND d.legal_entity_id = @legal_entity_id AND l.item_id = i.id
       ), 0)::numeric(15, 3)::text AS committed
FROM item.items i
WHERE i.id = ANY(@item_ids::bigint[]) AND i.tracking <> 'none'
ORDER BY i.id;
