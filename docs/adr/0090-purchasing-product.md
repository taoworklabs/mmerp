# 0090. Purchasing is a product mirroring Sales; supplier invoices come from the order by quantity

Status: accepted · 2026-11-19

## Context

Order to cash is closed ([ADR-0052](./0052-receipts-settle-named-invoices.md)); procure to pay is not. Stock comes in only through stock adjustments, and the payable seeded on 331 never moves. [ADR-0039](./0039-shared-master-data.md) left the supplier role, and how two areas share one partner, to Purchasing; [ADR-0049](./0049-shared-stock-ledger-and-sales-delivery.md) left receipts from suppliers out. Design: `docs/superpowers/specs/2026-11-19-purchasing-design.md`.

## Decision

- **A new product `purchase`**: module `internal/modules/purchase`, schema `purchase`, area `web/src/purchase`, enabled through `PRODUCTS`. It depends on no other product and never imports Sales; it uses `partner`, `item`, `tax`, `stock` and `posting` directly. Code that Sales also has (the amount calculator) is copied, not shared across products.
- **Supplier** (`purchase.suppliers`): Purchasing's partner role, one per partner per org unit, with an owning org unit, code, payment terms and active flag, as `sales.customers` is Sales'. A company that is both customer and supplier is one partner with two roles; each area links to the partner, never to the other area's role.
- **Purchase order** (`purchase.order`, `DM`): entered directly, approval fields `amount` and `items`. Goods receipts and supplier invoices are only made from a posted order; there is no purchase without one.
- **Goods receipt** (`purchase.goods_receipt`, `NK`): one warehouse, several per order, stock lines only; posting brings goods in through `stock.Record`, creating lots and serials named on its lines; cancelling calls `stock.Void`, refused when the goods were already issued.
- **Supplier invoice** (`purchase.invoice`, `HM`): from the order, several per order, **by quantity, independent of goods receipts**. Unlike the Sales invoice, unit price, discount and VAT rate are editable per line, because the supplier's paper is the truth; invoiced progress counts quantities only. It records the supplier's invoice symbol, number and date; a second invoice of the same supplier row with the same symbol and number, pending or posted, is refused.
- **Supplier payment** (`purchase.payment`, `PC`): money to one supplier, method `cash` or `bank`, allocated to posted invoices of the same supplier and org unit, as customer receipts are.
- **Progress is derived**: received, invoiced and paid on orders; paid and remaining on invoices. Cancelling runs backwards (payment, invoice, then receipt and order).
- **Business lines**: `purchase_goods` (net of stock-tracked lines), `purchase_expense` (net of other lines), `purchase_input_vat`, `purchase_payable` (names the supplier's partner; a payment writes it negative), `purchase_cash_payment`, `purchase_bank_payment`. Sample rules: 156, 642, 1331 debit; 331, 111, 112 credit.
- Receipts, invoices and payments reuse the order's permissions. Roles `staff`, `manager`, `viewer`, `approval_admin`, scoped by the org tree.
- Item, warehouse and stock screens stay in the Sales area; a tenant needs Sales to manage them until a stock area exists.

## Rejected alternatives

- **Supplier invoice from goods receipts (three-way match).** Stricter, but services have no receipt and need a second path, and it adds a link layer.
- **One purchase voucher that receives and bills at once.** Departs from Sales, where delivery and invoice are separate documents.
- **Purchasing inside Sales.** A Sales-only tenant would carry purchasing, and ADR-0039 gives each product its own role table.
- **Invoice prices fixed to the order, as in Sales.** A bill priced differently from the order could not be booked without cancelling the order.
- **Checking duplicate supplier invoices across org units.** It would reveal another team's invoice to a user who cannot open it.

## Consequences

- Goods reach 156 when the invoice is booked, not when they are received. Until stock valuation the ledger carries neither quantity nor cost, so this is enough; valuation may move the debit to the goods receipt with a "received, not invoiced" clearing account.
- A rounding gap of a đồng between the supplier's paper and the computed VAT is corrected with a manual entry.
- Prepayments, returns, debit notes, due dates, aging and a cash and bank catalogue need decisions of their own.
