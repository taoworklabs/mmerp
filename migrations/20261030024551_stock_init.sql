CREATE SCHEMA stock;

CREATE TABLE stock.warehouses (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code           text NOT NULL,
    name           text NOT NULL,
    legal_entity_id bigint NOT NULL REFERENCES iam.org_units,
    is_default     boolean NOT NULL DEFAULT false,
    active         boolean NOT NULL DEFAULT true
);
CREATE UNIQUE INDEX warehouses_code_key ON stock.warehouses (lower(code));
CREATE UNIQUE INDEX warehouses_default_key ON stock.warehouses (legal_entity_id) WHERE is_default;

CREATE TABLE stock.lots (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    item_id     bigint NOT NULL REFERENCES item.items,
    code        text NOT NULL,
    expiry      date,
    received_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX lots_code_key ON stock.lots (item_id, lower(code));

CREATE TABLE stock.serials (
    id      bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    item_id bigint NOT NULL REFERENCES item.items,
    code    text NOT NULL
);
CREATE UNIQUE INDEX serials_code_key ON stock.serials (item_id, lower(code));

-- Written once per document line; a reversal points at the move it undoes.
CREATE TABLE stock.moves (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    doc_type     text NOT NULL,
    doc_id       bigint NOT NULL,
    line_no      int NOT NULL,
    warehouse_id bigint NOT NULL REFERENCES stock.warehouses,
    item_id      bigint NOT NULL REFERENCES item.items,
    lot_id       bigint REFERENCES stock.lots,
    serial_id    bigint REFERENCES stock.serials,
    quantity     numeric(15,3) NOT NULL CHECK (quantity <> 0),
    date         date NOT NULL,
    reverses     bigint REFERENCES stock.moves
);
CREATE INDEX moves_doc_idx ON stock.moves (doc_type, doc_id);
CREATE INDEX moves_item_idx ON stock.moves (item_id);

CREATE TABLE stock.balances (
    warehouse_id bigint NOT NULL REFERENCES stock.warehouses,
    item_id      bigint NOT NULL REFERENCES item.items,
    lot_id       bigint REFERENCES stock.lots,
    serial_id    bigint REFERENCES stock.serials,
    quantity     numeric(15,3) NOT NULL
);
CREATE UNIQUE INDEX balances_key ON stock.balances (warehouse_id, item_id, coalesce(lot_id, 0), coalesce(serial_id, 0));

CREATE TABLE stock.adjustments (
    id           bigint PRIMARY KEY REFERENCES record.documents DEFERRABLE INITIALLY DEFERRED,
    warehouse_id bigint NOT NULL REFERENCES stock.warehouses,
    reason       text
);

CREATE TABLE stock.adjustment_lines (
    adjustment_id bigint NOT NULL REFERENCES stock.adjustments,
    line_no       int NOT NULL,
    item_id       bigint NOT NULL REFERENCES item.items,
    lot_code      text,
    lot_expiry    date,
    serial_code   text,
    quantity      numeric(15,3) NOT NULL CHECK (quantity <> 0),
    PRIMARY KEY (adjustment_id, line_no)
);

-- One default warehouse per legal entity; the series continues after them.
INSERT INTO stock.warehouses (code, name, legal_entity_id, is_default)
SELECT 'KHO' || lpad(row_number() OVER (ORDER BY id)::text, 5, '0'), 'Kho chính', id, true
FROM iam.org_units WHERE kind = 'company';

INSERT INTO numbering.series (code, prefix, last)
VALUES ('stock.warehouse', 'KHO', (SELECT count(*) FROM stock.warehouses))
ON CONFLICT DO NOTHING;
