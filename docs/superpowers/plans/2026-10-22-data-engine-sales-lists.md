# Data Engine — Sales Lists Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move the Sales customer, item, and quotation/order lists onto the data engine, each proven identical to its old query.

**Architecture:** Same pattern as `internal/modules/hrm/list_leave.go`:
- a parameterless `-- data:base` query in `internal/modules/sales/queries.sql`;
- a `data.Register`ed `List` in `list_<name>.go`;
- a `<name>Query(…) data.Query` builder;
- the service calls `IAM.DataScope` and then `List.Run`.

Each list gets an old-versus-new comparison test before its old query is deleted. The engine does not change.

**Tech Stack:** Go, pgx v5, sqlc v1.30.0 plus the `sqlc-gen-data` plugin, Postgres 18.

**Spec:** `docs/superpowers/specs/2026-10-21-data-engine-design.md` (§2 Dynamic lists, §Behaviour changes). Reference implementations: `internal/modules/hrm/list_leave.go`, `list_payroll.go`.

## Global Constraints

- No engine changes, no migrations, no indexes. If a list needs one, stop and report.
- **No performance measurement** (product-owner decision, 2026-10-22). Correctness is proven only by the comparison tests.
- **Behaviour must be identical to the old queries**, except two deliberate changes:
  - search escapes `%`, `_` and `\` (Sales already trims `q`);
  - a page past the end returns the true total.
- Never mock the database. The app pool is read-only outside `platform.InTx`; seed through a writable pool (`pgtest.System`) or through the service inside `InTx`.
- Commits follow Conventional Commits, stage explicit paths, and end with `Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>`.
- Comments are short and say why. They never reference specs, plans, tasks or commits.
- Branch `feat/data-engine-sales-lists`.
- `make gen` offline: `GOPROXY=file://$(go env GOMODCACHE)/cache/download make gen`.

## Review Focus

1. **A user with no customer permission anywhere** gets an empty customer, quote or order list, not an error. A user with no item permission gets `ErrForbidden` from `Items`, as today. Pinned by the comparison matrix (no-role actor) and the existing `TestItems`.
2. **`itemOptions`** (the approval-rule item picker) still returns retired items, at most 20, ordered by `lower(code)` then `id`. Pinned in Task 2 (`TestItemOptionsUnchanged`).
3. **The `ordered` flag** turns false again when the only order made from a quotation is cancelled. Pinned in Task 3 (`TestQuoteListOrderedFlag`).
4. **Orders and quotations never leak into each other's list.** Pinned in Task 3 (`TestDocListKindsSeparate`).
5. **Search `%`** matches literally on all three lists. Pinned by one `…SearchLiteralPercent` test per task.

---

### Task 1: Customer list on the engine

**Files:**
- Modify: `internal/modules/sales/queries.sql`. Add `CustomerListBase`; later delete `ListCustomers`.
- Create: `internal/modules/sales/list_customer.go`
- Create: `internal/modules/sales/list_customer_test.go`
- Modify: `internal/modules/sales/service_customer.go:37-53` (`Customers`)
- Regenerate: `make gen`

**Interfaces:**
- Consumes:
  - `iam.(*Service).DataScope(ctx, "sales", PermCustomerView) (data.Scope, error)`;
  - `boolFilter(v string) pgtype.Bool` (`service_customer.go:159`, unchanged; Task 2 uses it too).
- Produces:
  - `var customerList = data.Register(data.List[store.CustomerListBaseRow]{…})`;
  - `func customerQuery(sc data.Scope, f CustomerFilter) data.Query`.

- [ ] **Step 1: Write the base query**, then run `make gen`:

  ```sql
  -- name: CustomerListBase :many
  -- data:base
  SELECT c.id, c.code, c.name, c.tax_code, c.phone, c.payment_terms, c.org_unit_id, u.name AS org_unit_name, c.active
  FROM sales.customers c JOIN iam.org_units u ON u.id = c.org_unit_id;
  ```

- [ ] **Step 2: Write the failing tests** in `list_customer_test.go`:
  - `TestCustomerListMatchesOld`. While `ListCustomers` still exists, compare the old sqlc query with `Customers()` (which will call the engine). Matrix:
    - **Actors:** admin, a team-scoped staff member, a user with no role.
    - **Filters:**
      - none;
      - `q` matching code, name or tax code;
      - `q` with mixed case and diacritics;
      - `active` true / false / empty;
      - combinations of the above.
    - **Sorts:** all 4 (`code`, `-code`, `name`, `-name`).
    - **Pages:** 1 and 2 (size 20), plus a page past the end.
    - **Seed:** at least 45 customers across two teams, with mixed-case codes and repeated names.

    Assert the same ids in the same order. Assert the same `Total`, except a page past the end, where new = true count and old = 0.
  - `TestCustomerListSearchLiteralPercent`: `q = "%"` gives `Total == 0` when no code, name or tax code contains `%`.

  Run `go test ./internal/modules/sales -run 'CustomerList' -count=1`. Expected: FAIL. The `%` test fails on the old path.
- [ ] **Step 3: Implement `list_customer.go`:**
  - `Name: "sales.CustomerList"`
  - `Scope: data.ByColumn("org_unit_id")`
  - `Search: []string{"code", "name", "tax_code"}`
  - `Sorts`:
    - `code` → `Asc("code").Lower()`
    - `-code` → `Desc("code").Lower()`
    - `name` → `Asc("name")`
    - `-name` → `Desc("name")`
  - `Default: []data.Order{data.Asc("code").Lower()}`
  - `Tiebreak: data.Asc("id")`

  `customerQuery`'s `Where` is:

  ```go
  a := boolFilter(f.Active)
  data.When(a.Valid, data.Eq("active", a.Bool))
  ```

  Switch `Customers` to `DataScope` + `customerList.Run` and keep the row mapping. Drop the explicit `strings.TrimSpace(f.Q)`, because the engine trims. Keep it if it is used elsewhere.
- [ ] **Step 4:** Run `go test ./internal/modules/sales -count=1`. Expected: PASS.
- [ ] **Step 5: Remove the old path.**
  - Delete `ListCustomers` and run `make gen`.
  - Delete `TestCustomerListMatchesOld`. Paste its last passing output into the task report.
  - Run `make lint && go test ./internal/modules/sales/... ./internal/app/... -count=1`. Expected: PASS.
- [ ] **Step 6: Commit** `refactor(sales): customer list on the data engine`.

### Task 2: Item list on the engine

**Files:**
- Modify: `internal/modules/sales/queries.sql`. Add `ItemListBase`; later delete `ListItems`.
- Create: `internal/modules/sales/list_item.go`
- Create: `internal/modules/sales/list_item_test.go`
- Modify: `internal/modules/sales/service_item.go`:
  - `Items`, lines 16-33;
  - `itemOptions`, around line 97.
- Regenerate: `make gen`

**Interfaces:**
- Consumes:
  - `boolFilter`;
  - `s.d.IAM.Scope(ctx, "sales", PermItemView)`. This is the access check only, kept as today: `!sc.Any()` → `platform.ErrForbidden`.
- Produces:
  - `var itemList = data.Register(data.List[store.ItemListBaseRow]{…})`;
  - `func itemQuery(f ItemFilter) data.Query`.

- [ ] **Step 1: Write the base query.** It lists `sales.items`' columns explicitly, the same columns `SELECT *` returns today, so the row type keeps its fields. Run `make gen`.

  ```sql
  -- name: ItemListBase :many
  -- data:base
  SELECT <every column of sales.items, in table order> FROM sales.items;
  ```

  If any column type differs from the old `ListItemsRow`, record it in the report and adapt the mapping.
- [ ] **Step 2: Write the failing tests** in `list_item_test.go`:
  - `TestItemListMatchesOld`. Compare the old query with `Items()`. Matrix:
    - **Actors:** a viewer, a scoped viewer from another team (the catalogue is tenant-wide).
    - **Filters:** none, `q` on code / name, `active` true / false / empty.
    - **Sorts:** all 4, plus empty `Sort`. Empty must equal `code`.
    - **Pages:** 1 and 2, plus a page past the end.
    - **Seed:** at least 45 items, some retired.
  - `TestItemOptionsUnchanged`. Compare `itemOptions(ctx, q)` before and after: the same option ids, in the same order, for:
    - `q = ""`;
    - a code fragment;
    - a fragment matching only a retired item.

    Write it against the current code first, so it passes, and keep it after the switch.
  - `TestItemListSearchLiteralPercent`.

  Run `go test ./internal/modules/sales -run 'ItemList|ItemOptions' -count=1`. Expected: the comparison test fails to compile, or the `%` test fails.
- [ ] **Step 3: Implement `list_item.go`:**
  - `Name: "sales.ItemList"`
  - `Scope: data.Unscoped("tenant-wide catalogue")`
  - `Search: []string{"code", "name"}`
  - `Sorts`: the same 4 keys as customers
  - `Default: []data.Order{data.Asc("code").Lower()}`
  - `Tiebreak: data.Asc("id")`

  Wire `Items` and `itemOptions` as follows:
  - **`Items`**: keep the `PermItemView` `sc.Any()` check, then run `itemList.Run(ctx, itemQuery(f))`. An unscoped list ignores `Query.Scope`.
  - **`itemOptions`**: run `itemList.Run` with `data.Query{Q: q, Paging: platform.Paging{Page: 1, PageSize: 20}}`. Set no `active` condition, so retired items stay included. Map `Items` to options as today.
- [ ] **Step 4:** Run `go test ./internal/modules/sales -count=1`. Expected: PASS.
- [ ] **Step 5: Remove the old path.**
  - Delete `ListItems` and run `make gen`.
  - Delete `TestItemListMatchesOld`. Keep `TestItemOptionsUnchanged`.
  - Run `make lint && go test ./internal/modules/sales/... ./internal/app/... -count=1`.
- [ ] **Step 6: Commit** `refactor(sales): item list on the data engine`.

### Task 3: Quotation and order lists on the engine

**Files:**
- Modify: `internal/modules/sales/queries.sql`. Add `DocListBase`; later delete `ListHeaders`.
- Create: `internal/modules/sales/list_doc.go`
- Create: `internal/modules/sales/list_doc_test.go`
- Modify: `internal/modules/sales/service_doc.go:57-82` (`docs`)
- Regenerate: `make gen`

**Interfaces:**
- Consumes:
  - `kind{name, docType, prefix, view, edit}` (`service.go:30-36`);
  - `iam.(*Service).DataScope(ctx, "sales", k.view)`;
  - `expired(validUntil, today)`;
  - `s.d.Setting.Today(ctx)`.
- Produces:
  - `var docList = data.Register(data.List[store.DocListBaseRow]{…})`;
  - `func docQuery(sc data.Scope, k kind, f DocFilter) data.Query`.

- [ ] **Step 1: Write the base query.** It is `ListHeaders`'s select list without the window count, plus `h.kind` and `d.org_unit_id` so the engine can filter on them. Then run `make gen`.

  ```sql
  -- name: DocListBase :many
  -- data:base
  SELECT h.id, h.kind, d.number, d.status, d.date, d.org_unit_id, h.customer_id, h.customer_code, h.customer_name,
         h.valid_until, h.total, u.name AS org_unit_name, (h.quote_id IS NOT NULL)::bool AS from_quote,
         EXISTS (
             SELECT 1 FROM sales.headers oh JOIN record.documents od ON od.id = oh.id
             WHERE oh.quote_id = h.id AND od.status <> 'cancelled'
         ) AS ordered
  FROM sales.headers h
  JOIN record.documents d ON d.id = h.id
  JOIN iam.org_units u ON u.id = d.org_unit_id;
  ```

- [ ] **Step 2: Write the failing tests** in `list_doc_test.go`:
  - `TestDocListMatchesOld`. Compare the old query with `Quotes()` / `Orders()`. Matrix:
    - **Kinds:** quote, order.
    - **Actors:** admin, team staff, other-team staff, no role.
    - **Filters:**
      - none;
      - each `status`;
      - `customer_id`;
      - `q` on number / customer code / customer name;
      - combinations of the above.
    - **Sorts:** all 6 (`date`, `-date`, `number`, `-number`, `total`, `-total`).
    - **Pages:** 1 and 2, plus a page past the end.
    - **Seed:** enough documents of both kinds for 2+ pages, with equal dates and equal totals so the `id DESC` tiebreak matters, at least one order from a quotation, and one cancelled order from another quotation.

    Seed through the service, or by SQL through the writable pool as `hrm/list_payroll_test.go` does. Say which in the report.
  - `TestQuoteListOrderedFlag`:
    - a quotation with a live order shows `Ordered = true`;
    - after that order is cancelled, it shows `Ordered = false`;
    - an order made from a quotation shows `FromQuote = true`.
  - `TestDocListKindsSeparate`: `Quotes()` never returns an order id, and `Orders()` never returns a quotation id.
  - `TestDocListSearchLiteralPercent`.

  Run `go test ./internal/modules/sales -run 'DocList|QuoteList' -count=1`. Expected: FAIL.
- [ ] **Step 3: Implement `list_doc.go`:**
  - `Name: "sales.DocList"`
  - `Scope: data.ByColumn("org_unit_id")`
  - `Search: []string{"number", "customer_code", "customer_name"}`
  - `Sorts`:
    - `date` → `Asc("date")`
    - `-date` → `Desc("date")`
    - `number` → `Asc("number")`
    - `-number` → `Desc("number")`
    - `total` → `Asc("total")`
    - `-total` → `Desc("total")`
  - `Default: []data.Order{data.Desc("date")}`
  - `Tiebreak: data.Desc("id")`

  `docQuery`'s `Where`:
  - `Eq("kind", k.name)`, always present;
  - `When(f.Status != "", Eq("status", f.Status))`;
  - `When(f.CustomerID != 0, Eq("customer_id", f.CustomerID))`.

  Switch `docs` to `DataScope(ctx, "sales", k.view)` + `docList.Run`. Map `FromQuote`, `Ordered`, `ValidUntil` and `Expired` as today. `Total` comes from `Page.Total`.
- [ ] **Step 4:** Run `go test ./internal/modules/sales -count=1`. Expected: PASS.
- [ ] **Step 5: Remove the old path.**
  - Delete `ListHeaders` and run `make gen`.
  - Delete `TestDocListMatchesOld`, keeping its output in the report.
  - Run `make lint && make test`. Expected: PASS.
- [ ] **Step 6: Commit** `refactor(sales): quotation and order lists on the data engine`.
