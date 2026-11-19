CREATE TABLE sales.invoices (
    id             bigint PRIMARY KEY REFERENCES record.documents DEFERRABLE INITIALLY DEFERRED,
    order_id       bigint NOT NULL REFERENCES sales.headers,
    subtotal       bigint NOT NULL,
    discount_total bigint NOT NULL,
    vat_total      bigint NOT NULL,
    total          bigint NOT NULL,
    note           text
);

CREATE INDEX invoices_order_id_idx ON sales.invoices (order_id);

CREATE TABLE sales.invoice_lines (
    invoice_id       bigint NOT NULL REFERENCES sales.invoices ON DELETE CASCADE,
    position         int NOT NULL CHECK (position > 0),
    order_position   int NOT NULL CHECK (order_position > 0),
    item_id          bigint NOT NULL REFERENCES item.items,
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
    PRIMARY KEY (invoice_id, position)
);

CREATE INDEX invoice_lines_item_id_idx ON sales.invoice_lines (item_id);

CREATE TABLE sales.receipts (
    id          bigint PRIMARY KEY REFERENCES record.documents DEFERRABLE INITIALLY DEFERRED,
    customer_id bigint NOT NULL REFERENCES sales.customers,
    method      text NOT NULL CHECK (method IN ('cash', 'bank')),
    amount      bigint NOT NULL CHECK (amount > 0),
    note        text
);

CREATE INDEX receipts_customer_id_idx ON sales.receipts (customer_id);

CREATE TABLE sales.receipt_allocations (
    receipt_id bigint NOT NULL REFERENCES sales.receipts ON DELETE CASCADE,
    position   int NOT NULL CHECK (position > 0),
    invoice_id bigint NOT NULL REFERENCES sales.invoices,
    amount     bigint NOT NULL CHECK (amount > 0),
    PRIMARY KEY (receipt_id, position),
    UNIQUE (receipt_id, invoice_id)
);

CREATE INDEX receipt_allocations_invoice_id_idx ON sales.receipt_allocations (invoice_id);
