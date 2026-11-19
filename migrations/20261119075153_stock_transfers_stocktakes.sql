CREATE TABLE stock.transfers (
    id                bigint PRIMARY KEY REFERENCES record.documents DEFERRABLE INITIALLY DEFERRED,
    from_warehouse_id bigint NOT NULL REFERENCES stock.warehouses,
    to_warehouse_id   bigint NOT NULL REFERENCES stock.warehouses,
    note              text NOT NULL DEFAULT '',
    CHECK (from_warehouse_id <> to_warehouse_id)
);

CREATE TABLE stock.transfer_lines (
    transfer_id bigint NOT NULL REFERENCES stock.transfers,
    line_no     int NOT NULL,
    item_id     bigint NOT NULL REFERENCES item.items,
    lot_code    text,
    serial_code text,
    quantity    numeric(15,3) NOT NULL CHECK (quantity > 0),
    PRIMARY KEY (transfer_id, line_no)
);

CREATE TABLE stock.stocktakes (
    id           bigint PRIMARY KEY REFERENCES record.documents DEFERRABLE INITIALLY DEFERRED,
    warehouse_id bigint NOT NULL REFERENCES stock.warehouses,
    note         text NOT NULL DEFAULT ''
);

-- book and difference are set when the stocktake posts.
CREATE TABLE stock.stocktake_lines (
    stocktake_id bigint NOT NULL REFERENCES stock.stocktakes,
    line_no      int NOT NULL,
    item_id      bigint NOT NULL REFERENCES item.items,
    lot_code     text,
    lot_expiry   date,
    serial_code  text,
    counted      numeric(15,3) NOT NULL CHECK (counted >= 0),
    book         numeric(15,3),
    difference   numeric(15,3),
    PRIMARY KEY (stocktake_id, line_no),
    CHECK ((book IS NULL) = (difference IS NULL))
);
