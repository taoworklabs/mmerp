# Employee profiles from Excel Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** HR exports the employee list to Excel, edits it, and imports it back; the same import creates new employees and edits existing ones.

**Architecture:** Two `dataio` kinds registered by `hrm`, both `hrm.employees`: an export (`Run` returning one `dataio.Sheet`) and an import (`Header` + `Run`), in a new file `internal/modules/hrm/employee_io.go`. The import parses every row, merges non-empty cells over the current profile, then writes through the same `prepare` checks the form uses; manager cycles are checked once on the file's end state. The employee list gets an export button and an import dialog.

**Tech Stack:** Go, sqlc, `internal/core/dataio` (excelize underneath), `platform/data` lists, React + TanStack Query, `@/shared/jobs` (`ExportButton`, `ImportDialog`), Playwright.

**Spec:** `docs/superpowers/specs/2026-10-28-employee-import-design.md`

## Global Constraints

- Branch `feat/employee-import` off `main`; one commit per task, Conventional Commits with scope `hrm` (or `web` for the UI task).
- Services query only through `store.<Query>.Run(ctx, …)` or `data.List`; writes inside `platform.InTx`; never pass a tx.
- User-facing text only in i18n files, both `vi` and `en`. Backend row-error codes are translated as `hrm.import.<code>` (`internal/modules/hrm/i18n/{vi,en}.json`).
- Sensitive values never in logs or audit data.
- Tests use the real Postgres fixtures (`newFixture` in `service_test.go`, `newJobsFixture` in `internal/app/jobs_test.go`); never mock the DB.
- Run Go tests with `make db` first, then `go test ./internal/modules/hrm -run <Name>`. Do not run `make lint` or `make e2e` unless the user asks; tell them which to run.
- Comments short, English or Vietnamese, never referencing docs/ADRs.

## Review Focus

1. A file exported by someone without `hrm.employee.sensitive` and re-imported by someone with it: sensitive values stay exactly as stored (empty cells keep). Test in Task 2.
2. A file whose rows swap two people's managers is accepted; a file that closes a cycle is refused on the row that closes it. Test in Task 3.
3. Re-importing an unchanged export writes nothing: no audit rows, no `updated` entries, even for employees the importer may only view. Test in Task 2.
4. Editing an employee linked to an account keeps the link. Test in Task 2.
5. The same code twice (case differs: `e1` and `E1`) is a row error on the second row. Test in Task 2.

---

### Task 1: Employee export

**Files:**
- Create: `internal/modules/hrm/employee_io.go`
- Modify: `internal/modules/hrm/queries/employee.sql` (add `EmployeeExportBase`), `internal/modules/hrm/employee_list.go` (add `employeeExportList`), `internal/modules/hrm/import.go:registerDataIO`, `internal/modules/hrm/service.go:76` (restrict the new audit action), `internal/modules/hrm/i18n/{vi,en}.json`, `web/src/hrm/i18n/{vi,en}/main.json` (audit label)
- Test: `internal/modules/hrm/employee_io_test.go`

**Interfaces:**
- Produces:
  - `const employeeIO = "hrm.employees"` (export and import kind; equals `employeeType`, reuse that constant if it is the same string)
  - `type EmployeeExportParams struct { OrgUnitID int64 `json:"org_unit_id"`; Status string `json:"status"`; Q string `json:"q"` }` in `employee_types.go`
  - `func (s *Service) employeeColumns(ctx context.Context) ([]string, error)`: the 15 translated titles, in this order with these keys under `hrm.import.header.`: `code`, `name`, `date_of_birth`, `gender`, `phone`, `email`, `address`, `org_unit`, `manager`, `hire_date`, `termination_date`, `national_id`, `social_insurance_no`, `tax_code`, `bank_account` (`code`, `name` exist; vi titles: Mã NV, Họ tên, Ngày sinh, Giới tính, SĐT, Email, Địa chỉ, Phòng ban, Mã người quản lý trực tiếp, Ngày vào làm, Ngày nghỉ việc, CCCD/Hộ chiếu, Số sổ BHXH, Mã số thuế, Số tài khoản)
  - `func unitPaths(units []iam.OrgUnit) map[int64]string`: for every unit, its shortest unique path (`name`, or `parent / name`, … up the tree, joined by ` / `); units whose full path is still not unique map to their full path
  - `func resolveUnit(units []iam.OrgUnit, path string) (id int64, matches int)`: units whose name equals the last segment and whose ancestors' names equal the preceding segments in order (a suffix of the ancestor chain); trimmed, case-sensitive
  - `func genderLabel(loc string, g *string) string` and `const dateLayout = "02/01/2006"`

- [ ] **Step 1: Write the failing tests** in `employee_io_test.go` (package `hrm_test`), with a helper `func (f *fixture) exportEmployees(ctx context.Context, params string) [][]string` that calls `f.core.DataIO.StartExport(ctx, "hrm.employees", []byte(params))`, `f.wantJob(job, "completed")`, reads the file through `f.core.DataIO.Job`/`File` (as `printPayslip` in `me_test.go`), and returns `excelize` rows of `Sheet1` (`RawCellValue: true`).
  - `TestEmployeeExportColumns`: HR (`hr@*`) creates E1 in A with date of birth `1990-02-03`, gender `female`, manager E0. Rows[0] equals the 15 vi titles; E1's row has `"03/02/1990"`, `"Nữ"`, `"A"`, `"E0"`, hire date `"05/01/2026"`, empty sensitive cells.
  - `TestEmployeeExportFilters`: E1 in A, E2 in B, E3 terminated; `{"org_unit_id":A}` gives only E1; `{"status":"terminated"}` only E3; `{"q":"E2"}` only E2; `hr@A` with `{}` sees no E2.
  - `TestEmployeeExportSensitive`: E1 with all four sensitive values. Export by `hr@*` → the four cells empty and no `hrm.employee_sensitive_exported` audit row. Export by `hr@*` + `sensitive_viewer@*` → full values, and exactly one audit row `hrm.employee_sensitive_exported` on E1 whose data is `{"fields": ["national_id", …]}` (only the non-empty fields) and contains no value.
  - `TestEmployeeExportNeedsView`: a user with no HRM role gets the job ending `forbidden`.
  - `TestUnitPaths` (table test over `iam.OrgUnit` values, no DB): C→{HN→{Kinh doanh}, HCM→{Kinh doanh}, Kế toán}: `Kế toán` → `"Kế toán"`, the two → `"HN / Kinh doanh"` and `"HCM / Kinh doanh"`; `resolveUnit` of `"Kinh doanh"` → 2 matches, `"HN / Kinh doanh"` → 1, `"C / HN / Kinh doanh"` → 1, `"X"` → 0. Expose the two pure functions to `hrm_test` through `export_test.go` (`var UnitPaths = unitPaths`, `var ResolveUnit = resolveUnit`), as that file already does for other internals.

- [ ] **Step 2: Run** `go test ./internal/modules/hrm -run 'TestEmployeeExport|TestUnitPaths'`. Expected: FAIL (kind not registered / undefined).

- [ ] **Step 3: Implement.**
  - `EmployeeExportBase` (`-- data:base`): every `hrm.employees` column the sheet needs plus `m.code AS manager_code` (LEFT JOIN manager). `employeeExportList` reuses the scope, search and default order of `employeeList`; the export runs it with `employeeQuery(sc, EmployeeFilter{OrgUnitID, Status, Q}, today)` and `Paging{Page: 1, PageSize: 1 << 30}` (ponytail comment: one unpaged read; stream if tenants pass tens of thousands).
  - Export `Check`: `IAM.DataScope(ctx, "hrm", PermView)` must not be empty, else `platform.ErrForbidden`. `Run` builds rows: dates via `dateLayout`, gender via `genderLabel` (backend keys `hrm.import.gender.{male,female,other}`, vi Nam/Nữ/Khác), org unit via `unitPaths(IAM.OrgUnits(ctx, "", ""))`, sensitive cells only where `IAM.Allowed(ctx, "hrm", PermSensitive, row.OrgUnitID)`, opened with `open`. One `Audit.RecordFor(ctx, "hrm.employee_sensitive_exported", audit.Ref{Type: employeeType, ID: id}, map[string]any{"fields": fields})` per employee with at least one exported value. Sheet name: translated `hrm.import.sheet.employees` ("Nhân viên"/"Employees").
  - Register in `registerDataIO`; add the new action to the `RestrictHistory(…, record.ViewFiles)` loop; add its label `hrm.employee_sensitive_exported` ("Xuất thông tin nhạy cảm ra Excel" / "Exported sensitive data to Excel") to the web i18n files.

- [ ] **Step 4: Run** the same tests. Expected: PASS. Then `go test ./internal/modules/hrm` (whole package) passes.

- [ ] **Step 5: Commit** `feat(hrm): export employee profiles to Excel`.

---

### Task 2: Employee import (create, edit, sensitive, scope)

**Files:**
- Modify: `internal/modules/hrm/employee_io.go`, `internal/modules/hrm/employee.go` (split `UpdateEmployee`), `internal/modules/hrm/import.go:registerDataIO`, `internal/modules/hrm/i18n/{vi,en}.json`
- Test: `internal/modules/hrm/employee_io_test.go`, `internal/app/jobs_test.go`

**Interfaces:**
- Consumes: `employeeColumns`, `resolveUnit`, `dateLayout`, gender labels from Task 1.
- Produces:
  - `func (s *Service) importEmployees(ctx context.Context, _ json.RawMessage, rows [][]string) ([]dataio.RowError, error)` registered as import `hrm.employees` with `Header: func(ctx, _) { require edit somewhere; return s.employeeColumns(ctx) }`
  - `func (s *Service) updateEmployee(ctx context.Context, id int64, in EmployeeInput, checkCycle bool) error`: today's `UpdateEmployee` body; `UpdateEmployee` calls it with `true`. With `false` it skips only the `LockManagerTree`/`ManagerChain` block (`manager_self` is still refused).
  - `func cellDate(cell string) (string, bool)`: accepts `dd/mm/yyyy`, `yyyy-mm-dd` and an Excel serial number; returns ISO date. Blank → `("", true)`.
  - `func cellGender(cell string) (*string, bool)`: accepts the vi and en labels (case-insensitive) and `male`/`female`/`other`.
  - Row-error codes (vi/en text in backend i18n): `duplicate_employee` (exists), `name_required`, `org_unit_required`, `hire_date_required`, `invalid_date` {column, value}, `invalid_gender` {value}, `unknown_org_unit` {value}, `ambiguous_org_unit` {value}, `unknown_manager` {code}, `employee_code_taken` {code}, `sensitive_forbidden` {code}, `forbidden_row` {code}, `termination_before_hire`, `manager_self`, `manager_cycle` {name}. A service error from `prepare` maps to the row error of the same code with the row's code as param; any other error aborts the job.

Order inside `importEmployees` (all in the job's one transaction):
1. Parse every row into `parsedRow{row int; code string; cells [15]string}`; blank rows skipped. Look up codes with `EmployeesByCodes` (case-insensitive). Duplicate code → error on the later row.
2. For an existing code: outside `DataScope(PermView)` → `employee_code_taken`. Otherwise read `GetEmployee` (gives `user_login`) and build the current `EmployeeInput`; overlay non-empty plain cells. If the merged plain fields equal the current ones and no sensitive cell is filled, the row is **skipped** (no permission check, no write, no audit).
3. A non-empty sensitive cell needs `PermSensitive` at the target unit (and the old unit when moving), checked before comparing with the stored value → `sensitive_forbidden`. Filled cells go into `EmployeeInput.Sensitive`; an equal stored value is still not rewritten (compare after opening) so no audit diff appears.
4. The merged input always carries the current `UserLogin`, so the account link stays.
5. Collect row errors for every row before writing; return them if any.
6. Write creates first, then updates (Task 3 adds the manager ordering and the final cycle check; in this task a manager cell must name an employee that already exists, else `unknown_manager`). Creates call `CreateEmployee` directly (`platform.InTx` joins the job's transaction when one is in ctx). Updates use `updateEmployee(…, true)`.

- [ ] **Step 1: Write the failing tests.** Helper `func (f *fixture) importEmployees(ctx context.Context, rows ...[]any) dataio.Job` that writes an excelize file (header = Task 1 titles, prepended), calls `f.core.DataIO.StartImport(ctx, "hrm.employees", "e.xlsx", r, nil)`, `wantJob`, returns `f.core.DataIO.Job`.
  - `TestEmployeeImportCreatesAndEdits`: E1 exists in A. File: `E1` with new phone; `E9` new in `B`; empty code new in `A`. Job completed; E1's phone changed, other E1 fields unchanged; E9 exists with code `E9`; the third got a series code; audit has one `hrm.employee_updated` on E1 and two `hrm.employee_created`.
  - `TestEmployeeImportAllOrNothing`: one good row and one row with unknown unit `"Z"` → job failed, `RowErrors` has row 3 with the unit message; nothing created.
  - `TestEmployeeImportUnchangedExportWritesNothing`: export with `viewer@*` + `hr@A` (view everywhere, edit only A), import that same file back unchanged → completed, zero new audit rows (Review Focus 3).
  - `TestEmployeeImportKeepsSensitiveAndAccount`: E1 linked to user `u1`, with tax code `123`. HR without sensitive permission exports, changes E1's address, imports → address changed, tax code still `123`, still linked to `u1` (Review Focus 1, 4). The same file with a tax code typed in → `sensitive_forbidden`. HR with `sensitive_viewer@*` importing a tax code → stored, one `employee_updated` audit.
  - `TestEmployeeImportScope`: `hr@A` importing a row for E2 (in B) → `employee_code_taken`; a new row in B → `forbidden_row`.
  - `TestEmployeeImportDuplicateCode`: rows `e1` then `E1` → `duplicate_employee` on row 3 (Review Focus 5).
  - `TestEmployeeImportDatesAndGender`: `"03/02/1990"`, `"1990-02-03"` and the serial `32907` all store `1990-02-03`; `"nam"` stores `male`; `"31/02/1990"` → `invalid_date`.
  - In `internal/app/jobs_test.go`, `TestEmployeeImportOverHTTP`: through `f.upload(f.admin, "hrm.employees", "", head, row)`, one created employee and one row error message in Vietnamese, proving the HTTP route and translations.

- [ ] **Step 2: Run** `go test ./internal/modules/hrm -run TestEmployeeImport` and `go test ./internal/app -run TestEmployeeImportOverHTTP`. Expected: FAIL.

- [ ] **Step 3: Implement** as ordered above.

- [ ] **Step 4: Run** the tests, then `go test ./internal/modules/hrm ./internal/app`. Expected: PASS (existing `UpdateEmployee` tests unchanged).

- [ ] **Step 5: Commit** `feat(hrm): import employee profiles from Excel`.

---

### Task 3: Managers within the file and the end-state cycle check

**Files:**
- Modify: `internal/modules/hrm/employee_io.go`
- Test: `internal/modules/hrm/employee_io_test.go`

**Interfaces:**
- Consumes: `importEmployees`, `updateEmployee(…, checkCycle bool)` from Task 2.
- Produces: `func createOrder(rows []parsedRow, newCodes map[string]int) ([]int, error)`: indexes of new rows ordered so a new row's manager (when also new in the file) comes first; a cycle among new rows returns the index of a row in it (reported as `manager_cycle`).

- [ ] **Step 1: Write the failing tests.**
  - `TestEmployeeImportManagerInFile`: new `M1` and new `R1` with manager `M1`, written with `R1` first → both created, `R1.manager = M1`.
  - `TestEmployeeImportManagerSwap`: today A's manager is B. The file has B's row first with manager A, then A's row with manager C → completed, end state A→C, B→A (Review Focus 2). A per-row check would have refused B's row. (An empty cell keeps the manager, so the file cannot clear one.)
  - `TestEmployeeImportManagerCycle`: A→B exists; file sets B's manager to A → `manager_cycle` on B's row, nothing written. New rows `N1→N2`, `N2→N1` → `manager_cycle`.
  - `TestEmployeeImportManagerSelf`: E1's manager `E1` → `manager_self`.

- [ ] **Step 2: Run** `go test ./internal/modules/hrm -run TestEmployeeImportManager`. Expected: FAIL.

- [ ] **Step 3: Implement.** Resolve manager cells against existing codes plus new codes in the file. Write creates in `createOrder`, then updates with `updateEmployee(…, false)`. Then take `store.LockManagerTree`, and for each written row with a manager run `store.ManagerChain(manager)`; if it contains the row's employee, return `manager_cycle` on that row with the manager's `code · name` (the job's tx rolls back). Keep the ponytail comment from `UpdateEmployee` about the single lock.

- [ ] **Step 4: Run** the tests, then `go test ./internal/modules/hrm`. Expected: PASS.

- [ ] **Step 5: Commit** `feat(hrm): employee import resolves managers in the file`.

---

### Task 4: Employee list screen, docs, e2e

Use the `mmerp-ui` and `new-screen` skills; finish with the `docs/ui.md` checklist.

**Files:**
- Modify: `internal/modules/hrm/employee.go:EmployeeActions`, `internal/modules/hrm/service_test.go:TestEmployeeActions`, `web/src/hrm/pages/EmployeesPage.tsx`, `web/src/hrm/i18n/{vi,en}/main.json`, `products/hrm.md:58`
- Create: `web/e2e/employee-import.spec.ts`

**Interfaces:**
- Consumes: kinds `hrm.employees` from Tasks 1–3.
- Produces: action `import_employees` in `GET /hrm/employees/actions`: edit permission somewhere and `ProductGate(ctx, "hrm", platform.ClassWrite) == nil`, exactly like `import_leave_balances`.

- [ ] **Step 1: Failing Go test.** Extend `TestEmployeeActions`: `hr@A` gets `import_employees`; `viewer@*` does not. Run `go test ./internal/modules/hrm -run TestEmployeeActions` → FAIL; add the action; → PASS; `make gen` to refresh types.

- [ ] **Step 2: Screen.** On `EmployeesPage`, in the list's `action` area before the create button: `<ExportButton kind="hrm.employees" params={{ org_unit_id, status, q }} label={t('hrm.employees.export')} />` (always shown, since the page needs view); when the actions include `view_sensitive`, a second export entry "Xuất kèm thông tin nhạy cảm" / "Export with sensitive data" (`hrm.employees.export_sensitive`) passing `include_sensitive: true`; and when `import_employees`: a button opening `<ImportDialog kind="hrm.employees" title={t('hrm.employees.import')} description={t('hrm.employees.import_hint')} writes={[hrmKeys.employees.all()]} … />`. With three or more actions, follow `mmerp-ui` for grouping secondary actions (e.g. a menu for the two imports) rather than a row of four buttons. Copy (vi; en alongside):
  - `hrm.employees.export`: "Xuất Excel"
  - `hrm.employees.import`: "Nhập hồ sơ từ Excel"
  - `hrm.employees.import_hint`: "Mỗi dòng là một nhân viên. Mã đã có thì sửa hồ sơ đó, mã mới hoặc để trống thì tạo nhân viên mới. Ô để trống giữ nguyên giá trị hiện tại. Một dòng lỗi thì không dòng nào được ghi. Dùng file xuất từ danh sách này làm mẫu."

- [ ] **Step 3: e2e** `web/e2e/employee-import.spec.ts` (follow `m5.spec.ts`/`helpers.ts` patterns): HR exports the list, the download has the header row; HR uploads a file creating one employee and editing another's phone, sees the success state, and the list shows the new employee; a file with an unknown department shows the row error. Do not run it; tell the user to run `make e2e`.

- [ ] **Step 4: Docs.** `products/hrm.md` line 58: profile import/export now exists (employee export and import with the empty-cell rule); remove "there is no profile import/export yet". Add the `hrm.employees` export/import to the `dataio` list there.

- [ ] **Step 5: Verify** `go test ./internal/modules/hrm ./internal/app` and `cd web && pnpm vitest run` pass, `pnpm tsc --noEmit` clean. Commit `feat(web): export and import employee profiles from the employee list`.

---

After Task 4: tell the user to run `make lint` and `make e2e`; opening the PR (and the GitHub issue the workflow expects) waits for their go-ahead.
