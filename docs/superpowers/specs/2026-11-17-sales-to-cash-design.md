# Sales to cash (M17)

Decision record: ADR-0052 (to be written as the first task: receipts settle named invoices). This document is what a builder plans from.

## Why

mmerp is closing its core ERP loops one at a time; M17 closes order to cash. Today a confirmed order can be delivered, but nothing says it was billed or paid, and Sales writes no business lines, so accounting never sees a sale. After M17 a salesperson sees on each invoice and order whether the customer has paid nothing, part or all of it, and accounting gets the receivable, revenue, output VAT and cash on the ledger, per customer, through the partner subledger of M16.

## Goals

- **Sales invoice** (`sales.invoice`, document `HD`, vi: "Hoá đơn bán hàng"), made from a confirmed order; several per order, by quantity, independent of delivery.
- **Customer receipt** (`sales.receipt`, document `PT`, vi: "Phiếu thu"): money from one customer, cash or bank transfer, allocated to named posted invoices of that customer.
- Derived progress: invoiced quantities and invoice state on orders; paid amount and payment state on invoices and orders.
- Business lines from Sales, mapped by accounting like payroll's.
- Printing of invoices and receipts, seed and Playwright.

## Non-goals

- E-invoices (they need the Internet; a later job-based integration).
- Prepayments and unallocated money: a receipt always settles named invoices.
- Cost of goods sold (needs stock valuation, M19).
- A cash and bank catalogue (several bank accounts, balances per bank): comes with M18, when money also goes out.
- Due dates, aging, dunning, credit notes and returns, editing a posted invoice.
- A separate cashier role.

## Anti-goals

- A "paid" figure that is stored and can drift from the receipts.
- An invoice total, or a remaining receivable, that the screen computes instead of the server.
- Any change to how payroll posts, or to the ledger code of M15 and M16.
- Sales reading the accounting schema, or accounting importing Sales.

## Approaches

- **A. Invoices and receipts are Sales documents (chosen).** Sales owns both, derives progress from them, and writes business lines through the shared `posting` service, already wired to accounting (`internal/app/app.go`). A Sales-only tenant still records payments. Cost: receipts sit in Sales; a later cash and bank product would take them over.
- **B. Receipts in accounting, Sales asks through a dependency.** Rejected: a dependency in `deps.go` has no no-op, so every Sales tenant would need accounting, and the salesperson's view of payments would depend on another product.
- **C. No invoice; the order posts revenue on confirmation.** Rejected: it cannot bill an order in parts.

## Decided

Points the product owner chose in the conversation:

- Payment is tracked per invoice and shown to sales staff, on invoices and orders (this reopens a minimal open-item matching that ADR-0051 left out; ADR-0052 records it).
- An invoice is made from a confirmed order, several per order, by quantity, whatever has been delivered.
- A receipt states only its method, cash or bank transfer; each method is one business line kind, mapped by the accountant to one account.
- A receipt settles invoices of its own org unit only; a customer paying several teams' invoices at once gets one receipt per org unit.

## Design

### Sales invoice

- Created from a posted order ("Tạo hoá đơn", action `create_invoice`, while some line has quantity not yet taken by non-cancelled invoices). The draft gets the order's org unit, customer, copied customer details and terms, dated today, with every line's remaining quantity pre-filled.
- Each line names an order line by position and states a quantity (more than 0, 3 decimals, at most ordered less non-cancelled invoices). Unit price, discount and VAT rate come from the order line and cannot be changed on the invoice. Lines of services and goods alike.
- Amounts are computed by the server with the existing calculator ([products/sales.md](../../../products/sales.md#amounts)) on the invoice's lines, with the legal entity's rounding read at save. The sum of an order's invoices may differ from the order total by rounding; invoice state counts quantities, not amounts.
- Date: on or after the order date (`sales_invoice_before_order`). The invoice date is the revenue date and the period-lock date.
- Locks like a delivery: own row, then the order through `record.LockToDepend`; refused while an action on the order awaits approval. Posting re-checks the quantities under that lock.
- Approval field `amount` (total including VAT). No rule: sending posts at once.
- Permissions and org unit: those of the order (`sales.order.view`, `sales.order.edit`), as for deliveries. Attachments, discussion and printing (template `sales.invoice`, internal print, not an e-invoice).
- Into `posted`: writes business lines (below). Into `cancelled`: voids them; refused while a non-cancelled receipt allocates to the invoice (`sales_invoice_has_receipt`).
- The order cannot be cancelled while a non-cancelled invoice exists (`sales_order_has_invoice`), alongside the existing delivery rule.

### Customer receipt

- Created directly ("Phiếu thu" list, new), or from a posted invoice with remaining receivable ("Thu tiền", action `create_receipt`, pre-filling that invoice and its remaining amount).
- Header: customer, org unit (chosen as on orders, default the customer's owning org unit), date, method `cash` or `bank`, note.
- Lines are **allocations**: a posted invoice of the same customer and the same org unit as the receipt (`sales_receipt_invoice_org_unit`), and an amount (whole đồng, more than 0). An invoice appears at most once. The receipt amount is the sum of its allocations; there is no unallocated amount.
- An allocation never exceeds the invoice's total less the allocations of other non-cancelled receipts (`sales_receipt_over_invoice`, with the remaining amount as parameter). Receipt date on or after each invoice's date (`sales_receipt_before_invoice`).
- Locks: own row, then each allocated invoice through `record.LockToDepend`, sorted by id, so two receipts for the same invoices take turns. Posting re-checks under those locks.
- Approval field `amount`. Permissions: `sales.order.edit` at the receipt's org unit, and view on every allocated invoice; view needs `sales.order.view`. *My call*: no new permission until a cashier role is asked for.
- A receipt allocating to invoices the user cannot view answers not found for those lines' invoices when creating; reading a receipt shows invoice numbers only.
- Into `posted`: business lines. Into `cancelled`: void; the invoices' remaining receivable returns.
- Printing (template `sales.receipt`), attachments and discussion.

### Derived progress (never stored)

- Order line **invoiced** = sum over **posted** invoices; order **invoice state** `none`, `partial`, `full` by quantity over all lines. Quantity checks use non-cancelled invoices, display uses posted ones, as deliveries do.
- Drafts hold quantities and amounts back, as draft deliveries do: every check, the `create_invoice` and `create_receipt` actions and their pre-fills use what non-cancelled documents leave. The invoice page lists its non-posted receipts too, so a forgotten draft is found and deleted (*my call*).
- Invoice **paid** = sum of allocations of posted receipts; **remaining** = total − paid; **payment state** `unpaid`, `partial`, `paid`.
- Order **paid** = sum of its invoices' paid amounts; order **payment state** `unpaid`, `partial`, `paid`, where `paid` means invoice state `full` and every posted invoice `paid`.
- The order page shows delivered, invoiced and paid side by side; lists show the states as badges and filter by them.

### Business lines

Sales declares five kinds through `posting.DeclareKind` (*builder's call*: constants in Sales or in `posting`):

Amounts are positive and the mapping rule gives the side; a negative amount goes on the opposite side (`accounting/reconcile.go`).

| Kind | Partner | Written by | Amount | Seed rule |
| --- | --- | --- | --- | --- |
| `sales_receivable` | yes | invoice; receipt | invoice: +total; receipt: −amount | 131, debit |
| `sales_revenue` | no | invoice | +net (total less VAT) | 511, credit |
| `sales_output_vat` | no | invoice, when VAT > 0 | +VAT | 3331, credit |
| `sales_cash_receipt` | no | receipt, method `cash` | +amount | 111, debit |
| `sales_bank_receipt` | no | receipt, method `bank` | +amount | 112, debit |

The partner is the customer's partner. Invoice lines carry the invoice's org unit. A receipt's lines carry its org unit, which is every allocated invoice's, so 131 clears per org unit. Sales needs `posting` in its deps.

### Accounting

No backend change beyond what the kinds bring: the rules screen lists the new kinds, `sales_receivable` maps only to a tracked account. The five kind labels go in the accounting area's i18n (`web/src/accounting/i18n/{vi,en}`), where the payroll kinds' labels are (*my call*). The seed adds 3331 and 511 to its sample chart (111, 112 and 131 are there already) and rules for the five kinds.

### Screens (Sales area)

- Invoices list and `DocumentPage` (lines table: order line, item, quantity, price, discount, VAT, amounts; side: order, payment state, receipts).
- Receipts list and `DocumentPage` (allocations table: invoice number, invoice date, total, remaining, amount).
- Order page: invoiced and paid columns, invoice and payment state, "Tạo hoá đơn".
- Invoice page: "Thu tiền". Menu: invoices and receipts under Sales. Approval rules page gains both types.

### Glossary

`CONTEXT.md`, Sales section: Sales invoice, Customer receipt, Receipt allocation (vi: "Phân bổ thu"), Invoice state, Payment state. "Receipt" alone is avoided: goods receipts come with M18.

## Exit criteria (roadmap M17)

With `make seed` and `PRODUCTS=hrm,sales,accounting`:

1. A confirmed order of goods and a service gets two invoices for parts of its quantities; a third that exceeds the remaining quantity is refused; the order shows invoice state `partial`, then `full`.
2. After posting an invoice, the general ledger shows Dr 131 (with the customer) / Cr 511 / Cr 3331 for it, and the customer's partner balance grows by its total.
3. A receipt split across two invoices of the customer posts; each invoice shows `partial` or `paid` correctly, the order shows its paid amount, and the ledger shows Dr 111 or 112 / Cr 131.
4. Allocating more than an invoice's remaining amount is refused; cancelling an invoice with a receipt is refused; cancelling the receipt returns the remaining amounts, then the invoice can be cancelled and its lines are reversed.
5. With accounting disabled, invoices and receipts work and their business lines wait for posting; with Sales disabled, everything reads and prints.
6. Another team's staff sees none of the invoices or receipts. `recordtest` passes for both types. HRM and payroll posting unchanged.

## Left to the builder

Slicing and order of work; table and column names; whether invoice lines copy order line texts or join them; list filters beyond the states; print layouts; where the five kind constants live.
