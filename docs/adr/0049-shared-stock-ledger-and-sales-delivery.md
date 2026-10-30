# 0049. Stock is a shared ledger in `shared/stock`; Sales delivers through it

Status: accepted · 2026-10-30

## Context

POS needs stock: selling at a till takes goods off a shelf. Sales, POS and a later Purchase product will all move goods, and a tenant that buys Sales alone must still see its stock fall as it delivers. [products/sales.md](../../products/sales.md#scope) left stock, warehouses and delivery out of the first release.

Other ERPs agree on two numbers: **on hand**, changed only when goods physically move (Odoo and SAP Business One on validating a delivery, ERPNext on a delivery note, MISA on a stock issue), and **committed**, what confirmed orders have promised but not yet delivered. Retail POS (KiotViet) deducts on sale, since selling and handing over are one act.

## Decision

### Where it lives

- **`internal/shared/stock`** (schema `stock`) owns warehouses, lots and serials, the stock ledger and balances. It is master data and a ledger, like `posting`: products import it directly and call it in their own transaction. It never imports or names a product.
- It registers its own namespace `stock.` ([ADR-0039](./0039-shared-master-data.md)): `stock.view` (Anywhere), `stock.manage` (tenant-wide); roles `stock.viewer`, `stock.admin`. `ProductGate` never refuses it, so stock moves with whichever product writes them.
- Screens (warehouses, on-hand, stock card, stock adjustments) live in the Sales area for now, as "Items" does. A later `inventory` product (receipts, issues, transfers, stocktakes) gets its own area and writes through the same ledger.

### Ledger

- **Warehouse** (`stock.warehouses`): code, name, legal entity, active, a default flag (one per legal entity). The migration gives each existing legal entity a default warehouse; a company created later (or a unit changed into one) gets it in the same transaction, through a synchronous `iam` hook wired in `internal/app` to `stock`, so a company never lacks a place to move stock. A legal entity still without one makes its next new warehouse the default.
- **Stock move** (`stock.moves`): one row per (document, line, warehouse, item, lot or serial) with a signed quantity, `numeric(15,3)`, the document's date. Written once by `stock.Record(ctx, ref, moves)` when the document posts; `stock.Void(ctx, ref)` writes the reversal when it is cancelled. Moves are never edited, as for posting lines.
- **Balance** (`stock.balances`): on-hand per (warehouse, item, lot or serial), updated in the same transaction as the moves. It is the row locked to check availability; the ledger stays the source of truth and the balance can be rebuilt from it.
- A move's warehouse must belong to the document's legal entity.
- **Lock order** extends the existing one: after the document row and before other module-owned rows, `stock.balances` rows `FOR UPDATE`, in (warehouse, item, lot) order so two documents never deadlock.

### Items

`item.items` gains:

- `tracking`: `none` (no stock: services; lines write no moves), `quantity`, `lot` or `serial`. Existing items become `none`.
- `allow_negative`: for `tracking = quantity` only, whether on-hand may go below zero. Default false. Lot and serial items can never go negative: one cannot issue a lot that was never received.

Changing `tracking` is refused once the item has a move.

### Lots and serials

- **Lot** (`stock.lots`): per item, a code unique within the item, optional expiry date. Created when goods come in.
- **Serial** (`stock.serials`): per item, a code unique within the item; its on-hand is 0 or 1 across all warehouses.
- An outgoing line of a lot or serial item names its lots or serials; one document line may split into several.
- **Suggestion:** for a lot item the form pre-fills lots by earliest expiry, then earliest receipt, skipping expired lots and splitting across lots as needed (FEFO, then FIFO). Serials are never guessed: the user scans or picks from those on hand in the warehouse. The suggestion is only a pre-fill; posting re-checks every lot and serial under the balance lock.
- Availability is checked against the current balance only, never by date: the period lock already stops writes into closed periods. The same check runs when `stock.Void` reverses incoming moves, so cancelling an adjustment whose goods were already delivered is refused unless the item allows negative stock.
- Shortage is refused with the stable code `stock.insufficient` and parameters (warehouse, item, lot or serial, missing quantity).

### Stock adjustment

`stock.adjustment` (document, prefix `DC`): one warehouse, a reason, and lines each stating a signed change of quantity per item and lot or serial. It is how opening stock gets in until `inventory` exists. An incoming line may create its lot (code, expiry) or serial inline. Lines state a change, never a target quantity: counting to a target is a stocktake, which belongs to `inventory`. A record type of the `stock` namespace with attachments, discussion and printing through `record`; writing it needs `stock.manage`. It has no approval fields and no approval rules: sending posts at once. Approval can be added later through `record`'s gate without changing the ledger.

### Sales delivery

- **Delivery** (`sales.delivery`, document `GH`): made from a confirmed order; one warehouse per delivery; several deliveries per order (partial delivery). `posted` means goods left the warehouse: `OnTransition` into `posted` calls `stock.Record`, into `cancelled` calls `stock.Void`.
- Only lines of items with stock (`tracking` other than `none`) appear on a delivery. An order's delivery state counts those lines only; an order of services alone needs no delivery.
- A delivery never takes an order line beyond its ordered quantity less what non-cancelled deliveries already took.
- An order with a non-cancelled delivery cannot be cancelled.
- Deliveries reuse the order's permissions and org unit: whoever may edit the order may deliver it. Sales serves warehouses, lots and serials to its delivery form under those permissions, as it serves items to its line editor, so Sales users need no stock role.
- The on-hand screen (per warehouse, item, lot or serial) is a `stock` route. Committed and available are Sales routes joining `stock`, since `stock` never reads `sales`.
- **Delivered**, **committed** and **available** are derived, never stored ([ADR-0027](./0027-sales-product.md#progress-is-derived-not-stored)): delivered sums posted deliveries; committed is confirmed orders less delivered; available is on hand less committed. Confirming an order whose available quantity is short warns but does not refuse.

### Not in this decision

Stock valuation and cost (average, FIFO), units of measure conversion, hard reservations, transfers between warehouses, stocktakes, receipts from suppliers, POS.

## Rejected alternatives

- **Stock only in a `modules/inventory` product.** Sales and POS would reach it through `deps.go`, which has no no-op ([ADR-0005](./0005-enabled-products.md)), so every Sales tenant would have to buy inventory to deliver.
- **Deducting on order confirmation.** On-hand would no longer match the shelf between order and hand-over; every ERP looked at keeps the two apart.
- **Storing delivered or committed quantities.** They would drift from the documents they summarise, as quotation progress would have.
- **Negative stock as a tenant or warehouse setting.** The product owner wants it per item: some goods are sold before the paperwork arrives, others never may be.
- **Lot tracking added later.** Moves and balances keyed without a lot would need rewriting once lots exist; the key is cheap to have from the start.

## Consequences

- A Sales tenant without `inventory` sees warehouses, on-hand and adjustments in the Sales area.
- Every document that moves goods (delivery today, POS sale and purchase receipt later) calls `stock.Record`/`stock.Void` from its `OnTransition`.
- On-hand per date is the sum of moves up to that date; balances are a cache of the present.
- Costing can be added on top of the ledger without changing its key.
