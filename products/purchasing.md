# Purchasing product

Buying goods and services from suppliers: suppliers, purchase orders, goods receipts into stock, supplier invoices and payments. Purchasing is the fourth product. Hard-to-reverse decisions are in [ADR-0090](../docs/adr/0090-purchasing-product.md); terminology is in the Purchasing section of [CONTEXT.md](../CONTEXT.md#purchasing). It mirrors Sales ([products/sales.md](./sales.md)); this page states what differs.

## Scope

| In the first release | Not yet |
| --- | --- |
| Suppliers (Purchasing's partner role over a shared partner), with an owning org unit | Supplier price lists, requests for quotation, purchase requisitions |
| Purchase orders with approval, PDF print | Buying without a purchase order |
| Goods receipts from a confirmed order into one warehouse, creating lots and serials | Returns to suppliers, landed costs, stock valuation |
| Supplier invoices from a confirmed order, in parts, priced as the supplier billed; supplier payments allocated to invoices; business lines for accounting | Prepayments, debit notes, due dates and aging, a cash and bank catalogue, e-invoice intake |
| | Item, warehouse and stock screens of its own (they stay in the Sales area) |

## Module

A single module `internal/modules/purchase`, schema `purchase`, product `purchase`. Depends on no other product and never imports Sales. Uses `iam`, `record`, `audit`, `setting`, `printing`, `numbering` (supplier codes) and the shared `partner`, `item`, `tax`, `stock` (goods receipts) and `posting` (business lines) modules directly. Emits no hooks.

## Entities

| Table | Meaning |
| --- | --- |
| `purchase.suppliers` | Suppliers, Purchasing's partner role: `partner_id`, `org_unit_id`, code, payment terms, active flag; one per partner per org unit at most |
| `purchase.orders`, `purchase.order_lines` | Purchase orders keyed by the `record.documents` id: supplier details as copied, terms, totals; lines with item, copied code and unit, whether the item has stock (`stocked`), quantity, price, discount, VAT rate, computed amounts |
| `purchase.goods_receipts`, `purchase.goods_receipt_lines` | Goods receipts: the order, the warehouse; lines naming an order line by position, with lot (code, expiry) or serial and quantity |
| `purchase.invoices`, `purchase.invoice_lines` | Supplier invoices: the order, the supplier's invoice symbol, number and date, totals; lines naming an order line by position with their own price, discount and VAT rate |
| `purchase.payments`, `purchase.payment_allocations` | Supplier payments: the supplier, method (`cash`, `bank`), amount; one allocation per invoice settled |

## Record types

| Type code | Kind | `posted` means | Date for period lock | `OnTransition` | Approval fields |
| --- | --- | --- | --- | --- | --- |
| `purchase.supplier` | catalogue | — | — | — | — |
| `purchase.order` | document `DM` | Confirmed | Order date | Into `posted`: supplier still active. Into `cancelled`: refused while a non-cancelled goods receipt or invoice exists | `amount`, `items` |
| `purchase.goods_receipt` | document `NK` | Goods came in | Receipt date | Into `posted`: re-checks quantities, brings the goods in through `stock`. Into `cancelled`: takes them out again, refused when already issued | — |
| `purchase.invoice` | document `HM` | Booked | Invoice date | Into `posted`: re-checks quantities and the duplicate supplier invoice, writes business lines. Into `cancelled`: refused while a non-cancelled payment settles it; voids them | `amount` |
| `purchase.payment` | document `PC` | Money paid | Payment date | Into `posted`: re-checks allocations, writes business lines. Into `cancelled`: voids them | `amount` |

Approval rules are configured inside the Purchasing area (`/purchase/approval-rules`), needing `purchase.approval.manage` or `core.approval.manage`.

## Progress

Derived, never stored, as in Sales: an order's **received** quantity per line and **receipt state** (`none`, `partial`, `full`, `not_applicable`) from posted goods receipts; **invoiced** and **invoice state** from posted invoices, by quantity; an invoice's **paid**, **remaining** and **payment state** from posted payments; an order's **paid** and **payment state**. Checks and pre-fills count every non-cancelled document, so drafts hold quantities and amounts back.

## Amounts

The rules of [Sales](./sales.md#amounts): VND in đồng, quantities to 3 decimals, discount per line, VAT `none`, `0`, `5`, `8`, `10` %, rounding per the legal entity's setting, computed and stored by the server on each draft save. A supplier invoice line takes its quantity from the order line's remaining and its price, discount and VAT rate from the supplier's paper (pre-filled from the order line). A gap of a đồng against the paper is corrected by a manual journal entry.

## Supplier invoice duplicates

A supplier invoice must state the supplier's invoice symbol, number and date before it is sent. A second invoice of the same supplier (same partner and org unit) with the same symbol and number that is pending approval or posted is refused (`purchase_invoice_duplicate`). Numbers restart per symbol, so the same number under another symbol is accepted. Invoices of other org units are not checked.

## Business lines

| Kind | Partner | Written by | Amount | Sample rule |
| --- | --- | --- | --- | --- |
| `purchase_goods` | no | invoice | net of lines whose item has stock | 156, debit |
| `purchase_expense` | no | invoice | net of other lines | 642, debit |
| `purchase_input_vat` | no | invoice, when VAT > 0 | VAT | 1331, debit |
| `purchase_payable` | yes | invoice; payment | invoice +total; payment −amount | 331, credit |
| `purchase_cash_payment` | no | payment, `cash` | amount | 111, credit |
| `purchase_bank_payment` | no | payment, `bank` | amount | 112, credit |

## Permissions

| Role | Permissions | Granted |
| --- | --- | --- |
| `staff`, `manager` | `purchase.supplier.view`, `purchase.supplier.edit`, `purchase.order.view`, `purchase.order.edit` | At an org unit or tenant-wide |
| `viewer` | The `view` permissions above | At an org unit or tenant-wide |
| `approval_admin` | `purchase.approval.manage` | Tenant-wide only |

Goods receipts, invoices and payments reuse the order's permissions and org unit. Identity fields of a supplier need `partner.edit`. Picking items on an order needs no item role (`GET /purchase/line-items`); receiving goods needs no stock role. A record the user cannot view answers not found everywhere. With Purchasing disabled, everything stays readable and printable.

## Personal data

Supplier identity (a supplier may be a person) and its copies on documents: ordinary personal data, not encrypted, read per the org-unit scope, changes audited. No sensitive fields.

## Frontend

Area `web/src/purchase`, menu: suppliers, purchase orders, goods receipts, supplier invoices, payments; configuration: approval rules. Documents use `DocumentPage`; the approval inbox previews orders, invoices and payments.

## Proving flow

Supplier from a partner already a customer → purchase order of a lot item, a serial item and a service, approved → two goods receipts creating lots and serials → supplier invoice with a changed price → payment split across two invoices → cancel chain. Another team's staff sees none of it. Covered by `purchase` service tests and Playwright.
