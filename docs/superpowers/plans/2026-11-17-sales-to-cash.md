# Sales to cash (M17) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Sales invoices made from confirmed orders and customer receipts allocated to named invoices, with derived invoice and payment progress and business lines that accounting maps to 131, 5111, 33311, 1111 and 1121.

**Architecture:** Two new document types in `internal/modules/sales`, built on the pattern of `sales.delivery` (`delivery.go`, `delivery_types.go`, `delivery_list.go`, `delivery_handler.go`, `queries/delivery.sql`): `record` first, then the module's tables, the parent locked with `record.LockToDepend`, side effects in `OnTransition`. Business lines go through the shared `posting` service, added to Sales' deps. Progress is derived in SQL, never stored.

**Tech Stack:** Go, Huma, sqlc and the data engine, Postgres 18, React with `@mmerp/ui`, Vitest, Playwright.

**Spec:** `docs/superpowers/specs/2026-11-17-sales-to-cash-design.md`

## Global Constraints

- Money is `int64`/`bigint` in đồng; quantities `numeric(15,3)` via `shopspring/decimal`; rounding only through `platform.Round`/`tax.VAT`. Never `float`.
- Services query only through `store.<Query>.Run(ctx, …)` and `data.List`; transactions only through `platform.InTx`.
- Lock order: own document row (via `record.Create/Edit/Transition`) → the order (`LockToDepend`) for invoices; → each allocated invoice (`LockToDepend`, sorted by id) for receipts.
- `record.Create/Edit/Delete/Transition` before writing `sales.*` tables, in the same transaction.
- Error codes are stable strings with parameters; user text only in i18n, both `vi` and `en`.
- Migrations via `make migration name=sales_invoices_receipts`; never edit a released one.
- Comments short, English or Vietnamese, never referencing docs, ADRs or milestones.
- Commits: Conventional Commits, scope `sales`, `accounting`, `web`, `docs`; stage explicit paths only.
- `make gen` must leave no diff; `make lint` and `make test` pass at the end of each task.

## Review Focus

1. **Two receipts for the same invoice posted concurrently**: together they must never exceed the invoice total. Test: `TestReceiptConcurrentAllocations` (Task 4).
2. **An invoice whose order's cancel awaits approval**: creating or posting it is refused, as for deliveries. Test: `TestInvoiceRefusedWhileOrderCancelHeld` (Task 2).
3. **A forgotten draft receipt**: it holds the amount back, the pre-fill shows what remains after it, and the invoice page lists it. Test: `TestReceiptDraftHoldsAmount` (Task 5).
4. **Cancel order of documents**: receipt → invoice → order must be cancellable in that order and refused in any other. Test: `TestCancelChain` (Task 4).
5. **Accounting disabled or rule missing**: posting an invoice or a receipt never fails; its lines wait under awaiting posting. Test: `TestInvoicePostsWithoutAccounting` (Task 2).

---

### Task 0: Decision record and docs

**Files:**
- Create: `docs/adr/0052-receipts-settle-named-invoices.md`
- Modify: `docs/roadmap.md` (M17 section after M16; diagram and "Next" line), `CONTEXT.md` (Sales section), `products/sales.md` (scope table, entities, record types, statuses and actions, progress, permissions), `products/accounting.md` (scope: business lines from Sales), `AGENTS.md` (project status line: M17 in progress)

- [ ] **Step 1: Write ADR-0052** (use `mattpocock-skills:domain-modeling`): Context (ADR-0051 left out open-item matching; sales staff must see payment per invoice). Decision: receipts are Sales documents; each receipt allocates to posted invoices of the same customer and org unit; paid is derived from posted receipts; drafts hold amounts back; five kinds and their rules; receivable lines carry the partner. Rejected: receipts in accounting (deps have no no-op), order posting revenue (no partial billing), a cross-org-unit receipt (visibility). Consequences: a later cash product takes over receipts; prepayments need a new decision.
- [ ] **Step 2: Add the M17 section to `docs/roadmap.md`**: Goal, then **Done when** copied from the spec's six exit criteria.
- [ ] **Step 3: Add to `CONTEXT.md` Sales**: Sales invoice (`sales.invoice`, vi "Hoá đơn bán hàng"), Customer receipt (`sales.receipt`, vi "Phiếu thu"), Receipt allocation (vi "Phân bổ thu"), Invoice state, Payment state. Note that "receipt" alone is avoided.
- [ ] **Step 4: Update `products/sales.md` and `products/accounting.md`** to the spec (move "Payment, invoices" and "Accounting, business lines" into the first-release column; e-invoices stay "Not yet").
- [ ] **Step 5: Commit** `docs(sales): ADR-0052 and M17 scope`.

### Task 1: Tables, kinds and posting in Sales deps

**Files:**
- Create: `migrations/<ts>_sales_invoices_receipts.sql`
- Create: `internal/modules/sales/posting.go`
- Modify: `internal/modules/sales/deps.go`, `internal/modules/sales/service.go`, `internal/app/app.go:102`, `internal/modules/sales/service_test.go` (fixture)
- Test: `internal/modules/sales/posting_test.go`

**Interfaces:**
- Produces: `Deps.Posting *posting.Service`; kinds `KindReceivable = posting.Kind("sales_receivable")` (ByPartner true), `KindRevenue = "sales_revenue"`, `KindOutputVAT = "sales_output_vat"`, `KindCashReceipt = "sales_cash_receipt"`, `KindBankReceipt = "sales_bank_receipt"`, declared once by `declareKinds()` from `NewService` (guard with `sync.Once`, since tests build many services). Fixture field `f.posting *posting.Service`.

- [ ] **Step 1: Migration.** Tables, shaped like `sales.deliveries`:
  - `sales.invoices (id bigint PK REFERENCES record.documents DEFERRABLE INITIALLY DEFERRED, order_id bigint NOT NULL REFERENCES sales.headers, subtotal, discount_total, vat_total, total bigint NOT NULL, note text)`, index on `order_id`.
  - `sales.invoice_lines (invoice_id … ON DELETE CASCADE, position int, order_position int, item_id, item_code, description, unit text, quantity numeric(15,3) CHECK > 0, unit_price bigint, discount_percent numeric(5,2), vat_rate text, amount, discount, vat bigint, PK (invoice_id, position))`.
  - `sales.receipts (id PK REFERENCES record.documents DEFERRABLE INITIALLY DEFERRED, customer_id bigint NOT NULL REFERENCES sales.customers, method text NOT NULL CHECK (method IN ('cash','bank')), amount bigint NOT NULL CHECK (amount > 0), note text)`, index on `customer_id`.
  - `sales.receipt_allocations (receipt_id … ON DELETE CASCADE, position int, invoice_id bigint NOT NULL REFERENCES sales.invoices, amount bigint NOT NULL CHECK (amount > 0), PK (receipt_id, position), UNIQUE (receipt_id, invoice_id))`, index on `invoice_id`.
- [ ] **Step 2: Failing test** `TestSalesKindsDeclared`: `posting.Declared(sales.KindReceivable)` returns `ByPartner == true`; the other four `false`; building two services does not panic.
- [ ] **Step 3: Implement** `posting.go` and add `Posting` to `Deps`, `app.go` (`Posting: pst`), the test fixture (`posting.NewService(posting.Hooks{})`).
- [ ] **Step 4: Run** `make gen && go test ./internal/modules/sales -run TestSalesKindsDeclared` → PASS; `go test ./internal/app` → PASS.
- [ ] **Step 5: Commit** `feat(sales): invoice and receipt tables, sales posting kinds`.

### Task 2: Sales invoice document

**Files:**
- Create: `internal/modules/sales/invoice.go`, `invoice_types.go`, `invoice_handler.go`, `queries/invoice.sql`
- Modify: `service.go` (register; order cancel refusal in `transition`), `handler.go`
- Test: `internal/modules/sales/invoice_test.go`

**Interfaces:**
- Consumes: `compute(lines []LineInput, list []int64, mode platform.Rounding)`, `Deps.Posting`, kinds from Task 1.
- Produces:
  - `const invoiceType = "sales.invoice"`, prefix `HD`, `Perms` those of orders, `Fields` none beyond the built-in `amount` (set `record.Header.Amount = &total` on create and edit).
  - `InvoiceLineInput{OrderPosition int; Quantity string}`; `InvoiceFields{Date string; Note *string; Lines []InvoiceLineInput (1..500)}`; `NewInvoice{RequestID string; OrderID int64; InvoiceFields}`; `InvoiceUpdate{Version int32; InvoiceFields}`.
  - `InvoiceLine{InvoiceLineInput; ItemID int64; ItemCode, Description, Unit string; UnitPrice int64; DiscountPercent, VatRate string; Amount, Discount, Vat int64}`.
  - `Invoice{ID, OrderID int64; OrderNumber, Number, Status string; Version int32; Date string; OrgUnitID, CustomerID int64; Customer CustomerCopy (the posted order's copy, read through `sales.headers`, already final); Note *string; Lines []InvoiceLine; Totals Totals; AllowedActions []string}` (payment fields added in Task 5).
  - `(*Service).CreateInvoice(ctx, NewInvoice) (int64, error)`, `UpdateInvoice(ctx, id, InvoiceUpdate) error`, `Invoice(ctx, id) (Invoice, error)`, `InvoiceOptions(ctx, orderID int64) (InvoiceOptions, error)` returning each order line with `Ordered` and `Remaining` (ordered less every non-cancelled invoice).
  - Errors: `ErrInvoiceOverOrdered` (`sales_invoice_over_ordered`, params `line_no`, `order_position`, `remaining`), `ErrInvoicePosition` (`sales_invoice_position`), `ErrInvoiceBeforeOrder` (`sales_invoice_before_order`), `ErrOrderHasInvoice` (`sales_order_has_invoice`), `ErrInvoiceHasReceipt` (`sales_invoice_has_receipt`, used in Task 4).
  - Routes: `GET /sales/orders/{id}/invoice-options`, `POST /sales/invoices`, `GET /sales/invoices/{id}`, `PUT /sales/invoices/{id}`.
  - Query `InvoiceableLines(order_id, invoice_id)` like `DeliverableLines` (all lines, no tracking filter, `taken` from non-cancelled invoices other than this one), and `LiveInvoiceOfOrder`.

- [ ] **Step 1: Failing tests** in `invoice_test.go`, with a fixture `invoices` built like `deliveries` (posted order of team A: 10 goods at 1 000 đ and 8 %, 2 hours of repair at 250 000 đ and 10 %):
  - `TestInvoiceRecordContract`: `recordtest.Run` for `sales.invoice`, harness as in `delivery_test.go:289`.
  - `TestInvoicePartial`: invoice 4 goods + 1 hour posts; totals equal `compute` of those lines (goods 4 000 + VAT 320, repair 250 000 + VAT 25 000 → total 279 320); a second invoice for 7 goods is refused with `sales_invoice_over_ordered`, `remaining` `"6.000"`; for 6 goods and 1 hour it posts.
  - `TestInvoicePricesFromOrder`: line price, discount and VAT rate on the saved invoice equal the order line's even after the item's catalogue price changes.
  - `TestInvoiceBeforeOrder`: date before the order date → `sales_invoice_before_order`.
  - `TestInvoiceOrderNotPosted`: on a draft order → `sales_order_not_posted`.
  - `TestInvoiceRefusedWhileOrderCancelHeld`: order cancel pending approval → create refused with the record held error.
  - `TestInvoicePostingLines`: after posting, `posting.lines` for the invoice are `sales_receivable` 279 320 with the customer's `partner_id`, `sales_revenue` 254 000, `sales_output_vat` 25 320, all at team A; after cancelling, the lines are voided. A 0 % / `none` VAT invoice writes no `sales_output_vat` line.
  - `TestInvoicePostsWithoutAccounting`: with the fixture's no-op posting hooks the invoice posts (no accounting dependency).
  - `TestOrderCancelWithInvoice`: cancelling the order while an invoice is not cancelled → `sales_order_has_invoice`; after cancelling the invoice, the order cancels.
  - `TestInvoiceTeamBoundary`: team B staff gets not found on the invoice and on `InvoiceOptions` of team A's order.
- [ ] **Step 2: Run** `go test ./internal/modules/sales -run 'Invoice'` → FAIL (undefined).
- [ ] **Step 3: Implement.** `checkInvoice(ctx, id, orderID, f) ([]InvoiceLine, Totals, error)` mirrors `checkDelivery`: `LockToDepend` on the order, posted check, date check, per-line position and remaining check, copy price, discount, VAT and texts from the order line, then `compute` with zero list prices (invoices have no `max_discount` field) and the legal entity's rounding (`s.d.Setting`, as `doc.go` reads it). `OnTransition`: into `posted` re-run `checkInvoice` and call `Posting.Record` with three lines (skip zero amounts); into `cancelled` refuse with `ErrInvoiceHasReceipt` while query `LiveReceiptOfInvoice` (against `sales.receipt_allocations`, which exists from Task 1) finds a non-cancelled receipt, then call `Posting.Void`; Task 4 tests the refusal. `OnDelete` deletes the row.
- [ ] **Step 4: Run** `go test ./internal/modules/sales` → PASS.
- [ ] **Step 5: Commit** `feat(sales): sales invoices from confirmed orders`.

### Task 3: Invoice progress on orders and the invoice list

**Files:**
- Create: `internal/modules/sales/invoice_list.go`
- Modify: `queries/document.sql` (`invoice_state` next to `delivery_state`), `doc.go` (`orderProgress`, `create_invoice` action), `doc_types.go`, `invoice_handler.go`, `queries/invoice.sql`
- Test: `invoice_test.go`, `doc_list_test.go`

**Interfaces:**
- Produces: `Doc.InvoiceState string` (`none,partial,full`), `Line.Invoiced *string` (orders, posted invoices), `Doc.Invoices []DocRef` (only those the actor may view), action `create_invoice` (posted order, `canMakeOrder` at its unit, some line with remaining > 0 counting non-cancelled invoices). `DocFilter.InvoiceState`, `DocListItem.InvoiceState`. `InvoiceFilter{Q, Status, OrderID, CustomerID, PaymentState (Task 5), Sort, Paging}`, `InvoiceListItem{ID, Number, Status, Date, OrderID, OrderNumber, CustomerID, CustomerName, Total, OrgUnitName}`, `GET /sales/invoices`.

- [ ] **Step 1: Failing tests:** `TestOrderInvoiceState` (none → partial after a 4-goods invoice posts → full after the rest posts; a draft invoice leaves the state unchanged but removes `create_invoice` once it takes everything); `TestInvoiceList` (filter by order, status, `q` on number and customer name; team B sees none); `TestDocListInvoiceStateFilter`.
- [ ] **Step 2: Run** → FAIL.
- [ ] **Step 3: Implement** following `delivery_state` and `delivery_list.go`.
- [ ] **Step 4: Run** `go test ./internal/modules/sales` → PASS.
- [ ] **Step 5: Commit** `feat(sales): invoice state on orders, invoice list`.

### Task 4: Customer receipt document

**Files:**
- Create: `internal/modules/sales/receipt.go`, `receipt_types.go`, `receipt_handler.go`, `queries/receipt.sql`
- Modify: `service.go`, `handler.go`
- Test: `internal/modules/sales/receipt_test.go`

**Interfaces:**
- Produces:
  - `const receiptType = "sales.receipt"`, prefix `PT`, order permissions, `Header.Amount = &amount`.
  - `AllocationInput{InvoiceID int64; Amount int64 (minimum 1)}`; `ReceiptFields{Date string; OrgUnitID, CustomerID int64; Method string (enum cash,bank); Note *string; Allocations []AllocationInput (1..200)}`; `NewReceipt{RequestID string; ReceiptFields}`; `ReceiptUpdate{Version int32; ReceiptFields}`.
  - `Allocation{AllocationInput; InvoiceNumber, InvoiceDate string; InvoiceTotal, Remaining int64}`; `Receipt{ID int64; Number, Status string; Version int32; Date string; OrgUnitID, CustomerID int64; CustomerName, Method string; Amount int64; Note *string; Allocations []Allocation; AllowedActions []string}`.
  - `CreateReceipt(ctx, NewReceipt) (int64, error)`, `UpdateReceipt(ctx, id, ReceiptUpdate) error`, `Receipt(ctx, id) (Receipt, error)`, `OpenInvoices(ctx, customerID, orgUnitID, receiptID int64) ([]Allocation, error)` (posted invoices of that customer and org unit with remaining > 0 after non-cancelled receipts other than `receiptID`; `GET /sales/customers/{id}/open-invoices?org_unit_id&receipt_id`).
  - Errors: `ErrReceiptOverInvoice` (`sales_receipt_over_invoice`, params `line_no`, `invoice_id`, `remaining`), `ErrReceiptInvoice` (`sales_receipt_invoice`: not posted, other customer), `ErrReceiptInvoiceOrgUnit` (`sales_receipt_invoice_org_unit`), `ErrReceiptBeforeInvoice` (`sales_receipt_before_invoice`), `ErrReceiptDuplicateInvoice` (`sales_receipt_duplicate_invoice`).
  - Routes: `POST /sales/receipts`, `GET /sales/receipts/{id}`, `PUT /sales/receipts/{id}`.

- [ ] **Step 1: Failing tests** (fixture: two posted invoices of team A for the same customer, totals T1 and T2):
  - `TestReceiptRecordContract`: `recordtest.Run` for `sales.receipt`.
  - `TestReceiptSplit`: one receipt allocating T1 and half of T2 posts; amount = T1 + T2/2.
  - `TestReceiptOverInvoice`: allocating T1+1 → `sales_receipt_over_invoice`, `remaining` T1.
  - `TestReceiptRules`: invoice of another customer → `sales_receipt_invoice`; invoice of team B's order for a customer of B → `sales_receipt_invoice_org_unit` (or not found when the actor cannot view it); draft invoice → `sales_receipt_invoice`; date before an invoice → `sales_receipt_before_invoice`; same invoice twice → `sales_receipt_duplicate_invoice`.
  - `TestReceiptPostingLines`: method `cash` writes `sales_cash_receipt` +amount and `sales_receivable` −amount with the customer's partner, both at the receipt's org unit; method `bank` writes `sales_bank_receipt`; cancelling voids them.
  - `TestReceiptConcurrentAllocations`: two goroutines each `CreateReceipt` for the full T1 at once (as `delivery_test.go` races deliveries); exactly one succeeds, the other gets `sales_receipt_over_invoice`.
  - `TestCancelChain`: invoice cancel with a posted receipt → `sales_invoice_has_receipt`; cancel receipt → invoice cancels → order cancels.
- [ ] **Step 2: Run** → FAIL.
- [ ] **Step 3: Implement.** `checkReceipt(ctx, id, f) ([]Allocation, int64, error)`: customer must be viewable and active; sort the allocation invoice ids and `LockToDepend` each; each invoice posted, same customer, same org unit as the receipt, date not after the receipt; remaining = total − non-cancelled allocations of other receipts. `OnTransition`: posted → re-check, `Posting.Record` with two lines; cancelled → `Posting.Void`. `beforeSubmit` refuses an inactive customer (reuse `ErrCustomerInactive`).
- [ ] **Step 4: Run** `go test ./internal/modules/sales` → PASS.
- [ ] **Step 5: Commit** `feat(sales): customer receipts allocated to invoices`.

### Task 5: Payment progress and the receipt list

**Files:**
- Create: `internal/modules/sales/receipt_list.go`
- Modify: `invoice.go`, `invoice_types.go`, `invoice_list.go`, `doc.go`, `doc_types.go`, `queries/invoice.sql`, `queries/document.sql`, `queries/receipt.sql`, `receipt_handler.go`
- Test: `receipt_test.go`, `invoice_test.go`

**Interfaces:**
- Produces: `Invoice.Paid, Invoice.Remaining int64` (posted receipts), `Invoice.PaymentState` (`unpaid,partial,paid`), `Invoice.Receipts []ReceiptRef{ID, Number, Status, Amount}` (all non-cancelled, drafts included, only those viewable), action `create_receipt` (posted invoice, `canMakeOrder` at its unit, remaining after non-cancelled receipts > 0), `InvoiceListItem.PaymentState`, `InvoiceFilter.PaymentState`. `Doc.Paid *int64` and `Doc.PaymentState` for orders (`paid` only when `InvoiceState == "full"` and every posted invoice is paid; `unpaid` when nothing is paid). `ReceiptFilter{Q, Status, CustomerID, Method, Sort, Paging}`, `ReceiptListItem{ID, Number, Status, Date, CustomerID, CustomerName, Method, Amount, OrgUnitName}`, `GET /sales/receipts`.

- [ ] **Step 1: Failing tests:** `TestInvoicePaymentState` (unpaid → partial → paid as receipts post; cancelling the receipt returns to unpaid); `TestReceiptDraftHoldsAmount` (a draft receipt for all of T1: invoice stays `unpaid`, `create_receipt` disappears, `OpenInvoices` leaves T1 out, `Invoice.Receipts` lists the draft); `TestOrderPaymentState` (partly invoiced and fully paid invoices → `partial`; all invoiced and paid → `paid`); `TestReceiptList` (filters, team boundary).
- [ ] **Step 2: Run** → FAIL.
- [ ] **Step 3: Implement** as derived SQL next to `invoice_state`.
- [ ] **Step 4: Run** `go test ./internal/modules/sales` → PASS.
- [ ] **Step 5: Commit** `feat(sales): payment state on invoices and orders, receipt list`.

### Task 6: Printing invoices and receipts

**Files:**
- Modify: `internal/modules/sales/print.go`, `internal/modules/sales/i18n/{vi,en}.json`
- Test: `internal/modules/sales/print_test.go` (or the existing print tests' file)

**Interfaces:**
- Produces: templates `sales.invoice` (blocks `terms`, `footer`; data: number, date, order number, seller, customer copy, lines, totals) and `sales.receipt` (block `footer`; data: number, date, seller, customer name, method, amount, amount in words if the existing printing helper offers it, allocations with invoice numbers).

- [ ] **Step 1: Failing tests:** `TestInvoicePrintFrozen` (a posted invoice reprints identically after the customer and the legal entity are edited); `TestReceiptPrintNeedsView` (team B cannot print team A's receipt).
- [ ] **Step 2: Run** → FAIL. **Step 3: Implement** following `registerPrints`. **Step 4: Run** → PASS.
- [ ] **Step 5: Commit** `feat(sales): print invoices and receipts`.

### Task 7: Invoice screens

Use the `mmerp-ui` and `new-screen` skills; finish with the `docs/ui.md` checklist.

**Files:**
- Create: `web/src/sales/pages/InvoicesPage.tsx`, `InvoicePage.tsx`, `NewInvoicePage.tsx`, `web/src/sales/components/InvoiceForm.tsx`, `InvoicePreview.tsx`, `invoiceLines.ts` + `invoiceLines.test.ts`
- Modify: `web/src/sales/routes.tsx`, `index.ts` (nav `sales.nav.invoices`, permission `sales.order.view`; `recordTypes['sales.invoice']`), `hooks/useSales.ts`, `components/DocProgress.tsx` (invoiced column, invoice state), `pages/DocPage.tsx` (action `create_invoice` → `/sales/invoices/new?order=<id>`), `i18n/{vi,en}`
- Test: `invoiceLines.test.ts` (remaining per line, pre-fill, the over-ordered error mapped to the right row)

- [ ] **Step 1: Failing Vitest** for `invoiceLines.ts` (`prefill(options) → lines with remaining`, `lineError(err, lines)`), as `deliveryLines.test.ts` does.
- [ ] **Step 2–4:** implement pages from `DeliveriesPage`/`DeliveryPage`/`NewDeliveryPage`; amounts shown are the server's; `pnpm -C web test` and `make lint` → PASS.
- [ ] **Step 5: Commit** `feat(web): sales invoice screens`.

### Task 8: Receipt screens and payment progress

**Files:**
- Create: `web/src/sales/pages/ReceiptsPage.tsx`, `ReceiptPage.tsx`, `NewReceiptPage.tsx`, `components/ReceiptForm.tsx`, `ReceiptPreview.tsx`, `receiptAllocations.ts` + test
- Modify: `routes.tsx`, `index.ts` (nav `sales.nav.receipts`; `recordTypes['sales.receipt']`), `InvoicePage.tsx` (paid, remaining, payment state badge, receipts list, `create_receipt` → `/sales/receipts/new?invoice=<id>`), `DocProgress.tsx` (paid, payment state), list pages (payment state badge and filter), `i18n/{vi,en}`

- [ ] **Step 1: Failing Vitest** for `receiptAllocations.ts`: `total(allocs)`, pre-fill from one invoice with its remaining, the over-invoice error mapped to the row.
- [ ] **Step 2–4:** implement; the form loads `open-invoices` for the chosen customer and org unit; `pnpm -C web test`, `make lint` → PASS.
- [ ] **Step 5: Commit** `feat(web): customer receipt screens, payment progress`.

### Task 9: Accounting labels, seed and end to end

**Files:**
- Modify: `web/src/accounting/i18n/{vi,en}/main.json` (labels of the five kinds), `web/e2e/accounting.seed.ts` (accounts 1111, 1121, 5111, 33311 if absent; rules for the five kinds), `web/e2e/sales.seed.ts` (one order invoiced in two parts, one receipt split across two invoices, one invoice unpaid), `docs/roadmap.md` (nothing yet: marking done is a separate PR after acceptance)
- Create: `web/e2e/sales-to-cash.spec.ts`

- [ ] **Step 1: Write the Playwright spec** covering the spec's exit criteria 1 to 4 and 6 through the UI and the API: partial invoices and the refused third; general ledger and partner balance after the invoice; split receipt, payment states and ledger lines; over-allocation and cancel refusals, then the cancel chain; team B sees no invoice or receipt. Criterion 5 (accounting or Sales disabled) is covered on `:8091` in `product-disabled.spec.ts`: add invoices and receipts read-only there.
- [ ] **Step 2: Run** `make e2e` → FAIL until the seed and labels exist; add them; run → PASS. Also `make test`, `make lint`, `make gen` with no diff.
- [ ] **Step 3: Commit** `test(sales): sales to cash end to end`.

After Task 9: `code-review` on the branch, then open one PR `feat(sales): sales to cash` (body per house-style, lists the migration).
