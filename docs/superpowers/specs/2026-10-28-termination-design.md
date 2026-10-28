# Termination workflow and severance allowance

Date: 2026-10-28 · Status: draft, awaiting approval (lead session, on the user's behalf)

## Intent

Today an employee leaves when HR types a `termination_date` on the profile. Nothing records why, who decided, what was handed over, or what the company owes. HR keeps the decision in Word and the severance calculation in Excel, and nothing warns them that a contract is still in force or that a leave request runs past the last day.

This work makes a termination a document (`hrm.termination`, "Quyết định nghỉ việc"): reason, decision, handover, approval, attachments. Posting it is what sets the employee's termination date. The payroll of the termination month pays the severance allowance (trợ cấp thôi việc, Article 46 Labour Code 2019) or the job-loss allowance (trợ cấp mất việc làm, Article 47), next to the untaken leave it already pays. HR can print the decision.

The first release is the minimum that lets an SMB stop using Excel for this job.

## Goals

### The document

- Type `hrm.termination`, number prefix `NV`, numbered per legal entity and year like other HRM documents. One document per employee; its **date is the termination date**: the last day employed, the day the payroll stops paying (as `termination_date` means today). Its legal entity and org unit are the employee's (legal entity of the employee's org unit), like contracts.
- Fields:
  - employee (required)
  - termination date (required, not before the hire date: `termination_before_hire`)
  - reason (required, one of the fixed codes below)
  - decision date (optional; the date the decision is signed, printed on the decision)
  - pension eligible (bool, default false; Art 46.1 excludes it)
  - months in unemployment insurance (BHTN) (required integer ≥ 0, from the BHXH book)
  - months already paid severance or job-loss allowance by this employer (integer ≥ 0, default 0)
  - handover notes (free text)
  - note (free text)
- Attachments: the signed decision, the resignation letter, the handover minutes. View with view, attach with edit, also once posted (papers are often signed after entry), as contracts do.
- Discussion and audit like every HRM document. Approval field: `reason` (choice), so a tenant can route e.g. dismissals to a director.
- Permissions `hrm.termination.view`, `hrm.termination.edit`, both in the `hr` role. The employee's own account is a subject (never approves it) and has no grant (does not see it in the personal space) *(my call: the employee gets the printed decision)*.
- **At most one non-cancelled termination per employee.** Creating a second one while another is draft, pending or posted is refused (`termination_exists`, naming the other document). Posting re-checks it under the employee lock.

### Reasons and which allowance they pay

Reasons are fixed in code (law, not tenant data), stable snake_case codes stored in `hrm.terminations.reason` and documented in `products/hrm.md` so the BHXH/PIT reports (feature A) can read them. Labels are i18n keys. The allowance kind sits next to each code as data, and a test covers every row.

| Code | Labour Code 2019 | Allowance |
| --- | --- | --- |
| `contract_expired` | Art 34.1, contract expires | severance |
| `work_completed` | Art 34.2, work under the contract completed | severance |
| `mutual_agreement` | Art 34.3, both parties agree | severance |
| `imprisonment` | Art 34.4, sentenced to prison without suspension, or barred from the work | severance |
| `foreigner_expelled` | Art 34.5, foreign employee expelled | none |
| `death` | Art 34.6, employee died, declared dead, missing or without civil capacity | severance |
| `employer_ceased` | Art 34.7, individual employer died / employer ceased operating | severance |
| `dismissal` | Art 34.8, disciplinary dismissal (sa thải) | none |
| `employee_resignation` | Art 34.9, employee terminates unilaterally (Art 35) | severance |
| `employer_unilateral` | Art 34.10, employer terminates unilaterally (Art 36), except 36.1(e) | severance |
| `absent_without_reason` | Art 34.10 via Art 36.1(e), absent 5+ consecutive working days without reason | none (Art 46.1 excludes it) |
| `restructuring` | Art 34.11, Art 42/43 (restructuring, economic reasons, merger) | job loss (Art 47) |
| `work_permit_expired` | Art 34.12, foreign employee's work permit expires | none |
| `probation_failed` | Art 34.13, probation not passed or cancelled | none |

- `pension_eligible = true` turns a severance into none (Art 46.1: "trừ trường hợp đủ điều kiện hưởng lương hưu"). It does not touch the job-loss allowance, which Art 47 does not exclude.
- Retirement itself is not a separate reason *(my call)*: retiring is usually `contract_expired` or `mutual_agreement` with `pension_eligible`. The UI says so next to the checkbox.

### How the allowance is computed

All from Art 46–47 Labour Code 2019 and Art 8 Decree 145/2020 (NĐ 145/2020):

1. **Eligibility:** the employee worked for this employer for at least 12 months: months worked (step 2) ≥ 12. Otherwise none (Art 46.1, 47.1: "làm việc thường xuyên … từ đủ 12 tháng trở lên").
2. **Months worked** = whole months from the hire date to the termination date inclusive; leftover days are dropped *(assumption, Still open)*. The profile's hire date stands for the whole time worked for this employer *(assumption: probation and earlier contracts with the same employer are before it; a rehire after a gap is out of scope)*.
3. **Months counted** = months worked − months in BHTN − months already paid, floored at 0 (Art 46.2 Labour Code; Art 8.3 NĐ 145/2020). Months in BHTN are entered by HR from the BHXH book and are not derived from mmerp's payrolls: mmerp has no history before it was installed.
4. **Years counted** = whole years of months counted, plus a remainder of 1–6 months counted as ½ year and 7–11 months as 1 year (Art 8.3 NĐ 145/2020: "tháng lẻ ít hơn hoặc bằng 06 tháng được tính bằng 1/2 năm; trên 06 tháng được tính bằng 01 năm").
5. **Average salary** = average over the 6 consecutive calendar months before the termination month of the contract amount in force on each month's last day, salary + salary allowances (the payroll's insured amount, `ContractTerms.insured()`), benefits excluded (Art 8.5 NĐ 145/2020; Art 3 Circular 10/2020/TT-BLĐTBXH leaves allowances like meals, fuel, phone out of salary). Several contracts or addenda over those months: each month takes the one in force. A month with no contract in force is left out of the average *(assumption)*; none at all gives none.
6. **Amount:** severance = years × ½ × average; job loss = max(years × 1, 2) × average (Art 47.1: "nhưng ít nhất bằng 02 tháng tiền lương"). Rounded once, to the dong, with the platform rounding.
7. **Not subject to insurance** (not part of the insured salary). **Not taxable**: Art 2.2(b) Circular 111/2013/TT-BTC (the severance and job-loss allowances paid under the Labour Code are not counted in taxable income from wages); in the payroll it is gross income and exempt income, as untaken leave already is.
8. **Payment deadline** is Art 48.1: within 14 working days of termination (30 days in the listed exceptions). Operating note only; mmerp does not enforce or track payment.

### Payroll

- The payroll that has the employee's line in the period holding the termination date pays the allowance (the basis finds the termination by employee, whatever the payroll's legal entity), as it already pays untaken leave. A termination in a closed period cannot be posted (below), so no adjustment path is needed.
- **Inputs stay captured.** The non-sensitive inputs go into the payroll basis, hence into `inputs_hash`: per leaving employee the allowance kind, months worked, months in BHTN, months already paid, pension flag. The posted termination is a payroll source (`payroll_sources`, its version in the hash), so cancelling or re-posting it marks a draft payroll stale and blocks closing, under the existing rules.
- The 6 months of contract terms are decrypted in `payInputs` (each contract read is audited, as today) and captured, encrypted, in the payroll line's input as `severance: {kind, years, monthly: [6 amounts], average}`. `PayrollAmounts` gets `severance`; `gross` and `exempt` include it; insurance does not.
- Posting lines need nothing new: the allowance is part of `salary_expense` and `salary_payable`, aggregated like the rest.
- The payroll screen and payslip show a "Trợ cấp thôi việc / mất việc" line when non-zero. The payroll export gets the column.
- **Payroll samples:** two new C rows in 04/2026, each justified article by article in the "Giả định" sheet:
  - mutual agreement, hired 01/07/2007, terminated 15/04/2026, BHTN from 01/01/2009: months worked 225, BHTN 207 → 18 months counted → 1.5 years × ½ × average (with an addendum raising the salary in 01/2026 so the average spans two amounts);
  - restructuring, hired 01/03/2024, all months in BHTN: years 0 → the 2-month floor.
  They need the accountant's review like the existing ones.

### Effects when it is posted and cancelled

- `OnTransition` to `posted`, in this lock order: record row (by `record`) → `hrm.payroll_locks` row `FOR SHARE` with the payroll period check on the **affected range** [termination date, ∞) (a later termination date changes the pay of every period from its month on) → employee row `FOR NO KEY UPDATE` (`LockEmployee`) → nothing further (leave balances are only read by the payroll). Under the lock: refuse if another termination of the employee is posted (`termination_exists`) or the termination date is before the hire date; refuse if the employee's org unit now belongs to another legal entity than the document's (`termination_legal_entity_changed`: recreate it) *(my call)*; then keep the employee's current `termination_date` in the termination row (`previous_date`, usually null) and set it to the termination date, audited as an employee change.
- **Cancelling** a posted termination: same locks and the same payroll period check (so a termination already paid by a closed payroll cannot be cancelled: `payroll_period_closed`), then restore `termination_date` to `previous_date` (null for anyone terminated only through documents) *(my call: a legacy date survives a post and cancel)*.
- A draft or pending termination changes nothing on the employee.
- Posted contracts are **not** changed. The payroll already stops at the termination date. A contract in force after it is a warning.

### Warnings

The termination detail lists, without refusing anything, recomputed on every read:

- `contract_in_force_after`: a posted original contract of the employee whose end date is after the termination date, or has none (number, end date).
- `leave_after_termination`: a draft, pending or posted leave request whose end date is after the termination date (number, dates, status).
- `overtime_after_termination`: same for overtime requests.
- `leave_balance_to_pay`: the remaining balance of the termination year, which the payroll will pay (days). Shown as info. Accrual and pro rata belong to the leave-accrual work; this reads whatever the balance is.

The allowance preview (kind, months worked, months counted, years; and the amount and average only for holders of `hrm.salary.view`, audited as a contract terms read) shows on the detail so HR sees the figure before the payroll runs. The kind and years show to anyone who can view the termination.

### Employee profile

- `termination_date` becomes read-only on edit: the profile form shows it with a link to the termination; `UpdateEmployee` refuses a changed value with `termination_via_document` (422).
- Creating an employee may still carry a termination date *(my call: loading former employees' history, which has no termination document)*.
- Employee import: a changed termination date on an existing employee is a row error `termination_via_document`; empty or unchanged cells keep the current rules (empty keeps, unchanged rows skip). New rows may carry one, as on create.
- Existing employees with a date and no document stay as they are. They may still get a termination (the action shows for them too); posting it replaces the legacy date, which is how a wrong legacy date is corrected. Clearing a legacy date has no path *(accepted: rare, and a terminated profile is still readable)*.
- The profile page gets an action "Cho nghỉ việc" (shown when the actor may create terminations and the employee has no non-cancelled one) opening the new-termination form with the employee filled in; and shows the current termination (number, status) when there is one.

### Screens

- Menu Nhân sự → "Nghỉ việc" (`/hrm/terminations`): list with number, employee, termination date, reason, status; filters status, org unit, search. No money.
- New and detail pages per the `new-document-type` skill (`DocumentPage`, editable panels while draft, approval panel, activity, attachments). The detail shows the warnings block and the allowance preview.
- Area manifest `recordTypes` entry with `path`, `preview`, `invalidate` (termination detail and list, the employee profile).

### Printing (last task)

- Template `hrm.termination` ("Quyết định chấm dứt hợp đồng lao động"), same pattern as the contract print: editable blocks (preamble/grounds, articles, footer), data: decision number and date, employer name, employee name, code, position/unit, termination date, reason label with its article, allowance kind (no amount: the amount comes from the payroll). Printing needs view.

## Non-goals

- Notice-period checks (Art 35, 36), payment tracking of the allowance, the employer's damages or the employee's compensation for unlawful termination (Art 40, 41).
- Rehiring a former employee, multiple employment periods per profile.
- Changing posted contracts; social insurance book handover tracking; a handover checklist.
- BHXH decrease reports and PIT certificates (feature A reads `termination_date` and, later, `hrm.terminations.reason`).
- Leave accrual and pro rata of the balance (feature C).
- The employee seeing their termination in the personal space.

## Anti-goals

- Two sources of truth for an employee's termination date (except the legacy rows already there).
- A severance amount in any unencrypted place: not in the termination row, not in posting lines, not in audit data, not in logs.
- A termination that changes a closed pay period.

## Constraints

- One transaction writes one document; lock order record → `hrm.payroll_locks` (`FOR SHARE`) → employee (`FOR NO KEY UPDATE`) → leave balances.
- Payroll inputs are captured and hashed; never recalculated automatically.
- The type passes `recordtest`; `allowed_actions` on every response.
- Shared files touched minimally and listed in the PR: `employee.go`, `employee_io.go`, payroll files, `products/hrm.md`, `CONTEXT.md`, i18n, `web/src/hrm/index.ts`.
- One migration (`make migration name=hrm_terminations`): new table `hrm.terminations` only; no change to `hrm.employees`, `hrm.leave_balances` or `hrm.payroll_*`.

## Still open (accountant / labour office)

- Leftover days in months worked: dropped here; some practice rounds a part month up.
- A month without a contract in force among the 6: left out of the average here.
- Whether the 6 months end with the month before termination (here) or include the termination month when it ends on its last day.
- The two new payroll samples, like all samples, need the accountant's review. Not blocking: they ship marked as awaiting review, as the existing 28 did.

## Left to the builder

Table and column names, the shape of the warnings response, the exact rounding call, how `payInputs` reads the 6 months of contracts (a second query for leaving employees only, so other employees' audit rows do not grow), UI copy within `mmerp-ui`.
