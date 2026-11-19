# Purchasing (M18) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A new product `purchase`: suppliers, purchase orders with approval, goods receipts into stock, supplier invoices by quantity and supplier payments allocated to invoices, with derived receipt, invoice and payment progress and business lines that accounting maps to 156, 642, 1331, 331, 111 and 112.

**Architecture:** A new module `internal/modules/purchase` (schema `purchase`), the mirror of Sales: the order follows `sales.order` (`doc.go`, only one kind), the goods receipt follows `sales.delivery` with positive stock moves, the invoice follows `sales.invoice` with editable prices, the payment follows `sales.receipt`. `record` first, then module tables, the parent locked with `record.LockToDepend`, side effects in `OnTransition`, progress derived in SQL. Purchasing never imports Sales; it uses `partner`, `item`, `tax`, `stock`, `posting` directly. A new frontend area `web/src/purchase` mirrors `web/src/sales`.

**Tech Stack:** Go, Huma, sqlc and the data engine, Postgres 18, React with `@mmerp/ui`, Vitest, Playwright.

**Spec:** `docs/superpowers/specs/2026-11-19-purchasing-design.md`

## Global Constraints

- Money is `int64`/`bigint` in đồng; quantities `numeric(15,3)` via `shopspring/decimal`; rounding only through `platform.Round`/`tax.VAT`. Never `float`.
- Services query only through `store.<Query>.Run(ctx, …)` and `data.List`; transactions only through `platform.InTx`.
- Lock order: own document row → the order (`LockToDepend`) for receipts and invoices; → each allocated invoice (`LockToDepend`, sorted by id) for payments; `stock.balances` inside `stock.Record`; the supplier row `FOR UPDATE` last (invoice duplicate check).
- `record.Create/Edit/Delete/Transition` before writing `purchase.*` tables, in the same transaction.
- `internal/modules/purchase` never imports `internal/modules/sales` (depguard); Sales code is copied, not shared. Sales, payroll and ledger behaviour stay unchanged.
- Error codes are stable strings with parameters; user text only in i18n, both `vi` and `en`.
- Migrations via `make migration name=purchase_<description>`; never edit a released one.
- Comments short, English or Vietnamese, never referencing docs, ADRs or milestones.
- Commits: Conventional Commits, scope `purchase`, `accounting`, `web`, `docs`; stage explicit paths only.
- `make gen` leaves no diff and the touched Go/Vitest tests pass at the end of each task. `make lint` and `make e2e` run in CI; run them locally only when the user asks.

## Review Focus

1. **Two payments for the same invoice created concurrently**: together never above the invoice total. Test: `TestPaymentConcurrentAllocations` (Task 6).
2. **Two invoices with the same supplier symbol and number sent concurrently**: exactly one is accepted. Test: `TestSupplierInvoiceDuplicateConcurrent` (Task 5).
3. **Cancelling a goods receipt whose goods were already issued**: refused with `stock.insufficient` unless the item allows negative stock. Test: `TestGoodsReceiptCancelAfterIssue` (Task 4).
4. **A serial item received with a quantity other than one per serial, or a serial already on hand**: refused. Test: `TestGoodsReceiptSerials` (Task 4).
5. **Accounting disabled or rule missing**: posting an invoice or a payment never fails; lines wait. Test: `TestSupplierInvoicePostsWithoutAccounting` (Task 5).

---

### Task 0: Decision record and docs

**Files:**
- Create: `docs/adr/0090-purchasing-product.md`, `products/purchasing.md`
- Modify: `docs/roadmap.md` (M18 section after M17; "Next" line), `CONTEXT.md` (new Purchasing section), `products/accounting.md` (business lines from Purchasing), `AGENTS.md` (status line: M18 in progress, Purchasing the fourth product; read-before table row for `products/purchasing.md`)

- [ ] **Step 1: Write ADR-0090** (use `mattpocock-skills:domain-modeling`): Context (procure to pay is open; ADR-0039 left the supplier role and "how two areas share a partner" to Purchasing). Decision: product `purchase` mirroring Sales; supplier role table; invoice from the order by quantity, editable price, discount and VAT; receipts and invoices only from an order; six kinds; goods booked at invoice time to 156 (known ceiling until valuation); payments allocate to named invoices of one supplier and org unit; duplicate invoice guard by (supplier row, symbol, number); item and stock screens stay in Sales until M19. Rejected: B (from receipts), C (one voucher), inside Sales.
- [ ] **Step 2: Roadmap M18 section**: Goal, then **Done when** copied from the spec's six exit criteria.
- [ ] **Step 3: `CONTEXT.md` Purchasing**: Supplier, Purchase order (vi "Đơn mua hàng"), Goods receipt ("Phiếu nhập kho"), Supplier invoice ("Hoá đơn mua hàng"), Supplier payment ("Phiếu chi"), Payment allocation ("Phân bổ chi"), Receipt state. Note "receipt" alone is avoided.
- [ ] **Step 4: `products/purchasing.md`** in the shape of `products/sales.md` (scope, module, entities, record types, statuses and actions, progress, amounts by reference to Sales, permissions, personal data, frontend, proving flow), from the spec.
- [ ] **Step 5: Commit** `docs(purchase): ADR-0090 and M18 scope`.

### Task 1: Module skeleton, product registration, tables and kinds

Follow `.claude/skills/new-module` and the shape of commit `8d3a7b4c` (accounting skeleton).

**Files:**
- Create: `migrations/<ts>_purchase_init.sql`, `internal/modules/purchase/{service.go,types.go,deps.go,module.go,handler.go,posting.go,service_test.go}`, `internal/modules/purchase/queries/supplier.sql`, `internal/modules/purchase/i18n/{vi,en}.json`, `web/src/purchase/index.ts`, `web/src/purchase/i18n/{vi,en}/{meta,main}.json`, `web/src/shared/api/purchase.ts`
- Modify: `internal/app/app.go` (`"purchase": nil` in products; `purchase.NewService`, `purchase.Module`), `internal/app/auth_test.go`, `.golangci.yml` (depguard rule `purchase`, as `sales`), `sqlc.yaml`, `Makefile` (`PRODUCTS` adds `purchase`), `scripts/e2e-server.sh` (`:8090` `PRODUCTS=hrm,sales,accounting,purchase`), `web/src/app/areas.ts`, `web/src/shared/api/client.ts`
- Test: `internal/modules/purchase/service_test.go`

**Interfaces:**
- Produces:
  - `Deps{IAM *iam.Service; Record *record.Service; Audit *audit.Service; Setting *setting.Service; Printing *printing.Service; Numbering *numbering.Service; Partner *partner.Service; Stock *stock.Service; Posting *posting.Service}`.
  - Permissions `PermSupplierView = "purchase.supplier.view"`, `PermSupplierEdit`, `PermOrderView = "purchase.order.view"`, `PermOrderEdit`, `PermApprovalManage = "purchase.approval.manage"`; roles `staff`, `manager` (all work permissions), `viewer` (views), `approval_admin` (tenant-wide only), registered as in `sales/service.go`.
  - Kinds `KindGoods = posting.Kind("purchase_goods")`, `KindExpense = "purchase_expense"`, `KindInputVAT = "purchase_input_vat"`, `KindPayable = "purchase_payable"` (ByPartner true), `KindCashPayment = "purchase_cash_payment"`, `KindBankPayment = "purchase_bank_payment"`; `declareKinds` under `sync.OnceFunc`.
  - Test fixture `newFixture(t)` building the service with real `pgtest` DB, as `sales/service_test.go`.
- [ ] **Step 1: Migration** (all tables of the milestone, so later tasks only add queries):
  - `purchase.suppliers (id identity PK, code text NOT NULL, partner_id bigint NOT NULL REFERENCES partner.partners, org_unit_id bigint NOT NULL REFERENCES iam.org_units, payment_terms text, active boolean NOT NULL DEFAULT true)`; `suppliers_code_key` unique on `lower(code)`; `suppliers_partner_org_unit_key UNIQUE (partner_id, org_unit_id)`.
  - `purchase.orders (id PK REFERENCES record.documents DEFERRABLE INITIALLY DEFERRED, supplier_id REFERENCES purchase.suppliers, supplier_code, supplier_name text NOT NULL, supplier_tax_code, supplier_address, supplier_phone, supplier_email, contact_name text, expected_date date, payment_terms, note text, subtotal, discount_total, vat_total, total bigint NOT NULL, CHECK (total = subtotal - discount_total + vat_total))`.
  - `purchase.order_lines (order_id REFERENCES purchase.orders ON DELETE CASCADE, position int CHECK > 0, item_id REFERENCES item.items, item_code, description, unit text NOT NULL, stocked boolean NOT NULL, quantity numeric(15,3) CHECK > 0, unit_price bigint CHECK >= 0, discount_percent numeric(5,2) CHECK BETWEEN 0 AND 100, vat_rate text CHECK IN ('none','0','5','8','10'), amount, discount, vat bigint NOT NULL, PK (order_id, position))`.
  - `purchase.goods_receipts (id PK → record.documents, order_id → purchase.orders, warehouse_id → stock.warehouses, note)`; `purchase.goods_receipt_lines (receipt_id … CASCADE, position, order_position, item_id, item_code, description, unit, lot_code text, lot_expiry date, serial_code text, quantity numeric(15,3) CHECK > 0, PK)`.
  - `purchase.invoices (id PK → record.documents, order_id, supplier_invoice_symbol text, supplier_invoice_no text, supplier_invoice_date date, subtotal, discount_total, vat_total, total bigint NOT NULL, note)`; index on `(order_id)`; `purchase.invoice_lines` as `order_lines` plus `order_position`, keyed `(invoice_id, position)`.
  - `purchase.payments (id PK → record.documents, supplier_id, method text CHECK IN ('cash','bank'), amount bigint CHECK > 0, note)`; `purchase.payment_allocations (payment_id … CASCADE, position, invoice_id → purchase.invoices, amount bigint CHECK > 0, PK (payment_id, position), UNIQUE (payment_id, invoice_id))`; index on `invoice_id`.
- [ ] **Step 2: Failing tests** `TestPurchaseKindsDeclared` (`KindPayable` ByPartner, the other five not; two services do not panic) and in `internal/app` a check that `PRODUCTS=purchase` is accepted.
- [ ] **Step 3: Implement** the skeleton and wiring; web area manifest with product `purchase`, label `purchase.nav.group`, `basePath: '/purchase'`, empty nav, `homeIcon: IconShoppingBag`; `api = productClient<'/purchase/'>()`.
- [ ] **Step 4: Run** `make gen && go test ./internal/modules/purchase ./internal/app` → PASS.
- [ ] **Step 5: Commit** `feat(purchase): module skeleton, tables and posting kinds`.

### Task 2: Suppliers

**Files:**
- Create: `internal/modules/purchase/{supplier.go,supplier_handler.go,supplier_test.go}`
- Modify: `queries/supplier.sql`, `service.go`, `handler.go`

**Interfaces:**
- Produces: `const supplierType = "purchase.supplier"` (catalogue, code series `"NC"` via `Numbering.RegisterSeries`); `SupplierFields{OrgUnitID int64; PaymentTerms *string; Active bool}`, `NewSupplier{RequestID string; PartnerID *int64; Partner *partner.PartnerFields; Code string; SupplierFields}`, `Supplier`, `SupplierFilter`, `SupplierList` shaped exactly like their `sales/customer.go` counterparts; `CreateSupplier`, `UpdateSupplier`, `Supplier`, `Suppliers`. Errors `ErrSupplierCodeTaken` (`supplier_code_taken`), `ErrSupplierExists` (`supplier_exists`), `ErrSupplierInactive` (`supplier_inactive`). Routes `GET/POST /purchase/suppliers`, `GET/PUT /purchase/suppliers/{id}`.
- [ ] **Step 1: Failing tests** copied from the customer tests and renamed: create from a new partner and from an existing partner; second supplier for the same partner and org unit → `supplier_exists`; identity edit without `partner.edit` refused; team B staff gets not found; `recordtest` contract for `purchase.supplier` if customers have one.
- [ ] **Step 2: Run** → FAIL. **Step 3: Implement** from `sales/customer.go`. **Step 4: Run** `go test ./internal/modules/purchase` → PASS.
- [ ] **Step 5: Commit** `feat(purchase): suppliers`.

### Task 3: Purchase orders

**Files:**
- Create: `internal/modules/purchase/{calc.go,calc_test.go,order.go,order_types.go,order_list.go,order_handler.go,order_test.go,item.go}`, `queries/order.sql`, `queries/item.sql`

**Interfaces:**
- Produces:
  - `compute(lines []LineInput, mode platform.Rounding) ([]amounts, Totals, error)`: `sales/calc.go` without list prices and `max_discount`.
  - `const orderType = "purchase.order"`, prefix `DM`, approval fields `amount` and `items` (as Sales declares `items`).
  - `LineInput{ItemID int64; Description string; Quantity string; UnitPrice int64; DiscountPercent, VatRate string}`; `OrderFields{Date string; OrgUnitID, SupplierID int64; ExpectedDate, PaymentTerms, Note *string; Lines []LineInput (1..500)}`; `NewOrder{RequestID string; OrderFields}`; `OrderUpdate{Version int32; OrderFields}`; `Line{LineInput; Position int; ItemCode, Unit string; Stocked bool; Amount, Discount, Vat int64}`; `Order{ID int64; Number, Status string; Version int32; Supplier SupplierCopy; …; Lines []Line; Totals Totals; AllowedActions []string}` (progress fields added in Tasks 4–6).
  - `CreateOrder`, `UpdateOrder`, `Order`, `Orders(ctx, OrderFilter) (OrderList, error)`, `OrderActions`, `LineItems(ctx, q string)` (`GET /purchase/line-items`, open to `purchase.order.edit` anywhere).
  - Errors reuse Sales' codes where the meaning is the same (`invalid_quantity` etc. via `errLine`); `ErrOrderHasReceipt` (`purchase_order_has_receipt`), `ErrOrderHasInvoice` (`purchase_order_has_invoice`) defined here, exercised in Tasks 4 and 5.
- [ ] **Step 1: Failing tests:** `TestCalcMatchesSales` (the cases of `sales/calc_test.go` without discount-cut cases, both rounding modes); `TestOrderRecordContract` (`recordtest`); `TestOrderLifecycle` (draft → send with no rule posts; supplier details copied at save and again at send; `stocked` true for a lot item, false for a service); `TestOrderInactiveSupplier` (`supplier_inactive` on draft and on send); `TestOrderApprovalAmount` (a rule on `amount` holds a 10 000 000 đ order in `pending_approval`); `TestOrderTeamBoundary`; `TestOrderList` (filters `q`, status, supplier).
- [ ] **Step 2: Run** → FAIL. **Step 3: Implement** from `sales/doc.go` with one kind. **Step 4: Run** → PASS.
- [ ] **Step 5: Commit** `feat(purchase): purchase orders`.

### Task 4: Goods receipts and receipt state

**Files:**
- Create: `internal/modules/purchase/{receipt.go,receipt_types.go,receipt_list.go,receipt_handler.go,receipt_test.go}`, `queries/receipt.sql`
- Modify: `order.go` (cancel refusal, `create_goods_receipt` action, receipt progress), `order_types.go`, `queries/order.sql`

**Interfaces:**
- Produces:
  - `const goodsReceiptType = "purchase.goods_receipt"`, prefix `NK`, order permissions, no approval fields.
  - `ReceiptLineInput{OrderPosition int; Quantity string; Lot *stock.LotRef; Serial *string}`; `ReceiptFields{Date string; WarehouseID int64; Note *string; Lines []ReceiptLineInput (1..1000)}`; `NewGoodsReceipt{RequestID string; OrderID int64; ReceiptFields}`; `GoodsReceiptUpdate`, `GoodsReceipt`, `GoodsReceiptFilter`, `GoodsReceiptList` mirroring `sales/delivery_types.go`.
  - `CreateGoodsReceipt`, `UpdateGoodsReceipt`, `GoodsReceipt`, `GoodsReceipts`, `ReceiptOptions(ctx, orderID) (ReceiptOptions, error)` (warehouses of the legal entity, stock lines with ordered and remaining, item tracking).
  - `Order.ReceiptState` (`none,partial,full,not_applicable`), `Line.Received *string`, `OrderFilter.ReceiptState`, action `create_goods_receipt`.
  - Errors `purchase_receipt_over_order` (params `line_no`, `order_position`, `remaining`), `purchase_receipt_position`, `purchase_receipt_before_order`, `purchase_receipt_lot_required`, `purchase_receipt_serial_required` (serial item line without serial or quantity ≠ 1).
  - Routes `GET /purchase/orders/{id}/receipt-options`, `GET/POST /purchase/goods-receipts`, `GET/PUT /purchase/goods-receipts/{id}`.
- [ ] **Step 1: Failing tests** (fixture: posted order of team A: lot item 10 × 50 000 đ at 8 %, serial item 2 × 3 000 000 đ at 10 %, transport service 1 × 500 000 đ at 10 %):
  - `TestGoodsReceiptRecordContract`.
  - `TestGoodsReceiptPartial`: 6 of the lot item in lot `L1` (expiry set) and serial `S1` posts; on hand at the warehouse is 6 in `L1` and 1 for `S1`; a second receipt of 5 → `purchase_receipt_over_order`, `remaining` `"4.000"`; 4 in lot `L2` and `S2` post; order receipt state `partial` then `full`; the service line is not offered.
  - `TestGoodsReceiptSerials`: serial line with quantity 2 → `purchase_receipt_serial_required`; `S1` again after it is on hand → the ledger's serial error.
  - `TestGoodsReceiptCancelAfterIssue`: after a stock adjustment issues 6 of `L1`, cancelling the first receipt → `stock.insufficient`.
  - `TestGoodsReceiptRefusedWhileOrderCancelHeld`; `TestOrderCancelWithReceipt` (`purchase_order_has_receipt`, then allowed after the receipt is cancelled); `TestGoodsReceiptTeamBoundary`; `TestGoodsReceiptDraftHoldsQuantity`.
- [ ] **Step 2: Run** → FAIL. **Step 3: Implement** from `sales/delivery.go`, quantities positive in `stock.Record`; lots and serials named by code (created by the ledger). **Step 4: Run** → PASS.
- [ ] **Step 5: Commit** `feat(purchase): goods receipts into stock`.

### Task 5: Supplier invoices and invoice state

**Files:**
- Create: `internal/modules/purchase/{invoice.go,invoice_types.go,invoice_list.go,invoice_handler.go,invoice_test.go}`, `queries/invoice.sql`
- Modify: `order.go`, `order_types.go`, `queries/order.sql`, `queries/supplier.sql` (`LockSupplier`)

**Interfaces:**
- Produces:
  - `const invoiceType = "purchase.invoice"`, prefix `HM`, order permissions, `Header.Amount = &total`.
  - `InvoiceLineInput{OrderPosition int; Quantity string; UnitPrice int64; DiscountPercent, VatRate string}`; `InvoiceFields{Date string; SupplierInvoiceSymbol, SupplierInvoiceNo, SupplierInvoiceDate, Note *string; Lines []InvoiceLineInput (1..500)}`; `NewInvoice{RequestID string; OrderID int64; InvoiceFields}`; `InvoiceUpdate`; `InvoiceLine`, `Invoice`, `InvoiceFilter`, `InvoiceList` mirroring `sales/invoice_types.go` plus the three supplier fields.
  - `CreateInvoice`, `UpdateInvoice`, `Invoice`, `Invoices`, `InvoiceOptions(ctx, orderID)` (each order line: ordered, remaining, price, discount, VAT rate as pre-fill).
  - `Order.InvoiceState`, `Line.Invoiced *string`, action `create_invoice`, `OrderFilter.InvoiceState`.
  - Errors `purchase_invoice_over_order`, `purchase_invoice_position`, `purchase_invoice_before_order`, `purchase_invoice_supplier_ref_required` (symbol, number or date missing on send), `purchase_invoice_duplicate` (params `number`: the other invoice's document number), `purchase_invoice_has_payment` (used in Task 6).
  - Routes `GET /purchase/orders/{id}/invoice-options`, `GET/POST /purchase/invoices`, `GET/PUT /purchase/invoices/{id}`.
- [ ] **Step 1: Failing tests** (Task 4's order fixture):
  - `TestSupplierInvoiceRecordContract`.
  - `TestSupplierInvoicePostingLines`: invoice of 4 lot items at 52 000 đ (changed from 50 000), 2 serials, the service; posts `purchase_goods` 6 208 000, `purchase_expense` 500 000, `purchase_input_vat` 666 640, `purchase_payable` 7 374 640 with the supplier's partner, all at team A; cancel voids them; a 0 % / `none` invoice writes no VAT line.
  - `TestSupplierInvoicePartial`: over-quantity refused with `remaining`; invoice state `partial` then `full`, counting quantities while amounts differ from the order.
  - `TestSupplierInvoiceDuplicate`: second invoice of the same supplier row with symbol `1C26TAA`, number `0000123` → `purchase_invoice_duplicate` on send; same number with symbol `1C26TBB` posts; same symbol and number for a supplier of team B posts.
  - `TestSupplierInvoiceDuplicateConcurrent`: two goroutines send the two duplicates at once; exactly one succeeds.
  - `TestSupplierInvoiceSupplierRef`: send without symbol → `purchase_invoice_supplier_ref_required`; saving a draft without it works.
  - `TestSupplierInvoicePostsWithoutAccounting`; `TestOrderCancelWithInvoice`; `TestSupplierInvoiceTeamBoundary`.
- [ ] **Step 2: Run** → FAIL. **Step 3: Implement** from `sales/invoice.go`: the line takes item, texts and `stocked` from the order line and price, discount, VAT rate from the input; `compute` on them. Duplicate check in `beforeSubmit` and in `OnTransition` into `posted`: `LockSupplier` (`SELECT … FOR UPDATE`) then query other invoices of the supplier row in status `pending_approval` or `posted` with the same symbol and number. Goods vs expense split by `stocked`. **Step 4: Run** → PASS.
- [ ] **Step 5: Commit** `feat(purchase): supplier invoices from purchase orders`.

### Task 6: Supplier payments and payment state

**Files:**
- Create: `internal/modules/purchase/{payment.go,payment_types.go,payment_list.go,payment_handler.go,payment_test.go}`, `queries/payment.sql`
- Modify: `invoice.go`, `invoice_types.go`, `invoice_list.go`, `order.go`, `order_types.go`, `queries/invoice.sql`, `queries/order.sql`

**Interfaces:**
- Produces:
  - `const paymentType = "purchase.payment"`, prefix `PC`, order permissions, `Header.Amount = &amount`.
  - `AllocationInput{InvoiceID int64; Amount int64}`; `PaymentFields{Date string; OrgUnitID, SupplierID int64; Method string (cash,bank); Note *string; Allocations []AllocationInput (1..200)}`; `NewPayment`, `PaymentUpdate`, `Allocation`, `Payment`, `PaymentFilter`, `PaymentList` mirroring `sales/receipt_types.go`.
  - `CreatePayment`, `UpdatePayment`, `Payment`, `Payments`, `OpenInvoices(ctx, supplierID, orgUnitID, paymentID int64)` (`GET /purchase/suppliers/{id}/open-invoices`).
  - `Invoice.Paid, Remaining int64`, `Invoice.PaymentState`, `Invoice.Payments []PaymentRef`, action `create_payment`; `Order.Paid *int64`, `Order.PaymentState`; filters on payment state.
  - Errors `purchase_payment_over_invoice`, `purchase_payment_invoice`, `purchase_payment_invoice_org_unit`, `purchase_payment_before_invoice`, `purchase_payment_duplicate_invoice`.
  - Routes `GET/POST /purchase/payments`, `GET/PUT /purchase/payments/{id}`.
- [ ] **Step 1: Failing tests** (two posted invoices of team A for one supplier, totals T1, T2): `TestPaymentRecordContract`; `TestPaymentSplit`; `TestPaymentOverInvoice`; `TestPaymentRules` (other supplier, other org unit, draft invoice, date before invoice, same invoice twice); `TestPaymentPostingLines` (`cash` writes `purchase_cash_payment` +amount and `purchase_payable` −amount with the partner; `bank` writes `purchase_bank_payment`; cancel voids); `TestPaymentConcurrentAllocations`; `TestPaymentInactiveSupplier` (posts); `TestCancelChain` (payment → invoice → order); `TestInvoicePaymentState`; `TestPaymentDraftHoldsAmount`; `TestOrderPaymentState`; `TestPaymentList`.
- [ ] **Step 2: Run** → FAIL. **Step 3: Implement** from `sales/receipt.go`. **Step 4: Run** → PASS.
- [ ] **Step 5: Commit** `feat(purchase): supplier payments allocated to invoices`.

### Task 7: Printing

**Files:**
- Create: `internal/modules/purchase/{print.go,print_test.go}`
- Modify: `i18n/{vi,en}.json`, `service.go`

**Interfaces:**
- Produces: templates `purchase.order`, `purchase.goods_receipt` (warehouse, lines with lots and serials, signature blocks for storekeeper and deliverer), `purchase.invoice` (supplier symbol, number, date), `purchase.payment` (method, amount, allocations), following `sales/print.go`.
- [ ] **Step 1: Failing tests:** `TestOrderPrintFrozen` (reprint identical after supplier and legal entity edits); `TestPaymentPrintNeedsView` (team B cannot print).
- [ ] **Step 2–4:** implement, run `go test ./internal/modules/purchase` → PASS.
- [ ] **Step 5: Commit** `feat(purchase): print orders, goods receipts, invoices and payments`.

### Task 8: Supplier and order screens

Use the `mmerp-ui` and `new-screen` skills; finish with the `docs/ui.md` checklist.

**Files:**
- Create: `web/src/purchase/{routes.tsx,keys.ts}`, `hooks/usePurchase.ts`, `pages/{SuppliersPage,SupplierPage,NewSupplierPage,OrdersPage,OrderPage,NewOrderPage,ApprovalRulesPage}.tsx`, `components/{SupplierForm,OrderForm,OrderPreview,OrderProgress}.tsx`
- Modify: `web/src/purchase/index.ts` (nav suppliers, orders, approval rules; `recordTypes` for `purchase.supplier`, `purchase.order`), `web/src/shared/api/purchase.ts` (types), `i18n/{vi,en}`
- Test: Vitest next to any pure helper extracted (e.g. order line editing), as `web/src/sales/*.test.ts`.

- [ ] **Step 1: Failing Vitest** for the extracted helpers.
- [ ] **Step 2–4:** implement from the Sales customer and order pages; amounts are the server's; `pnpm -C web test` and `pnpm -C web exec tsc --noEmit` → PASS.
- [ ] **Step 5: Commit** `feat(web): purchase supplier and order screens`.

### Task 9: Goods receipt, invoice and payment screens

**Files:**
- Create: `pages/{GoodsReceiptsPage,GoodsReceiptPage,NewGoodsReceiptPage,InvoicesPage,InvoicePage,NewInvoicePage,PaymentsPage,PaymentPage,NewPaymentPage}.tsx`, `components/{GoodsReceiptForm,GoodsReceiptPreview,InvoiceForm,InvoicePreview,PaymentForm,PaymentPreview}.tsx`, `receiptLines.ts`, `invoiceLines.ts`, `paymentAllocations.ts` with tests
- Modify: `index.ts` (nav goods receipts, invoices, payments; their `recordTypes` with `invalidate` as Sales does), `OrderPage.tsx` (actions `create_goods_receipt` → `/purchase/goods-receipts/new?order=<id>`, `create_invoice`; received, invoiced, paid columns and states), `InvoicePage.tsx` (`create_payment`), `i18n/{vi,en}`

- [ ] **Step 1: Failing Vitest:** `receiptLines` (remaining pre-fill; a serial line expands to one row per unit; over-order error mapped to the row), `invoiceLines` (pre-fill price, discount, VAT from options; error mapping), `paymentAllocations` (total, pre-fill from one invoice).
- [ ] **Step 2–4:** implement from the Sales delivery, invoice and receipt screens; the receipt form takes lot code and expiry or serial per row; `pnpm -C web test` and `pnpm -C web exec tsc --noEmit` → PASS.
- [ ] **Step 5: Commit** `feat(web): goods receipt, supplier invoice and payment screens`.

### Task 10: Accounting labels, seed and end to end

**Files:**
- Modify: `web/src/accounting/i18n/{vi,en}/main.json` (six kind labels), `web/e2e/accounting.seed.ts` (accounts 156, 1331, 642 if absent; rules for the six kinds), `web/e2e/demo.seed.ts` (call the purchase seed after Sales), `web/e2e/product-disabled.spec.ts` (Purchasing read-only on `:8091`)
- Create: `web/e2e/purchase.seed.ts` (two suppliers, one of them a partner already a customer; an order received and invoiced in parts; one payment split across two invoices; one invoice unpaid), `web/e2e/purchasing.spec.ts`

- [ ] **Step 1: Write the Playwright spec** covering exit criteria 1 to 4 and 6 through the UI and API: supplier from a customer's partner; order approved; two receipts with lots and serials, the refused third, on hand in the Sales stock screen; invoice with a changed price and the ledger lines and partner balance; duplicate symbol and number refused; split payment, states and ledger; over-allocation and cancel refusals; team B sees nothing. Criterion 5's goods-receipt cancel refusal is covered by `TestGoodsReceiptCancelAfterIssue`.
- [ ] **Step 2: Run** `make test` and `make gen` with no diff; `make e2e` runs in CI on the PR (locally only if the user asks).
- [ ] **Step 3: Commit** `test(purchase): purchasing end to end`.

After Task 10: `code-review` on the branch, then one PR `feat(purchase): purchasing` (body per house-style, lists the migration). Marking M18 done in the roadmap is a separate PR after acceptance.
