# 0100. Stock gets its own area over `shared/stock`; transfers and stocktakes are stock documents; valuation waits

Status: accepted · 2026-11-19

## Context

[ADR-0049](./0049-shared-stock-ledger-and-sales-delivery.md) put the stock ledger in `shared/stock` and its screens in the Sales area, and left transfers, stocktakes and valuation to a later `inventory` product. Since M18 a tenant buying Purchasing alone receives goods but cannot see its warehouses, on hand or stock card without also enabling Sales ([ADR-0090](./0090-purchasing-product.md)). Valuation (cost per move, cost of goods sold, 156 following the warehouse) touches every document that moves goods and the ledger's posting; transfers and stocktakes only add documents to the existing ledger. Design: `docs/superpowers/specs/2026-11-19-stock-design.md`.

## Decision

- **No `inventory` product.** Transfers and stocktakes are record types of the `stock` namespace in `shared/stock`, next to the stock adjustment, under `stock.manage`. `ProductGate` never refuses them, as for every stock write: whichever product brings goods in may count and move them.
- **A frontend area `web/src/stock`** (product key `stock`, home tile "Kho"), entered by any `stock.*` or `item.*` permission. Items, warehouses, on hand, stock card and stock adjustments move there from the Sales area; Sales and Purchasing keep no stock or item screens. The area calls only `shared/api/core`, where stock and item routes already are.
- **Transfer** (`stock.transfer`, `CK`): one source and one destination warehouse of the same legal entity, different from each other; lines of item, lot or serial and a positive quantity. Posting writes, through one `stock.Record`, an outgoing move from the source and an incoming one into the destination per line; cancelling calls `stock.Void`. No goods in transit: both sides move on the transfer's date.
- **Stocktake** (`stock.stocktake`, `KK`): one warehouse, lines of item, lot or serial and a counted quantity (≥ 0). Only listed lines are counted. Posting compares each counted quantity with on hand **under the balance lock** and writes the difference as moves, through a new ledger call `stock.RecordCounts`; the book quantity and difference are kept on the line once posted. Cancelling calls `stock.Void`.
- Neither has approval: sending posts at once, as for adjustments.
- **Valuation is a milestone of its own** (M20): costing method, cost on moves, cost of goods sold and moving the 156 debit from the supplier invoice to the goods receipt need their own decision.

## Rejected alternatives

- **An `inventory` product holding transfers and stocktakes.** A product could be disabled, then a Purchasing-only tenant could not count what it received; and the ledger already lives in `shared/`.
- **Stock screens duplicated in the Sales and Purchasing areas.** Areas never share components, so each screen would exist twice.
- **Goods in transit (two-step transfer).** Needs an in-transit location and a receiving step; SMB warehouses of one company are usually close enough to move on one date.
- **A stocktake computing its difference when saved.** On hand may change between saving and posting; only the balance lock gives a difference that holds.
- **Valuation inside M19.** It changes Sales and Purchasing posting; done together it would hold up the screens that Purchasing-only tenants need now.

## Consequences

- A tenant with Purchasing alone manages items and stock in the Stock area.
- `stock.Record` already accepts moves of several warehouses of one legal entity; a transfer needs no ledger change. `stock.RecordCounts` locks in the same order as `Record`.
- Cancelling a transfer or stocktake whose incoming goods were already issued is refused unless the item allows negative stock, as for adjustments.
- Moves stay quantity-only until M20.
