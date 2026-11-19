-- name: InsertInvoice :exec
INSERT INTO purchase.invoices (id, order_id, supplier_invoice_symbol, supplier_invoice_no, supplier_invoice_date,
    subtotal, discount_total, vat_total, total, note)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: UpdateInvoice :exec
UPDATE purchase.invoices SET supplier_invoice_symbol = $2, supplier_invoice_no = $3, supplier_invoice_date = $4,
    subtotal = $5, discount_total = $6, vat_total = $7, total = $8, note = $9
WHERE id = $1;

-- name: DeleteInvoice :exec
DELETE FROM purchase.invoices WHERE id = $1;

-- name: InvoiceOrder :one
SELECT order_id FROM purchase.invoices WHERE id = $1;

-- name: GetInvoice :one
-- Paid counts posted payments; held counts every payment not cancelled, drafts included.
SELECT v.order_id, v.supplier_invoice_symbol, v.supplier_invoice_no, v.supplier_invoice_date,
       v.subtotal, v.discount_total, v.vat_total, v.total, v.note,
       coalesce((SELECT sum(a.amount) FROM purchase.payment_allocations a JOIN record.documents pd ON pd.id = a.payment_id
                 WHERE a.invoice_id = v.id AND pd.status = 'posted'), 0)::bigint AS paid,
       coalesce((SELECT sum(a.amount) FROM purchase.payment_allocations a JOIN record.documents pd ON pd.id = a.payment_id
                 WHERE a.invoice_id = v.id AND pd.status <> 'cancelled'), 0)::bigint AS held,
       d.number, d.status, d.version, d.date, d.org_unit_id, u.name AS org_unit_name, o.number AS order_number,
       h.supplier_id, s.partner_id
FROM purchase.invoices v
JOIN record.documents d ON d.id = v.id
JOIN record.documents o ON o.id = v.order_id
JOIN purchase.orders h ON h.id = v.order_id
JOIN purchase.suppliers s ON s.id = h.supplier_id
JOIN iam.org_units u ON u.id = d.org_unit_id
WHERE v.id = $1;

-- name: GetInvoiceLines :many
SELECT order_position, item_id, item_code, description, unit, stocked, quantity::text AS quantity, unit_price,
       discount_percent::text AS discount_percent, vat_rate, amount, discount, vat
FROM purchase.invoice_lines WHERE invoice_id = $1 ORDER BY position;

-- name: InvoiceableLines :many
-- An order's lines with what invoices other than this one, not cancelled, already take of each.
SELECT l.position, l.item_id, l.item_code, l.description, l.unit, l.stocked, l.quantity::text AS quantity, l.unit_price,
       l.discount_percent::text AS discount_percent, l.vat_rate,
       coalesce((
           SELECT sum(il.quantity)
           FROM purchase.invoice_lines il
           JOIN purchase.invoices v ON v.id = il.invoice_id
           JOIN record.documents d ON d.id = v.id
           WHERE v.order_id = l.order_id AND il.order_position = l.position AND il.invoice_id <> @invoice_id AND d.status <> 'cancelled'
       ), 0)::text AS taken
FROM purchase.order_lines l
WHERE l.order_id = @order_id
ORDER BY l.position;

-- name: DeleteInvoiceLines :exec
DELETE FROM purchase.invoice_lines WHERE invoice_id = $1;

-- name: InsertInvoiceLines :exec
INSERT INTO purchase.invoice_lines (invoice_id, position, order_position, item_id, item_code, description, unit, stocked,
    quantity, unit_price, discount_percent, vat_rate, amount, discount, vat)
SELECT @invoice_id::bigint, unnest(@positions::int[]), unnest(@order_positions::int[]), unnest(@item_ids::bigint[]),
    unnest(@item_codes::text[]), unnest(@descriptions::text[]), unnest(@units::text[]), unnest(@stocked::bool[]),
    unnest(@quantities::text[])::numeric, unnest(@unit_prices::bigint[]), unnest(@discount_percents::text[])::numeric,
    unnest(@vat_rates::text[]), unnest(@amounts::bigint[]), unnest(@discounts::bigint[]), unnest(@vats::bigint[]);

-- name: LiveInvoiceOfOrder :one
-- Any invoice of the order that is not cancelled.
SELECT v.id FROM purchase.invoices v JOIN record.documents d ON d.id = v.id
WHERE v.order_id = $1 AND d.status <> 'cancelled' LIMIT 1;

-- name: LivePaymentOfInvoice :one
-- Any payment settling the invoice that is not cancelled.
SELECT a.payment_id FROM purchase.payment_allocations a JOIN record.documents d ON d.id = a.payment_id
WHERE a.invoice_id = $1 AND d.status <> 'cancelled' LIMIT 1;

-- name: OrderInvoices :many
-- The order's invoices not cancelled.
SELECT d.id, d.number, d.status
FROM purchase.invoices v JOIN record.documents d ON d.id = v.id
WHERE v.order_id = $1 AND d.status <> 'cancelled'
ORDER BY d.id;

-- name: DuplicateInvoice :one
-- Another invoice of the same supplier with the same supplier symbol and number, pending or
-- posted; symbols compare without case or surrounding spaces. Read under the supplier's row lock.
SELECT d.number
FROM purchase.invoices v
JOIN purchase.orders h ON h.id = v.order_id
JOIN record.documents d ON d.id = v.id
WHERE h.supplier_id = @supplier_id AND v.id <> @invoice_id AND d.status IN ('pending_approval', 'posted')
  AND upper(btrim(v.supplier_invoice_symbol)) = upper(btrim(@symbol::text)) AND btrim(v.supplier_invoice_no) = btrim(@number::text)
LIMIT 1;

-- name: InvoiceListBase :many
-- data:base
SELECT v.id, d.number, d.status, d.date, d.org_unit_id, v.order_id, o.number AS order_number,
       h.supplier_id, h.supplier_name, v.supplier_invoice_symbol, v.supplier_invoice_no, v.total, u.name AS org_unit_name,
       -- Posted invoices only; one of nothing owes nothing.
       (CASE WHEN d.status <> 'posted' THEN '' WHEN p.paid >= v.total THEN 'paid' WHEN p.paid = 0 THEN 'unpaid' ELSE 'partial' END)::text AS payment_state
FROM purchase.invoices v
CROSS JOIN LATERAL (
    SELECT coalesce(sum(a.amount), 0) AS paid FROM purchase.payment_allocations a JOIN record.documents pd ON pd.id = a.payment_id
    WHERE a.invoice_id = v.id AND pd.status = 'posted'
) p
JOIN record.documents d ON d.id = v.id
JOIN record.documents o ON o.id = v.order_id
JOIN purchase.orders h ON h.id = v.order_id
JOIN iam.org_units u ON u.id = d.org_unit_id;
