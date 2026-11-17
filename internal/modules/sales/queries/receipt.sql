-- name: InsertReceipt :exec
INSERT INTO sales.receipts (id, customer_id, method, amount, note) VALUES ($1, $2, $3, $4, $5);

-- name: UpdateReceipt :exec
UPDATE sales.receipts SET customer_id = $2, method = $3, amount = $4, note = $5 WHERE id = $1;

-- name: DeleteReceipt :exec
DELETE FROM sales.receipts WHERE id = $1;

-- name: GetReceipt :one
SELECT r.customer_id, r.method, r.amount, r.note, c.code AS customer_code, p.name AS customer_name, c.partner_id,
       d.number, d.status, d.version, d.date, d.org_unit_id, u.name AS org_unit_name
FROM sales.receipts r
JOIN record.documents d ON d.id = r.id
JOIN sales.customers c ON c.id = r.customer_id
JOIN partner.partners p ON p.id = c.partner_id
JOIN iam.org_units u ON u.id = d.org_unit_id
WHERE r.id = $1;

-- name: GetAllocations :many
-- A receipt's allocations with their invoice and what receipts other than this one, not
-- cancelled, leave of it.
SELECT a.invoice_id, a.amount, d.number AS invoice_number, d.date AS invoice_date, v.total AS invoice_total,
       (v.total - coalesce((
           SELECT sum(o.amount) FROM sales.receipt_allocations o JOIN record.documents od ON od.id = o.receipt_id
           WHERE o.invoice_id = a.invoice_id AND o.receipt_id <> a.receipt_id AND od.status <> 'cancelled'
       ), 0))::bigint AS remaining
FROM sales.receipt_allocations a
JOIN sales.invoices v ON v.id = a.invoice_id
JOIN record.documents d ON d.id = a.invoice_id
WHERE a.receipt_id = $1
ORDER BY a.position;

-- name: AllocatableInvoices :many
-- The named invoices with their status, date, org unit, customer and what receipts other than
-- this one, not cancelled, leave of each.
SELECT v.id, d.number, d.status, d.date, d.org_unit_id, h.customer_id, v.total,
       (v.total - coalesce((
           SELECT sum(o.amount) FROM sales.receipt_allocations o JOIN record.documents od ON od.id = o.receipt_id
           WHERE o.invoice_id = v.id AND o.receipt_id <> @receipt_id AND od.status <> 'cancelled'
       ), 0))::bigint AS remaining
FROM sales.invoices v
JOIN record.documents d ON d.id = v.id
JOIN sales.headers h ON h.id = v.order_id
WHERE v.id = ANY(@invoice_ids::bigint[]);

-- name: OpenInvoices :many
-- Posted invoices of a customer at an org unit that receipts other than this one, not
-- cancelled, leave something of; oldest first.
SELECT v.id, d.number, d.date, v.total,
       (v.total - coalesce((
           SELECT sum(o.amount) FROM sales.receipt_allocations o JOIN record.documents od ON od.id = o.receipt_id
           WHERE o.invoice_id = v.id AND o.receipt_id <> @receipt_id AND od.status <> 'cancelled'
       ), 0))::bigint AS remaining
FROM sales.invoices v
JOIN record.documents d ON d.id = v.id
JOIN sales.headers h ON h.id = v.order_id
WHERE h.customer_id = @customer_id AND d.org_unit_id = @org_unit_id AND d.status = 'posted'
ORDER BY d.date, v.id;

-- name: DeleteAllocations :exec
DELETE FROM sales.receipt_allocations WHERE receipt_id = $1;

-- name: InsertAllocations :exec
INSERT INTO sales.receipt_allocations (receipt_id, position, invoice_id, amount)
SELECT @receipt_id::bigint, unnest(@positions::int[]), unnest(@invoice_ids::bigint[]), unnest(@amounts::bigint[]);

-- name: ReceiptCustomerActive :one
SELECT c.active FROM sales.receipts r JOIN sales.customers c ON c.id = r.customer_id WHERE r.id = $1;

-- name: ReceiptListBase :many
-- data:base
SELECT r.id, d.number, d.status, d.date, d.org_unit_id, r.customer_id, p.name AS customer_name, r.method, r.amount,
       u.name AS org_unit_name
FROM sales.receipts r
JOIN record.documents d ON d.id = r.id
JOIN sales.customers c ON c.id = r.customer_id
JOIN partner.partners p ON p.id = c.partner_id
JOIN iam.org_units u ON u.id = d.org_unit_id;
