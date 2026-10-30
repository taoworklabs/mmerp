# Stock ledger (part 1 of 3) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `internal/shared/stock` (warehouses, lots, serials, moves, balances, stock adjustment) and the item tracking columns, backend only.

**Architecture:** A shared module in the style of `shared/posting` (ledger written once, voided by reversal) plus `shared/partner` (own namespace, `Shared` record type). Documents of any product call `stock.Record`/`stock.Void` in their `OnTransition`. Part 2 (Sales delivery) and part 3 (screens) get their own plans once this part's interfaces exist.

**Tech Stack:** Go, Huma, sqlc with the data engine, Postgres 18, `shopspring/decimal`.

**Spec:** `docs/adr/0049-shared-stock-ledger-and-sales-delivery.md`

## Global Constraints

- Layering: `stock` is in `internal/shared`; it imports `core` and `shared/item`, never `modules/*`, never names `sales`.
- Namespace `stock.`: permissions `stock.view` (Anywhere) and `stock.manage` (tenant-wide); roles `stock.viewer`, `stock.admin`. `ProductGate` never refuses it.
- Quantities are `numeric(15,3)` in SQL and `decimal.Decimal` in Go (strings in JSON, same pattern as Sales `LineInput.Quantity`). Never float.
- Moves are written once; cancelling writes a reversal, never an update or delete of a move.
- Lock order: advisory request lock → period lock → `record.documents` row → `stock.balances` rows `FOR UPDATE` ordered by (warehouse_id, item_id, lot_id, serial_id) → other module rows.
- Shortage error: `&platform.Error{Status: 409, Code: "stock_insufficient", Params: {warehouse_id, item_id, lot_id, serial_id, missing}}`. Codes follow the repo's snake_case style (`item_code_taken`).
- Migrations through `make migration name=<module>_<description>`; SQL only; every kind column has a `CHECK`.
- Adjustment: record type `stock.adjustment`, prefix `DC`, no approval fields; writing needs `stock.manage`.
- All user-facing text through i18n (`vi` and `en`); stable error codes, never sentences.

## Review Focus

- Two adjustments issuing the last unit of the same lot concurrently: exactly one posts, the other gets `stock_insufficient`. Test in Task 4.
- Cancelling an incoming adjustment whose goods were already issued: refused with `stock_insufficient` unless the item is `quantity` with `allow_negative`. Test in Task 5.
- A serial received twice (second receipt while on hand anywhere): refused. Test in Task 4.
- A move into a warehouse of another legal entity than the document's: refused with `stock_warehouse_entity`. Test in Task 4.
- Changing an item's `tracking` after it has a move: refused with `item_tracking_locked`. Test in Task 6.

---

### Task 1: Item tracking columns

**Files:**
- Create: `migrations/<ts>_item_stock_tracking.sql`
- Modify: `internal/shared/item/types.go`, `internal/shared/item/service.go`, `internal/shared/item/queries/item.sql`
- Test: `internal/shared/item/service_test.go`

**Interfaces:**
- Produces: `item.ItemFields.Tracking string` (`json:"tracking" enum:"none,quantity,lot,serial"`), `item.ItemFields.AllowNegative bool` (`json:"allow_negative"`); constants `item.TrackNone`, `TrackQuantity`, `TrackLot`, `TrackSerial`; `item.ErrNegativeNotQuantity` (code `item_negative_not_quantity`, 422).

- [ ] **Step 1: Failing test** `TestTrackingAndNegative`: create an item with `Tracking: "lot", AllowNegative: false` and read it back with both values; create one with `Tracking: "lot", AllowNegative: true` → `errors.Is(err, item.ErrNegativeNotQuantity)`; an existing-style item created with `Tracking: "none"` reads back `none`.
- [ ] **Step 2: Run** `go test ./internal/shared/item -run TestTrackingAndNegative` → FAIL (unknown field).
- [ ] **Step 3: Migration** `ALTER TABLE item.items ADD tracking text NOT NULL DEFAULT 'none' CHECK (tracking IN ('none','quantity','lot','serial')), ADD allow_negative boolean NOT NULL DEFAULT false, ADD CONSTRAINT items_negative_quantity CHECK (NOT allow_negative OR tracking = 'quantity');`. Register `item.items_negative_quantity` → `ErrNegativeNotQuantity` in `NewService`. Extend the create/update/list queries and the audit payload; `make gen`.
- [ ] **Step 4: Run** the test and `go test ./internal/shared/item ./internal/modules/sales` → PASS.
- [ ] **Step 5: Commit** `feat(item): stock tracking and negative stock per item`.

### Task 2: `stock` module, warehouses and default warehouse

**Files:**
- Create: `migrations/<ts>_stock_init.sql` (every `stock` table of this plan), `internal/shared/stock/{service.go,types.go,deps.go,module.go,handler.go,warehouse.go,queries/warehouse.sql}`, `internal/shared/stock/warehouse_test.go`
- Modify: `sqlc.yaml` (entry like `item`), `internal/app/app.go`, `internal/shared/stock/i18n` if errors need labels

**Interfaces:**
- Consumes: `iam.Service.RegisterRoles`, `iam.Service.Scope`, `iam.Service.LegalEntityOf`, `numbering.Service.RegisterSeries`, `record.Service.Register`.
- Produces:
  - `type Deps struct { IAM *iam.Service; Record *record.Service; Audit *audit.Service; Numbering *numbering.Service }`
  - `func NewService(d Deps) *Service`; `func Module(s *Service) platform.Module` (Name `stock`, no Product)
  - `type WarehouseFields struct { Name string; LegalEntityID int64; Active bool }`, `type Warehouse struct { ID int64; Code string; IsDefault bool; WarehouseFields }`
  - `func (s *Service) Warehouses(ctx, f WarehouseFilter) (WarehouseList, error)` (needs `stock.view` anywhere)
  - `func (s *Service) CreateWarehouse(ctx, in NewWarehouse) (int64, error)`, `UpdateWarehouse(ctx, id int64, in WarehouseFields) error` (need `stock.manage` tenant-wide; code series `stock.warehouse`, prefix `KHO`)
  - `func (s *Service) DefaultWarehouse(ctx context.Context, legalEntity int64) (int64, error)`: returns the default warehouse, creating it (name from i18n `stock.warehouse.default_name`) inside the caller's transaction when missing; requires a tx in ctx.
  - Routes `GET/POST /stock/warehouses`, `PUT /stock/warehouses/{id}` (core client side).

Tables in the migration: `stock.warehouses (id, code, name, legal_entity_id → iam.org_units, is_default, active)` with a unique partial index on `legal_entity_id WHERE is_default` and unique `lower(code)`; `stock.lots (id, item_id, code, expiry date NULL, received_at timestamptz)` unique `(item_id, lower(code))`; `stock.serials (id, item_id, code)` unique `(item_id, lower(code))`; `stock.moves (id, doc_type, doc_id, line_no, warehouse_id, item_id, lot_id NULL, serial_id NULL, quantity numeric(15,3) <> 0, date, reverses bigint NULL → stock.moves)`; `stock.balances (warehouse_id, item_id, lot_id, serial_id, quantity numeric(15,3))` with a unique index on `(warehouse_id, item_id, coalesce(lot_id,0), coalesce(serial_id,0))`; `stock.adjustments (id → record.documents deferred, warehouse_id, reason)`, `stock.adjustment_lines (adjustment_id, line_no, item_id, lot_code, lot_expiry, serial_code, quantity numeric(15,3) <> 0)`. The migration inserts one default warehouse per `iam.org_units` row of kind `company`.

- [ ] **Step 1: Failing tests** in `warehouse_test.go`: `TestDefaultWarehouseBackfilledAndLazy`: a company created before `NewService` has no default until `DefaultWarehouse` runs in `platform.InTx`; calling twice returns the same id; `TestWarehousePermissions`: a `stock.viewer` lists, gets `platform.ErrForbidden` on create; a `stock.admin` creates `KHO00001`; a second default for the same legal entity is impossible (only one `is_default`).
- [ ] **Step 2: Run** → FAIL (package missing).
- [ ] **Step 3: Implement** the module shape (as `shared/item`), the migration, sqlc entry, roles, series, wiring in `app.go` after `item`; depguard needs nothing new (shared rule covers it).
- [ ] **Step 4: Run** `go test ./internal/shared/stock ./internal/app` and `make gen` → PASS, no diff after gen.
- [ ] **Step 5: Commit** `feat(stock): warehouses with a default per legal entity`.

### Task 3: Balances check, `Record` and lot suggestion

**Files:**
- Create: `internal/shared/stock/ledger.go`, `internal/shared/stock/ledger_types.go`, `internal/shared/stock/queries/ledger.sql`, `internal/shared/stock/ledger_test.go`

**Interfaces:**
- Produces:
  - `type Move struct { LineNo int; WarehouseID, ItemID int64; Lot *LotRef; Serial *string; Quantity decimal.Decimal }` (positive in, negative out)
  - `type LotRef struct { Code string; Expiry *string }` (an incoming move creates the lot when missing; an outgoing one must name an existing lot)
  - `func (s *Service) Record(ctx context.Context, ref record.Ref, moves []Move) error`
  - `func (s *Service) SuggestLots(ctx context.Context, warehouse, item int64, qty decimal.Decimal, today string) ([]LotPick, error)`, `type LotPick struct { LotID int64; Code string; Expiry *string; Quantity decimal.Decimal }`
  - Errors: `ErrInsufficient` (`stock_insufficient`, 409), `ErrWarehouseEntity` (`stock_warehouse_entity`, 422), `ErrUntracked` (`stock_untracked`, 422: item `tracking = none`), `ErrLotRequired` / `ErrSerialRequired` (422), `ErrSerialQuantity` (`stock_serial_quantity`: serial moves are ±1), `ErrAlreadyRecorded` (409).

`Record` algorithm: read the document's legal entity and date from `record.documents`; refuse when any move's warehouse is of another legal entity; resolve lots and serials per item tracking; lock balance rows `FOR UPDATE` in (warehouse, item, lot, serial) order, inserting missing rows first with `ON CONFLICT DO NOTHING`; apply deltas; refuse when a result is below zero unless the item is `quantity` with `allow_negative`; a serial's total on hand across warehouses must stay 0 or 1; insert the moves. `SuggestLots` orders on-hand lots by `expiry NULLS LAST, received_at, id`, skipping `expiry < today`, and takes from each until `qty` is covered (the last pick partial); it returns what it found even when short.

- [ ] **Step 1: Failing tests** (the test registers `record.Type{Code: "stock.testdoc", Product: "stock", Shared: true, Kind: record.Document, NumberPrefix: "TT", Perms: {View: stock.view, Edit: stock.manage}, OnDelete: no-op}` and creates documents of it with `record.Create` in `platform.InTx`; Tasks 4 to 6 reuse this helper): `TestRecordInAndOut` (in 10 of lot L1, out 4 → balance 6); `TestRecordRefusesShortage` (out 7 of 6 → `ErrInsufficient` with `missing = 1`); `TestNegativeAllowedForQuantityItem` (item `quantity`, `allow_negative` → out 3 of 0 → balance −3); `TestSerialOnHandOnce` (receive S1 twice → `ErrInsufficient`/`ErrSerialQuantity`); `TestWarehouseOfOtherEntity` → `ErrWarehouseEntity`; `TestSuggestLotsFEFO` (lots A exp 2026-12-01 qty 2, B exp 2026-11-15 qty 2, C no expiry qty 5, D expired qty 9; ask 5 today 2026-11-01 → B 2, A 2, C 1).
- [ ] **Step 2: Run** `go test ./internal/shared/stock -run 'TestRecord|TestNegative|TestSerial|TestWarehouseOf|TestSuggest'` → FAIL.
- [ ] **Step 3: Implement** `ledger.go` per the algorithm above; balance updates in one `unnest` statement.
- [ ] **Step 4: Run** → PASS.
- [ ] **Step 5: Commit** `feat(stock): record stock moves under balance locks`.

### Task 4: Concurrency of `Record`

**Files:**
- Test: `internal/shared/stock/ledger_test.go`

- [ ] **Step 1: Failing test** `TestConcurrentIssueOfLastUnit`: on hand 1 of lot L1; two goroutines each in their own `platform.InTx` record out 1; exactly one returns nil, the other `ErrInsufficient`; final balance 0. `TestOppositeOrderNoDeadlock`: two documents moving items X then Y and Y then X concurrently both succeed within 5 s.
- [ ] **Step 2: Run** → FAIL if Task 3 locked out of order or checked before locking; otherwise passes immediately, which is acceptable (the test pins the property).
- [ ] **Step 3: Fix** lock ordering if needed.
- [ ] **Step 4: Run** `go test -race ./internal/shared/stock` → PASS.
- [ ] **Step 5: Commit** `test(stock): concurrent issues of the same balance`.

### Task 5: `Void`

**Files:**
- Modify: `internal/shared/stock/ledger.go`, `queries/ledger.sql`
- Test: `internal/shared/stock/ledger_test.go`

**Interfaces:**
- Produces: `func (s *Service) Void(ctx context.Context, ref record.Ref) error`: inserts one reversing move per non-reversed move of `ref` (`reverses` set), through the same lock-and-check path as `Record`.

- [ ] **Step 1: Failing tests** `TestVoidRestores` (out 4 then void → balance back to 10, moves table has 2 rows for that doc pair); `TestVoidRefusedWhenGoodsGone` (doc A in 5 of L1, doc B out 5, void A → `ErrInsufficient`); `TestVoidTwiceNoop` (second void inserts nothing).
- [ ] **Step 2: Run** → FAIL.
- [ ] **Step 3: Implement.**
- [ ] **Step 4: Run** → PASS.
- [ ] **Step 5: Commit** `feat(stock): void stock moves by reversal`.

### Task 6: Tracking locked once moved

**Files:**
- Modify: `internal/shared/item/queries/item.sql`, `internal/shared/item/service.go`, `internal/shared/item/types.go`
- Test: `internal/shared/item/service_test.go` (needs a stock move: insert one row into `stock.moves` through the system pool; `item` tests must not import `stock`)

**Interfaces:**
- Produces: `item.ErrTrackingLocked` (`item_tracking_locked`, 409).

- [ ] **Step 1: Failing test** `TestTrackingLockedOnceMoved`: item with one move; update with a different `Tracking` → `ErrTrackingLocked`; update of the name alone succeeds.
- [ ] **Step 2: Run** → FAIL.
- [ ] **Step 3: Implement** in the update query: `EXISTS (SELECT 1 FROM stock.moves WHERE item_id = $id)` compared against a tracking change (same tier join, allowed).
- [ ] **Step 4: Run** `go test ./internal/shared/item` and `make lint`'s query scan (`node scripts/check-queries.mjs`) → PASS.
- [ ] **Step 5: Commit** `feat(item): lock stock tracking once an item has moved`.

### Task 7: Stock adjustment document

**Files:**
- Create: `internal/shared/stock/adjustment.go`, `adjustment_types.go`, `adjustment_list.go`, `adjustment_handler.go`, `queries/adjustment.sql`, `adjustment_test.go`
- Modify: `internal/shared/stock/service.go` (register type), `internal/shared/stock/i18n/{vi,en}.json` (print labels)

**Interfaces:**
- Consumes: `Record`, `Void`, `DefaultWarehouse`.
- Produces:
  - `type AdjustmentLine struct { ItemID int64; Lot *LotRef; Serial *string; Quantity string /* signed, up to 3 decimals */ }`
  - `type AdjustmentFields struct { Date string; WarehouseID int64; Reason string /* 1..500 */; Lines []AdjustmentLine /* 1..500 */ }`
  - `CreateAdjustment(ctx, NewAdjustment{RequestID; AdjustmentFields}) (int64, error)`, `UpdateAdjustment(ctx, id, AdjustmentUpdate{Version; AdjustmentFields}) error`, `Adjustment(ctx, id) (Adjustment, error)` (with `allowed_actions`), `Adjustments(ctx, AdjustmentFilter) (AdjustmentList, error)`
  - Record type: `{Code: "stock.adjustment", Product: "stock", Shared: true, Kind: record.Document, NumberPrefix: "DC", Perms: {View: stock.view, Edit: stock.manage, ViewFiles: stock.view, Attach: stock.manage}, OnTransition: posted → Record, cancelled → Void, OnDelete}`. Header org unit = the warehouse's legal entity.
  - Routes `/stock/adjustments`, `/stock/adjustments/{id}`; a print template like Sales' `registerPrints`.

- [ ] **Step 1: Failing tests**: `recordtest.Run` contract for `stock.adjustment` (harness as in `modules/sales/service_test.go:contract`, `Table: "stock.adjustments"`, `Effects` = count of non-reversed moves; `BreakPosting` = make an outgoing line exceed on hand); `TestAdjustmentCreatesLotsInline` (posting an incoming line with lot `L9`, expiry `2027-01-31` creates the lot and balance); `TestAdjustmentNeedsManage` (`stock.viewer` create → `platform.ErrForbidden`).
- [ ] **Step 2: Run** `go test ./internal/shared/stock -run 'Adjustment|Contract'` → FAIL.
- [ ] **Step 3: Implement** following the `new-document-type` skill steps 2 to 5; lines written in one `unnest`.
- [ ] **Step 4: Run** `go test ./internal/shared/stock ./internal/app` → PASS; `make gen` clean.
- [ ] **Step 5: Commit** `feat(stock): stock adjustment document`.

### Task 8: On-hand read API

**Files:**
- Create: `internal/shared/stock/onhand.go`, `onhand_list.go`, `queries/onhand.sql`, `onhand_test.go`
- Modify: `internal/shared/stock/handler.go`

**Interfaces:**
- Produces: `OnHand(ctx, OnHandFilter{WarehouseID, ItemID *int64; Q string; ExpiringBefore *string; Paging}) (OnHandList, error)` rows `{WarehouseID, WarehouseCode, ItemID, ItemCode, ItemName, Unit, LotCode, Expiry, SerialCode, Quantity string}`, zero balances hidden; `StockCard(ctx, warehouse, item int64, from, to string) (StockCard, error)`: opening quantity at `from`, moves in date order with document number and running balance, closing quantity. Routes `GET /stock/on-hand`, `GET /stock/card`. Both need `stock.view`.

- [ ] **Step 1: Failing tests** `TestOnHandHidesZero`, `TestStockCardRunningBalance` (in 10 on 11-01, out 4 on 11-05, void of the out on 11-06; card 11-02..11-30 → opening 10, rows −4 (6), +4 (10), closing 10).
- [ ] **Step 2: Run** → FAIL.
- [ ] **Step 3: Implement** as `data.List` over `-- data:base` queries.
- [ ] **Step 4: Run** `go test ./internal/shared/stock` → PASS.
- [ ] **Step 5: Commit** `feat(stock): on-hand and stock card`.

### Task 9: Docs

**Files:**
- Modify: `docs/backend.md` (lock order line adds `stock.balances`), `AGENTS.md` (same lock-order sentence), `docs/architecture.md` (shared modules list)

- [ ] **Step 1:** Edit the three lock-order mentions to: "… → document row → `stock.balances` rows (`FOR UPDATE`, by warehouse, item, lot, serial) → module-owned rows".
- [ ] **Step 2: Run** `make gen && go test ./...` → PASS, no gen diff.
- [ ] **Step 3: Commit** `docs(stock): lock order and module list`.

---

Next plans, written once this one lands: **part 2** Sales delivery (`sales.delivery`, `GH`, derived delivered/committed/available, order cancel guard, warehouse/lot/serial options under Sales permissions); **part 3** screens (warehouses, on-hand, stock card, adjustments in the Sales area; delivery form with lot suggestion; item form tracking fields).
