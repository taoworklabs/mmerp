-- name: InsertPayment :exec
INSERT INTO purchase.payments (id, supplier_id, method, amount, note) VALUES ($1, $2, $3, $4, $5);

-- name: UpdatePayment :exec
UPDATE purchase.payments SET supplier_id = $2, method = $3, amount = $4, note = $5 WHERE id = $1;

-- name: DeletePayment :exec
DELETE FROM purchase.payments WHERE id = $1;

-- name: GetPayment :one
SELECT r.supplier_id, r.method, r.amount, r.note, c.code AS supplier_code, p.name AS supplier_name, c.partner_id,
       d.number, d.status, d.version, d.date, d.org_unit_id, u.name AS org_unit_name
FROM purchase.payments r
JOIN record.documents d ON d.id = r.id
JOIN purchase.suppliers c ON c.id = r.supplier_id
JOIN partner.partners p ON p.id = c.partner_id
JOIN iam.org_units u ON u.id = d.org_unit_id
WHERE r.id = $1;

-- name: GetAllocations :many
-- A payment's allocations with their invoice and what payments other than this one, not
-- cancelled, leave of it.
SELECT a.invoice_id, a.amount, d.number AS invoice_number, d.date AS invoice_date, v.total AS invoice_total,
       (v.total - coalesce((
           SELECT sum(o.amount) FROM purchase.payment_allocations o JOIN record.documents od ON od.id = o.payment_id
           WHERE o.invoice_id = a.invoice_id AND o.payment_id <> a.payment_id AND od.status <> 'cancelled'
       ), 0))::bigint AS remaining
FROM purchase.payment_allocations a
JOIN purchase.invoices v ON v.id = a.invoice_id
JOIN record.documents d ON d.id = a.invoice_id
WHERE a.payment_id = $1
ORDER BY a.position;

-- name: AllocatableInvoices :many
-- The named invoices with their status, date, org unit, supplier and what payments other than
-- this one, not cancelled, leave of each.
SELECT v.id, d.number, d.status, d.date, d.org_unit_id, h.supplier_id, v.total,
       (v.total - coalesce((
           SELECT sum(o.amount) FROM purchase.payment_allocations o JOIN record.documents od ON od.id = o.payment_id
           WHERE o.invoice_id = v.id AND o.payment_id <> @payment_id AND od.status <> 'cancelled'
       ), 0))::bigint AS remaining
FROM purchase.invoices v
JOIN record.documents d ON d.id = v.id
JOIN purchase.orders h ON h.id = v.order_id
WHERE v.id = ANY(@invoice_ids::bigint[]);

-- name: OpenInvoices :many
-- Posted invoices of a supplier at an org unit that payments other than this one, not
-- cancelled, leave something of; oldest first.
SELECT v.id, d.number, d.date, v.total,
       (v.total - coalesce((
           SELECT sum(o.amount) FROM purchase.payment_allocations o JOIN record.documents od ON od.id = o.payment_id
           WHERE o.invoice_id = v.id AND o.payment_id <> @payment_id AND od.status <> 'cancelled'
       ), 0))::bigint AS remaining
FROM purchase.invoices v
JOIN record.documents d ON d.id = v.id
JOIN purchase.orders h ON h.id = v.order_id
WHERE h.supplier_id = @supplier_id AND d.org_unit_id = @org_unit_id AND d.status = 'posted'
ORDER BY d.date, v.id;

-- name: DeleteAllocations :exec
DELETE FROM purchase.payment_allocations WHERE payment_id = $1;

-- name: InsertAllocations :exec
INSERT INTO purchase.payment_allocations (payment_id, position, invoice_id, amount)
SELECT @payment_id::bigint, unnest(@positions::int[]), unnest(@invoice_ids::bigint[]), unnest(@amounts::bigint[]);

-- name: PaymentListBase :many
-- data:base
SELECT r.id, d.number, d.status, d.date, d.org_unit_id, r.supplier_id, p.name AS supplier_name, r.method, r.amount,
       u.name AS org_unit_name
FROM purchase.payments r
JOIN record.documents d ON d.id = r.id
JOIN purchase.suppliers c ON c.id = r.supplier_id
JOIN partner.partners p ON p.id = c.partner_id
JOIN iam.org_units u ON u.id = d.org_unit_id;
