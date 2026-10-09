-- name: Today :one
-- Today in the tenant time zone.
SELECT (now() AT TIME ZONE @tz::text)::date;

-- name: CreateCustomer :one
INSERT INTO sales.customers (code, name, tax_code, address, phone, email, contact_name, payment_terms, org_unit_id, active)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id;

-- name: UpdateCustomer :exec
UPDATE sales.customers
SET code = $2, name = $3, tax_code = $4, address = $5, phone = $6, email = $7, contact_name = $8,
    payment_terms = $9, org_unit_id = $10, active = $11
WHERE id = $1;

-- name: GetCustomer :one
SELECT c.*, u.name AS org_unit_name
FROM sales.customers c JOIN iam.org_units u ON u.id = c.org_unit_id
WHERE c.id = $1;

-- name: CustomerActive :one
-- Read under a share lock, so the customer cannot be deactivated while a document posts.
SELECT active FROM sales.customers WHERE id = $1 FOR SHARE;

-- name: ListCustomers :many
-- Scope filter: @all_units or org_unit_id in @units.
SELECT c.id, c.code, c.name, c.tax_code, c.phone, c.payment_terms, c.org_unit_id, u.name AS org_unit_name, c.active, count(*) OVER () AS total
FROM sales.customers c JOIN iam.org_units u ON u.id = c.org_unit_id
WHERE (@all_units::bool OR c.org_unit_id = ANY(@units::bigint[]))
  AND (@q::text = '' OR c.code ILIKE '%' || @q || '%' OR c.name ILIKE '%' || @q || '%' OR c.tax_code ILIKE '%' || @q || '%')
  AND (sqlc.narg(active)::bool IS NULL OR c.active = sqlc.narg(active))
ORDER BY
    CASE WHEN @sort::text = 'code' THEN lower(c.code) END,
    CASE WHEN @sort = '-code' THEN lower(c.code) END DESC,
    CASE WHEN @sort = 'name' THEN c.name END,
    CASE WHEN @sort = '-name' THEN c.name END DESC,
    c.id
LIMIT @lim OFFSET @off;

-- name: CreateItem :one
INSERT INTO sales.items (code, name, unit, price, vat_rate, active) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id;

-- name: UpdateItem :execrows
UPDATE sales.items SET code = $2, name = $3, unit = $4, price = $5, vat_rate = $6, active = $7 WHERE id = $1;

-- name: GetItem :one
SELECT * FROM sales.items WHERE id = $1;

-- name: ItemsByID :many
SELECT * FROM sales.items WHERE id = ANY(@ids::bigint[]);

-- name: ListItems :many
SELECT *, count(*) OVER () AS total
FROM sales.items
WHERE (@q::text = '' OR code ILIKE '%' || @q || '%' OR name ILIKE '%' || @q || '%')
  AND (sqlc.narg(active)::bool IS NULL OR active = sqlc.narg(active))
ORDER BY lower(code), id
LIMIT @lim OFFSET @off;

-- name: InsertHeader :exec
INSERT INTO sales.headers (id, kind, request_id, customer_id, customer_code, customer_name, customer_tax_code,
    customer_address, customer_phone, customer_email, contact_name, valid_until, delivery_date, quote_id,
    payment_terms, delivery_terms, note, subtotal, discount_total, vat_total, total)
VALUES (@id, @kind, CAST(@request_id::text AS uuid), @customer_id, @customer_code, @customer_name, @customer_tax_code,
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

-- name: HeaderByRequest :one
SELECT id, kind FROM sales.headers WHERE request_id = CAST(@request_id::text AS uuid);

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

-- name: InsertLine :exec
INSERT INTO sales.lines (doc_id, position, item_id, item_code, description, unit, quantity, unit_price,
    discount_percent, vat_rate, amount, discount, vat)
VALUES (@doc_id, @position, @item_id, @item_code, @description, @unit, CAST(@quantity::text AS numeric), @unit_price,
    CAST(@discount_percent::text AS numeric), @vat_rate, @amount, @discount, @vat);

-- name: GetLines :many
SELECT item_id, item_code, description, unit, quantity::text AS quantity, unit_price,
       discount_percent::text AS discount_percent, vat_rate, amount, discount, vat
FROM sales.lines WHERE doc_id = $1 ORDER BY position;

-- name: ListHeaders :many
-- Scope filter: @all_units or the document's org unit in @units.
SELECT h.id, d.number, d.status, d.date, h.customer_id, h.customer_code, h.customer_name, h.valid_until, h.total,
       u.name AS org_unit_name, q.number AS quote_number, coalesce(o.number, '')::text AS order_number, count(*) OVER () AS total_rows
FROM sales.headers h
JOIN record.documents d ON d.id = h.id
JOIN iam.org_units u ON u.id = d.org_unit_id
LEFT JOIN record.documents q ON q.id = h.quote_id
LEFT JOIN LATERAL (
    SELECT od.number
    FROM sales.headers oh JOIN record.documents od ON od.id = oh.id
    WHERE oh.quote_id = h.id AND od.status <> 'cancelled'
    ORDER BY od.id DESC LIMIT 1
) o ON true
WHERE h.kind = @kind::text
  AND (@all_units::bool OR d.org_unit_id = ANY(@units::bigint[]))
  AND (@status::text = '' OR d.status = @status)
  AND (sqlc.narg(customer_id)::bigint IS NULL OR h.customer_id = sqlc.narg(customer_id))
  AND (@q::text = '' OR d.number ILIKE '%' || @q || '%' OR h.customer_name ILIKE '%' || @q || '%' OR h.customer_code ILIKE '%' || @q || '%')
ORDER BY
    CASE WHEN @sort::text = 'date' THEN d.date END,
    CASE WHEN @sort = '-date' THEN d.date END DESC,
    CASE WHEN @sort = 'number' THEN d.number END,
    CASE WHEN @sort = '-number' THEN d.number END DESC,
    CASE WHEN @sort = 'total' THEN h.total END,
    CASE WHEN @sort = '-total' THEN h.total END DESC,
    h.id DESC
LIMIT @lim OFFSET @off;

-- name: PrintLegalEntity :one
SELECT coalesce(legal_name, name)::text AS name, tax_code, address FROM iam.org_units WHERE id = $1;
