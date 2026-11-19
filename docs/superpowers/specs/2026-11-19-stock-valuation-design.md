# Stock valuation (M20)

Decision record: [ADR-0110](../../adr/0110-moving-average-stock-valuation.md). This document is what a builder plans from.

## Why

Stock moves carry quantities only. The ledger never books cost of goods sold, and goods reach 156 at the supplier invoice, so the stock account and the warehouse never agree. M20 puts a value on every move and books it.

The product owner left the business choices to the builder ("tôi k rõ nghiệp vụ này bạn đề xuất đi"); every choice below is the builder's call, recorded so it can be checked.

## Goals

- A value on every stock move, at moving average per legal entity and item.
- Business lines for goods receipts, supplier invoices (stock lines), sales deliveries, stock adjustments and stocktakes.
- A unit cost on the incoming lines of stock adjustments (how opening stock is valued).
- The stock card shows each move's value.

## Non-goals

- FIFO, period-end average, standard cost; revaluing on backdated documents.
- Landed cost, consignment, returns, goods in transit.
- A stock value report per warehouse (the 156 balance is the value per legal entity); on-hand value screens.

## Design

### Ledger (`shared/stock`)

- `stock.moves.value bigint NOT NULL DEFAULT 0`, signed like the quantity. `warehouse_id` becomes nullable: a **value-only move** (quantity 0, no warehouse, lot or serial) changes the value only. The check becomes `quantity <> 0 OR value <> 0`, and `warehouse_id IS NOT NULL OR quantity = 0`.
- `stock.costs (legal_entity_id, item_id, quantity numeric(15,3), value bigint, unit_cost numeric)`, primary key on the pair: running totals of the legal entity's moves of the item, and the last unit cost seen with a positive quantity. The migration fills it from `stock.balances` with value 0 and no unit cost; existing moves keep value 0.
- `Move.Value *int64`: the value of an incoming move when the caller knows it (goods receipt, adjustment increase, value-only move). Nil means "at average": outgoing moves, transfers, stocktake differences.
- Valuation runs in `apply`, after the balances are updated and before the moves are inserted, per item of the document:
  1. Lock the cost rows `FOR UPDATE`, sorted by item (inserting missing ones first).
  2. Average = value / quantity when quantity > 0, else the last unit cost, else zero.
  3. Each nil-valued move: `platform.Round(quantity × average)`.
  4. If the item's new quantity is zero, the last nil-valued outgoing move of the item takes the residue so the new value is zero.
  5. Value-only moves apply only when the quantity before the document is > 0; otherwise they are not written and their value is 0.
  6. New totals; unit cost = new value / new quantity when new quantity > 0, else unchanged.
- `Void` reverses each open move with `-value`, through the same `apply`, so totals and unit cost move back the same way; the residue rule does not apply to reversals (they carry their value).
- `Record` returns the value written for each move, in move order (`[]int64`); `CountResult` gains `Value`. `Void` is unchanged.
- Lock order: after the `stock.balances` rows, `stock.costs` rows `FOR UPDATE` sorted by item. `AGENTS.md` and `docs/documents.md` gain the step.

### Business lines

`shared/stock` declares `stock_inventory`, `stock_adjustment_gain`, `stock_adjustment_loss`, `stocktake_surplus`, `stocktake_shortage`; Sales declares `sales_cost_of_goods_sold`; Purchasing declares `purchase_received_not_invoiced`, `purchase_price_difference`. Amounts follow the existing convention: positive, the rule gives the side, negative goes opposite. Sample rules (localisation pack, not shipped): 156 debit; 711 credit; 811 debit; 3381 credit; 1381 debit; 632 debit; 3388 credit; 632 debit.

| Document | Lines on posting (V = sum of move values) |
| --- | --- |
| Goods receipt | `stock_inventory` +V, `purchase_received_not_invoiced` +V |
| Supplier invoice, stock lines | `purchase_received_not_invoiced` −O (O = order price of the invoiced quantities), `stock_inventory` +A (value-only moves applied), `purchase_price_difference` +(D − A) (D = invoice net − O) |
| Sales delivery | `stock_inventory` V (negative), `sales_cost_of_goods_sold` −V |
| Stock adjustment | increase: `stock_inventory` +V⁺, `stock_adjustment_gain` +V⁺; decrease: `stock_inventory` V⁻, `stock_adjustment_loss` −V⁻ |
| Stocktake | the same split with `stocktake_surplus` / `stocktake_shortage` |
| Transfer | none (values cancel) |

- Order price of a quantity: `platform.Round((amount − discount) × quantity / order quantity)` of the order line, for receipts and invoices alike.
- Each document writes lines through one `posting.Record` and voids them with `posting.Void` on cancelling, after `stock.Void`. Zero-amount lines are dropped (`nonZero`); a document with no non-zero line still records (empty recording), as today.
- Invoices write `purchase_goods` no more; the kind stays declared.
- `shared/stock` gains `Posting` in its deps.

### Stock adjustment

- Line field `unit_cost` (int64, minor unit, ≥ 0): required on increase lines (`stock_unit_cost_required`), refused on decrease lines (`stock_unit_cost_not_allowed`), both on save, params `line_no`. Column `stock.adjustment_lines.unit_cost bigint` (null on decreases; existing increase lines get 0).
- Value of an increase line: `platform.Round(quantity × unit_cost)`.
- Form: a unit cost column, enabled when the quantity is positive; the posted view shows it. Print template gains the column.

### Stock card

- The stock card API returns each move's `value`; the page shows a value column. Value-only moves have no warehouse and do not appear on a warehouse's card.

### Seed and tests

- `make seed` opening adjustments carry unit costs; Playwright adjustment flows fill them.
- Go tests: valuation in `ledger_value_test.go` (average after receipts, issue at average, residue on emptying, negative stock with last cost, no cost at all, void at original value, value-only moves with and without stock, transfer values cancel); business lines per document in each document's tests.

## Exit criteria (roadmap M20)

With `make seed` data and `PRODUCTS=hrm,sales,purchase,accounting`:

1. Opening stock entered by adjustment with unit costs; the stock card shows the values.
2. Two goods receipts of one item at different order prices give the weighted average; a delivery then books cost of goods sold at it, and the delivery emptying the stock leaves value zero.
3. A goods receipt books 156 against received-not-invoiced; the supplier invoice clears it; a price difference goes to stock when on hand and to the price difference kind when not.
4. Stocktake differences and adjustments book their kinds; transfers book nothing.
5. Cancelling any of these reverses the value it posted; the average returns to what it was.
6. A delivery of an item allowing negative stock and never received posts at value zero; the next receipt absorbs it. `recordtest` passes for every type touched.

## Left to the builder

Slicing and order; test names; how the stock card formats values; seed prices.
