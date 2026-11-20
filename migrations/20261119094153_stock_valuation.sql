-- Every move carries a value; a value-only move (a price difference) has no warehouse and no quantity.
ALTER TABLE stock.moves ADD COLUMN value bigint NOT NULL DEFAULT 0;
ALTER TABLE stock.moves ALTER COLUMN warehouse_id DROP NOT NULL;
ALTER TABLE stock.moves DROP CONSTRAINT moves_quantity_check;
ALTER TABLE stock.moves ADD CHECK (quantity <> 0 OR value <> 0);
ALTER TABLE stock.moves ADD CHECK (warehouse_id IS NOT NULL OR (quantity = 0 AND lot_id IS NULL AND serial_id IS NULL));

-- Running totals of a legal entity's moves of an item; unit_cost is the last average seen
-- with a positive quantity, used when there is none on hand.
CREATE TABLE stock.costs (
    legal_entity_id bigint NOT NULL REFERENCES iam.org_units,
    item_id         bigint NOT NULL REFERENCES item.items,
    quantity        numeric(15,3) NOT NULL,
    value           bigint NOT NULL,
    unit_cost       numeric,
    PRIMARY KEY (legal_entity_id, item_id)
);
CREATE INDEX costs_item_idx ON stock.costs (item_id);

INSERT INTO stock.costs (legal_entity_id, item_id, quantity, value)
SELECT w.legal_entity_id, b.item_id, sum(b.quantity), 0
FROM stock.balances b JOIN stock.warehouses w ON w.id = b.warehouse_id
GROUP BY w.legal_entity_id, b.item_id;

-- Unit cost of an increase line; decreases are valued at average.
ALTER TABLE stock.adjustment_lines ADD COLUMN unit_cost bigint CHECK (unit_cost >= 0);
UPDATE stock.adjustment_lines SET unit_cost = 0 WHERE quantity > 0;
ALTER TABLE stock.adjustment_lines ADD CHECK ((quantity > 0) = (unit_cost IS NOT NULL));
