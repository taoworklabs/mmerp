-- name: InsertOrder :exec
INSERT INTO purchase.orders (id, supplier_id, supplier_code, supplier_name, supplier_tax_code, supplier_address,
    supplier_phone, supplier_email, contact_name, expected_date, payment_terms, note, subtotal, discount_total, vat_total, total)
VALUES (@id, @supplier_id, @supplier_code, @supplier_name, @supplier_tax_code, @supplier_address,
    @supplier_phone, @supplier_email, @contact_name, @expected_date, @payment_terms, @note, @subtotal, @discount_total, @vat_total, @total);

-- name: UpdateOrder :exec
UPDATE purchase.orders
SET supplier_id = $2, supplier_code = $3, supplier_name = $4, supplier_tax_code = $5, supplier_address = $6,
    supplier_phone = $7, supplier_email = $8, contact_name = $9, expected_date = $10, payment_terms = $11,
    note = $12, subtotal = $13, discount_total = $14, vat_total = $15, total = $16
WHERE id = $1;

-- name: DeleteOrder :exec
DELETE FROM purchase.orders WHERE id = $1;

-- name: GetOrder :one
SELECT o.*, d.number, d.status, d.version, d.date, d.org_unit_id, d.legal_entity_id, u.name AS org_unit_name
FROM purchase.orders o
JOIN record.documents d ON d.id = o.id
JOIN iam.org_units u ON u.id = d.org_unit_id
WHERE o.id = $1;

-- name: DeleteOrderLines :exec
DELETE FROM purchase.order_lines WHERE order_id = $1;

-- name: InsertOrderLines :exec
INSERT INTO purchase.order_lines (order_id, position, item_id, item_code, description, unit, stocked, quantity, unit_price,
    discount_percent, vat_rate, amount, discount, vat)
SELECT @order_id::bigint, unnest(@positions::int[]), unnest(@item_ids::bigint[]), unnest(@item_codes::text[]),
    unnest(@descriptions::text[]), unnest(@units::text[]), unnest(@stocked::bool[]), unnest(@quantities::text[])::numeric,
    unnest(@unit_prices::bigint[]), unnest(@discount_percents::text[])::numeric, unnest(@vat_rates::text[]),
    unnest(@amounts::bigint[]), unnest(@discounts::bigint[]), unnest(@vats::bigint[]);

-- name: GetOrderLines :many
SELECT l.position, l.item_id, l.item_code, l.description, l.unit, l.stocked, l.quantity::text AS quantity, l.unit_price,
       l.discount_percent::text AS discount_percent, l.vat_rate, l.amount, l.discount, l.vat
FROM purchase.order_lines l
WHERE l.order_id = $1 ORDER BY l.position;

-- name: OrderListBase :many
-- data:base
SELECT o.id, d.number, d.status, d.date, d.org_unit_id, o.supplier_id, o.supplier_code, o.supplier_name,
       o.total, u.name AS org_unit_name
FROM purchase.orders o
JOIN record.documents d ON d.id = o.id
JOIN iam.org_units u ON u.id = d.org_unit_id;

-- name: CopySupplier :one
-- Takes the supplier's code and its partner's identity as they are now onto an order being
-- sent; returns whether the supplier is active.
UPDATE purchase.orders o
SET supplier_code = s.code, supplier_name = p.name, supplier_tax_code = p.tax_code, supplier_address = p.address,
    supplier_phone = p.phone, supplier_email = p.email, contact_name = p.contact_name
FROM purchase.suppliers s JOIN partner.partners p ON p.id = s.partner_id
WHERE o.id = $1 AND s.id = o.supplier_id
RETURNING s.active;
