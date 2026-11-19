CREATE SCHEMA purchase;

-- Purchasing's role of a partner: identity stays on the partner.
CREATE TABLE purchase.suppliers (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code          text NOT NULL,
    partner_id    bigint NOT NULL REFERENCES partner.partners,
    org_unit_id   bigint NOT NULL REFERENCES iam.org_units,
    payment_terms text,
    active        boolean NOT NULL DEFAULT true,
    -- Its index also serves lookups by partner.
    CONSTRAINT suppliers_partner_org_unit_key UNIQUE (partner_id, org_unit_id)
);

CREATE UNIQUE INDEX suppliers_code_key ON purchase.suppliers (lower(code));
CREATE INDEX suppliers_org_unit_id_idx ON purchase.suppliers (org_unit_id);

CREATE TABLE purchase.orders (
    id                bigint PRIMARY KEY REFERENCES record.documents DEFERRABLE INITIALLY DEFERRED,
    supplier_id       bigint NOT NULL REFERENCES purchase.suppliers,
    supplier_code     text NOT NULL,
    supplier_name     text NOT NULL,
    supplier_tax_code text,
    supplier_address  text,
    supplier_phone    text,
    supplier_email    text,
    contact_name      text,
    expected_date     date,
    payment_terms     text,
    note              text,
    subtotal          bigint NOT NULL,
    discount_total    bigint NOT NULL,
    vat_total         bigint NOT NULL,
    total             bigint NOT NULL,
    CHECK (total = subtotal - discount_total + vat_total)
);

CREATE INDEX orders_supplier_id_idx ON purchase.orders (supplier_id);

-- stocked: whether the item had stock tracking when the line was saved; goods receipts take
-- only those lines, and invoices book them as goods rather than expenses.
CREATE TABLE purchase.order_lines (
    order_id         bigint NOT NULL REFERENCES purchase.orders ON DELETE CASCADE,
    position         int NOT NULL CHECK (position > 0),
    item_id          bigint NOT NULL REFERENCES item.items,
    item_code        text NOT NULL,
    description      text NOT NULL,
    unit             text NOT NULL,
    stocked          boolean NOT NULL,
    quantity         numeric(15, 3) NOT NULL CHECK (quantity > 0),
    unit_price       bigint NOT NULL CHECK (unit_price >= 0),
    discount_percent numeric(5, 2) NOT NULL CHECK (discount_percent BETWEEN 0 AND 100),
    vat_rate         text NOT NULL CHECK (vat_rate IN ('none', '0', '5', '8', '10')),
    amount           bigint NOT NULL,
    discount         bigint NOT NULL,
    vat              bigint NOT NULL,
    PRIMARY KEY (order_id, position)
);

CREATE INDEX order_lines_item_id_idx ON purchase.order_lines (item_id);

CREATE TABLE purchase.goods_receipts (
    id           bigint PRIMARY KEY REFERENCES record.documents DEFERRABLE INITIALLY DEFERRED,
    order_id     bigint NOT NULL REFERENCES purchase.orders,
    warehouse_id bigint NOT NULL REFERENCES stock.warehouses,
    note         text
);

CREATE INDEX goods_receipts_order_id_idx ON purchase.goods_receipts (order_id);
CREATE INDEX goods_receipts_warehouse_id_idx ON purchase.goods_receipts (warehouse_id);

-- The order line is (order_id of the receipt, order_position); item and texts are copied from it.
CREATE TABLE purchase.goods_receipt_lines (
    receipt_id     bigint NOT NULL REFERENCES purchase.goods_receipts ON DELETE CASCADE,
    position       int NOT NULL CHECK (position > 0),
    order_position int NOT NULL CHECK (order_position > 0),
    item_id        bigint NOT NULL REFERENCES item.items,
    item_code      text NOT NULL,
    description    text NOT NULL,
    unit           text NOT NULL,
    lot_code       text,
    lot_expiry     date,
    serial_code    text,
    quantity       numeric(15, 3) NOT NULL CHECK (quantity > 0),
    PRIMARY KEY (receipt_id, position),
    CHECK (lot_code IS NULL OR serial_code IS NULL),
    CHECK (lot_expiry IS NULL OR lot_code IS NOT NULL)
);

CREATE INDEX goods_receipt_lines_item_id_idx ON purchase.goods_receipt_lines (item_id);

-- The supplier's own invoice reference is empty on a draft and required to send.
CREATE TABLE purchase.invoices (
    id                      bigint PRIMARY KEY REFERENCES record.documents DEFERRABLE INITIALLY DEFERRED,
    order_id                bigint NOT NULL REFERENCES purchase.orders,
    supplier_invoice_symbol text,
    supplier_invoice_no     text,
    supplier_invoice_date   date,
    subtotal                bigint NOT NULL,
    discount_total          bigint NOT NULL,
    vat_total               bigint NOT NULL,
    total                   bigint NOT NULL,
    note                    text,
    CHECK (total = subtotal - discount_total + vat_total)
);

CREATE INDEX invoices_order_id_idx ON purchase.invoices (order_id);
CREATE INDEX invoices_supplier_ref_idx ON purchase.invoices (supplier_invoice_no, supplier_invoice_symbol);

CREATE TABLE purchase.invoice_lines (
    invoice_id       bigint NOT NULL REFERENCES purchase.invoices ON DELETE CASCADE,
    position         int NOT NULL CHECK (position > 0),
    order_position   int NOT NULL CHECK (order_position > 0),
    item_id          bigint NOT NULL REFERENCES item.items,
    item_code        text NOT NULL,
    description      text NOT NULL,
    unit             text NOT NULL,
    stocked          boolean NOT NULL,
    quantity         numeric(15, 3) NOT NULL CHECK (quantity > 0),
    unit_price       bigint NOT NULL CHECK (unit_price >= 0),
    discount_percent numeric(5, 2) NOT NULL CHECK (discount_percent BETWEEN 0 AND 100),
    vat_rate         text NOT NULL CHECK (vat_rate IN ('none', '0', '5', '8', '10')),
    amount           bigint NOT NULL,
    discount         bigint NOT NULL,
    vat              bigint NOT NULL,
    PRIMARY KEY (invoice_id, position)
);

CREATE INDEX invoice_lines_item_id_idx ON purchase.invoice_lines (item_id);

CREATE TABLE purchase.payments (
    id          bigint PRIMARY KEY REFERENCES record.documents DEFERRABLE INITIALLY DEFERRED,
    supplier_id bigint NOT NULL REFERENCES purchase.suppliers,
    method      text NOT NULL CHECK (method IN ('cash', 'bank')),
    amount      bigint NOT NULL CHECK (amount > 0),
    note        text
);

CREATE INDEX payments_supplier_id_idx ON purchase.payments (supplier_id);

CREATE TABLE purchase.payment_allocations (
    payment_id bigint NOT NULL REFERENCES purchase.payments ON DELETE CASCADE,
    position   int NOT NULL CHECK (position > 0),
    invoice_id bigint NOT NULL REFERENCES purchase.invoices,
    amount     bigint NOT NULL CHECK (amount > 0),
    PRIMARY KEY (payment_id, position),
    UNIQUE (payment_id, invoice_id)
);

CREATE INDEX payment_allocations_invoice_id_idx ON purchase.payment_allocations (invoice_id);
