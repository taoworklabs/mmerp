-- name: InsertHeader :exec
INSERT INTO sales.headers (id, kind, customer_id, customer_code, customer_name, customer_tax_code,
    customer_address, customer_phone, customer_email, contact_name, valid_until, delivery_date, quote_id,
    payment_terms, delivery_terms, note, subtotal, discount_total, vat_total, total)
VALUES (@id, @kind, @customer_id, @customer_code, @customer_name, @customer_tax_code,
    @customer_address, @customer_phone, @customer_email, @contact_name, @valid_until, @delivery_date, @quote_id,
    @payment_terms, @delivery_terms, @note, @subtotal, @discount_total, @vat_total, @total);

-- name: UpdateHeader :exec
UPDATE sales.headers
SET customer_id = $2, customer_code = $3, customer_name = $4, customer_tax_code = $5, customer_address = $6,
    customer_phone = $7, customer_email = $8, contact_name = $9, valid_until = $10, delivery_date = $11,
    payment_terms = $12, delivery_terms = $13, note = $14, subtotal = $15, discount_total = $16,
    vat_total = $17, total = $18
WHERE id = $1;

-- name: DeleteHeader :exec
DELETE FROM sales.headers WHERE id = $1;

-- name: GetHeader :one
SELECT h.*, d.number, d.status, d.version, d.date, d.org_unit_id, d.legal_entity_id, u.name AS org_unit_name,
       q.number AS quote_number
FROM sales.headers h
JOIN record.documents d ON d.id = h.id
JOIN iam.org_units u ON u.id = d.org_unit_id
LEFT JOIN record.documents q ON q.id = h.quote_id
WHERE h.id = $1;

-- name: LiveOrder :one
-- The order made from a quotation that is not cancelled; at most one, since making another
-- holds the quotation's row lock and finds this one first.
SELECT d.id, d.number, d.status
FROM sales.headers h JOIN record.documents d ON d.id = h.id
WHERE h.quote_id = $1 AND d.status <> 'cancelled';

-- name: DeleteLines :exec
DELETE FROM sales.lines WHERE doc_id = $1;

-- name: InsertLines :exec
INSERT INTO sales.lines (doc_id, position, item_id, item_code, description, unit, quantity, unit_price,
    discount_percent, vat_rate, amount, discount, vat)
SELECT @doc_id::bigint, unnest(@positions::int[]), unnest(@item_ids::bigint[]), unnest(@item_codes::text[]),
    unnest(@descriptions::text[]), unnest(@units::text[]), unnest(@quantities::text[])::numeric,
    unnest(@unit_prices::bigint[]), unnest(@discount_percents::text[])::numeric, unnest(@vat_rates::text[]),
    unnest(@amounts::bigint[]), unnest(@discounts::bigint[]), unnest(@vats::bigint[]);

-- name: GetLines :many
SELECT item_id, item_code, description, unit, quantity::text AS quantity, unit_price,
       discount_percent::text AS discount_percent, vat_rate, amount, discount, vat
FROM sales.lines WHERE doc_id = $1 ORDER BY position;

-- name: DocListBase :many
-- data:base
SELECT h.id, h.kind, d.number, d.status, d.date, d.org_unit_id, h.customer_id, h.customer_code, h.customer_name,
       h.valid_until, h.total, u.name AS org_unit_name, (h.quote_id IS NOT NULL)::bool AS from_quote,
       EXISTS (
           SELECT 1 FROM sales.headers oh JOIN record.documents od ON od.id = oh.id
           WHERE oh.quote_id = h.id AND od.status <> 'cancelled'
       ) AS ordered
FROM sales.headers h
JOIN record.documents d ON d.id = h.id
JOIN iam.org_units u ON u.id = d.org_unit_id;

-- name: CopyCustomer :one
-- Takes the customer's code and its partner's identity as they are now onto a document
-- being sent; returns whether the customer is active.
UPDATE sales.headers h
SET customer_code = c.code, customer_name = p.name, customer_tax_code = p.tax_code, customer_address = p.address,
    customer_phone = p.phone, customer_email = p.email, contact_name = p.contact_name
FROM sales.customers c JOIN partner.partners p ON p.id = c.partner_id
WHERE h.id = $1 AND c.id = h.customer_id
RETURNING c.active;
