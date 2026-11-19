# 0052. Sales invoices and customer receipts are Sales documents; a receipt settles named invoices

Status: accepted · 2026-11-17

## Context

A confirmed order can be delivered ([ADR-0049](./0049-shared-stock-ledger-and-sales-delivery.md)), but nothing says it was billed or paid, and Sales writes no business lines. The partner subledger ([ADR-0051](./0051-business-lines-name-a-partner.md)) keeps what each customer owes, but only per account: sales staff cannot see whether a given invoice or order is paid. ADR-0051 left open-item matching out. The product owner wants payment shown per invoice and per order to sales staff. Design: `docs/superpowers/specs/2026-11-17-sales-to-cash-design.md`.

## Decision

- **Sales invoice** (`sales.invoice`, document `HB`): made from a confirmed order, several per order, by quantity, whatever has been delivered. Prices, discounts and VAT rates are the order line's. `posted` means billed; the invoice date is the revenue date.
- **Customer receipt** (`sales.receipt`, document `PT`): money from one customer, method `cash` or `bank`, **allocated** to posted invoices of the same customer and the same org unit as the receipt. Its amount is the sum of its allocations: there is no unallocated money. An allocation never exceeds what other non-cancelled receipts leave of the invoice.
- **Paid is derived**: an invoice's paid amount sums the allocations of posted receipts; its payment state is `unpaid`, `partial` or `paid`. An order is `paid` once fully invoiced and every invoice is paid. Drafts hold quantities and amounts back, as draft deliveries do.
- **Cancelling runs backwards**: an invoice with a non-cancelled receipt cannot be cancelled, nor an order with a non-cancelled invoice.
- **Business lines**: Sales declares `sales_receivable` (names the customer's partner), `sales_revenue`, `sales_output_vat`, `sales_cash_receipt`, `sales_bank_receipt`. Amounts are positive; the accountant's rules give accounts and sides (in the sample chart: 131 debit, 511 credit, 3331 credit, 111 and 112 debit). A receipt writes the receivable negative. Lines carry the document's org unit.
- Both documents reuse the order's permissions, like deliveries, and declare the `amount` approval field.
- The invoice prefix is `HB` (hoá đơn bán): `HD` already numbers employment contracts. Registering a document type whose prefix another type uses is refused at startup, since each type counts its own numbers.

## Rejected alternatives

- **Receipts in accounting.** Sales would need accounting through `deps.go`, which has no no-op ([ADR-0005](./0005-enabled-products.md)), so every Sales tenant would have to enable accounting.
- **The order posting revenue on confirmation.** It cannot bill an order in parts.
- **A receipt across org units.** Staff of one team would see their invoice paid by a receipt they cannot open. A customer paying several teams at once gets one receipt per org unit.
- **Storing paid amounts.** They would drift from the receipts, as progress would ([ADR-0027](./0027-sales-product.md)).

## Consequences

- Accounting needs no code: the kinds appear on its rules screen, and `sales_receivable` maps only to a tracked account.
- Prepayments, credit notes, due dates, aging and e-invoices need decisions of their own. A later cash and bank product would take receipts over.
- Cost of goods sold waits for stock valuation.
