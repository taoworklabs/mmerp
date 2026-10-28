# Insurance and personal income tax reports: D02-LT, 05/KK-TNCN, 05/QTT-TNCN

Date: 2026-10-28 · Status: approved by the lead session, 2026-10-28

## Intent

Once payroll is closed in mmerp, HR and the accountant still rebuild three filings by hand in Excel from the payroll export: the monthly list of insurance increases, decreases and salary adjustments for the social insurance (BHXH) office, the quarterly personal income tax (PIT) return, and the year-end PIT finalisation with its per-employee and dependant appendices. Every figure they need is already in the posted payrolls and the employee profiles. This work produces those filings as flat Excel sheets that carry every figure of the official forms under their official column and indicator codes, so they can be checked, keyed into eBHXH/HTKK/eTax (or imported where those tools accept Excel) and filed, without a parallel spreadsheet. The printed layout (title, merged headings, signature block) stays with the filing tools.

The forms changed in 2026, and this design follows the versions in force on 2026-10-28 (sources in [Legal basis](#legal-basis)):

- **D02-LT** "Danh sách tham gia, điều chỉnh đóng BHXH, BHYT, BHTN", Quyết định 366/QĐ-BHXH of 29/4/2026, in force since 01/05/2026 (replaces Quyết định 595/2017 and its amendments 1040/2020, 490/2023, 948/2023). One flat list; each row says its movement in column 18.
- **05/KK-TNCN**, **05/QTT-TNCN**, **05-1/BK-QTT-TNCN**, **05-3/BK-QTT-TNCN**, Thông tư 89/2026/TT-BTC of 30/6/2026, in force since 01/07/2026 (replaces Thông tư 80/2021). Since 01/07/2026 every employer files 05/KK **quarterly** (Thông tư 89/2026 Art. 22.1.a.1); Q3/2026 is the first such quarter.

## Goals

### One "Báo cáo" page

- A new HRM menu group "Báo cáo" with one page, `/hrm/reports`, listing the reports. Each report is one row: what it is (the form number and its plain name), the inputs it needs (legal entity, then month, quarter or year), and a download button that starts a `dataio` export job and follows it like the payroll export.
- The page and the menu entry show for actors who hold both `hrm.salary.view` and `hrm.employee.sensitive` somewhere (from an `actions` endpoint, as other HRM lists do). The legal entity picker lists the legal entities where the actor holds `hrm.salary.view`.

### Five exports, all per legal entity

Each is a `dataio` export (`Run` + `Check`) with its own kind, one sheet each, because `dataio` writes one sheet per file and that stays *(my call: five small kinds over teaching core multi-sheet files)*:

| Kind | Form | Period param |
| --- | --- | --- |
| `hrm.report.d02lt` | D02-LT (QĐ 366/2026) | month |
| `hrm.report.pit_05kk` | 05/KK-TNCN (TT 89/2026) | quarter |
| `hrm.report.pit_05qtt` | 05/QTT-TNCN (TT 89/2026) | year |
| `hrm.report.pit_05_1bk` | 05-1/BK-QTT-TNCN, table 1 (TT 89/2026) | year |
| `hrm.report.pit_05_3bk` | 05-3/BK-QTT-TNCN (TT 89/2026) | year |

- **Source data.** Only **posted** payrolls of the legal entity (the payroll in each period's `posted_payroll_id`), their decrypted lines, the employee profile (name, date of birth, hire and termination dates, and the decrypted social insurance number, tax code and national ID) and dependants. Amounts are decrypted and summed in the service, per request; nothing aggregated is stored. Per-person rows exist only in the export files.
- **Every period the report covers must have a posted payroll** for that legal entity; otherwise the export fails with `report_payroll_missing` naming the missing months. For D02-LT that includes month M−1. A company that starts using mmerp mid-year therefore cannot produce the finalisation for that year from mmerp (see [Still open](#still-open)).
- **Layouts are data.** Each form's layout (title, header fields, indicator codes and labels, column codes and headings) is a Go value named after the form and its legal basis, e.g. `layoutD02LT_QD366_2026`, `layout05KK_TT89_2026`. A new version of a form is a new layout value chosen by the period it covers, never an edit of the old one. Only the TT 89/2026 and QĐ 366/2026 layouts are built now. D02-LT before 05/2026 and 05/KK before Q3/2026 are refused with `report_form_not_supported` *(my call: mmerp has no posted payrolls that old at any customer yet)*. The year reports pick their layout by the year they finalise: tax year 2026 onward uses TT 89, because it is filed in 2027 under TT 89 and the 2026 PIT law applies to salaries for the whole of 2026; earlier years are refused.
- **Flat sheets, not printed forms.** `dataio.Sheet` is one header row plus rows, and that stays: no core change. A list form (D02-LT, 05-1/BK, 05-3/BK) is one row per person under a header whose cells are the official headings prefixed by their column code (e.g. `[12] Tổng thu nhập chịu thuế phát sinh trong kỳ`, `(18) Loại tăng, giảm, điều chỉnh`), then a total row where the form has one. An indicator form (05/KK, 05/QTT) has the header `STT | Chỉ tiêu | Mã | Giá trị`, one row per indicator, and the header fields (legal entity, period) as the first rows. The file name carries the form, legal entity and period. The form's title, signature block and merged headings are left to the filing tools.
- Labels on the forms are the official Vietnamese text and are not translated; the report names on the page and error messages go through i18n in `vi` and `en`.
- **Header fields** come from the legal entity (`iam.org_units` of kind `company`): name (`legal_name`, else `name`), tax code, address. The BHXH unit code, the tax agent and "lần đầu/bổ sung" are not in mmerp and are left blank for HR to fill in, as are columns mmerp has no data for.
- Money cells are integer đồng; dates are `dd/mm/yyyy` text and months `mm/yyyy` text, like the employee export; the social insurance number, tax codes and ID numbers are text columns so leading zeros survive.

### D02-LT: increases, decreases, adjustments of month M

Rows come from comparing the posted payrolls of M and M−1 of the legal entity, by employee, on the payroll line's `insurance_base` (the insured salary: salary plus salary allowances, uncapped; zero when the 14-day rule waived contributions):

| M−1 | M | Row | Column 18 (Loại) | Column 13 (effective date) | Columns 14–15 |
| --- | --- | --- | --- | --- | --- |
| absent or 0, and not employed by this legal entity in M−1 | > 0 | increase | "Tăng lao động" | the later of the hire date and the start of the employee's first contract with this legal entity in force in M (a move from another legal entity starts with the contract) | from M |
| 0 (employed, contributions waived), and insured on an earlier posted payroll of this legal entity (or hired before its payroll history starts) | > 0 | increase | "Đi làm lại" | first day of M | from M |
| > 0 | absent or 0, `termination_date` in M−1 or M | decrease | "Giảm lao động" | termination date | from M |
| > 0 | absent, not terminated (moved to another legal entity) | decrease | "Giảm lao động", column 19 "Chuyển đơn vị" | end date of the last contract with this legal entity, else blank | from M |
| > 0 | 0, not terminated | decrease | "Nghỉ không lương, ốm đau từ 14 ngày" | first day of M | from M to M |
| > 0 | > 0, different | adjustment | "Điều chỉnh tiền lương" | effective date of the addendum in force for M | from M |

- Column 19 (Ghi chú) of an adjustment holds the old and new insured salary ("Từ X lên Y"). Columns 4 and 9 hold the new contract salary and the sum of salary allowances (from the line's `InsuredTerms`); a decrease shows the last values from M−1.
- Columns 10–12 mark "x" by the contract in force: no end date → 10; fixed-term → 11. Column 12 is never marked (no other agreement kinds in mmerp).
- Employees are ordered by employee code within each movement; increases, then decreases, then adjustments.
- The decrease reason uses `termination_date` only. The termination workflow (`feat/termination`) adds `hrm.terminations` with a stable reason code that this branch does not have; the lead wires the reason in after both land.
- Assumption: a mid-month termination counts as a decrease from the following month when the termination month still has a contribution (the payroll's 14-day rule decides this, not this report) *(see [Still open](#still-open))*.

### 05/KK-TNCN: quarter Q

Indicators [15]–[45] from the three posted payrolls of Q, one row per indicator (STT, label, code, value). Per person, summed over the quarter: taxable income `TNCT = gross − exempt`, assessable income `TNTT = taxable` (the payroll's `taxable`, after deductions), tax `income_tax`.

- [15] people with a payroll line in Q; [16] the same (every employee is a resident with a labour contract: the payroll supports nothing else).
- [17] = [18]: people with tax > 0 in Q; [19] = 0.
- [20] = [21]: sum of TNCT; [22], [23] = 0.
- [24] = [28]: sum of `exempt` (overtime premium within the limits, untaken leave paid on leaving, tax-free benefits); [25]–[27] = 0.
- [29] = [30]: sum of TNCT of people with tax > 0 in Q; [31] = 0.
- [32] = [33]: sum of TNTT; [34] = 0.
- [35], [42], [43]: sum of tax; [36]–[41], [44], [45] = 0.
- Back pay and clawbacks count in the period of the payroll that pays them (taxable income is recognised when paid); a payroll's period is treated as its payment period.

### 05/QTT-TNCN and its appendices: year Y

**Who is finalised** (assumption, [Still open](#still-open)): everyone with a line on the posted payroll of December Y has authorised the company to finalise for them (column [10] ticked); everyone else paid in Y is listed with their income and tax withheld but not finalised. There is no per-employee flag.

**05-1/BK table 1:** one row per employee paid in Y (authorised or not), ordered by employee code, then a total row ([26]–[39]):

- [08] tax code; [09] national ID only when there is no tax code.
- [12] sum of TNCT over Y; [13], [14] = 0; [11] blank.
- [15] number of dependants on the December line (or the last line in Y); [16] sum of `dependent_deduction` + `personal_deduction` over Y *(builder confirms against HTKK whether [16] includes the personal deduction; it does in the TT 80 instructions)*; [17], [19] = 0; [18] sum of the employee's insurance.
- [20] = max(0, [12] − [13] − [14] − [16] − [17] − [18] − [19]).
- [21] sum of tax withheld.
- Only for authorised people: [22] = yearly tax on [20] under the progressive schedule in `hrm.legal_params` in force at 31/12/Y (for 2026 the baseline's 5 brackets, 5/10/20/30/35%) with the monthly bracket bounds × 12, rounded to the đồng with the platform rounding; [23] = max(0, [21] − [22]); [24] = max(0, [22] − [21]); [25] ticked when 0 < [24] ≤ 50,000 đồng.
- Tables 2 and 3 are not produced (no transfers between entities, no exempt categories in mmerp).

**05/QTT-TNCN:** section I is computed like 05/KK over the twelve months, plus [20] = 0 and [21] = people with a dependant deduction in Y. Section II: [37] authorised people; [38] their tax withheld; [39] = 0; [40] sum of [22]; [41] sum of [24] of people with [25] ticked; [42], [43] = 0; [44] = max(0, [40] − [38] − [41] − [42] − [43]); [45] = max(0, −([40] − [38] − [41] − [42] − [43])).

**05-3/BK:** one row per dependant of an employee paid in Y whom a posted payroll of Y deducted: [07], [08] the employee's name and tax code; [09] name; [10] date of birth; [11] the ID number when it reads as a tax code (10 or 13 digits after removing a dash), otherwise [12] "Số ĐDCN" and [13] the number; [14] relationship (translated); [15], [16] the first and last month a posted payroll of Y deducted them, by the payroll's own rule (a set first month, only months the employee was paid in); a dependant no payroll of Y deducted is not listed.

### Permissions, audit, files

- Every report needs **both** `hrm.salary.view` at the legal entity and `hrm.employee.sensitive` at the **current** org unit of every employee it lists, checked when the job runs and again at every download *(my call: the same rule as the employee export; the identifiers the file reveals belong to the profile, which follows the current unit)*.
- The file pins what it holds (`dataio.Sheet.Pin`: legal entity and employee ids), and `Check` re-checks the pin, the way `checkEmployeeFile` does for the employee export: moving an employee out of the actor's reach or revoking either permission closes the file.
- Each export is audited by `dataio` (`dataio.exported`, with the params). The decrypted identifiers are audited like the employee export audits sensitive values, never with the values themselves.
- Export is an export operation for `ProductGate`: it works while HRM is disabled.

## Non-goals

- **Withholding certificates** (chứng từ khấu trừ thuế TNCN). Since Decree 123/2020/NĐ-CP (amended by Decree 70/2025/NĐ-CP) they are electronic documents with a numbering series registered with the tax office and a digital signature. A spreadsheet version is not legally usable; doing it properly is its own feature.
- Electronic filing to eBHXH, HTKK or eTax (XML, I-VAN, signatures). `products/hrm.md` already lists BHXH e-filing as "not yet".
- The TT 80/2021 forms and D02-LT before QĐ 366: periods before 07/2026 (PIT) and 05/2026 (D02-LT).
- Tables 2 and 3 of 05-1/BK, appendix 05-2/BK (non-residents), non-residents and contracts under 3 months: the payroll supports none of them.
- Changing how payroll computes, the 2026 PIT schedule or the 2026 family deduction amounts in `hrm.legal_params` (see [Still open](#still-open)).
- Storing reports, report history or an authorisation flag per employee.
- The termination record and its reason codes (feature B).

## Anti-goals

- A figure in a file that does not match the posted payrolls. If the data is incomplete (a missing month), refuse rather than produce a plausible but wrong file.
- Any way for an actor to get per-person amounts or decrypted identifiers they could not see on the payroll and profile screens.
- Aggregated or per-person report data stored anywhere, logged, or written to the audit data.

## Constraints

- No new table and no migration. Reads only `hrm` and lower tiers.
- Services read through the data engine (`store.<Query>.Run`); new queries go in a new `queries/report.sql`; the code in `report.go`, `report_types.go`, `report_handler.go`, `report_test.go` (area-first naming).
- Money is `int64` đồng; the yearly tax uses `decimal` and the single platform rounding.
- Tests against a real Postgres: one per movement kind of D02-LT, the missing-month refusals, the 05/KK and 05/QTT figures on a small fixture checked by hand (including a back pay and a 50,000 đồng case), the dependant months clipped to the year, permission refusal at run and at download (revoked, employee moved); a D02-LT for a month in which an employee moves between two legal entities (a decrease on one, an increase on the other); the refusal when M−1 has no posted payroll; the yearly tax of 05-1/BK under the 5-bracket schedule × 12. An e2e spec for the page: pick a legal entity and month, download D02-LT. The lead runs it.

## Legal basis

- D02-LT: Quyết định 366/QĐ-BHXH ngày 29/4/2026 (official template PDF/xlsx from baohiemxahoidientu.vn).
- 05/KK-TNCN, 05/QTT-TNCN, 05-1/BK-QTT-TNCN, 05-3/BK-QTT-TNCN: Thông tư 89/2026/TT-BTC ngày 30/6/2026 (form files from luatvietnam.vn; the TT 89 text itself not read from chinhphu.vn or gdt.gov.vn).
- Quarterly PIT filing for every employer from Q3/2026: Thông tư 89/2026 Art. 22.1.a.1 (secondary sources). Before, monthly by default and quarterly for those filing VAT quarterly (Decree 126/2020 Art. 8–9). The system offers quarters only and does not decide anything else.
- Withholding certificates: Decree 123/2020/NĐ-CP, Decree 70/2025/NĐ-CP.
- PIT law 109/2025/QH15 (5-bracket schedule for 2026) and Resolution 110/2025/UBTVQH15 (family deductions 15.5 and 6.2 million đồng from 01/01/2026).

## Still open

For an accountant or the BHXH office before loading real data:

- Authorisation: everyone on the December payroll is assumed to authorise the company to finalise for them.
- Under that assumption, someone who left in October but gets back pay on the December payroll counts as authorising.
- Whether the BHXH office accepts the D02-LT layout of the QĐ 366 PDF (no part A) or the eBHXH xlsx (with part A, unit details).
- The free-text values of D02-LT column 18; QĐ 366 lists kinds but no fixed codes.
- Whether a mid-month termination is a decrease from that month or the next (tied to the open "hired after the 15th" question in `products/hrm.md`).
- Whether [16] of 05-1/BK includes the personal deduction, and the formula for [36] of 05/KK.
- The employer's finalisation deadline for tax year 2026 under the Law on Tax Administration 108/2025/QH15.
- A company that starts using mmerp mid-year needs its earlier months' income and tax to finalise; importing that history is not built.
- The TT 89 forms were read from form files and secondary sources, not from the text on chinhphu.vn or gdt.gov.vn; the accountant checks the indicator mapping against the official text.
- HTKK does not yet ship the TT 89 05/KK (reported for HTKK 5.7.9); the files are checked against the form, not against HTKK's import.
