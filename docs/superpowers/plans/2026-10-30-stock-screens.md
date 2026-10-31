# Stock and delivery screens (part 3 of 3) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Users of the Sales area can manage warehouses, see on hand and a stock card, enter stock adjustments, set an item's stock tracking, and deliver a confirmed order with lot suggestion; orders show their delivery state and a shortage warning. Two small backend rules come first.

**Architecture:** Everything lives in the Sales area (`web/src/sales`), built from `shared/ui/page` templates and `shared/document`. Stock and item calls go through the core client (`@/shared/api/core`), Sales calls through `@/shared/api/sales`. Buttons come from `allowed_actions`; the screen computes no quantity the server already gives.

**Tech Stack:** React, TypeScript, TanStack Query, react-hook-form, `@mmerp/ui`, openapi-fetch; Go for Task 1.

**Spec:** `docs/adr/0049-shared-stock-ledger-and-sales-delivery.md`, `products/sales.md`, `docs/ui.md`, the `mmerp-ui` and `new-screen` skills.

## Global Constraints

- Page templates: lists `ListPage`; documents `DocumentScreen` (detail) and `CreatePage` (new); catalogue create/edit `FormModal` like `ItemsPage`. No `style`, no CSS file, no colour code in the area.
- Every string is an i18n key in `vi` and `en`, using `CONTEXT.md` terms (Kho, Tồn kho thực tế, Thẻ kho, Phiếu điều chỉnh kho, Phiếu giao hàng, Lô hàng, Hạn sử dụng, Số serial, Đã giao, Đã đặt, Khả dụng).
- Error codes: codes raised by `stock` and `item` (not products) go in `web/src/shared/i18n/locales/{vi,en}.json` as `shared.error.<code>`; codes raised by Sales go in `web/src/sales/i18n/{vi,en}/meta.json` as `sales.error.<code>`. Params interpolate (`{{line_no}}`, `{{missing}}`, `{{remaining}}`).
- Quantities are decimal strings: input with `DecimalField scale={3}` (`allowNegative` for adjustment lines), display with `formatDecimal(value, 3)`. Never `Number()` a quantity for arithmetic.
- The URL is the state (filters, sort, page, stock-card parameters).
- Permissions: stock pages need `stock.view` (nav and route); warehouse writes need `stock.manage` (`useCan`, catalogue without `allowed_actions`); adjustments and deliveries use `allowed_actions` and the `/actions` collection endpoints.
- New record types in the Sales manifest `recordTypes`: `stock.adjustment` (path `stock/adjustments/${id}`) and `sales.delivery` (path `deliveries/${id}`, `preview`, `invalidate` the deliveries list, the order detail and order lists).
- Verification per task: Vitest for logic the task adds (`pnpm -C web test -- <file>`), `go test` for Task 1, `make gen` clean. Lint, tsc and e2e run only if the user allows them (see the handoff).

## Review Focus

- A lot line on the delivery form after the warehouse changes: suggestions and serials reload for the new warehouse; stale lot codes are cleared, not sent. Test in Task 7.
- An order whose `allowed_actions` lacks `create_delivery` shows no delivery button, even when it is posted. Test in Task 8.
- A stock-card URL with a range over 366 days or `from > to` shows the server's error next to the date filter, not a blank table. Covered in Task 5.
- An adjustment line with a negative quantity on a lot item shows the lot field required before sending (the server refuses with `stock_lot_required` otherwise; the error lands on the line). Test in Task 6.
- A user holding Sales roles but no stock role sees no Kho menu group and gets `ForbiddenPage` on a typed stock URL. Covered in Task 3.

---

### Task 1: Backend: expired lots and delivery date

**Files:** `internal/modules/sales/delivery.go`, `delivery_types.go`, `delivery_test.go`

**Interfaces:**
- Produces: `ErrLotExpired` (`sales_lot_expired`, 422, params `line_no`, `lot_code`, `expiry`): a delivery line's lot whose expiry is before the delivery date, checked at save and again at posting. `ErrDeliveryDate` (`sales_delivery_date`, 422): delivery date before the order date. Stock adjustments are unaffected (they may write off expired lots).

- [ ] **Step 1: Failing tests** `TestDeliveryRefusesExpiredLot` (lot expiry 2026-11-01, delivery dated 2026-11-02 → `ErrLotExpired`; dated 2026-11-01 → saves), `TestDeliveryDateBeforeOrder` → `ErrDeliveryDate`.
- [ ] **Step 2: Run** `go test ./internal/modules/sales -run 'Expired|DeliveryDate'` → FAIL.
- [ ] **Step 3: Implement** in `checkDelivery` (lot expiry read through a stock read function, not a raw query of `stock` tables from sales if one exists; otherwise a sales query joining `stock.lots` is allowed).
- [ ] **Step 4: Run** `go test ./internal/modules/sales ./internal/app` → PASS; `make gen` clean.
- [ ] **Step 5: Commit** `feat(sales): refuse expired lots and early dates on deliveries`.

### Task 2: Error messages

**Files:** `web/src/shared/i18n/locales/{vi,en}.json`, `web/src/sales/i18n/{vi,en}/meta.json`

- [ ] **Step 1:** List every stable code the backend added in #155, #156 and Task 1: grep `Code: "` in `internal/shared/stock`, `internal/shared/item`, `internal/modules/sales/delivery_types.go`.
- [ ] **Step 2:** Add `shared.error.<code>` for `stock_*` and `item_*`, and `sales.error.<code>` for `sales_*` and `sales_delivery_*`, in both languages, short, telling the user what to change (e.g. vi `stock_insufficient`: "Không đủ hàng trong kho: thiếu {{missing}}." ; `sales_delivery_over_ordered`: "Dòng {{line_no}} vượt số còn phải giao ({{remaining}})."). Follow `mmerp-ui` `references/content.md`.
- [ ] **Step 3: Run** `node web/scripts/check-i18n.mjs` → complete.
- [ ] **Step 4: Commit** `feat(web): messages for stock and delivery errors`.

### Task 3: Warehouses and item tracking

**Files:**
- Create: `web/src/sales/pages/WarehousesPage.tsx`
- Modify: `web/src/sales/pages/ItemsPage.tsx`, `web/src/sales/index.ts` (nav group `sales.nav.group_stock` with on hand, stock card, adjustments, warehouses; `permission: 'stock.view'`), `web/src/sales/routes.tsx`, `web/src/sales/keys.ts` (`salesKeys.stock.{warehouses,onHand,card,adjustments}`), `web/src/sales/i18n/{vi,en}/main.json`, `meta.json` (nav labels)
- Modify (backend, small): `internal/shared/stock` warehouse update input without `legal_entity_id` (`WarehouseUpdate{Name, Active}`), `make gen`.

**Interfaces:**
- Produces: `WarehousesPage` (`ListPage`: code, name, legal entity, default, status; create/edit in a `FormModal` with `TextField` name, `LegalEntityField` on create only, `CheckboxField` active; writes gated by `useCan('stock.manage')`). Item form gains `SelectField` tracking (Không theo dõi / Theo số lượng / Theo lô / Theo serial) and `CheckboxField` allow negative shown only when tracking is `quantity`; `item_tracking_locked` maps to the tracking field via `errorFields`. Items list gains a tracking column.

- [ ] **Step 1:** Backend change and a Go test that `PUT /stock/warehouses/{id}` accepts `{name, active}`.
- [ ] **Step 2:** Build the page, route (`useCan('stock.view') ? … : <ForbiddenPage/>`), nav group, item form fields.
- [ ] **Step 3: Verify** `go test ./internal/shared/stock`; Vitest for any helper added; `make gen` clean.
- [ ] **Step 4: Commit** `feat(sales): warehouses and item stock tracking screens`.

### Task 4: On hand

**Files:** `web/src/sales/pages/OnHandPage.tsx`, routes, keys, i18n

**Interfaces:**
- Produces: `OnHandPage` (`ListPage` over `GET /stock/on-hand`): columns warehouse, item code and name, unit, lot, expiry (`formatDate`, expired in danger tone with text "Hết hạn"), serial, quantity (`formatDecimal`, right aligned); filters: `SearchInput` (q), warehouse `SelectField` filter, "expiring before" `DateField`; row `href` opens the stock card for that warehouse and item with the current month as range.

- [ ] **Step 1–3:** Build, wire route `stock/on-hand`, verify as Task 3.
- [ ] **Step 4: Commit** `feat(sales): on-hand list`.

### Task 5: Stock card

**Files:** `web/src/sales/pages/StockCardPage.tsx`, routes, keys, i18n

**Interfaces:**
- Produces: `StockCardPage` at `stock/card?warehouse_id=&item_id=&from=&to=` (`ListPage` without paging): warehouse and item pickers (`ItemField` on the core `/items` list), `DateField` from/to defaulting to the current month; an opening row, the moves (date, document number linking through `recordTypes` path when `doc_id` ≠ 0, otherwise the document type label without a link, lot, serial, quantity signed, running balance), a closing row. `stock_card_range_invalid`/`_too_long` render beside the dates.

- [ ] **Step 1–3:** Build, wire, verify.
- [ ] **Step 4: Commit** `feat(sales): stock card`.

### Task 6: Stock adjustments

**Files:** `web/src/sales/pages/{AdjustmentsPage,NewAdjustmentPage,AdjustmentPage}.tsx`, `web/src/sales/components/AdjustmentForm.tsx`, `AdjustmentForm.test.ts`, manifest `recordTypes['stock.adjustment']`, routes, keys, i18n

**Interfaces:**
- Produces: list (`DocumentStatusFilter docType="stock.adjustment"`, create from `GET /stock/adjustments/actions`); `CreatePage` + `AdjustmentForm` (date, warehouse `SelectField`, reason, `LinesTable` columns item, lot code, expiry, serial, quantity `DecimalField allowNegative`); detail `DocumentScreen` with the form read-only unless `edit` is allowed. Lot/serial inputs show per the picked item's `tracking` (from the item list row). Pure helper `adjustmentLineErrors(line, tracking): string[]` (lot required for lot items, serial required and quantity ±1 for serial items, none for quantity items) used to block sending.

- [ ] **Step 1: Failing test** `AdjustmentForm.test.ts` for `adjustmentLineErrors` (four tracking cases, negative lot line without lot → `lot_required`).
- [ ] **Step 2: Run** `pnpm -C web test -- AdjustmentForm` → FAIL.
- [ ] **Step 3: Implement** pages and form.
- [ ] **Step 4: Run** → PASS.
- [ ] **Step 5: Commit** `feat(sales): stock adjustment screens`.

### Task 7: Delivery form and pages

**Files:** `web/src/sales/pages/{DeliveriesPage,NewDeliveryPage,DeliveryPage}.tsx`, `web/src/sales/components/{DeliveryForm,DeliveryPreview}.tsx`, `deliveryLines.ts`, `deliveryLines.test.ts`, manifest `recordTypes['sales.delivery']`, routes `deliveries`, `deliveries/new?order=`, `deliveries/:id`, nav item Phiếu giao hàng (`sales.order.view`), keys, i18n

**Interfaces:**
- Consumes: `GET /sales/orders/{id}/delivery-options?warehouse_id=`, `POST /sales/deliveries`, `PUT /sales/deliveries/{id}`, `GET /sales/deliveries`, `GET /sales/deliveries/{id}`.
- Produces: pure `linesFromOptions(options): DeliveryLineInput[]` — one line per quantity item with `remaining`; for a lot item one line per suggested lot (code and quantity from the pick; if suggestions cover less than remaining, the last line keeps the shortfall without a lot so the user must pick); for a serial item no prefilled lines, a serial `SelectField` per added line from `serials`. `DeliveryForm`: order facts (number, customer), warehouse `SelectField` (changing it refetches options and rebuilds lines after a confirm if lines were edited), `LinesTable` (order line description, unit, remaining as a hint, lot select from options, serial select, quantity `DecimalField`), date default today. List page with status filter and order filter from the URL. Detail `DocumentScreen` with read-only lines; `DeliveryPreview` via `DocumentPreview`.

- [ ] **Step 1: Failing test** `deliveryLines.test.ts`: quantity line → one line with remaining; lot item with picks 2 + 2 and remaining 5 → three lines (2 L1, 2 L2, 1 without lot); serial item → no lines; service lines never present.
- [ ] **Step 2: Run** → FAIL.
- [ ] **Step 3: Implement** helper, form, pages, manifest entry.
- [ ] **Step 4: Run** → PASS.
- [ ] **Step 5: Commit** `feat(sales): delivery screens`.

### Task 8: Order page: delivery state, delivered, shortage

**Files:** `web/src/sales/pages/DocPage.tsx`, `web/src/sales/components/{DocProgress,docColumns,DocForm}.tsx`, `DocProgress.test.tsx` (or a pure helper test), i18n

**Interfaces:**
- Produces: `DocProgress` shows the order's `delivery_state` (`partial` info "Giao một phần", `full` positive "Đã giao đủ", `none` and `not_applicable` nothing); orders list gains a delivery state column and a `delivery_state` `FilterChip`; the order's read-only lines show "Đã giao" per stock line; while draft or pending, a stock line whose `available` is below its quantity shows a warning text "Khả dụng {{available}}" beside the quantity (never blocks); `CreateDeliveryButton` shown only when `allowed_actions` includes `create_delivery`, navigating to `deliveries/new?order={id}`; the order's deliveries listed as links in the header badges.

- [ ] **Step 1: Failing test** for the pure part (state → label/tone; shortage check comparing decimal strings without floats, e.g. with `decimal.js` if installed, else a string comparison helper with tests).
- [ ] **Step 2–4:** Implement, run tests.
- [ ] **Step 5: Commit** `feat(sales): delivery state and shortage warning on orders`.

### Task 9: End-to-end test and seed

**Files:** `web/e2e/deliveries.spec.ts`, `web/e2e/sales.seed.ts`

- [ ] **Step 1:** Spec: an admin with Sales and stock roles makes a lot item, enters opening stock by an adjustment with a new lot, confirms an order, creates a delivery from the order (lot prefilled), sends it, sees on hand drop on the on-hand page and the order show "Đã giao đủ"; cancelling the order is refused with the delivery message.
- [ ] **Step 2:** Seed: give the demo catalogue tracking, one opening adjustment per warehouse, one partly delivered order.
- [ ] **Step 3:** Run the spec only if the user allows e2e; otherwise leave it for CI.
- [ ] **Step 4: Commit** `test(sales): delivery flow end to end`.
