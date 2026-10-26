# Shared master data Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move partner identity, the item catalogue and VAT computation into `shared/` as first-class master data, with Sales keeping the customer role.

**Architecture:** `iam` already treats a role/permission prefix as a namespace (`core.` works today), so `shared/partner` and `shared/item` register roles under `partner` and `item`. The core change is in `record`: a type may be `Shared` (never product-gated, never a product with data) and `Anywhere` (a permission held at any org unit covers every record). Sales keeps `sales.customers` as the customer role (gains `partner_id`, loses identity columns), reads `partner.partners` and `item.items` in SQL, and calls `shared/tax` for VAT.

**Tech Stack:** Go, Huma, sqlc + data engine, Postgres 18, React + TanStack Query, dependency-cruiser.

**Spec:** `docs/superpowers/specs/2026-10-23-shared-master-data-design.md`

## Global Constraints

- Layering `modules → shared → core → platform`; `shared/partner`, `shared/item`, `shared/tax` never import or name `sales`.
- Data engine only (`store.<Query>.Run`, `data.List`), `platform.InTx`, one schema per module, writes only to the module's own schema.
- New migrations only (`make migration name=<module>_<description>`); never edit `20261023000000_all_baseline.sql`.
- Money `int64`; every rounding through `platform.Round`/`platform.Allocate`.
- Error codes stable, with params; user text only in `vi` and `en` i18n.
- Comments short, never cite docs/ADRs/milestones. Commits: Conventional Commits, scopes `record`, `partner`, `item`, `tax`, `sales`, `web`; end with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.
- Branch `feat/shared-master-data` off `main`; one commit per task.
- Do not run `make lint` or `make e2e` without asking the user first (user preference). `make gen` and `make test` are fine.
- Nobody loses an action they have today; existing quotations and orders read and print identically.

## Review Focus

1. Two customers in different org units with the same tax code (or the same code differing in case/whitespace) at migration time → one partner, both customers kept with their own codes. Test in Task 5 (migration test).
2. A user holding `sales.customer.edit` but not `partner.edit` saves a customer without touching identity → succeeds; changing the name → `forbidden`. Test in Task 5.
3. Sales disabled → customer writes refused (`product_not_enabled`), partner and item writes still succeed. Tests in Tasks 4 and 5.
4. Tenant administrator (`core.admin`) holds partner and item permissions with no extra grant. Test in Task 2.
5. A quotation's `items` approval field still resolves labels and options after items move. Test in Task 4.

---

### Task 1: ADR and glossary

**Files:**
- Create: `docs/adr/0039-shared-master-data.md`
- Modify: `CONTEXT.md` (Customer entries near line 123), `docs/adr/0027-sales-product.md` (one line under "Who owns customers and items": "Superseded by ADR-0039.")

- [ ] **Step 1: Write ADR-0039** (use `mattpocock-skills:domain-modeling`). Sections: Context (Odoo-like direction; ADR-0027 deferred promotion), Decision (the spec's Goals 1–8, "Deliberate behaviour changes" and the designer's calls, verbatim in substance), Consequences (each later product adds a role table in its own schema pointing to `partner.partners`; `shared/master` frontend folder; how two areas share `partner.partner` links is decided with Purchasing).
- [ ] **Step 2: Update `CONTEXT.md`.** Add **Partner** (`partner.partner`, vi "Đối tác"): a company or person known to the whole tenant; identity only. **Partner role** (vi "Vai trò đối tác"): a product's relation to a partner, holding that product's fields and ownership (customer today). Redefine **Customer** as Sales' partner role, one per partner per org unit, owning org unit and payment terms, `code` unchanged. Add **Partner code** (series `partner.partner`, prefix `DT`).
- [ ] **Step 3: Commit** `docs(adr): shared master data for partners, items and tax`.

### Task 2: Shared record types in core

**Files:**
- Modify: `internal/core/record/types.go` (`Type` struct), `internal/core/record/service.go` (`Register` ~L82, `ProductsWithData` ~L149, `answer` ~L203), Catalog visibility/list helpers that call `t.Locate` (grep `Locate(` in `internal/core/record`)
- Test: `internal/core/record/contract_test.go` or a new `internal/core/record/shared_test.go`; `internal/core/iam/service_test.go`

**Interfaces:**
- Produces: `record.Type.Shared bool` — "Product names a shared module's namespace (e.g. partner): never product-gated, never listed in products_with_data." `record.Type.Anywhere bool` — "Catalogs only: the permission held at any org unit covers every record; Locate only reports existence (return 0, nil)."

- [ ] **Step 1: Write failing tests** in `shared_test.go`, registering a fake namespace `"widget"` with roles `{viewer: [widget.view], editor: [widget.view, widget.edit]}` and a type `{Code: "widget.widget", Product: "widget", Kind: Catalog, Shared: true, Anywhere: true, Perms: {View: "widget.view", Edit: "widget.edit"}, Locate: func(...) (0, nil)}`:
  - `TestSharedTypeNotProductGated`: ctx with `platform.WithProducts(ctx, nil)`; user with `widget.editor` granted at some org unit; `Can(ctx, "widget.widget", 1, record.Edit)` is `true`.
  - `TestAnywhereGrantCoversEveryRecord`: user granted `widget.viewer` at unit A; `Can(..., View)` true while Locate returns 0; user without grants → false.
  - `TestSharedTypeNotProductWithData`: `HasData` returns true; `ProductsWithData` does not contain `"widget"`.
  - In iam tests, `TestAdminHoldsSharedNamespace`: `core.admin` user, `Scope(ctx, "widget", "widget.edit").All` is true.
- [ ] **Step 2: Run** `go test ./internal/core/record ./internal/core/iam -run 'Shared|Anywhere|AdminHoldsShared'` — expect FAIL (unknown fields).
- [ ] **Step 3: Implement.** Add the two fields. In `answer`: when `t.Anywhere`, check `IAM.Scope(ctx, t.Product, perm)` `.Any()` instead of `Allowed(..., unit)`; skip `ProductGate` when `t.Shared`. In `ProductsWithData`, skip shared types. `Register` keeps the `<product>.<type>` prefix rule; reject `Anywhere` on a `Document`. Any other `ProductGate(ctx, t.Product, …)` call in `record` (e.g. `WriteGate`, `TypeActions`) also skips shared types — grep `ProductGate` in `internal/core/record`.
- [ ] **Step 4: Run** the same command — PASS; then `go test ./internal/core/...` — PASS.
- [ ] **Step 5: Commit** `feat(record): shared and anywhere catalog types`.

### Task 3: `shared/tax`

**Files:**
- Create: `internal/shared/tax/service.go`, `internal/shared/tax/service_test.go`
- Modify: `internal/modules/sales/calc.go`, `internal/modules/sales/types.go` (drop `VatRates`), every `VatRates`/`vatRate` user in `internal/modules/sales`

**Interfaces:**
- Produces (package-level, no service, no tables):
  - `var Rates = []string{"none", "0", "5", "8", "10"}` — "none is not subject to VAT."
  - `func Valid(rate string) bool`
  - `func VAT(nets []int64, rates []string, mode platform.Rounding) []int64` — VAT of each line's net amount; `platform.RoundLine` rounds each line, `platform.RoundTotal` computes each rate over the document once and allocates it with `platform.Allocate`.

- [ ] **Step 1: Write failing tests** `TestVATPerLine` and `TestVATTotalAllocates`: move the VAT cases from `internal/modules/sales/calc_test.go` (same nets, rates and expected per-line VAT), plus `TestValid` (`"10"` true, `"7"` false).
- [ ] **Step 2: Run** `go test ./internal/shared/tax` — FAIL.
- [ ] **Step 3: Implement** by moving `vatRate` and the rounding branch of `compute`; `compute` builds `nets` (`amount − discount`) and calls `tax.VAT`; validation uses `tax.Valid`.
- [ ] **Step 4: Run** `go test ./internal/shared/tax ./internal/modules/sales` — PASS (sales calc tests unchanged in outcome).
- [ ] **Step 5: Commit** `refactor(tax): move VAT computation to shared/tax`.

### Task 4: `shared/item`

**Files:**
- Create: `migrations/<ts>_item_init.sql`, `internal/shared/item/{module.go,service.go,types.go,deps.go,handler.go,list.go,service_test.go}`, `internal/shared/item/queries/item.sql`
- Modify: `sqlc.yaml` (entry like posting's), `internal/app/app.go` (construct, pass to Sales deps, add `item.Module`), `internal/modules/sales/{item.go,item_handler.go,item_list.go,item_list_test.go,service.go,types.go,deps.go}`, `internal/modules/sales/queries/{item.sql,document.sql}`, `web/src/sales/pages/ItemsPage.tsx`, `web/src/sales/index.ts` (nav permission), `web/src/sales/hooks/useSales.ts`, `web/src/sales/components/pickers.tsx`, `web/src/core/i18n/{vi,en}/meta.json` (role labels `item.role.viewer`, `item.role.admin`), `web/src/sales/i18n/{vi,en}/meta.json` (drop `sales.role.catalog_admin`)

**Interfaces:**
- Consumes: Task 2 (none needed: items are not a record type).
- Produces: package `item`, `NewService(Deps{IAM, Audit, Numbering}) *Service`; `PermView = "item.view"` (anywhere), `PermManage = "item.manage"` (tenant-wide); roles under namespace `item`: `viewer: [item.view]`, `admin: [item.view, item.manage]` (admin tenant-wide); series `item.item` (prefix `HH`, product `""`); routes `GET /items`, `POST /items`, `PUT /items/{id}` (types `ItemFields`, `NewItem`, `Item`, `ItemFilter`, `ItemList` moved verbatim from Sales); `ErrCodeTaken` code `item_code_taken` on constraint `item.items_code_key`; audit action `item.item_saved`. Table `item.items` (same columns and ids as `sales.items`).

- [ ] **Step 1: Migration.** `CREATE SCHEMA item; ALTER TABLE sales.items SET SCHEMA item;` rename series `UPDATE numbering.series SET code = 'item.item' WHERE code = 'sales.item'`; grants: for every `iam.user_roles` row with `product='sales' AND role IN ('staff','manager','viewer','catalog_admin')` insert `('item','viewer', same org_unit_id)`; for `catalog_admin` also `('item','admin', NULL)`; `ON CONFLICT DO NOTHING`; then `DELETE ... WHERE product='sales' AND role='catalog_admin'`; `UPDATE iam.authz SET version = version + 1`.
- [ ] **Step 2: Write failing tests** in `internal/shared/item/service_test.go` (real Postgres via `pgtest`, as Sales tests do): `TestItemCRUD` (create with empty code gets `HH00001`, update, list filters `active`), `TestItemManageTenantWide` (manage granted at an org unit → `forbidden`), `TestItemWriteWithSalesDisabled` (`WithProducts(ctx, nil)` → create succeeds), `TestItemCodeTaken` (`item_code_taken`).
- [ ] **Step 3: Run** `go test ./internal/shared/item` — FAIL.
- [ ] **Step 4: Implement** the module (move code from `sales/item.go`, `item_list.go`, `item_handler.go`; list name `item.ItemList`; no `ProductGate`). In Sales: delete the moved code, `PermItemView`/`PermItemManage`, role `catalog_admin` and `PermItemView` from `staff`/`manager`/`viewer`; Sales queries read `item.items` (`ItemsByID` for document prepare and approval labels; `itemOptions` uses a Sales query over `item.items` or `item.Service` search — keep it in Sales since it is Sales' approval field); `AnyCatalog` counts customers only. Run `make gen`.
- [ ] **Step 5: Add a Sales test** `TestItemsApprovalFieldAfterMove`: create an item via `item.Service`, quotation with it; `itemLabels` and `itemOptions` return it.
- [ ] **Step 6: Frontend.** Items page and item picker call `/items` through `shared/api/core`; nav permission `item.view`; create/edit buttons on `item.manage` (no `useProductOn('sales')` gate for items).
- [ ] **Step 7: Run** `make gen && go test ./internal/shared/item ./internal/modules/sales ./internal/app && cd web && pnpm tsc --noEmit && pnpm vitest run` — PASS.
- [ ] **Step 8: Commit** `feat(item): item catalogue in shared/item`.

### Task 5: `shared/partner` and the customer role

**Files:**
- Create: `migrations/<ts>_partner_init.sql`, `internal/shared/partner/{module.go,service.go,types.go,deps.go,handler.go,list.go,service_test.go}`, `internal/shared/partner/queries/partner.sql`
- Modify: `sqlc.yaml`, `internal/app/app.go`, `internal/modules/sales/{customer.go,customer_handler.go,customer_list.go,customer_list_test.go,deps.go,service.go,doc.go,service_test.go}`, `internal/modules/sales/queries/{customer.sql,document.sql}`, `internal/app/sales_test.go`

**Interfaces:**
- Consumes: `record.Type{Shared, Anywhere}` (Task 2).
- Produces: package `partner`, `NewService(Deps{IAM, Record, Audit, Numbering}) *Service`.
  - Permissions `PermView = "partner.view"`, `PermEdit = "partner.edit"`, both "anywhere"; roles under `partner`: `viewer: [partner.view]`, `editor: [partner.view, partner.edit]`.
  - Record type `partner.partner`: `Kind: Catalog, Shared: true, Anywhere: true, Perms{View: PermView, Edit: PermEdit, ViewFiles: PermView, Attach: PermEdit}`.
  - Series `partner.partner`, prefix `DT`, product `""`.
  - `type Fields struct { Name string; TaxCode, Address, Phone, Email, ContactName *string }` (same tags/limits as today's `CustomerFields` identity fields).
  - `func (s *Service) Create(ctx, code string, f Fields) (int64, error)` — needs `partner.edit`; runs in the caller's tx if any (`platform.InTx` nests).
  - `func (s *Service) Update(ctx, id int64, f Fields) error` — needs `partner.edit`; no-op without the permission check when `f` equals the stored fields.
  - `func (s *Service) Get(ctx, id int64) (Partner, error)` — `Partner{ID, Code, Fields, AllowedActions}`.
  - `func (s *Service) List(ctx, Filter) (List, error)` — `Filter{Q, TaxCode, Sort, Paging}`; needs `partner.view` anywhere.
  - Routes `GET /partners`, `GET /partners/{id}`, `POST /partners`, `PUT /partners/{id}`.
  - `ErrTaxCodeTaken` = 409 `partner_tax_code_taken`, `Params{"partner_id": id}` (constraint `partner.partners_tax_code_key`, unique on `lower(btrim(tax_code))` where not null); `ErrCodeTaken` 409 `partner_code_taken`.
  - Audit `partner.partner_created`, `partner.partner_updated` via `RecordChanges` with `audit.Ref{Type: "partner.partner"}`.
- Sales customer API changes:
  - `CustomerFields{PaymentTerms *string; OrgUnitID int64; Active bool}`.
  - `NewCustomer{Code string; PartnerID *int64; Partner *partner.Fields; CustomerFields}` — exactly one of `PartnerID`/`Partner` (else 422 `customer_partner_required`).
  - `UpdateCustomer` body `{CustomerFields; Partner partner.Fields}`; calls `partner.Update` in the same tx.
  - `Customer{ID, Code, PartnerID, PartnerCode, Partner partner.Fields, CustomerFields, OrgUnitName, AllowedActions}`; list items keep flat `name`, `tax_code`, `phone` from the join; `CustomerFilter` gains `PartnerID *int64` (`query:"partner_id"`).
  - `ErrCustomerExists` 409 `customer_exists_for_partner` on constraint `sales.customers_partner_org_unit_key`.

- [ ] **Step 1: Migration.** `CREATE SCHEMA partner; CREATE TABLE partner.partners (id identity PK, code text NOT NULL, name text NOT NULL, tax_code, address, phone, email, contact_name text)`, unique `lower(code)` and `lower(btrim(tax_code)) WHERE tax_code IS NOT NULL`. Fill: one partner per distinct `lower(btrim(tax_code))`, identity from the customer with the highest id in the group (no `updated_at` exists; highest id is the stand-in for "most recent"); one partner per customer without tax code; codes `DT` + 5-digit sequence in customer id order; insert series `('partner.partner','DT', <count>)`. `ALTER TABLE sales.customers ADD partner_id bigint REFERENCES partner.partners`, fill, `SET NOT NULL`, drop identity columns, `UNIQUE (partner_id, org_unit_id)` named `customers_partner_org_unit_key`, index on `partner_id`. If two customers of one partner share an org unit, keep both by giving the later one its own partner (simulated data; document in the migration comment). Grants: `sales` `staff`/`manager` → `partner.editor`, `viewer` → `partner.viewer`, same org unit, `ON CONFLICT DO NOTHING`; bump `iam.authz`.
- [ ] **Step 2: Write failing tests** in `internal/shared/partner/service_test.go`: `TestPartnerCRUD`, `TestPartnerTaxCodeTaken` (`" 0101234567 "` vs `"0101234567"` → `partner_tax_code_taken` with `partner_id`), `TestPartnerVisibleAnywhere` (viewer granted at unit A reads a partner whose only customer is in unit B), `TestPartnerWriteWithSalesDisabled`, and the `recordtest` contract suite for `partner.partner`.
- [ ] **Step 3: Write failing Sales tests** in `service_test.go`: `TestCustomerOnExistingPartner` (two org units, one partner, two customers with own codes), `TestCustomerSamePartnerSameUnit` (`customer_exists_for_partner`), `TestCustomerEditWithoutPartnerEdit` (identity unchanged → OK; name changed → `forbidden`), `TestCustomerWriteSalesDisabled` (`product_not_enabled`), `TestDocumentCopiesPartnerIdentity` (submit copies name/tax code/address from the partner, code from the customer). Keep the existing `recordtest` run for `sales.customer` passing.
- [ ] **Step 4: Write the migration test** `TestPartnerMigration` in `internal/app/sales_test.go` (or a `migrations` test if one exists): on a database migrated to just before `<ts>_partner_init`, insert customers `A(unit1, "0101 ")`, `B(unit2, "0101")`, `C(unit1, NULL)`, apply the migration, assert two partners, A and B share one, codes unchanged. If the pgtest helper cannot stop before a migration, assert the same through the migration's SQL run on a scratch schema; say which in the commit.
- [ ] **Step 5: Run** `go test ./internal/shared/partner ./internal/modules/sales ./internal/app` — FAIL.
- [ ] **Step 6: Implement** the partner module, then Sales: `GetCustomer`, `CustomerListBase`, `CopyCustomer` and the document prepare join `partner.partners`; `CreateCustomer` creates or reuses the partner in its tx (partner code from the partner series, customer code from `sales.customer` as today); `UpdateCustomer` keeps its lock order (customer row, then partner via `partner.Update`); customer audit records role fields only. Wire in `app.go`. Run `make gen`.
- [ ] **Step 7: Run** `go test ./...` — PASS.
- [ ] **Step 8: Commit** `feat(partner): partner identity in shared/partner, customers as its Sales role`.

### Task 6: Frontend over partners

**Files:**
- Create: `web/src/shared/master/PartnerFields.tsx` (identity inputs for React Hook Form), `web/src/shared/master/PartnerPicker.tsx`, `web/src/shared/master/ItemPicker.tsx` (moved from `sales/components/pickers.tsx`), `web/src/shared/master/index.ts`, `web/src/sales/pages/PartnerPage.tsx`, lint fixture `web/lint-fixtures/<area>/…` importing `shared/api/sales` from `shared/master`
- Modify: `web/.dependency-cruiser.*` (rule `master-api-only-core`: from `shared/master/` to `shared/api/(?!core\.)` forbidden), `web/lint-fixtures/expected.json`, `web/src/sales/components/{CustomerForm.tsx,CustomerProfile.tsx,CustomerOverview.tsx}`, `web/src/sales/pages/{CustomerPage.tsx,CustomersPage.tsx}`, `web/src/sales/routes.tsx`, `web/src/sales/index.ts` (`recordTypes['partner.partner']: { path: id => \`partners/${id}\`, invalidate: () => [salesKeys.customers.all()] }`), `web/src/sales/keys.ts`, `web/src/sales/i18n/{vi,en}/*.json`, `web/src/core/i18n/{vi,en}/meta.json` (`partner.role.viewer`, `partner.role.editor`, error codes `partner_tax_code_taken`, `partner_code_taken`, `customer_exists_for_partner`, `customer_partner_required`, record type name `partner.partner`), `web/e2e/sales.seed.ts`, `web/e2e/m14.spec.ts` (body shapes only)

Use the `mmerp-ui` and `new-screen` skills; finish with the `ui.md` checklist.

- [ ] **Step 1: Write failing Vitest** for `CustomerForm`: on a `409 partner_tax_code_taken` with `partner_id`, the form shows the existing partner's name and a "use this partner" action; confirming resubmits with `partner_id` and no `partner`. Identity inputs are disabled when the partner's `allowed_actions` lacks `edit`.
- [ ] **Step 2: Run** `cd web && pnpm vitest run src/sales` — FAIL.
- [ ] **Step 3: Implement** the shared folder, form flow (new customer: pick existing partner or type identity; edit: one form, one save), `PartnerPage` (identity form, attachments, discussion, history of `partner.partner`, list of the actor's customers of this partner via `GET /sales/customers?partner_id=`), customer page link to its partner; dependency-cruiser rule and fixture.
- [ ] **Step 4: Run** `make gen && cd web && pnpm tsc --noEmit && pnpm vitest run` — PASS.
- [ ] **Step 5: Commit** `feat(web): Sales screens over shared partners and items`.

### Task 7: Docs and final verification

**Files:**
- Modify: `backend.md` (shared tier now hosts namespaces with roles and record types; "Promoting an entity to shared" points to role tables), `products/sales.md` (customers as a role, items and VAT in shared, permissions), `CLAUDE.md` project-status line only if it lists shared modules.

- [ ] **Step 1: Update the docs** listed above.
- [ ] **Step 2: Run** `make gen` (no diff) and `make test` — PASS.
- [ ] **Step 3: Ask the user** whether to run `make lint` and `make e2e`; run them only if they agree, and report results as they are.
- [ ] **Step 4: Commit** `docs: shared master data in backend and sales docs`.
