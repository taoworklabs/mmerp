# Stock (M19)

Decision record: [ADR-0100](../../adr/0100-stock-area-transfers-stocktakes.md). This document is what a builder plans from.

## Why

Since M18 goods come in through goods receipts and go out through deliveries, but the stock screens live in the Sales area: a tenant with Purchasing alone cannot see what it received. Warehouses cannot exchange goods except by two hand-made adjustments, and counting the shelf means working out each difference by hand. M19 gives stock its own area and adds the two documents every warehouse uses: transfers and stocktakes.

The product owner asked to proceed without a design conversation ("làm luôn"); every choice below is the builder's call, recorded so it can be checked.

## Goals

- **Stock area** (`web/src/stock`, home tile "Kho"): items, warehouses, on hand, stock card, stock adjustments, transfers, stocktakes. The Sales area loses its stock and item screens.
- **Transfer** (`stock.transfer`, document `CK`, vi: "Phiếu chuyển kho").
- **Stocktake** (`stock.stocktake`, document `KK`, vi: "Phiếu kiểm kê kho").
- Print templates, attachments and discussion for both; seed and Playwright.

## Non-goals

- **Valuation** (costing method, cost on moves, cost of goods sold, 156 following the goods receipt): M20, with its own decision.
- Goods in transit and two-step transfers; transfers between legal entities.
- Approval of stock documents; freezing a warehouse during a count; blind counts; counting by bin or location.
- Units of measure conversion, reservations, reorder points.

## Anti-goals

- A stocktake difference computed anywhere but under the balance lock at posting.
- A stock screen existing in two areas.
- Any change to how Sales or Purchasing documents move stock, or to their posting.

## Approaches

Where transfers and stocktakes live:

- **A. Record types of `shared/stock`, beside the adjustment (chosen).** The ledger, the `stock` namespace and `stock.manage` exist; no product gate, so any tenant that has goods may count and move them.
- **B. A new `inventory` product** (as ADR-0049 foresaw). Rejected: disabling it would leave a Purchasing-only tenant unable to count its stock, and it adds a product with nothing of its own but two documents.

How a stocktake gets its difference:

- **A. Counted quantity stored; difference computed at posting under the balance lock (chosen).**
- **B. Difference stored when the draft is saved.** Rejected: on hand moves between saving and posting.

Valuation in M19 or later: **later (chosen)**, see ADR-0100.

## Design

### Stock area

- Manifest `stock`: `product: 'stock'`, label "Kho", base path `/stock`, entered by any permission in `stock.` or `item.` (the default rule of `useAreasEntitled` already counts master-data nav permissions).
- Menu: on hand, stock card, transfers, stocktakes, adjustments; configuration group: items, warehouses.
- Pages move from `web/src/sales/pages` (`ItemsPage`, `WarehousesPage`, `OnHandPage`, `StockCardPage`, `AdjustmentsPage`, `AdjustmentPage`, `NewAdjustmentPage`) with their components, hooks, keys and i18n; their `recordTypes` entries (`stock.adjustment`, items if declared) move to the stock manifest. Sales keeps what its own documents need (delivery form pickers through Sales routes).
- Links that pointed to `/sales/stock/...` or `/sales/items` now point to `/stock/...`. *Builder's call*: no redirects from the old paths (no released installs depend on them).
- Home tile order: after Purchasing, before Accounting.

### Transfer

- Header: date, source warehouse, destination warehouse, note (optional, ≤ 500). Both warehouses active and of the document's legal entity (the legal entity is the source warehouse's); source ≠ destination (`stock_transfer_same_warehouse`).
- Lines (1..500): item with stock tracking, lot code or serial as the item's tracking asks (`stock.CheckShape` on save), quantity > 0, up to 3 decimals. The form suggests lots from the source warehouse (`SuggestLots`) and lists serials on hand there (`SerialsOnHand`).
- Into `posted`: one `stock.Record` with, per line, a move of −quantity in the source and +quantity in the destination, same line number, same lot code (no expiry: the lot exists) or serial. Shortage in the source is refused with `stock_insufficient`. Into `cancelled`: `stock.Void`, refused when the goods have left the destination since (unless negative stock is allowed).
- No approval; `stock.manage` tenant-wide to write, `stock.view` to read. Actions as the adjustment: edit, delete, submit, cancel, print.
- Print template `stock.transfer` (lines with lot or serial; signatures: issued by, received by).

### Stocktake

- Header: date, warehouse, note (optional, ≤ 500).
- Lines (1..2000): item with stock tracking, lot code or serial as its tracking asks, counted quantity ≥ 0 (up to 3 decimals; serials 0 or 1). One line per (item, lot, serial) (`stock_stocktake_duplicate_line`). Lines not listed are not counted and do not change.
- **Load on hand**: a form action fills lines from the warehouse's current balances (non-zero, optionally one item), with counted = on hand, through `GET /stock/warehouses/{id}/on-hand-lines` (*builder's call* on the route shape). The user then edits counts and adds lots or serials found on the shelf.
- While draft, the page shows each line's current on hand and the difference it would make (display only).
- Into `posted`: `stock.RecordCounts(ctx, ref, warehouse, counts)`: resolves lots and serials (a counted lot or serial that does not exist yet is created, as for incoming moves), locks serials then balances in the usual order, reads on hand per key, writes a move of `counted − on hand` for each non-zero difference, and returns the book quantities. The stocktake stores `book_quantity` and `difference` on its lines in the same transaction. A serial counted 1 that is on hand in another warehouse is refused (`stock_serial_quantity`): transfer it first.
- Into `cancelled`: `stock.Void`, as adjustments.
- No approval; permissions and actions as the transfer. Posted lines show book, counted and difference.
- Print template `stock.stocktake` (count sheet: lines, book, counted, difference; signatures).

### Ledger

- `RecordCounts` shares `apply` with `Record` and `Void`: the targets turn into deltas between locking the balances and applying them, so lock order and shortage rules stay the same. Negative stock never arises from a count (counted ≥ 0).
- Transfers need no ledger change: `Record` already checks each move's warehouse against the legal entity and orders incoming before outgoing for one code; serial on-hand is checked after all deltas.

### Glossary

`CONTEXT.md`, Stock section: Transfer, Stocktake, Counted quantity, Book quantity, Stocktake difference, Stock area.

## Exit criteria (roadmap M19)

With `make seed` data and `PRODUCTS=hrm,sales,purchase`:

1. A user with `stock.admin` opens the Stock area and sees items, warehouses, on hand and the stock card; the Sales area has no stock or item screens. With `PRODUCTS=hrm,purchase` the Stock area still opens and writes.
2. A transfer of a quantity item, a lot and a serial from the default warehouse to a second one posts; on hand falls in one and grows in the other; the stock card of each shows the move. A transfer taking more than on hand is refused with the line named.
3. Cancelling the transfer returns both sides; cancelling after the goods left the destination is refused.
4. A stocktake loaded from on hand, with one count lowered, one raised and one new lot found, posts; on hand equals the counts, the posted lines show book quantity and difference, and lines not listed are unchanged.
5. A delivery posted between saving and posting the stocktake is taken into account: the difference is computed at posting.
6. Users without `stock.manage` cannot write either document through the API. `recordtest` passes for both types. Sales and Purchasing stock moves unchanged.

## Left to the builder

Slicing and order; table and column names; route shapes; print layouts; list filters beyond status and warehouse; seed contents; whether transfer and stocktake share a lines component with the adjustment.
