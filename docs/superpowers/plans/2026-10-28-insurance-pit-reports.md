# Insurance and PIT reports Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Five `dataio` Excel exports per legal entity (D02-LT, 05/KK-TNCN, 05/QTT-TNCN, 05-1/BK-QTT-TNCN, 05-3/BK-QTT-TNCN) computed from posted payrolls, and an HRM "Báo cáo" page to start them.

**Architecture:** All in `internal/modules/hrm`, area-first files `report*.go`. One reader (`reportBasis`) loads the posted payroll of every month a report covers (refusing a missing month), decrypts the lines in the service and checks the salary permission; one reader (`reportPeople`) loads the listed employees with decrypted identifiers, checks and audits the sensitive permission. Each export builds a flat `dataio.Sheet` from those, pins `{legal entity, employee ids}` and shares one `Check`. Layouts are Go values named after form and legal basis. No table, no migration, no core change.

**Tech Stack:** Go, sqlc, `shopspring/decimal`, `dataio`; React + `@mmerp/ui`, `shared/jobs.ExportButton`; Playwright.

**Spec:** `docs/superpowers/specs/2026-10-28-insurance-pit-reports-design.md`

## Global Constraints

- Only posted payrolls: the period's `hrm.payroll_periods.posted_payroll_id` per legal entity and month.
- Missing posted payroll for any month covered (for D02-LT including M−1): fail with `report_payroll_missing`, `Params{"months": "MM/YYYY, MM/YYYY"}`.
- Unsupported period: `report_form_not_supported` — D02-LT before 2026-05, 05/KK before 2026-Q3, year reports before 2026.
- Both `hrm.salary.view` at the legal entity and `hrm.employee.sensitive` at each listed employee's **current** org unit; at run time and at every download (pin + `Check`).
- Nothing aggregated is stored; per-person amounts appear only in the file; identifiers are audited as `hrm.employee_sensitive_exported` with field names, never values.
- Money cells `int64` đồng; dates `dd/mm/yyyy` text (`dateLayout`); months `mm/yyyy` text; identifiers in text columns.
- Form labels are the official Vietnamese text, not translated; page text and error codes go through `vi` and `en` i18n.
- Everything English in code; comments short, why not what, no doc references.
- Do not run `make e2e`, `make dev`, `make reset-db`, `make seed`.

## Review Focus

1. An employee with a payroll line in some months but not others (hired or left mid-year) — year totals sum only the months they have; no zero rows for months they were absent. Test in Task 3 (`TestPIT05_1BKPartialYear`).
2. A payroll line with a negative adjustment (clawback) making a quarter's TNCT lower — sums stay signed, [17] counts only people with tax > 0. Test in Task 2.
3. An employee with no tax code — 05-1/BK [09] holds the national ID; with both empty the row still appears. Test in Task 3.
4. A dependant whose deduction months are open-ended (`deduction_to` null) or start before Y — clipped to 01/Y–12/Y; a dependant entirely outside Y is not listed. Test in Task 4.
5. A legal entity with no `legal_name`/`tax_code` — header falls back to `name`, blank tax code, export still succeeds. Test in Task 2.

---

### Task 1: Report reader, permissions, pin, and D02-LT

**Files:**
- Create: `internal/modules/hrm/queries/report.sql`, `internal/modules/hrm/report.go`, `internal/modules/hrm/report_types.go`, `internal/modules/hrm/report_layouts.go`, `internal/modules/hrm/report_d02lt.go`, `internal/modules/hrm/report_test.go`
- Modify: `internal/modules/hrm/types.go` (errors), `internal/modules/hrm/import.go` (`registerDataIO` calls `s.registerReports()`), `sqlc.yaml` only if queries files are listed explicitly, `internal/modules/hrm/i18n/{vi,en}.json` (error codes), `web/src/core/i18n` or wherever `hrm.errors.*` live (check with `grep -rn payroll_not_computed web/src`)

**Interfaces:**
- Produces:
  - `type ReportParams struct { LegalEntityID int64 `json:"legal_entity_id"`; Month string `json:"month,omitempty"` /* YYYY-MM */; Quarter string `json:"quarter,omitempty"` /* YYYY-Qn */; Year int `json:"year,omitempty"` }` in `report_types.go`.
  - `ErrReportPayrollMissing = &platform.Error{Status: http.StatusConflict, Code: "report_payroll_missing"}`, `ErrReportFormNotSupported = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "report_form_not_supported"}` in `types.go`.
  - `type reportMonth struct { Start time.Time; Lines map[int64]payrollLine; Units map[int64]int64 /* org unit on the line */ }`
  - `type reportBasis struct { Entity iam.OrgUnit; Months []reportMonth }`
  - `func (s *Service) reportBasis(ctx context.Context, entityID int64, first time.Time, months int) (reportBasis, error)` — refuses without `hrm.salary.view` at the entity (`platform.ErrForbidden`), then `ErrReportPayrollMissing` listing every missing month.
  - `type reportPerson struct { ID int64; Code, Name string; DateOfBirth *time.Time; HireDate time.Time; TerminationDate *time.Time; OrgUnitID int64; SocialInsuranceNo, TaxCode, NationalID string }`
  - `func (s *Service) reportPeople(ctx context.Context, ids []int64) (map[int64]reportPerson, error)` — refuses unless `hrm.employee.sensitive` at every person's current unit; decrypts with the existing `open`/`sealedOf`; audits `hrm.employee_sensitive_exported` per person with the non-empty field names, in `platform.InTx`, as `exportEmployees` does.
  - `type reportFile struct { LegalEntityID int64 `json:"legal_entity_id"`; EmployeeIDs []int64 `json:"employee_ids"` }`, `func (s *Service) checkReportFile(ctx context.Context, pin json.RawMessage) error` — same shape as `checkEmployeeFile`: no pin → forbidden; salary at entity; sensitive at each current unit; an id that no longer exists → forbidden.
  - `func (s *Service) reportSheet(name string, header []string, rows [][]any, text []int, entity int64, ids []int64) (dataio.Sheet, error)` — builds the sheet with the pin.
  - `func entityHeader(u iam.OrgUnit) (name, taxCode, address string)` — `legal_name` else `name`.
  - Export kinds registered in `registerReports()`: `hrm.report.d02lt` now; later tasks add theirs to the same function.
- Queries in `report.sql`: `ReportEmployees :many` (the columns of `reportPerson`, `WHERE id = ANY(@ids)`); `ReportContracts :many` (posted contracts of `@ids` whose `record.documents.legal_entity_id = @legal_entity_id`: id, employee_id, parent_id, start_date, end_date, the type's `fixed_term`). Reuse `store.PostedPayroll`, `store.PayrollLines`, `store.EmployeeUnits`.

- [ ] **Step 1: Write the failing tests** in `report_test.go`, on a fixture built from `newPayrollFixture` with posted payrolls of 2026-05 and 2026-06 (contracts, timesheets, `payroll(month)`, `post`). Helper `func (f *fixture) exportReport(ctx context.Context, kind, params string) [][]string` modelled on `exportEmployees` (RawCellValue rows), and `exportReportEnding(ctx, kind, params, want string)`.
  - `TestD02LTIncrease`: E hired 2026-06-01 with a contract from that day; D02-LT for 2026-06 has one row: B = E's name, (1) = E's social insurance number, (13) `01/06/2026`, (14) `06/2026`, (18) `Tăng lao động`, (11) `x`.
  - `TestD02LTDecrease`: E terminated 2026-05-31 (on May's line, not on June's): row (18) `Giảm lao động`, (13) `31/05/2026`.
  - `TestD02LTUnpaidMonth`: E has ≥ 14 unpaid working days in June (unpaid leave): row (18) `Nghỉ không lương, ốm đau từ 14 ngày`, (14) and (15) `06/2026`; July with E back is (18) `Đi làm lại`.
  - `TestD02LTAdjustment`: an addendum effective 2026-06-01 raises the salary from 22,000,000 to 25,000,000: (18) `Điều chỉnh tiền lương`, (4) `25000000`, (13) `01/06/2026`, (19) contains `22.000.000` and `25.000.000`.
  - `TestD02LTMoveBetweenEntities`: a second company C2; E's contract with C ends 2026-05-31 and one with C2 starts 2026-06-01; payrolls of both entities posted for May and June. C's D02-LT June: (18) `Giảm lao động`, (19) `Chuyển đơn vị`, (13) `31/05/2026`. C2's: (18) `Tăng lao động`, (13) `01/06/2026`. For C2, May's posted payroll exists and has no line for E.
  - `TestD02LTPreviousMonthMissing`: only June posted → job fails `report_payroll_missing`; the error's params name `05/2026`.
  - `TestD02LTBeforeForm`: month 2026-04 → `report_form_not_supported`.
  - `TestReportNeedsBothPermissions`: actor with `payroll@*` only → job fails `forbidden`; with `sensitive_viewer@*` only → `forbidden`.
  - `TestReportDownloadRechecks`: export as an actor with both roles granted at A; then move E to department B; `DataIO.File` → `forbidden`. Same after revoking `sensitive_viewer`.
  - `TestReportAuditsIdentifiers`: one `hrm.employee_sensitive_exported` audit row per listed employee (use `auditCount`).
- [ ] **Step 2: Run them to see them fail**: `go test ./internal/modules/hrm -run 'TestD02LT|TestReport' ` → compile errors (`unknown kind` once compiled).
- [ ] **Step 3: Implement the reader, permission check, pin and `checkReportFile`** in `report.go`, queries in `report.sql`, `make gen` (sqlc only is enough: run `make gen`, then check `git diff api/ web/src/shared/api/schema.gen.ts` shows nothing unexpected).
- [ ] **Step 4: Implement `layoutD02LT_QD366_2026`** in `report_layouts.go` — a `[]string` header, verbatim with codes:
  `STT`, `Họ và tên`, `(1) Mã số BHXH`, `(2) Ngày/ tháng/ năm sinh`, `(3) Chức vụ, chức danh nghề, nghề nghiệp, nơi làm việc`, `(4) Hệ số lương/ Tiền lương`, `(5) Hệ số chênh lệch bảo lưu`, `(6) Phụ cấp chức vụ`, `(7) Phụ cấp thâm niên VK (%)`, `(8) Phụ cấp thâm niên nghề (%)`, `(9) Các khoản phụ cấp lương và thu nhập bổ sung`, `(10) HĐLĐ không xác định thời hạn`, `(11) HĐLĐ xác định thời hạn`, `(12) HĐ, thỏa thuận khác`, `(13) Ngày Quyết định, HĐLĐ, HĐLV có hiệu lực`, `(14) Từ tháng/ năm`, `(15) Đến tháng/ năm`, `(16) Nơi đăng ký KCB ban đầu`, `(17) Hình thức nhận kết quả`, `(18) Loại tăng, giảm, điều chỉnh`, `(19) Ghi chú`.
  List forms (D02-LT, 05-1/BK, 05-3/BK) have no rows above the people: the column header is the sheet's first row, so the file stays importable. The entity and period go in the file name, `D02-LT <entity name> MM-YYYY` (05-1/BK: `05-1BK-QTT-TNCN <entity> YYYY`). Indicator forms carry the header fields as their first rows (Task 2).
- [ ] **Step 5: Implement `exportD02LT`** in `report_d02lt.go` following the spec's movement table. Decide per employee in `prev.Lines ∪ cur.Lines`, base = `Result.InsuranceBase`:
  ```
  prevBase, inPrev := prev[e]; curBase, inCur := cur[e]
  switch {
  case curBase > 0 && (!inPrev || prevBase == 0):
      increase; "Đi làm lại" when inPrev (employed, waived), else "Tăng lao động"
  case prevBase > 0 && (!inCur || curBase == 0):
      decrease; terminated in M−1 or M → "Giảm lao động"; !inCur → "Giảm lao động" + "Chuyển đơn vị";
      else "Nghỉ không lương, ốm đau từ 14 ngày" with (15) = M
  case prevBase > 0 && curBase > 0 && prevBase != curBase:
      adjustment
  }
  ```
  Effective dates and (4)/(9)/(10)/(11) from `ReportContracts` and the line's `Input.InsuredTerms` (allowances = sum of `Lines` with `Kind == "allowance"`); old/new amounts in (19) formatted with dot thousands (`"Từ 22.000.000 lên 25.000.000"`). Order: increases, decreases, adjustments, each by employee code.
- [ ] **Step 6: Add error codes** `report_payroll_missing` ("Chưa chốt bảng lương các tháng: {{months}}" / "No posted payroll for: {{months}}") and `report_form_not_supported` to the i18n where `payroll_not_computed` lives, both languages.
- [ ] **Step 7: Run** `go test ./internal/modules/hrm -run 'TestD02LT|TestReport'` → PASS; then `go test ./internal/modules/hrm` → PASS.
- [ ] **Step 8: Commit** `feat(hrm): export the D02-LT insurance movement list from posted payrolls`.

### Task 2: 05/KK-TNCN for a quarter

**Files:**
- Create: `internal/modules/hrm/report_pit.go`
- Modify: `report_layouts.go`, `report.go` (`registerReports`), `report_test.go`

**Interfaces:**
- Consumes: `reportBasis`, `reportSheet`, `entityHeader`, `checkReportFile` (Task 1).
- Produces:
  - `type indicator struct { STT, Label, Code string }`; `layout05KK_TT89_2026 []indicator` — the 31 rows [15]–[45] verbatim from the spec's research (labels as on the TT 89 form, e.g. `{"1", "Tổng số người lao động:", "[15]"}`, `{"", "Trong đó: Cá nhân cư trú có hợp đồng lao động", "[16]"}`, …, `{"9.3", "Trong đó: Tổng số thuế thu nhập cá nhân đã khấu trừ trên tiền phí mua bảo hiểm nhân thọ, …", "[45]"}`).
  - `type pitSums struct { People, Withheld, WithDependants int64; TNCT, Exempt, TNCTWithheld, Assessable, Tax int64 }` and `func sumPIT(months []reportMonth) (pitSums, map[int64]pitPerson)` where `pitPerson{ TNCT, Exempt, Insurance, Personal, DependentDed, Assessable, Tax int64; Dependents int; LastMonth int }` — per person sums over the months they have a line: TNCT = `Gross − Exempt`, Insurance = `employeeInsurance()`, Assessable = `Taxable`, Tax = `IncomeTax`; `Dependents` from the last month's `Input.Dependents`.
  - `func indicatorSheet(layout []indicator, values map[string]int64, head [][]any) [][]any` — rows `STT | Chỉ tiêu | Mã | Giá trị`; head rows first (`[04] Tên người nộp thuế`, `[05] Mã số thuế`, `[06] Địa chỉ`, `[01] Kỳ tính thuế` = `Quý n năm Y`), missing codes = 0.
  - Kind `hrm.report.pit_05kk`, params `{legal_entity_id, quarter: "2026-Q3"}`.

- [ ] **Step 1: Failing tests**
  - `TestPIT05KK`: two employees over Q3/2026 (payrolls July–September posted): one above the tax threshold, one below; one month with an adjustment of −1,000,000 on the taxed one. Assert, against sums of the posted lines read with `f.hrm.Payroll(f.pay, id)` lines: [15]=[16]=2, [17]=[18]=1, [19]=0, [20]=[21]=Σ(gross−exempt), [24]=[28]=Σexempt, [29]=[30]=Σ(gross−exempt) of the taxed one, [32]=[33]=Σtaxable, [35]=[42]=[43]=Σtax, [22],[23],[25]–[27],[31],[34],[36]–[41],[44],[45]=0.
  - `TestPIT05KKQuarterMissingMonth`: September not posted → `report_payroll_missing` naming `09/2026`.
  - `TestReportEntityHeader`: C has no `legal_name`/`tax_code`: the 05/KK head rows hold `C` and an empty tax code; after `iam.UpdateOrgUnit` sets legal name, tax code and address, they appear.
  - `TestPIT05KKBeforeTT89`: `2026-Q2` → `report_form_not_supported`.
- [ ] **Step 2: Run** `go test ./internal/modules/hrm -run TestPIT05KK` → FAIL.
- [ ] **Step 3: Implement** layout, `sumPIT`, `indicatorSheet`, `exportPIT05KK` in `report_pit.go`; register the kind. Pin all employee ids of the quarter.
- [ ] **Step 4: Run** → PASS; whole package → PASS.
- [ ] **Step 5: Commit** `feat(hrm): export the quarterly PIT return 05/KK-TNCN`.

### Task 3: 05/QTT-TNCN and 05-1/BK for a year

**Files:**
- Modify: `report_pit.go`, `report_layouts.go`, `report.go`, `report_test.go`, `payroll_calc.go` (only if `incomeTax` needs exporting within the package — it does not; same package)

**Interfaces:**
- Consumes: `sumPIT`, `indicatorSheet`, `reportPeople` (Tasks 1–2); `legalParamsAt(ctx, "Y-12-31")`, `parseBrackets`, `incomeTax` (existing).
- Produces:
  - `func yearBrackets(b []bracket) []bracket` — every `Upper` × 12.
  - `type finalised struct { Authorised bool; Due, Over, Under int64; Waived bool }`, `func finalise(p pitPerson, authorised bool, brackets []bracket) finalised` — Due = `platform.Round(incomeTax(yearBrackets(b), Assessable))` where Assessable = max(0, TNCT − (Personal+DependentDed) − Insurance); Over = max(0, Tax−Due); Under = max(0, Due−Tax); Waived = 0 < Under ≤ 50_000.
  - `layout05_1BK_TT89_2026 []string` — `[06] STT`, `[07] Họ và tên`, `[08] Mã số thuế`, `[09] Số ĐDCN/ Hộ chiếu (trường hợp chưa có MST)`, `[10] Cá nhân uỷ quyền quyết toán thay`, `[11] CN nước ngoài ủy quyền quyết toán dưới 12 tháng`, `[12] Tổng thu nhập chịu thuế phát sinh trong kỳ`, `[13] Trong đó: TNCT được miễn theo Hiệp định`, `[14] Trong đó: TNCT được miễn theo quy định của Hợp đồng dầu khí`, `[15] Số lượng NPT tính giảm trừ`, `[16] Tổng số tiền giảm trừ gia cảnh`, `[17] Từ thiện, nhân đạo, khuyến học`, `[18] Bảo hiểm được trừ`, `[19] Quĩ hưu trí tự nguyện được trừ`, `[20] Thu nhập tính thuế`, `[21] Tổng số thuế TNCN đã khấu trừ`, `[22] Tổng số thuế phải nộp`, `[23] Số thuế đã nộp thừa`, `[24] Số thuế còn phải nộp`, `[25] Cá nhân có số thuế được miễn do có số thuế còn phải nộp từ 50.000 đồng trở xuống`.
  - `layout05QTT_TT89_2026 []indicator` — [15]–[36] (section I) and [37]–[45] (section II) verbatim, including the published STT jump from 6 to 8.
  - Kinds `hrm.report.pit_05_1bk`, `hrm.report.pit_05qtt`, params `{legal_entity_id, year: 2026}`. Authorised = has a line in December's posted payroll. Ticks are `"x"`.

- [ ] **Step 1: Failing tests**
  - `TestFinaliseUsesYearSchedule` (internal test in package `hrm`, file `report_internal_test.go`): with the baseline 2026 brackets (`[[10000000,0.05],[30000000,0.1],[60000000,0.2],[100000000,0.3],[null,0.35]]`), `finalise` of Assessable 500,000,000 → Due = 120M×5% + 240M×10% + 140M×20% = 6,000,000 + 24,000,000 + 28,000,000 = 58,000,000; with Tax 57,960,000 → Under 40,000, Waived true; with Tax 58,100,000 → Over 100,000, Waived false.
  - `TestPIT05_1BK`: a 12-month fixture is costly; post January–December 2026 for one employee with a dependant from 2026-03 (all months need timesheets: use a helper `postYear(f, emps)` looping `timesheet`+`payroll`+`post`). Row for E: [08] tax code, [10] `x`, [12] = Σ(gross−exempt), [15] = 1, [16] = Σ(personal+dependent deduction), [18] = Σ employee insurance, [20] = [12]−[16]−[18], [21] = Σ tax, [22]–[24] from `finalise`; total row sums [12]–[24].
  - `TestPIT05_1BKPartialYear`: F terminated 2026-06-30: listed with income of January–June, [10] empty, [22]–[25] empty.
  - `TestPIT05_1BKNoTaxCode`: F has only a national ID → [08] empty, [09] the ID.
  - `TestPIT05QTT`: same fixture; [15]=[16]= employees paid in Y; [21] people with a dependant deduction; [37] authorised count; [38] their Σtax; [40] Σ[22]; [41] Σ[24] of waived; [44]/[45] by the formula.
  - `TestPIT05QTTYearBefore2026`: year 2025 → `report_form_not_supported`; year 2026 with November missing → `report_payroll_missing` naming `11/2026`.
- [ ] **Step 2: Run** `go test ./internal/modules/hrm -run 'TestFinalise|TestPIT05_1BK|TestPIT05QTT'` → FAIL.
- [ ] **Step 3: Implement** `yearBrackets`, `finalise`, the two layouts and the two exports; brackets from `legalParamsAt(ctx, fmt.Sprintf("%d-12-31", y))`.
- [ ] **Step 4: Run** → PASS; package → PASS.
- [ ] **Step 5: Commit** `feat(hrm): export the PIT finalisation 05/QTT-TNCN and its 05-1/BK list`.

### Task 4: 05-3/BK dependants

**Files:**
- Modify: `report_pit.go`, `report_layouts.go`, `report.go`, `report_test.go`

**Interfaces:**
- Consumes: `reportBasis`, `reportPeople`, `openDependent`, `store.PayrollDependents`.
- Produces: `layout05_3BK_TT89_2026 []string` — `[06] STT`, `[07] Họ và tên người nộp thuế là người lao động tại tổ chức trả thu nhập`, `[08] MST của người nộp thuế là người lao động tại tổ chức trả thu nhập`, `[09] Họ và tên người phụ thuộc`, `[10] Ngày sinh người phụ thuộc`, `[11] MST của người phụ thuộc`, `[12] Loại giấy tờ (Số ĐDCN/Hộ chiếu/GKS) người phụ thuộc`, `[13] Số giấy tờ`, `[14] Quan hệ với người nộp thuế`, `[15] Từ tháng`, `[16] Đến tháng`; kind `hrm.report.pit_05_3bk`; `func clipMonths(from, to *string, year int) (first, last string, ok bool)` — `YYYY-MM` bounds clipped to the year, `ok` false when outside; output `mm/yyyy`.
- Rules: ID number with 10 or 13 digits after removing `-` → [11]; otherwise [12] `Số ĐDCN`, [13] the number. Relationship labels: `child` "Con", `spouse` "Vợ/chồng", `parent` "Cha/mẹ", `other` "Khác" (official form text, Vietnamese in code as form data). The dependants read is audited as `hrm.dependents_viewed` per employee (existing action name).

- [ ] **Step 1: Failing tests** `TestPIT05_3BK`: dependants with (2025-06, null) → `01/2026`–`12/2026`; (2026-03, 2026-08) → `03/2026`–`08/2026`; (2024-01, 2025-12) → not listed; ID `0123456789` → [11]; ID `001099012345` (12 digits) → [12]/[13]. Plus `TestClipMonths` table test in `report_internal_test.go`.
- [ ] **Step 2: Run** → FAIL. **Step 3: Implement.** **Step 4: Run** → PASS.
- [ ] **Step 5: Commit** `feat(hrm): export the PIT dependant list 05-3/BK`.

### Task 5: "Báo cáo" page

Use the `mmerp-ui` and `new-screen` skills; finish with `docs/ui.md`'s checklist.

**Files:**
- Create: `web/src/hrm/pages/ReportsPage.tsx`, `web/e2e/hrm-reports.spec.ts`
- Modify: `web/src/hrm/index.ts` (nav item `{ label: 'hrm.nav.reports', path: 'reports', icon: IconReportAnalytics, permission: 'hrm.salary.view', group: 'hrm.nav.group_reports' }` appended after payrolls), `web/src/hrm/routes.tsx` (`reports` route: `useCan('hrm.salary.view') && useCan('hrm.employee.sensitive')` else `ForbiddenPage`), `web/src/hrm/i18n/{vi,en}.json` (append keys only)

**Interfaces:**
- Consumes: `ExportButton` (`@/shared/jobs`), `LegalEntityField`/`useLegalEntities({ product: 'hrm', permission: 'hrm.salary.view' })`, `MonthField`, a page template from `shared/ui/page`.
- The page: one legal-entity picker at the top; five rows (form number, plain name, period input, `ExportButton` with `kind` and `params`). Period inputs: month (D02-LT), quarter select `YYYY-Qn` (05/KK) for the last 8 quarters from 2026-Q3, year select from 2026 to the current year (three year reports). The button is disabled until entity and period are chosen. Plain names (vi): "Danh sách tham gia, điều chỉnh BHXH, BHYT, BHTN (D02-LT)", "Tờ khai khấu trừ thuế TNCN theo quý (05/KK-TNCN)", "Tờ khai quyết toán thuế TNCN (05/QTT-TNCN)", "Bảng kê thu nhập, thuế từng người (05-1/BK-QTT-TNCN)", "Bảng kê người phụ thuộc (05-3/BK-QTT-TNCN)"; one line under the title: "Số liệu lấy từ các bảng lương đã chốt."

- [ ] **Step 1: Write the e2e spec** `hrm-reports.spec.ts` (the lead runs it): as the admin with `payroll` and `sensitive_viewer` granted, open `/hrm/reports`, pick the company and a month with no posted payroll, click D02-LT's export → the error notification names the missing month. A user with only `payroll` sees the forbidden page.
- [ ] **Step 2: Implement** page, route, nav, i18n.
- [ ] **Step 3: Run** `cd web && pnpm tsc --noEmit && pnpm vitest run` → PASS; `make gen` leaves no diff.
- [ ] **Step 4: Commit** `feat(web): HRM reports page for insurance and PIT filings`.

### Task 6: Docs and finish

**Files:**
- Modify: `products/hrm.md` (append a "Reports" section after "Business lines": the five exports, the source rule, the permission rule, the forms' legal basis; the "Still open" items from the spec appended to the payroll "Still open" list), `CONTEXT.md` (HRM section: append "Insurance movement list (D02-LT)", "PIT return (05/KK)", "PIT finalisation (05/QTT)", "Finalisation authorisation" with one-line meanings), `web/src/hrm/index.ts` already done.

- [ ] **Step 1:** Edit the docs (append only).
- [ ] **Step 2:** `go test ./...` → PASS; `make lint` → PASS; `cd web && pnpm tsc --noEmit && pnpm vitest run` → PASS.
- [ ] **Step 3:** Commit `docs(hrm): document the insurance and PIT reports`.
- [ ] **Step 4:** Fresh reviewer over `git diff main...HEAD` (pr-review-toolkit:code-reviewer); fix what holds; record its verdict for the PR body.
- [ ] **Step 5:** Push `feat/insurance-pit-reports` to `github`; open the PR into `main` per house style, ending with what the lead must check: no migrations; shared files touched (`products/hrm.md`, `CONTEXT.md`, `web/src/hrm/index.ts`, `routes.tsx`, i18n); e2e spec `web/e2e/hrm-reports.spec.ts` added, not run; the termination-reason integration point in `report_d02lt.go`.
