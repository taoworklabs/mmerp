# 0110. Stock is valued at moving average per legal entity and item; goods reach stock at the goods receipt

Status: accepted · 2026-11-19

## Context

Stock moves carry quantities only ([ADR-0100](./0100-stock-area-transfers-stocktakes.md)). The ledger has no cost of goods sold, and goods reach the stock account when the supplier invoice is booked, not when they arrive ([ADR-0090](./0090-purchasing-product.md#consequences)), so the stock account and the warehouse never agree. Vietnamese SMBs (Circular 133/200) mostly use moving average or period-end average; FIFO is allowed but rare in this segment and needs cost layers. Moves are written once and never edited (`stock.Record`/`stock.Void`); valuation must keep that.

## Decision

- **Moving average, perpetual.** Each incoming move recomputes the unit cost; each outgoing move is valued at the unit cost of that moment.
- **One average per legal entity and item.** Not per warehouse, lot or serial. A transfer changes no value and writes no business line.
- **Cost follows posting order, not document date.** A document dated earlier than moves already posted takes the average at the time it posts; later moves are never revalued. The stock card shows the value each move was written with.
- **Every move carries a value** (`bigint`, minor unit, signed like the quantity), written with the move. `shared/stock` keeps a running quantity, value and last unit cost per legal entity and item, updated under lock in the same transaction. An outgoing move is valued `round(quantity × value / quantity on hand)` through the platform rounding function; the move that brings on hand to zero takes the whole remaining value.
- **Where the value of an incoming move comes from:**
  - goods receipt: the net unit price of its purchase order line;
  - stock adjustment, increase: a unit cost entered on the line (how opening stock is valued); decrease: the average;
  - stocktake surplus or shortage: the average;
  - transfer: the average, out of the source and into the destination at the same value.
- **No known cost.** A move valued at average when the legal entity has none of the item on hand (negative stock) uses the last unit cost; if the item never had one, it is valued at zero and the next receipt absorbs the difference. Refusing would make negative stock useless for a new item.
- **Cancelling** reverses each move at the value it was written with, not at the current average.
- **Business lines.** `shared/stock` exports one kind, `stock_inventory` (sample rule: 156), that every document moving goods writes for the value of its moves, with its own counterpart kind:
  - goods receipt: `purchase_received_not_invoiced` credit (sample: 3388), so goods reach 156 when they arrive;
  - supplier invoice: stock lines now debit `purchase_received_not_invoiced` at the order price instead of `purchase_goods`; the difference between the invoice's net and the order price becomes a value-only move on the item (so `stock_inventory`) while the legal entity has the item on hand, and `purchase_price_difference` (sample: 632) when it has none;
  - sales delivery: `sales_cost_of_goods_sold` (sample: 632);
  - stock adjustment: `stock_adjustment_gain` / `stock_adjustment_loss` (samples: 711, 811);
  - stocktake: `stocktake_surplus` (sample: 3381) / `stocktake_shortage` (sample: 1381).
- `purchase_goods` stays a known kind for invoices posted before this change; new invoices no longer write it on stock lines.

## Rejected alternatives

- **FIFO.** Needs cost layers per receipt and their consumption on every issue and reversal; little demand among the target SMBs.
- **Period-end weighted average.** Issues have no cost until the period closes, so deliveries cannot post cost of goods sold and stock value is unknown during the month.
- **Average per warehouse.** Transfers would carry cost between warehouses and 156 would be split per warehouse; SMBs keep one 156 per company.
- **Revaluing later moves when a document is backdated.** Breaks "written once" and would rewrite posted entries; a recompute job at period lock was rejected for the same reason.
- **Keeping 156 at the supplier invoice.** Moving average needs the cost when goods arrive, and invoices are by quantity, independent of receipts.
- **Booking the receipt to 331 at a provisional price.** Puts a payable on the supplier before an invoice exists, which payments cannot allocate to.

## Consequences

- `stock.moves` gains `value`; a value-only move (quantity zero) is allowed for price differences, so the non-zero quantity check moves to "quantity or value".
- A new cost row per legal entity and item is locked `FOR UPDATE`, sorted by item, after the `stock.balances` rows; the documented lock order gains this step.
- Sales deliveries, goods receipts, adjustments and stocktakes start writing business lines; accounting picks them up through the existing jobs. Transfers do not.
- A price difference on an invoice posted before its goods are received adjusts the average of the stock already on hand, not of those goods. Accepted for SMBs; exact matching would need invoices tied to receipts, which ADR-0090 rejected.
- Cancelling an incoming move after part was issued (negative stock only) can leave value with zero quantity; it stays in the running value and is absorbed by the next receipt.
- Opening stock entered by adjustment before the legal entity's accounting start date never reaches the ledger; its 156 balance is part of the opening manual entry ([ADR-0050](./0050-accounting-product-and-general-ledger.md)).
- No tenant is in production: existing moves get value zero, and dev and demo data are reseeded with valued opening stock.
- Consignment, landed cost (freight added to receipts) and returns need decisions of their own.
