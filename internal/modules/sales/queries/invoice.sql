-- name: InsertInvoice :exec
INSERT INTO sales.invoices (id, order_id, subtotal, discount_total, vat_total, total, note)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: UpdateInvoice :exec
UPDATE sales.invoices SET subtotal = $2, discount_total = $3, vat_total = $4, total = $5, note = $6 WHERE id = $1;

-- name: DeleteInvoice :exec
DELETE FROM sales.invoices WHERE id = $1;

-- name: InvoiceOrder :one
SELECT order_id FROM sales.invoices WHERE id = $1;

-- name: GetInvoice :one
SELECT v.order_id, v.subtotal, v.discount_total, v.vat_total, v.total, v.note,
       d.number, d.status, d.version, d.date, d.org_unit_id, u.name AS org_unit_name, o.number AS order_number
FROM sales.invoices v
JOIN record.documents d ON d.id = v.id
JOIN record.documents o ON o.id = v.order_id
JOIN iam.org_units u ON u.id = d.org_unit_id
WHERE v.id = $1;

-- name: GetInvoiceLines :many
SELECT order_position, item_id, item_code, description, unit, quantity::text AS quantity, unit_price,
       discount_percent::text AS discount_percent, vat_rate, amount, discount, vat
FROM sales.invoice_lines WHERE invoice_id = $1 ORDER BY position;

-- name: InvoiceableLines :many
-- An order's lines with what invoices other than this one, not cancelled, already take of each.
SELECT l.position, l.item_id, l.item_code, l.description, l.unit, l.quantity::text AS quantity, l.unit_price,
       l.discount_percent::text AS discount_percent, l.vat_rate,
       coalesce((
           SELECT sum(il.quantity)
           FROM sales.invoice_lines il
           JOIN record.documents d ON d.id = il.invoice_id
           JOIN sales.invoices v ON v.id = il.invoice_id
           WHERE v.order_id = l.doc_id AND il.order_position = l.position AND il.invoice_id <> @invoice_id AND d.status <> 'cancelled'
       ), 0)::text AS taken
FROM sales.lines l
WHERE l.doc_id = @order_id
ORDER BY l.position;

-- name: DeleteInvoiceLines :exec
DELETE FROM sales.invoice_lines WHERE invoice_id = $1;

-- name: InsertInvoiceLines :exec
INSERT INTO sales.invoice_lines (invoice_id, position, order_position, item_id, item_code, description, unit,
    quantity, unit_price, discount_percent, vat_rate, amount, discount, vat)
SELECT @invoice_id::bigint, unnest(@positions::int[]), unnest(@order_positions::int[]), unnest(@item_ids::bigint[]),
    unnest(@item_codes::text[]), unnest(@descriptions::text[]), unnest(@units::text[]), unnest(@quantities::text[])::numeric,
    unnest(@unit_prices::bigint[]), unnest(@discount_percents::text[])::numeric, unnest(@vat_rates::text[]),
    unnest(@amounts::bigint[]), unnest(@discounts::bigint[]), unnest(@vats::bigint[]);

-- name: LiveInvoiceOfOrder :one
-- Any invoice of the order that is not cancelled.
SELECT v.id FROM sales.invoices v JOIN record.documents d ON d.id = v.id
WHERE v.order_id = $1 AND d.status <> 'cancelled' LIMIT 1;

-- name: LiveReceiptOfInvoice :one
-- Any receipt settling the invoice that is not cancelled.
SELECT a.receipt_id FROM sales.receipt_allocations a JOIN record.documents d ON d.id = a.receipt_id
WHERE a.invoice_id = $1 AND d.status <> 'cancelled' LIMIT 1;

-- name: InvoicePartner :one
-- The partner of the invoice's customer, named on its receivable line.
SELECT c.partner_id FROM sales.invoices v
JOIN sales.headers h ON h.id = v.order_id
JOIN sales.customers c ON c.id = h.customer_id
WHERE v.id = $1;
