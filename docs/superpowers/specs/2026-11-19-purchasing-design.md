# Purchasing (M18)

Decision record: ADR-0090 (to be written as the first task: Purchasing is a product mirroring Sales). This document is what a builder plans from.

## Why

mmerp closes its core ERP loops one at a time. M17 closed order to cash; M18 closes procure to pay. Today stock only comes in through stock adjustments, and nothing records what is owed to suppliers, so the payable seeded on 331 never moves. After M18 a buyer orders from a supplier, the warehouse receives the goods in parts, the supplier's invoice is booked and paid, and everyone sees on each order whether it is received, invoiced and paid. Accounting gets goods, expenses, input VAT, the payable per supplier and cash out on the ledger, through the partner subledger of M16.

## Goals

- **Supplier** (`purchase.supplier`, catalogue): Purchasing's partner role over a shared partner, as `sales.customers` is Sales'.
- **Purchase order** (`purchase.order`, document `DM`, vi: "Đơn mua hàng"), entered directly, with approval.
- **Goods receipt** (`purchase.goods_receipt`, document `NK`, vi: "Phiếu nhập kho"), made from a confirmed order; one warehouse, several per order; it brings goods into stock and may create lots and serials.
- **Supplier invoice** (`purchase.invoice`, document `HM`, vi: "Hoá đơn mua hàng"), made from a confirmed order, several per order, by quantity, independent of receipts.
- **Supplier payment** (`purchase.payment`, document `PC`, vi: "Phiếu chi"): money to one supplier, cash or bank transfer, allocated to named posted invoices of that supplier.
- Derived progress on orders (received, invoiced, paid) and on invoices (paid, remaining).
- Business lines from Purchasing, mapped by accounting like Sales'.
- A new product `purchase` with its own module, schema, frontend area, roles and approval rules page; printing, seed and Playwright.

## Non-goals

- A cash and bank catalogue (several bank accounts, balances per bank); M17 deferred it to M18, and M18 defers it again: a payment only states its method.
- Prepayments to suppliers and unallocated money: a payment always settles named invoices.
- Buying without a purchase order (receipts and invoices always come from an order).
- Stock valuation and cost of goods, landed costs (M19).
- Due dates, aging, returns to suppliers, debit notes, editing a posted invoice.
- Purchase requisitions, requests for quotation, supplier price lists, units of measure conversion, foreign currencies.

## Anti-goals

- A received, invoiced or paid figure that is stored and can drift from the documents.
- Totals or remaining amounts computed by the screen instead of the server.
- Purchasing importing Sales, or Sales importing Purchasing; either reading the accounting schema, or accounting importing either.
- Any change to how Sales, payroll or the ledger post.

## Approaches

The product owner chose where the supplier invoice comes from:

- **A. Invoice from the purchase order, by quantity, independent of goods receipts (chosen).** Mirrors M17 exactly (order → invoices by quantity, payments allocated to invoices); received and invoiced state show side by side on the order. Cost: an invoice may bill goods not yet received; the buyer sees it on the order.
- **B. Invoice from goods receipts (three-way match).** Rejected: stricter, but adds a link layer and services, which have no receipt, need a separate path.
- **C. One purchase voucher that both receives and bills** (the MISA style). Rejected: departs from the Sales pattern where delivery and invoice are separate documents.

Where it lives (one sensible way):

- **A new product `purchase` (chosen)**, symmetric to Sales: `internal/modules/purchase`, schema `purchase`, frontend area `web/src/purchase`, enabled through `PRODUCTS`. It depends on no other product.
- Inside Sales: rejected; a Sales-only tenant would carry purchasing, and ADR-0039 already gives each product its own partner role table.
- Inside an `inventory` product: none exists yet; receipts write through the shared `stock` ledger anyway (ADR-0049).

## Decided

Points the product owner chose in the conversation:

- Purchasing is a new product, mirroring Sales.
- The supplier invoice comes from the purchase order by quantity, independent of goods receipts (approach A).
- Goods receipts and invoices always come from a purchase order; there is no purchase without one.
- A payment states only its method, cash or bank transfer; no cash and bank catalogue yet.
- A supplier invoice books goods (stock-tracked lines) and expenses (lines of items without stock) to two separate kinds.
- A supplier invoice line may differ from its order line in unit price, discount and VAT rate (the supplier's paper is the truth); invoiced progress counts quantities only. A rounding gap of a đồng against the paper is corrected by a manual entry, not on the invoice.
- Items, warehouses, on-hand, stock card and stock adjustments stay in the Sales area for M18; a tenant needs Sales to manage them. M19 moves them to a stock area of their own.

## Design

### Supplier

- `purchase.suppliers`: `partner_id`, `org_unit_id` (owning org unit, decides who sees it), code (issued by a code series when empty, ADR-0032), payment terms text, active flag; one per partner per org unit at most. Identity lives in `partner.partners`, so one company can be both a customer and a supplier.
- Same rules as customers: identity fields editable only with `partner.edit`; inactive suppliers cannot go on a draft (`supplier_inactive`); a document whose supplier was deactivated after saving cannot be sent or posted. Payments have no active-supplier check (what is owed is still paid).
- Attachments and discussion through `record`.

### Purchase order

- Header: supplier, org unit (default the supplier's owning org unit), date, expected date (optional), terms, note; copied supplier details (code, name, tax code, address, phone, email, contact person) on each draft save and again on sending, as Sales documents do.
- Lines: item (goods or service, active), copied code and unit, description, quantity (> 0, 3 decimals), unit price (whole đồng, ≥ 0), discount %, VAT rate; amounts computed by the server with the Sales calculator rules ([products/sales.md](../../../products/sales.md#amounts)) and the legal entity's rounding read at save. *Builder's call*: share the calculator (move it below both products, e.g. `shared/`) or duplicate it; it must not be imported across products.
- Approval fields: `amount` (total including VAT) and `items`. With no rule, sending posts (confirms) at once. Rules configured at `/purchase/approval-rules`.
- `posted` means confirmed. Cancel is refused while a non-cancelled goods receipt (`purchase_order_has_receipt`) or invoice (`purchase_order_has_invoice`) exists.
- Actions on a posted order: `create_goods_receipt` (while a stock line has quantity no non-cancelled receipt takes), `create_invoice` (while some line has quantity no non-cancelled invoice takes), print.
- Print template `purchase.order`; attachments and discussion.

### Goods receipt

- Mirror of the Sales delivery with the sign reversed: made from a posted order for one warehouse of the order's legal entity; several per order; only lines of items with stock (`tracking` other than `none`).
- Each line names an order line by position and a quantity; never more than ordered less other non-cancelled receipts (`purchase_receipt_over_order`).
- Lot items state lot code and optional expiry per line (one order line may split into several lots); serial items state one serial per unit. Lots and serials are created through `stock.Record` (`LotRef` creates a missing lot), as stock adjustments do. A serial already on hand is refused by the ledger.
- Date on or after the order date (`purchase_receipt_before_order`).
- Locks like a delivery: own row, then the order through `record.LockToDepend`; refused while an action on the order awaits approval; re-checks quantities under the lock.
- Into `posted`: `stock.Record` with positive quantities. Into `cancelled`: `stock.Void`, which the ledger refuses (`stock.insufficient`) when the goods were already issued and the item does not allow negative stock.
- No approval fields. Permissions and org unit of the order. Purchasing serves warehouses to its receipt form under the order's permissions, so buyers need no stock role.
- Print template `purchase.goods_receipt` (*my call*: the receipt is the paper the storekeeper signs; Sales deliveries have no print yet).

### Supplier invoice

- Created from a posted order (`create_invoice`, pre-filled with every line's remaining quantity and the order line's price, discount and VAT rate); each line takes a quantity of an order line, never more than ordered less other non-cancelled invoices (`purchase_invoice_over_order`).
- Unlike the Sales invoice, unit price, discount and VAT rate are editable per line, so the invoice matches the supplier's paper; amounts are computed by the server from them with the legal entity's rounding. Invoiced progress counts quantities, never amounts; the order total and the sum of its invoices may differ.
- Header adds the **supplier's invoice symbol, number and date** (`supplier_invoice_symbol`, `supplier_invoice_no`, `supplier_invoice_date`, all required on send; numbers restart per symbol under Decree 123). *My call*: a second non-cancelled invoice of the same supplier row (supplier and org unit) with the same symbol and number is refused (`purchase_invoice_duplicate`), checked on send and post under a `FOR UPDATE` lock on the supplier row (a module-owned row, last in the lock order). Not checked across org units: that would reveal another team's invoice.
- Date on or after the order date (`purchase_invoice_before_order`); the invoice date is the booking date and the period-lock date.
- Locks: own row, then the order through `record.LockToDepend`; re-checks quantities under that lock.
- Approval field `amount`. Permissions and org unit of the order.
- Into `posted`: business lines. Into `cancelled`: refused while a non-cancelled payment allocates to it (`purchase_invoice_has_payment`); voids them.
- Print template `purchase.invoice` (internal voucher).

### Supplier payment

- Mirror of the customer receipt: created directly or from a posted invoice with a remaining amount (`create_payment`).
- Header: supplier, org unit, date, method `cash` or `bank`, note.
- Allocations: posted invoices of the same supplier and org unit (`purchase_payment_invoice_org_unit`), whole đồng > 0, each invoice at most once; never more than the invoice's total less other non-cancelled payments (`purchase_payment_over_invoice`, remaining as parameter); payment date on or after each invoice's date (`purchase_payment_before_invoice`). The payment amount is the sum of its allocations.
- Locks: own row, then each allocated invoice through `record.LockToDepend`, sorted by id.
- Approval field `amount` (payments are where approval matters most for SMBs).
- Permissions: `purchase.order.edit` at the payment's org unit and view on every allocated invoice (*my call*, as in M17: no separate cashier permission).
- Into `posted`: business lines. Into `cancelled`: voids them; remaining amounts return.
- Print template `purchase.payment`; attachments and discussion.

### Derived progress (never stored)

- Order line **received** = sum of posted receipts; order **receipt state** `none`, `partial`, `full`, or `not_applicable` without stock lines.
- Order line **invoiced** = sum of posted invoices; order **invoice state** `none`, `partial`, `full`.
- Invoice **paid** = sum of allocations of posted payments; **remaining** = total − paid; **payment state** `unpaid`, `partial`, `paid`. Order **paid** and **payment state** as in M17.
- Checks and pre-fills use what non-cancelled documents leave; display uses posted ones.

### Business lines

Purchasing declares six kinds through `posting.DeclareKind`. Amounts are positive; the rule gives the side.

| Kind | Partner | Written by | Amount | Seed rule |
| --- | --- | --- | --- | --- |
| `purchase_goods` | no | invoice | +net of lines whose item has stock tracking (net as billed) | 156, debit |
| `purchase_expense` | no | invoice | +net of lines whose item has no stock tracking | 642, debit |
| `purchase_input_vat` | no | invoice, when VAT > 0 | +VAT | 1331, debit |
| `purchase_payable` | yes | invoice; payment | invoice: +total; payment: −amount | 331, credit |
| `purchase_cash_payment` | no | payment, method `cash` | +amount | 111, credit |
| `purchase_bank_payment` | no | payment, method `bank` | +amount | 112, credit |

The partner is the supplier's partner. Lines carry the document's org unit. The goods/expense split reads the item's tracking as copied on the invoice line at save (*builder's call* where it is copied), so a later catalogue change never moves a posted invoice.

**Known ceiling**: goods go to 156 when the invoice is booked, not when they are received. Until M19 the ledger carries no quantity or cost, so this is enough; M19 valuation may move the debit to the goods receipt with a "received, not invoiced" clearing account. ADR-0090 records this.

### Accounting

No backend change beyond the kinds. The six kind labels go in the accounting area's i18n, next to the Sales kinds. The seed adds 156, 1331 and 642 to its sample chart (111, 112, 331 exist; 331 is already tracked by partner) and rules for the six kinds.

### Product, permissions, roles

- Product `purchase` in `PRODUCTS`; disabled means readable and printable, no writes. `make e2e` serves `PRODUCTS=hrm,sales,purchase` on `:8090` (*builder's call* whether accounting joins).
- Permissions: `purchase.supplier.view/edit`, `purchase.order.view/edit` (receipts, invoices and payments reuse the order's), `purchase.approval.manage`.
- Roles: `staff`, `manager` (same permissions; approval rules route to `manager`), `viewer` at an org unit or tenant-wide; `approval_admin` tenant-wide only. The tenant administrator holds them (ADR-0028).
- Scope by the org tree as in Sales: another team's buyer sees none of the suppliers or documents.
- Writing an order line reads items through a Purchasing endpoint (as `GET /sales/line-items`), so buyers need no item role.

### Screens (Purchasing area)

- Home tile and header title "Mua hàng"; menu: suppliers, purchase orders, goods receipts, supplier invoices, payments; configuration: approval rules. No item or stock screens (they stay in Sales until M19).
- Order `DocumentPage`: lines with received and invoiced columns, paid amount, the three states, actions "Nhập kho", "Lập hoá đơn".
- Goods receipt, invoice and payment pages mirroring Sales delivery, invoice and receipt pages; invoice page "Chi tiền".
- Lists with state badges and filters. Approval inbox previews orders, invoices and payments.

### Glossary

`CONTEXT.md`, new Purchasing section: Supplier, Purchase order, Goods receipt, Supplier invoice, Supplier payment, Payment allocation (vi: "Phân bổ chi"), Receipt state. Avoid "receipt" alone (Sales' customer receipt).

## Exit criteria (roadmap M18)

With `make seed` data and `PRODUCTS=hrm,sales,purchase,accounting` (Sales provides the item and stock screens until M19):

1. A supplier is created from an existing partner that is already a customer; a purchase order of a lot item, a serial item and a service is sent, approved and confirmed.
2. Two goods receipts take parts of the stock lines into a warehouse, creating lots and serials; on hand grows; a third that exceeds the remaining quantity is refused; the order shows receipt state `partial`, then `full`.
3. A supplier invoice for part of the quantities, with one unit price changed from the order, posts; the ledger shows Dr 156, Dr 642, Dr 1331 / Cr 331 with the supplier, and the supplier's partner balance grows by its total. A second invoice of the supplier with the same symbol and number is refused.
4. A payment split across two invoices posts; each invoice shows `partial` or `paid`, the order shows its paid amount, the ledger shows Dr 331 / Cr 111 or 112.
5. Over-allocating is refused; cancelling an invoice with a payment is refused; cancelling a goods receipt whose goods were delivered is refused unless the item allows negative stock.
6. With accounting disabled, purchasing works and its lines wait for posting; with Purchasing disabled, everything reads and prints. Another team's staff sees nothing. `recordtest` passes for the four document types. Sales and payroll posting unchanged.

## Left to the builder

Slicing and order of work; table and column names; sharing or duplicating the amount calculator (never a cross-product import); print layouts; list filters beyond states; seed contents; where the kind constants live.
