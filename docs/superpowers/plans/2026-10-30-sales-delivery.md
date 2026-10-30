# Sales delivery (part 2 of 3) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the Sales delivery document that takes goods of a confirmed order out of one warehouse through `stock.Record`, with delivered, committed and available quantities derived from documents. Backend only.

**Architecture:** `sales.delivery` is a third Sales document type next to quotations and orders, in its own area files (`delivery.go`, `delivery_types.go`, `delivery_list.go`, `delivery_handler.go`, `queries/delivery.sql`). Its `OnTransition` calls `stock.Record`/`stock.Void` (Sales imports `shared/stock`, a lower tier). Progress is derived by SQL over posted deliveries, never stored. Part 3 builds the screens.

**Tech Stack:** Go, Huma, sqlc with the data engine, Postgres 18, `shopspring/decimal`.

**Spec:** `docs/adr/0049-shared-stock-ledger-and-sales-delivery.md` (section "Sales delivery"); `products/sales.md`.

## Global Constraints

- Record type `sales.delivery`, `Product: "sales"`, `Kind: record.Document`, `NumberPrefix: "GH"`, no approval `Fields`. Perms reuse the order's: View `sales.order.view`, Edit `sales.order.edit`, ViewFiles view, Attach edit (constants `PermOrderView`, `PermOrderEdit`). Header `OrgUnitID` = the order's org unit, `Date` = delivery date, `Amount` nil.
- A delivery is made from a **posted** order; one warehouse per delivery, belonging to the order's legal entity and active; several deliveries per order.
- Only lines of items with `tracking` other than `none` appear on a delivery. An order's delivery state counts those lines only; an order with no stock lines has delivery state `not_applicable`.
- A delivery line never takes an order line beyond its ordered quantity less what other non-cancelled deliveries already take (drafts and pending included, so two drafts cannot together over-deliver at posting; posting re-checks under the order's lock).
- An order with a non-cancelled delivery (any status but `cancelled`) cannot be cancelled: stable code `sales_order_has_delivery` (409).
- Delivered, committed, available are derived, never stored. Delivered = sum of posted delivery lines per order line. Committed per (legal entity, item) = posted orders' stock-line quantities less delivered. Available = on hand in that legal entity's warehouses less committed.
- Confirming an order whose available quantity is short warns, never refuses: the order detail carries `available` per stock line while the order is `draft` or `pending_approval`.
- Quantities `numeric(15,3)` / `decimal.Decimal`, strings in JSON. Never float.
- Lock order: … → delivery's `record.documents` row → order's `record.documents` row via `record.Lock` → stock locks (items `FOR SHARE`, serials, balances, all inside `stock.Record`/`Void`) → `sales` rows.
- Sales users need no stock role: Sales serves warehouses, remaining quantities, lot suggestions and serials to its delivery form under `sales.order.edit`.
- Migrations via `make migration name=sales_delivery`; SQL only; CHECKs on kind columns.
- Do not run `make lint`, tsc, eslint, check-queries or e2e unless the user asks; verification is `go test` and a clean `make gen`.

## Review Focus

- Two deliveries for the same order line saved concurrently, each within the remaining quantity alone but over it together: one is saved, the other fails `sales_delivery_over_ordered` (the order's row lock serializes them). Test in Task 3.
- Cancelling an order while a delivery is being created for it: serialized by the order's row lock; either the delivery exists and the cancel fails, or the order is cancelled and the delivery create fails `sales_order_not_posted`. Test in Task 3.
- Cancelling a posted delivery restores on hand and lowers delivered; the order can then be cancelled once every delivery is cancelled. Test in Task 3.
- A delivery line for a lot item naming a lot not on hand in that warehouse: `stock_lot_not_found` (422, `line_no`), not `stock_insufficient`. Test in Task 5.
- The stock card shows a delivery's number only to someone who may view that delivery. Test in Task 6.

---

### Task 1: Delivery schema, record type and draft write path

**Files:**
- Create: `migrations/<ts>_sales_delivery.sql`, `internal/modules/sales/{delivery.go,delivery_types.go,delivery_handler.go,queries/delivery.sql,delivery_test.go}`
- Modify: `internal/modules/sales/service.go` (register type), `deps.go` (add `Stock *stock.Service`), `handler.go`, `internal/app/app.go`, `.golangci.yml` only if depguard needs it (sales importing `internal/shared/stock` should already be allowed), `internal/modules/sales/i18n/{vi,en}.json` if labels are needed

**Interfaces:**
- Produces:
  - Tables: `sales.deliveries (id → record.documents DEFERRABLE INITIALLY DEFERRED, order_id → sales.headers, warehouse_id → stock.warehouses, note text NULL)` with an index on `order_id`; `sales.delivery_lines (delivery_id → sales.deliveries ON DELETE CASCADE, position int > 0, order_position int, item_id → item.items, lot_code text NULL, serial_code text NULL, quantity numeric(15,3) > 0, PRIMARY KEY (delivery_id, position))`. The order line is (`sales.deliveries.order_id`, `order_position`), checked in Go at save.
  - `type DeliveryLineInput struct { OrderPosition int; LotCode *string; SerialCode *string; Quantity string }` (optional pointers `omitempty`; quantity pattern as `LineInput.Quantity`)
  - `type DeliveryFields struct { Date string; WarehouseID int64; Note *string; Lines []DeliveryLineInput /* 1..500 */ }`
  - `type NewDelivery struct { RequestID string; OrderID int64; DeliveryFields }`, `type DeliveryUpdate struct { Version int32; DeliveryFields }`
  - `type Delivery struct { ID, OrderID int64; OrderNumber, Number, Status string; Version int32; Date string; OrgUnitID int64; WarehouseID int64; WarehouseCode, WarehouseName string; Note *string; Lines []DeliveryLine; AllowedActions []string }`, `DeliveryLine` = input + `ItemID, ItemCode, Description, Unit`
  - `CreateDelivery(ctx, NewDelivery) (int64, error)`, `UpdateDelivery(ctx, id, DeliveryUpdate) error`, `Delivery(ctx, id) (Delivery, error)`; routes `POST /sales/deliveries`, `PUT /sales/deliveries/{id}`, `GET /sales/deliveries/{id}`
  - Errors: `ErrOrderNotPosted` (`sales_order_not_posted`, 409), `ErrDeliveryOverOrdered` (`sales_delivery_over_ordered`, 409, params `line_no`, `order_position`, `remaining`), `ErrDeliveryLineNotStock` (`sales_delivery_line_not_stock`, 422, `line_no`), `ErrDeliveryWarehouse` (`sales_delivery_warehouse`, 422: inactive or other legal entity)

Save path (create and update, same transaction): `record.Lock` the order → order must be `posted` → warehouse active and of the order's legal entity → each line's `order_position` exists on the order, its item has `tracking` ≠ `none`, quantity > 0 with ≤ 3 decimals → per order line, the sum of this delivery's lines + other non-cancelled deliveries' lines ≤ ordered quantity → `record.Create`/`Edit` → write lines in one `unnest`.

- [ ] **Step 1: Failing tests** in `delivery_test.go` (fixture copied from `service_test.go`: a company, a warehouse from `stock.DefaultWarehouse`, items with `quantity` and `lot` tracking and one `none`, a posted order with three lines 10 / 5 lot / 2 service): `TestDeliveryFromPostedOrderOnly` (draft order → `ErrOrderNotPosted`); `TestDeliveryRefusesServiceLine` → `ErrDeliveryLineNotStock`; `TestDeliveryOverOrdered` (draft A takes 6 of line 1, draft B asks 5 → `ErrDeliveryOverOrdered` with `remaining` "4.000"); `TestDeliveryWarehouseOfOtherEntity`; `TestDeliveryReadBack` (fields, order number, warehouse code, `allowed_actions` for a draft = edit, delete, submit).
- [ ] **Step 2: Run** `go test ./internal/modules/sales -run Delivery` → FAIL.
- [ ] **Step 3: Implement** per the save path above.
- [ ] **Step 4: Run** `go test ./internal/modules/sales ./internal/app` → PASS; `make gen` clean.
- [ ] **Step 5: Commit** `feat(sales): delivery documents from confirmed orders`.

### Task 2: Posting and cancelling a delivery

**Files:**
- Modify: `internal/modules/sales/delivery.go`, `service.go` (transition hook for the delivery type), `queries/delivery.sql`
- Test: `internal/modules/sales/delivery_test.go`, `internal/modules/sales/service_test.go` (recordtest contract)

**Interfaces:**
- Consumes: `stock.Service.Record(ctx, ref, []stock.Move)`, `stock.Service.Void(ctx, ref)`; `record.Service.Lock`.
- Produces: delivery `OnTransition`: into `posted` → `record.Lock` the order, order still `posted`, re-run the over-ordered check counting only other non-cancelled deliveries, build moves (`LineNo` = delivery position, `WarehouseID`, `ItemID`, lot/serial from the line, `Quantity` negated) and call `Record`; into `cancelled` → `Void`. `OnDelete` deletes the delivery row.

- [ ] **Step 1: Failing tests**: recordtest `contract(t, "sales.delivery")` variant (Table `sales.deliveries`; `Effects` = number of live stock moves of the delivery > 0 ? 1 : 0; `BreakPosting` = issue the on hand elsewhere so posting fails `stock_insufficient`; `RequireApproval`/`Approve`/`Reject` as for orders); `TestDeliveryPostsStock` (on hand 10, deliver 4 → on hand 6); `TestDeliveryCancelRestores` (cancel posted → on hand 10).
- [ ] **Step 2: Run** → FAIL.
- [ ] **Step 3: Implement.**
- [ ] **Step 4: Run** `go test ./internal/modules/sales ./internal/app` → PASS.
- [ ] **Step 5: Commit** `feat(sales): deliveries take goods out of stock`.

### Task 3: Order guard and concurrency

**Files:**
- Modify: `internal/modules/sales/service.go` (`transition`: orders into `cancelled`), `queries/delivery.sql`
- Test: `internal/modules/sales/delivery_test.go`

**Interfaces:**
- Produces: `ErrOrderHasDelivery` (`sales_order_has_delivery`, 409); the order's `allowed_actions` keeps `cancel` (the refusal happens at the transition, as quotations do with `ErrQuoteHasOrder`).

- [ ] **Step 1: Failing tests**: `TestOrderCancelRefusedWithDelivery` (a draft delivery exists → cancel order → `ErrOrderHasDelivery`; delete the draft → cancel succeeds); `TestConcurrentDeliverySaves` (line of 10; two deliveries of 6 created concurrently, each in its own `platform.InTx` → exactly one succeeds, the other `ErrDeliveryOverOrdered`); `TestCancelOrderVsCreateDelivery` (run both concurrently in their own `platform.InTx`; exactly one of: order cancelled and delivery refused `ErrOrderNotPosted`, or delivery created and cancel refused `ErrOrderHasDelivery`).
- [ ] **Step 2: Run** → FAIL.
- [ ] **Step 3: Implement**; reuse the `race` helper pattern from `internal/shared/stock/ledger_test.go`.
- [ ] **Step 4: Run** `go test -race ./internal/modules/sales -run 'Delivery|CancelOrder'` → PASS.
- [ ] **Step 5: Commit** `feat(sales): an order with deliveries cannot be cancelled`.

### Task 4: Derived progress and availability

**Files:**
- Create: `internal/modules/sales/delivery_list.go`, `internal/modules/sales/queries/availability.sql`
- Modify: `doc.go`, `doc_types.go` (order detail and list fields), `delivery_handler.go`
- Test: `internal/modules/sales/delivery_test.go`, `doc_list_test.go`

**Interfaces:**
- Produces:
  - `Line` (order lines) gains `Delivered *string` (`json:"delivered,omitempty"`: present for stock lines of orders) and `Available *string` (present only while the order is `draft` or `pending_approval`, for stock lines: available in the order's legal entity).
  - `Doc` gains `DeliveryState string` (`json:"delivery_state,omitempty" enum:"none,partial,full,not_applicable"`, orders only) and `Deliveries []DocRef` (non-cancelled deliveries the actor may view).
  - `DocFilter` gains `DeliveryState string` (orders list filter).
  - `Deliveries(ctx, DeliveryFilter{Q, Status, OrderID, WarehouseID, Sort, Paging}) (DeliveryList, error)` as a `data.List` over a `-- data:base` query scoped by `org_unit_id` through `iam.DataScope`; `GET /sales/deliveries`.
  - `Availability(ctx, legalEntity int64, items []int64) ([]ItemAvailability, error)` with `ItemAvailability{ItemID int64; OnHand, Committed, Available string}`; `GET /sales/availability?org_unit_id=&item_id=…` needing `sales.order.view` at that unit (legal entity = that unit's).

Committed SQL: for posted orders of the legal entity, per item, `sum(order line quantity) - sum(posted delivery line quantity for those order lines)`, stock items only; on hand = sum of `stock.balances` over warehouses of the legal entity.

- [ ] **Step 1: Failing tests**: `TestDeliveryStateDerived` (none → partial after delivering 4 of 10 → full after delivering the rest of every stock line; cancel one → partial; an order of only services → `not_applicable`); `TestAvailableOnDraftOrder` (on hand 10, posted order A commits 6, draft order B for 5 shows `available` "4.000"; posted order C has no `available`); `TestAvailabilityEndpoint` (on hand, committed, available per item); `TestDeliveriesList` (filter by order, scoped by org unit: a user of another branch sees none).
- [ ] **Step 2: Run** → FAIL.
- [ ] **Step 3: Implement.**
- [ ] **Step 4: Run** `go test ./internal/modules/sales ./internal/app` → PASS; `make gen` clean.
- [ ] **Step 5: Commit** `feat(sales): delivered, committed and available quantities`.

### Task 5: Delivery form options and lot errors

**Files:**
- Modify: `internal/shared/stock/ledger.go`, `ledger_types.go`, `queries/ledger.sql` (lot not found; serials on hand), `internal/modules/sales/delivery.go`, `delivery_handler.go`
- Test: `internal/shared/stock/ledger_test.go`, `internal/modules/sales/delivery_test.go`

**Interfaces:**
- Produces in `stock`:
  - `ErrLotNotFound` (`stock_lot_not_found`, 422, `line_no`) and `ErrSerialNotFound` (`stock_serial_not_found`, 422, `line_no`): an outgoing move naming a lot or serial that does not exist for the item is refused before any write (no lot is created for an out move).
  - `func (s *Service) SerialsOnHand(ctx context.Context, warehouse, item int64) ([]string, error)` (codes, sorted).
- Produces in `sales`: `DeliveryOptions(ctx, orderID int64, warehouseID int64) (DeliveryOptions, error)` under `sales.order.edit` at the order's unit: `Warehouses []WarehouseOption` (active, of the order's legal entity, default first), `Lines []DeliveryOptionLine{OrderPosition, ItemID, ItemCode, Description, Unit, Tracking, Ordered, Remaining string, Lots []stock.LotPick /* SuggestLots for Remaining, today in the tenant time zone */, Serials []string}`; route `GET /sales/orders/{id}/delivery-options?warehouse_id=` (warehouse optional: default warehouse when absent).

- [ ] **Step 1: Failing tests**: in stock `TestOutMoveUnknownLot` → `ErrLotNotFound` and no lot row created; `TestOutMoveUnknownSerial`; `TestSerialsOnHand`. In sales `TestDeliveryOptions` (remaining after a draft delivery, FEFO lots filled for the remaining quantity, serials listed, service lines absent, a user with only `sales.order.view` → forbidden).
- [ ] **Step 2: Run** → FAIL.
- [ ] **Step 3: Implement**; find the existing "today in tenant time zone" helper used for quotation expiry and reuse it.
- [ ] **Step 4: Run** `go test ./internal/shared/stock ./internal/modules/sales ./internal/app` → PASS.
- [ ] **Step 5: Commit** `feat(sales): delivery form options with lot suggestion`.

### Task 6: Stock card hides numbers of unreadable documents

**Files:**
- Modify: `internal/shared/stock/onhand.go`, `types.go`
- Test: `internal/shared/stock/onhand_test.go`, `internal/app/stock_test.go` (needs Sales deliveries)

**Interfaces:**
- Produces: `StockCardRow.Number` and `DocID` are blanked (`""`, `0`) unless `record.Service.Visible(ctx, ref, record.View)` succeeds for that document; `DocType` stays. Check visibility once per distinct document of the card.

- [ ] **Step 1: Failing test** in `internal/app/stock_test.go`: a `stock.viewer` without Sales roles reads a card containing a posted delivery and an adjustment → the delivery row has empty number and doc id 0, the adjustment row keeps its number; a Sales staff of that unit who also holds `stock.viewer` sees the delivery number.
- [ ] **Step 2: Run** → FAIL.
- [ ] **Step 3: Implement.**
- [ ] **Step 4: Run** `go test ./internal/shared/stock ./internal/app` → PASS.
- [ ] **Step 5: Commit** `fix(stock): stock card shows only readable document numbers`.

### Task 7: Docs

**Files:**
- Modify: `products/sales.md` (scope table: delivery moves from "Not yet"; record types table gains `sales.delivery`; statuses and actions; progress bullets for delivered, committed, available; entities table gains `sales.deliveries`, `sales.delivery_lines`), `CONTEXT.md` only if a term used in code is missing

- [ ] **Step 1:** Edit per the list, in the file's existing voice.
- [ ] **Step 2: Run** `make gen` → no diff.
- [ ] **Step 3: Commit** `docs(sales): deliveries`.

---

Next plan, after this lands: **part 3** screens (warehouses, on-hand, stock card, adjustments in the Sales area; delivery form from an order with lot suggestion; order delivery state and availability warning; item tracking fields; web i18n for every `stock_*`, `item_*` and new `sales_*` error code; warehouse update API without `legal_entity_id`).
