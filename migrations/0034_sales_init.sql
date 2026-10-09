CREATE SCHEMA sales;

-- A customer is seen by whoever holds a sales role at its owning org unit or above.
CREATE TABLE sales.customers (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code          text NOT NULL,
    name          text NOT NULL,
    tax_code      text,
    address       text,
    phone         text,
    email         text,
    contact_name  text,
    payment_terms text,
    org_unit_id   bigint NOT NULL REFERENCES iam.org_units,
    active        boolean NOT NULL DEFAULT true
);

CREATE UNIQUE INDEX customers_code_key ON sales.customers (lower(code));
CREATE INDEX customers_org_unit_id_idx ON sales.customers (org_unit_id);

-- One catalogue for the tenant: goods and services alike.
CREATE TABLE sales.items (
    id       bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code     text NOT NULL,
    name     text NOT NULL,
    unit     text NOT NULL,
    price    bigint NOT NULL CHECK (price >= 0),
    vat_rate text NOT NULL CHECK (vat_rate IN ('none', '0', '5', '8', '10')),
    active   boolean NOT NULL DEFAULT true
);

CREATE UNIQUE INDEX items_code_key ON sales.items (lower(code));

-- A quotation or order; the id, number, status, date and org unit live in record.documents
-- (deferred FK: record inserts its row first in the same transaction). The customer columns
-- are copies taken on each draft save, so a document keeps what it said.
CREATE TABLE sales.headers (
    id                bigint PRIMARY KEY REFERENCES record.documents DEFERRABLE INITIALLY DEFERRED,
    kind              text NOT NULL CHECK (kind IN ('quote', 'order')),
    request_id        uuid NOT NULL,
    customer_id       bigint NOT NULL REFERENCES sales.customers,
    customer_code     text NOT NULL,
    customer_name     text NOT NULL,
    customer_tax_code text,
    customer_address  text,
    customer_phone    text,
    customer_email    text,
    contact_name      text,
    valid_until       date,
    delivery_date     date,
    quote_id          bigint REFERENCES sales.headers,
    payment_terms     text,
    delivery_terms    text,
    note              text,
    subtotal          bigint NOT NULL,
    discount_total    bigint NOT NULL,
    vat_total         bigint NOT NULL,
    total             bigint NOT NULL,
    CHECK (total = subtotal - discount_total + vat_total),
    CHECK (kind = 'quote' AND valid_until IS NOT NULL AND delivery_date IS NULL AND quote_id IS NULL
        OR kind = 'order' AND valid_until IS NULL)
);

CREATE UNIQUE INDEX headers_request_id_key ON sales.headers (request_id);
CREATE INDEX headers_customer_id_idx ON sales.headers (customer_id);
CREATE INDEX headers_quote_id_idx ON sales.headers (quote_id);

CREATE TABLE sales.lines (
    doc_id           bigint NOT NULL REFERENCES sales.headers ON DELETE CASCADE,
    position         int NOT NULL CHECK (position > 0),
    item_id          bigint NOT NULL REFERENCES sales.items,
    item_code        text NOT NULL,
    description      text NOT NULL,
    unit             text NOT NULL,
    quantity         numeric(15, 3) NOT NULL CHECK (quantity > 0),
    unit_price       bigint NOT NULL CHECK (unit_price >= 0),
    discount_percent numeric(5, 2) NOT NULL CHECK (discount_percent BETWEEN 0 AND 100),
    vat_rate         text NOT NULL CHECK (vat_rate IN ('none', '0', '5', '8', '10')),
    amount           bigint NOT NULL,
    discount         bigint NOT NULL,
    vat              bigint NOT NULL,
    PRIMARY KEY (doc_id, position)
);

CREATE INDEX lines_item_id_idx ON sales.lines (item_id);
