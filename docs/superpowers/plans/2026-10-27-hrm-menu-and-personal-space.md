# HRM Menu and Personal Space Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rename and regroup the HRM menu in HR's words, add a "cần xử lý" block to the HRM overview, and add a "Cá nhân" home tile where every linked employee sees their own days, payslips, requests, profile and the org chart.

**Architecture:** The backend adds an HRM "own" area (`me.go`) serving only the actor's linked employee, a month-close count for the overview, and an "own part" path in core `printing`, so an employee's payslip PDF is cut from the snapshot frozen at posting. The frontend adds a second manifest from `web/src/hrm` (base path `/me`). Its tile is decided by an optional `useEntitled` hook on `AreaManifest`, so core never learns HRM data.

**Tech Stack:** Go, Huma, sqlc, Postgres; React, TanStack Query, react-i18next, Vitest, Playwright.

**Spec:** `docs/superpowers/specs/2026-10-27-hrm-menu-and-personal-space-design.md`

## Global Constraints

- Vietnamese menu labels exactly as in the spec table; group "Cấu hình" becomes "Thiết lập"; HRM tile "Nhân sự"; personal tile "Cá nhân"; personal items "Chấm công", "Phiếu lương", "Nghỉ phép & tăng ca", "Hồ sơ cá nhân", "Sơ đồ tổ chức" (no "của tôi").
- English domain terms, API paths and record-type codes do not change.
- Personal space endpoints never take an employee id from the client; "own" = `hrm.employees.user_id = actor`.
- Payslips only for payrolls with `record.documents.status = 'posted'`.
- Every own payslip read (list, detail, PDF) is audited; amounts never go into audit data or logs.
- Profile in the personal space: ordinary fields only, no sensitive fields.
- Services query through `store.<Query>.Run(ctx, …)`; audit writes inside `platform.InTx`.
- All text through i18n, `vi` and `en`; screens follow `docs/ui.md` and the `mmerp-ui` skill; layouts from `shared/ui/page`.
- Spec correction: leave balances are per employee and **year**, not per leave type (`hrm.leave_balances`). The requests page shows the single balance of the current year.

## Review Focus

1. **Another employee's payslip by id:** `GET /hrm/me/payslips/{payroll_id}` for a payroll the actor has no line on must be 404, and the print export with `parts:[otherEmployeeId]` must be refused. Tested in Tasks 2 and 3.
2. **A payroll cancelled after being posted:** its month vanishes from the list and its detail is 404. Tested in Task 3.
3. **An account with no employee profile** (e.g. `root`) calls `/hrm/me/*`: `422 not_an_employee`, no tile. Tested in Tasks 3 and 5.
4. **A week view across two months:** both months are fetched and merged; standard days show per month only in the month view. Tested in Task 6.
5. **HRM product disabled:** the personal tile still shows and reads work; the "new request" buttons follow `allowed_actions` (no `create`). Tested in Task 9.

---

### Task 1: ADRs and glossary

**Files:**
- Create: `docs/adr/0042-personal-space-tile.md`
- Create: `docs/adr/0043-printing-own-part.md`
- Modify: `CONTEXT.md` (HRM section)

- [ ] **Step 1:** Write ADR-0042: a product area may export more than one manifest; each manifest is one home tile with its own base path and menu; a manifest may declare `useEntitled: () => boolean | undefined` to decide its own tile from its product's API, replacing the permission rule; `recordTypes` stay on the product's main manifest only.
- [ ] **Step 2:** Write ADR-0043: a print template may declare `Owner(ctx, id) (int64, error)`, the part key the actor owns in a record (0 = none). An actor without View+Print on the record may print exactly `parts: [owner key]` of a **posted** record, from its frozen snapshot; the print is audited as any other. HRM payslips use it, with the employee id as key.
- [ ] **Step 3:** Add "Personal space" (Cá nhân) to `CONTEXT.md`: the home tile where a user linked to an employee profile sees their own HRM data.
- [ ] **Step 4: Commit** `docs(hrm): ADRs for the personal space tile and own-part prints`

### Task 2: Own-part prints in core `printing`

**Files:**
- Modify: `internal/core/printing/types.go` (Template), `internal/core/printing/service.go` (`check`)
- Modify: `internal/modules/hrm/print.go` (payslip template registers `Owner`)
- Test: `internal/core/printing/service_test.go`, `internal/modules/hrm/me_test.go`

**Interfaces:**
- Produces: `printing.Template.Owner func(ctx context.Context, id int64) (int64, error)`, optional. HRM: `func (s *Service) payslipOwner(ctx context.Context, payrollID int64) (int64, error)` returns the actor's employee id if that employee has a line on the payroll, else 0.

- [ ] **Step 1: Write failing tests** in `service_test.go` (follow its existing test record type): `TestOwnPartPrint`:
  - an actor without View on a posted record, template `Owner` returning 7: export params `{"id": X, "parts": [7]}` passes `Check` and renders;
  - same with `parts: [8]`, `parts: []`, or `parts: [7, 8]` → `platform.ErrNotFound` (whatever `Visible` returns today for an unseen record);
  - the record in `draft` with `parts: [7]` → refused;
  - an audit row `printing.printed` with `parts: [7]` exists after the render.
- [ ] **Step 2:** Run `go test ./internal/core/printing -run TestOwnPartPrint`. Expected: FAIL.
- [ ] **Step 3:** In `check`, when `Visible` or `Can(Print)` refuses, and `t.Owner != nil`, and the record is `posted`, and `in.Parts` is exactly one key equal to `Owner(ctx, in.ID)` (non-zero), allow. Otherwise return the original error. `render` then uses the frozen snapshot unchanged.
- [ ] **Step 4:** Register `Owner: s.payslipOwner` on the `hrm.payslip` template. Add to `me_test.go`: `TestOwnPayslipPrint`: an employee with no role prints their own part of a posted payroll and gets a PDF; another employee's part is refused.
- [ ] **Step 5:** Run `go test ./internal/core/printing ./internal/modules/hrm -run 'OwnPart|OwnPayslipPrint'`. Expected: PASS.
- [ ] **Step 6: Commit** `feat(printing): print one's own part of a posted record`

### Task 3: HRM "own" endpoints

**Files:**
- Create: `internal/modules/hrm/me.go`, `me_types.go`, `me_handler.go`, `queries/me.sql`, `me_test.go` (extend)
- Modify: `internal/modules/hrm/handler.go` (register), `sqlc.yaml` only if queries are listed per file (check)

**Interfaces:**
- Produces (Go, JSON tags snake_case):
  - `func (s *Service) MyProfile(ctx) (MyProfile, error)`, `GET /hrm/me`. `MyProfile{EmployeeID int64; Code, FullName string; DateOfBirth, Gender, Phone, Email, Address *string; OrgUnitName string; ManagerName *string; HireDate string; TerminationDate *string; Contract *MyContract}`, `MyContract{TypeName, StartDate string; EndDate *string}` (the contract in effect today, no amounts).
  - `func (s *Service) MyDays(ctx, month string /*YYYY-MM*/) (MyDays, error)`, `GET /hrm/me/days?month=`. `MyDays{Month string; StandardDays int; Days []MyDay}`, `MyDay{Date, Kind string /*work|rest|holiday*/; Worked *string /*"1","0.5"*/; WorkedClosed bool; Leave *MyDayLeave; OvertimeHours *string}`, `MyDayLeave{TypeName, Days string; Paid bool}`. Worked comes from the timesheet lines of any non-cancelled timesheet for the month (`WorkedClosed` = that timesheet is `posted`). Leave and overtime come from **posted** requests only. Kind and standard days come from `workCalendar` of `s.d.IAM.LegalEntityOf(ctx, employee.OrgUnitID)`. A leave request's days are spread across its dates with the same allocation the payroll uses.
  - `func (s *Service) MyPayslips(ctx) ([]MyPayslipSummary, error)`, `GET /hrm/me/payslips`. `MyPayslipSummary{PayrollID int64; PeriodStart string; Net int64}`, newest first.
  - `func (s *Service) MyPayslip(ctx, payrollID int64) (MyPayslip, error)`, `GET /hrm/me/payslips/{payroll_id}`. `MyPayslip{PayrollID int64; Number, PeriodStart string; StandardDays int; PaidDays, OvertimeHours string; Amounts PayrollAmounts; Adjustments []PayrollAdjustmentInput}`.
  - Every endpoint: no linked employee → `ErrNotAnEmployee`. Payslip list audits `hrm.own_payslips_viewed` on the employee ref; detail audits `hrm.own_payslip_viewed` on the payroll ref (data: employee id only).
- Reuses as-is: `/hrm/leaves?employee_id=`, `/hrm/overtimes?employee_id=`, `/hrm/employees/{id}/leave-balances` and `/hrm/leaves/actions` (`self_employee_id`) already serve the actor's own data.

- [ ] **Step 1: Write failing tests** in `me_test.go`:
  - `TestMyPayslips`: employee A with a posted payroll and a draft one sees one entry with `Net` equal to the payroll line's net; after cancelling the posted payroll the list is empty and the detail is `platform.ErrNotFound`.
  - `TestMyPayslipMatchesPrint`: `MyPayslip(p).Amounts`, `PaidDays`, `Adjustments` equal the `payslipPrint` part of A in `payslips(ctx, p)`.
  - `TestMyPayslipOtherEmployee`: A asking for a payroll where only B has a line → `platform.ErrNotFound`.
  - `TestMyPayslipAudited`: one `hrm.own_payslip_viewed` row after a detail read, its data holding no amount.
  - `TestMyDays`: a month with a draft timesheet (A worked 1 on the 1st, 0.5 on the 2nd), a posted annual-leave request on the 3rd, posted overtime of 2h on the 4th, a holiday on the 5th: days match field by field, `WorkedClosed` false, `StandardDays` equals `calendar.standardDays` for the month; after posting the timesheet `WorkedClosed` is true. B's lines never appear.
  - `TestMyProfile`: ordinary fields and the current contract's type and term; the JSON has no `id_number`, `tax_code`, `bank_account` or salary key.
  - `TestMeNotAnEmployee`: an unlinked actor gets `ErrNotAnEmployee` from all four.
- [ ] **Step 2:** Run `go test ./internal/modules/hrm -run 'TestMy|TestMe'`. Expected: FAIL (undefined).
- [ ] **Step 3:** Add queries in `queries/me.sql`: `MyPostedPayrollLines(employee_id)` (payroll id, number, period_start, line data; joins `record.documents` status `posted`), `MyPayrollLine(employee_id, payroll_id)` (same, one row), `MyTimesheetLines(employee_id, period_start)` (date, days, timesheet status; non-cancelled timesheets), `MyLeaves(employee_id, from, to)` and `MyOvertime(employee_id, from, to)` (posted only), `MyCurrentContract(employee_id, today)`. Run `make gen`.
- [ ] **Step 4:** Implement the four service methods and handlers. Decrypt with `linesAAD`/`adjustmentsAAD` as `payrollLines`/`adjustments` do; reuse their unmarshal into `PayrollLine` fields.
- [ ] **Step 5:** Run `go test ./internal/modules/hrm -run 'TestMy|TestMe'`. Expected: PASS. Then `make lint`.
- [ ] **Step 6: Commit** `feat(hrm): own profile, days and payslips for the personal space`

### Task 4: Month-close counts for the overview

**Files:**
- Create: `internal/modules/hrm/overview.go`, `overview_handler.go`, `queries/overview.sql`, `overview_test.go`
- Modify: `internal/modules/hrm/handler.go` (register)

**Interfaces:**
- Produces: `GET /hrm/overview/month-close` → `MonthClose{PeriodStart string; OpenTimesheetUnits *int; OpenPayrollEntities *int}` for the previous calendar month in the tenant time zone. `OpenTimesheetUnits` counts org units the actor holds `hrm.timesheet.view` in, having an employee employed during the period, with no `posted` timesheet for it (nil without that permission anywhere). `OpenPayrollEntities` counts legal entities the actor holds `hrm.payroll.view` in, with an employee holding a posted original contract with the entity over the period (as `PayrollEmployees`), and no `posted_payroll_id` for the period (nil without the permission).

- [ ] **Step 1: Write failing test** `TestMonthClose`: two units with employees last month, one with a posted timesheet → `OpenTimesheetUnits == 1`; after posting the payroll of the entity → `OpenPayrollEntities == 0`; an actor with neither permission → both nil.
- [ ] **Step 2:** Run `go test ./internal/modules/hrm -run TestMonthClose`. Expected: FAIL.
- [ ] **Step 3:** Implement. Filter units and entities by `s.allowed(ctx, perm, unit)`.
- [ ] **Step 4:** Run it again. Expected: PASS.
- [ ] **Step 5: Commit** `feat(hrm): month-close counts for the overview`

### Task 5: `useEntitled` on area manifests

**Files:**
- Modify: `web/src/shared/area/index.ts`, `web/src/core/pages/HomePage.tsx`
- Test: `web/src/core/pages/HomePage.test.tsx` (create)

**Interfaces:**
- Produces: `AreaManifest.useEntitled?: () => boolean | undefined` (a React hook; `undefined` while loading = tile not shown yet). When present it replaces the permission rule for that tile. The tile still needs a first nav item the user may see.

- [ ] **Step 1: Write failing test** `HomePage.test.tsx`: an area with `useEntitled: () => true` and no permissions shows its tile; with `() => false` or `() => undefined` it does not; areas without the hook keep today's rule. Two areas with the same `product` both render (React keys by `basePath`).
- [ ] **Step 2:** Run `pnpm -C web vitest run src/core/pages/HomePage.test.tsx`. Expected: FAIL.
- [ ] **Step 3:** Render one `AreaTile` component per area so each calls its own hook. Key tiles by `basePath`.
- [ ] **Step 4:** Run it again. Expected: PASS.
- [ ] **Step 5: Commit** `feat(web): an area may decide its own home tile`

### Task 6: Personal space area: manifest, days page

**Files:**
- Create: `web/src/hrm/me/index.ts` (manifest `hrmMe`), `web/src/hrm/me/routes.tsx`, `web/src/hrm/me/DaysPage.tsx`, `web/src/hrm/me/days.ts` + `days.test.ts`
- Modify: `web/src/app/areas.ts` (`[hrm, hrmMe, sales, core]`), `web/src/hrm/keys.ts` (`hrmKeys.me.*`), `web/src/hrm/i18n/{vi,en}/{meta,main}.json`

**Interfaces:**
- Consumes: Task 3 endpoints via `@/shared/api/hrm` (after `make gen`); Task 5 `useEntitled`.
- Produces: manifest `{ product: 'hrm', label: 'hrm.me.nav.group', basePath: '/me', homeIcon: IconUserCircle, nav: [days, payslips, requests, profile, org], useEntitled }`. `useEntitled` reads `GET /hrm/leaves/actions` and returns `self_employee_id != null`. No `recordTypes`. Shares the HRM i18n bundles.
- `days.ts`: `weekOf(date: string): string[]` (Monday to Sunday) and `monthsOf(dates: string[]): string[]` (distinct `YYYY-MM`), pure.

- [ ] **Step 1: Write failing tests** `days.test.ts`: `weekOf('2026-10-01')` → `2026-09-28…2026-10-04`; `monthsOf` of that week → `['2026-09','2026-10']`.
- [ ] **Step 2:** Run `pnpm -C web vitest run src/hrm/me/days.test.ts`. Expected: FAIL.
- [ ] **Step 3:** Implement `days.ts`, the manifest, routes and `DaysPage`: a Month / Week switch, previous/next navigation, a calendar grid (worked, leave type, holiday, rest day, overtime hours), and a "chưa chốt" label on days whose `worked_closed` is false. The month view has a summary: days worked, paid leave days, overtime hours, `standard_days`. The week view fetches every month in `monthsOf(week)` and has no standard-days figure. Use the `mmerp-ui` skill; the grid must work on a phone-width screen.
- [ ] **Step 4:** Run vitest and `make lint`. Expected: PASS.
- [ ] **Step 5: Commit** `feat(hrm): personal space with own days`

### Task 7: Personal space: payslips, requests, profile, org chart

**Files:**
- Create: `web/src/hrm/me/PayslipsPage.tsx`, `PayslipPage.tsx`, `RequestsPage.tsx`, `ProfilePage.tsx`
- Modify: `web/src/hrm/me/routes.tsx`, i18n files

**Interfaces:**
- Consumes: Task 3 endpoints; `ExportButton` with `kind="printing.hrm.payslip"`, `params={{ id: payrollId, parts: [selfEmployeeId] }}` (Task 2); existing `NewLeavePage`, `NewOvertimePage`, `LeavePage`, `OvertimePage`, `OrgStructurePage` mounted under `/me/...`.

- [ ] **Step 1:** `PayslipsPage`: months newest first with net pay, opening `PayslipPage` (`/me/payslips/:payrollId`), which shows the sections of the printed payslip (income, deductions, tax, adjustments, net) and the PDF button. An empty list has an empty state saying payslips appear once payroll is closed.
- [ ] **Step 2:** `RequestsPage`: the current year's leave balance, own leave requests and own overtime requests (lists filtered by `employee_id = self_employee_id`), and "new leave" / "new overtime" buttons shown only when `/hrm/leaves/actions` (resp. overtimes) contains `create`. The new-request and detail routes live under `/me/requests/...` and reuse the existing pages; after creating, stay under `/me`.
- [ ] **Step 3:** `ProfilePage` from `GET /hrm/me`; `org` route renders `OrgStructurePage`.
- [ ] **Step 4:** Run `make lint` and `pnpm -C web vitest run`. Expected: PASS.
- [ ] **Step 5: Commit** `feat(hrm): payslips, requests and profile in the personal space`

### Task 8: HRM menu in HR's words, and the overview's "cần xử lý"

**Files:**
- Modify: `web/src/hrm/index.ts` (nav order unchanged except group key `group_config` label), `web/src/hrm/i18n/vi/meta.json`, `web/src/hrm/i18n/vi/main.json`, `en` counterparts, `web/src/hrm/pages/OverviewPage.tsx`, every `web/e2e/*.spec.ts` matching old labels (`grep -rln "Đơn nghỉ\|Bảng công\|Tham số pháp lý\|Quy tắc duyệt\|Cơ cấu tổ chức\|Quản lý nhân sự\|Loại nghỉ\|Lịch làm việc\|Tăng ca" web/e2e`), `products/hrm.md` (Frontend section)

**Interfaces:**
- Consumes: Task 4 `GET /hrm/overview/month-close`; core `GET /approvals/inbox` through `@/shared/api/core` (count items whose doc type is `hrm.leave_request` or `hrm.overtime_request`).

- [ ] **Step 1:** Change the `vi` labels to the spec table, and make page titles, breadcrumbs, record-type names, empty states and buttons use the same words (e.g. every user-facing "Đơn nghỉ" → "Đơn nghỉ phép", "Bảng công" → "Bảng chấm công", "Tham số pháp lý" → "Mức đóng bảo hiểm & thuế", "Quy tắc duyệt" → "Quy trình duyệt"). Update the `en` labels to match in meaning ("Employee records", "Employment contracts", "Org chart", "Attendance sheets", "Leave requests", "Overtime requests", "Payrolls", "Setup", "Insurance & tax rates", "Approval workflows", "Work calendar & holidays"); English record names stay CONTEXT.md terms. Run the i18n key check in `make lint`.
- [ ] **Step 2:** `OverviewPage`: above today's tiles, a "Cần xử lý" section with up to three entries, each hidden when zero or not permitted: requests waiting for the actor's approval (to `/approvals`), units without a closed attendance sheet last month (to `/hrm/timesheets` filtered to that period), legal entities without a closed payroll last month (to `/hrm/payrolls`). The whole section is hidden when all are zero.
- [ ] **Step 3:** Update e2e selectors to the new labels. Run `make e2e`. Expected: PASS.
- [ ] **Step 4:** Update `products/hrm.md`: the Frontend section (menu, personal space) and Personal data (own payslips readable by the employee, audited).
- [ ] **Step 5: Commit** `feat(hrm): menu and screens in HR's words, overview needs-attention block`

### Task 9: End-to-end personal space

**Files:**
- Create: `web/e2e/personal-space.spec.ts`

- [ ] **Step 1: Write the test** (helpers from `web/e2e/helpers.ts` / `api.ts`): set up an employee linked to a no-role account, a posted payroll with their line, and a draft timesheet day. That user signs in and sees exactly the "Cá nhân" tile. Under "Chấm công" they see the day marked "chưa chốt". Under "Phiếu lương" the net pay matches. The PDF download job succeeds. Visiting `/me/payslips/<other payroll id>` shows not-found. A second test signs the same user in on `:8091` (same database, no product enabled, no jobs): the tile shows, the payslip list reads, and there is no "new request" button.
- [ ] **Step 2:** Run `make e2e`. Expected: PASS.
- [ ] **Step 3: Commit** `test(hrm): personal space end to end`
