CREATE TABLE sales.deliveries (
    id           bigint PRIMARY KEY REFERENCES record.documents DEFERRABLE INITIALLY DEFERRED,
    order_id     bigint NOT NULL REFERENCES sales.headers,
    warehouse_id bigint NOT NULL REFERENCES stock.warehouses,
    note         text
);

CREATE INDEX deliveries_order_id_idx ON sales.deliveries (order_id);
CREATE INDEX deliveries_warehouse_id_idx ON sales.deliveries (warehouse_id);

-- The order line is (order_id of the delivery, order_position); item and texts are copied from it.
CREATE TABLE sales.delivery_lines (
    delivery_id    bigint NOT NULL REFERENCES sales.deliveries ON DELETE CASCADE,
    position       int NOT NULL CHECK (position > 0),
    order_position int NOT NULL CHECK (order_position > 0),
    item_id        bigint NOT NULL REFERENCES item.items,
    item_code      text NOT NULL,
    description    text NOT NULL,
    unit           text NOT NULL,
    lot_code       text,
    serial_code    text,
    quantity       numeric(15, 3) NOT NULL CHECK (quantity > 0),
    PRIMARY KEY (delivery_id, position)
);

CREATE INDEX delivery_lines_item_id_idx ON sales.delivery_lines (item_id);
