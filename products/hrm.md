# HRM product

Human resources management: employee profiles, contracts, leave and overtime, timekeeping, payroll. HRM is the first product, and every core capability is proven by an HRM flow ([backend.md](../docs/backend.md#layers), [ADR-0023](../docs/adr/0023-build-core-first.md)). Hard-to-reverse decisions are in [ADR-0007](../docs/adr/0007-hrm-product.md); terminology is in the HRM section of [CONTEXT.md](../CONTEXT.md#hrm).

## Scope

| In the first release | Not yet |
| --- | --- |
| Employee profiles, dependants | Recruitment, performance review, training |
| Employment contracts and addenda | Salary payment through banks |
| Leave requests, leave balances, overtime requests | Time clock integration |
| Timesheets: manual entry and Excel import | Electronic social insurance (BHXH) filing |
| Payroll: salary, social insurance (BHXH), health insurance (BHYT), unemployment insurance (BHTN), personal income tax (PIT, thuế TNCN) | Pay periods other than the calendar month |

## Module

A single module `internal/modules/hrm`, schema `hrm`, belonging to the `hrm` product. It is not split into several modules: all four parts share employees and pay periods. If it were split, then, because modules in `modules/` may not import each other, payroll would have to call timekeeping through an interface. Split only when another product needs one part on its own. Then apply the "promote an entity to shared" rule ([backend.md](../docs/backend.md#data-ownership)), e.g. employees move to `shared/`.

### Dependencies

- **Dependency on other products:** none.
- **Used directly:** `iam` (users, org tree, `Scope`), `record`, `setting`, `audit`, `dataio`, `shared/posting`. `numbering` issues employee codes (series `hrm.employee`); `approval` and document numbers work only through `record`.
- **Hooks emitted:** none yet. The first hook may be "employee terminated", but it is added only when another module needs to react.

## Entities

| Table | Meaning |
| --- | --- |
| `hrm.employees` | Employee profile. Optional link to `iam.users.id`: an employee with an account submits their own requests; one without has HR enter them on their behalf. `org_unit_id` is the department; `manager_id` is the direct manager |
| `hrm.dependents` | Dependants for the family deduction (giảm trừ gia cảnh) |
| `hrm.contract_types` | Contract types, with a fixed-term flag |
| `hrm.contracts` | Employment contracts and addenda: type, term, salary, allowances |
| `hrm.leave_types` | Leave types, with a deducts-from-balance flag, a company-paid flag, an optional cap on the days of one request and an attachment-required flag |
| `hrm.leave_requests` | Leave requests |
| `hrm.leave_balances` | Leave balances per employee and year |
| `hrm.overtime_requests` | Overtime requests |
| `hrm.timesheets`, `hrm.timesheet_lines` | Timesheets per pay period and org unit |
| `hrm.payroll_locks` | One row per legal entity, used only for locking (see [Payroll lock](#payroll-lock)) |
| `hrm.payroll_periods` | Pay periods per legal entity, with `posted_payroll_id`: the payroll currently in effect for the period |
| `hrm.payrolls`, `hrm.payroll_lines`, `hrm.payroll_sources` | Payrolls per period and legal entity, with `inputs_hash` (see [Input snapshot](#input-snapshot)); captured inputs; list of source documents with their versions |
| `hrm.payroll_adjustments` | Back-pay and clawback amounts on a draft payroll; amount and reason encrypted |
| `hrm.legal_params` | Legal parameters with effective dates |
| `hrm.work_weeks`, `hrm.holidays` | Work calendar per legal entity: work-week versions; public holidays and substitute days off |
| `hrm.terminations` | Terminations: employee, reason code, decision date, months in unemployment insurance and already paid, pension flag, handover; the termination date is the document's date (see [Termination](#termination-1)) |
| `hrm.pit_authorisations` | Who authorised which legal entity to finalise their PIT for a year, one per employee and year (see [Reports](#reports)) |

Departments and branches are nodes in the `iam` permission-scope tree, with `kind` `department` and `branch` respectively. Which department an employee belongs to, and who their direct manager is, are HRM business relations, independent of the permission-scope tree.

## Record types

| Type code | Kind | `posted` means | Date used for period lock | `OnTransition` | Approval fields |
| --- | --- | --- | --- | --- | --- |
| `hrm.employee` | master data | — | — | — | — |
| `hrm.contract` | document | Contract in effect | Effective date | Payroll period check | `contract_kind` |
| `hrm.leave_request` | document | Request granted | Leave start date | Deduct from leave balance (cancelling refunds it); payroll period check | `days`, `leave_type` |
| `hrm.overtime_request` | document | Overtime granted | Overtime date | Payroll period check | `hours`, `day_kind` |
| `hrm.timesheet` | document | Timesheet closed | `period_end` | Payroll period check | — |
| `hrm.payroll` | document | Payroll closed | `period_end` | Close the pay period; `posting.Record` (cancelling runs `posting.Void` and reopens the period) | — |
| `hrm.termination` | document | Termination in force | Termination date | Payroll period check from the earlier of the new and replaced dates on; set the employee's termination date (cancelling restores the one it replaced) | `reason` |

- Every document type uses approval and audit. Employee profiles use audit. Excel import/export through `dataio` currently covers: employee profile export and import (`hrm.employees`: the export's 15 columns are the import template; the employee code is the key, an empty cell keeps the current value, a changed termination date on an existing employee is a row error (only a termination changes it), sensitive columns are exported only on request and need `hrm.employee.sensitive`), timesheet import and export, payroll export, leave balance adjustment import.
- Discussion is available on every record type. Attachments are only on contracts, leave and overtime requests, terminations (any viewer of the termination; it holds no amount) and employee profiles ([ADR-0024](../docs/adr/0024-attachments-and-discussion.md)):

  | Type | View attachments | Add attachments |
  | --- | --- | --- |
  | Contract | View the contract, plus `hrm.salary.view` (scans show the salary) | Edit the contract (already includes `hrm.salary.view`), even when `posted` |
  | Leave request, overtime request | Same as viewing the request | The submitter, and HR who can edit the request |
  | Employee profile | View the profile, plus `hrm.employee.sensitive` (ID card scans contain sensitive fields) | Edit the profile, plus `hrm.employee.sensitive` |

- Numbering scope: per legal entity and year, separate for each document type. The number is assigned when the draft is created; deleting a draft leaves a gap. This is acceptable, because HRM documents do not require gap-free numbering.
- "Payroll period check" means rejecting if the document's **affected date range** overlaps a closed pay period (see [Payroll lock](#payroll-lock)).
- Only payrolls produce business lines.

## Approval

### Approval fields

| Document type | Fields |
| --- | --- |
| `hrm.leave_request` | `days` (number), `leave_type` (list: annual leave, sick leave, unpaid, statutory leave…) |
| `hrm.overtime_request` | `hours` (number), `day_kind` (list: working day, rest day, public holiday) |
| `hrm.contract` | `contract_kind` (list) |
| `hrm.termination` | `reason` (list: the termination reasons) |

`day_kind` is entered by hand but must be the kind the legal entity's work calendar gives the date (a public holiday wins over a weekly day off), checked on save and again when the request goes into force; otherwise `overtime_day_kind_mismatch` names the right kind. Deriving it from the calendar comes later. Overtime requests record day-shift and night-shift hours separately; `hours` is their sum.

This lets tenants configure rules such as "leave over 3 days needs director approval" or "public-holiday overtime needs an extra approval step" ([documents.md](../docs/documents.md#shared-features)). HRM approval rules are configured inside the HRM area and need `hrm.approval.manage` (role `approval_admin`, grantable only tenant-wide) or `core.approval.manage`, both at tenant-wide scope.

### Direct manager

- The direct manager may not form a cycle: you cannot choose the employee themselves, or anyone under that employee (directly or indirectly). The manager picker does not list these people (`GET /hrm/employees?manager_of=<id>`); the backend still checks on save and returns `manager_cycle` with the name of the chosen person.
- `hrm` implements `Approvers` ([documents.md](../docs/documents.md#record-functions)) for leave requests and overtime requests (not for contracts and timesheets: approval rules for those two types go by role): `level = 1` returns the user of the direct manager, `level = 2` returns that person's manager, and so on up through `manager_id`.
- A manager without an account, or an employee without a manager: that step goes to the fallback role in the rule.
- An employee submits their own request and is themselves the manager for that step: `approval` excludes the submitter and moves up to the next level.

## Contracts

- A `posted` contract is not edited. Changing salary or terms means creating an **addendum**, which is also an `hrm.contract` pointing to the original contract.
- The contract in effect on a date is the `posted` original contract whose term contains that date; amounts come from the latest `posted` addendum whose effective date is no later than that date, or from the original contract if there is none.
- Contract amounts consist of salary, **salary allowances** (position, responsibility, hazard…: subject to insurance, included in the overtime hourly rate) and **benefits** (fuel, phone, meals, housing…: not subject to insurance, not included in the overtime hourly rate). Each benefit has a taxable or tax-exempt flag. These are contract lines, not fixed columns, because each customer names its items differently.
- **Contract type** is master data entered by the tenant (fixed-term, indefinite-term, probation…), following the same pattern as leave types. A type has a **fixed-term** flag: an original contract of such a type must have an end date, one of an indefinite type must not ([ADR-0018](../docs/adr/0018-fixed-term-contract-types-and-contract-list.md)).
- The company-wide contract list can filter contracts **expiring soon** (within the next 30 days), and has no money columns.
- An addendum carries **all** amounts, not just the changed ones; its effective date lies within the original contract's term, and it cannot change the end date (changing the term is a new contract). An addendum can only be approved once the original contract is `posted`; an original contract that still has a `posted` addendum cannot be cancelled.
- Two `posted` original contracts of the same employee may not have overlapping terms.
- The amounts are a single encrypted block in `hrm.contracts.terms`. Viewing needs the `payroll` role (permission `hrm.salary.view`); creating and editing contracts needs that permission too. Details in [ADR-0017](../docs/adr/0017-contracts-overtime-payroll-lock.md).

## Leave

- Leave balances are per employee and year, in `hrm.leave_balances`.
- A leave request starts and ends within the same year.
- The number of leave days is **computed** from the work calendar: the working days from the start date to the end date (neither a weekly rest day nor a public holiday, under the calendar in force for those dates), less half a day for "first day: afternoon only" (`start_half`) and for "last day: morning only" (`end_half`); a one-day request has a single "half day" flag. A range holding no working day is rejected (`leave_no_working_days`); both halves of a single working date are rejected (`leave_half_days`). A client may send the count it expects; it is refused with `leave_days_changed` (and the computed count) if it differs. Approval recomputes the count and refuses with `leave_days_changed` if the calendar has changed it since, so approved days never move. Requests saved before counts were computed keep their typed count, capped by the working days (`leave_days_exceed_working_days`) ([ADR-0047](../docs/adr/0047-leave-accrual-and-computed-days.md), superseding the typed count of [ADR-0030](../docs/adr/0030-leave-day-allocation.md)).
- The day count is **allocated** across the working days of the range, in date order, at most one day per date, half a day on the first working date with `start_half` and on the last with `end_half`. Nothing is ever dropped, so the days deducted from the balance and the days paid are the same days.
- **A date holds at most one day of absence.** Approving a request is rejected with `leave_overlaps` if any of its dates would hold more than one day once the employee's other approved leave is counted; the error names the date and the other request. Only approved requests count, so two overlapping requests can both be submitted and the second one approved is rejected. Half days compose: a morning and an afternoon on the same date are allowed, a half day plus a full day are not. Detecting an overlap needs no morning/afternoon marker. The check runs in `OnTransition` with the employee row locked (`FOR UPDATE`), after the payroll period check and before the leave balance, so two simultaneous approvals cannot both pass.
- **Granting and adjusting leave balances:** HR grants or adjusts a balance when needed, per employee in the UI, or in bulk from Excel. This needs a dedicated permission, a reason is mandatory, and every adjustment is audited. An adjustment locks the same `leave_balances` row (`FOR UPDATE`) as a deduction, so it never interleaves with approving a leave request; an adjustment that would make the balance negative is rejected.
- **Accrual** ([ADR-0047](../docs/adr/0047-leave-accrual-and-computed-days.md)): per legal entity, `hrm.leave_accrual` is `monthly` (default) or `manual` (HR grants, as before). Monthly, the system job `hrm.leave_accrual` (hourly and at start, catching up any missed month) credits each month on the 1st of the next: (12 + 1 per 5 full years since hire) / 12 per counted month (Labour Code 2019, Articles 113 and 114; Decree 145/2020, Article 66), the year's cumulative figure rounded down to a half day. A month counts when the employee is employed on at least half of its working days. Each decided month is a row of `hrm.leave_accruals` (0 days for an uncounted month), so no month is credited twice. Turning `monthly` on mid-year credits the year so far on top of any hand grant; turning it back to `manual` credits nothing more and removes nothing. After the termination date only months up to the termination month are credited.
- **Carry-over**: on 1 January last year's remaining days move into this year as **carried days**, lasting until the end of month `hrm.leave_carry_over_months` (`0`, `3` default, `6`, `12`) of the new year; `0` lapses them at once. Leave dated up to that date spends carried days first; leave dated later cannot spend them; what is left lapses when they expire. Carried days still left at the next new year lapse before the carry-over, which moves only that year's own days. A leaver's balance is neither carried past the year they left nor expired after they left; when a termination dated last year is entered after the carry-over, the next run moves the carried days back into the termination year, so its payroll pays them. The job writes nothing while HRM is disabled. Every credit, carry-over, expiry and reversal is audited on the employee.
- A `posted` leave request deducts from the leave balance in `OnTransition`; cancelling refunds it, carried days going back as carried while they last.
- Deducting locks the `leave_balances` row (`FOR UPDATE`), and the table has a non-negative constraint. Two requests approved at the same time cannot make the balance negative: the later one fails, stays pending approval, and the approver gets an "insufficient leave days" error.
- Only leave types that deduct from the balance (e.g. annual leave) touch the leave balance.
- Leave types also have a **company-paid** flag: annual leave has it; sick leave (paid by social insurance (BHXH), outside payroll) and unpaid leave do not. This flag decides whether a leave day counts as paid workdays.
- A leave type may cap the days of **one request** (`max_days`, e.g. 3 for one's own marriage under Article 115); a request over it is refused with `leave_days_exceed_type_max`, on entry and again on approval. A type may also require an attachment (e.g. a medical certificate for sick leave): a draft saves without files, but sending it, and approving it, are refused with `leave_attachment_required` while it has none ([ADR-0033](../docs/adr/0033-leave-type-limits.md)).

## Timekeeping

- Timesheets are per pay period and org unit; each unit has one non-cancelled timesheet per period. A daily grid: each line is the actual days worked (0.5 or 1) by one employee on one day ([ADR-0020](../docs/adr/0020-timesheets.md)). Leave days come from leave requests, public holidays from the work calendar; the grid shades weekly rest days and public holidays.
- The timesheet warns, per cell, of entries the payroll treats differently from what they say, and refuses neither: work on a weekly day off or a holiday (`work_on_day_off`: not paid without an overtime request) and work on a day an approved leave request covers (`work_on_leave_day`: the leave takes the balance, the work earns nothing more). The payroll line carries `work_on_leave_day` too.
- Data sources in the first release are manual entry and Excel import through `dataio`. Excel import is a job on behalf of the user: it runs with the importing user's permissions and re-checks them when it runs. The imported file replaces all lines of the draft timesheet; Excel export has the same layout, so it doubles as the template to fill in.
- Time clocks will be an integration in `internal/integration/`, built when there is a need for it, and will only produce data for `draft` timesheets.

## Work calendar

- Per legal entity ([ADR-0021](../docs/adr/0021-payroll.md)): weekly rest days by versions with effective dates (`hrm.work_weeks`; with no version yet, Saturday and Sunday are off) and a list of public holidays, including substitute days off (`hrm.holidays`).
- Editing the calendar edits a payroll source: it is rejected if the affected range overlaps a closed pay period (holiday: that day itself; work week: from the effective date until before the next version). Needs permission `hrm.calendar.manage`; every edit is audited.
- Paid workdays for a working day: a public holiday is 1; any other day is actual days worked plus company-paid leave days, capped at 1. A whole month does not exceed the standard workdays: a month with more than 26 working days still pays only one full month's salary. A leave request is spread across its working days that are not public holidays, in date order.

## Pay periods

- The first release only supports calendar months.
- The schema stores periods as date ranges (`period_start`, `period_end`) for timesheets, payrolls and pay periods alike, not as `year, month`. First-release logic always generates periods from the 1st to the last day of the month.
- Supporting other periods later (e.g. the 26th to the 25th) only needs a per-legal-entity setting and a change to the period generator, with no data migration.

## Payroll

### Which data it takes

- A payroll covers every employee with a `posted` original contract with that legal entity that is in effect during the period and who is employed during the period; plus everyone with an adjustment on that payroll.
- A payroll can only be created once every org unit with employees on the list has a `posted` timesheet for that period.

### Input snapshot

- On creation, the payroll reads contracts, timesheets, overtime requests (from the start of the year, to accumulate the year's tax-exempt hours), leave requests and legal parameters, then stores all inputs in `payroll_lines`.
- `inputs_hash` is a hash of every non-sensitive input: legal parameters, work calendar, rounding mode, minimum wage region, hire and termination dates, dependants hash, source documents with their `version`, actual days worked, leave requests, overtime requests. The legal parameter version is not stored separately; the parameters go into the hash.
- The list of source documents and each one's `version` is also written to `payroll_sources`, but not used for comparison.

### When sources change

Sources can change while the payroll is still `draft` or pending approval: cancelled, given a new version, or a new source document appears in the period (e.g. an overtime request that was just approved).

- On open, the payroll recomputes the input hash from current data and compares it with `inputs_hash`; if they differ it shows a "source data has changed" warning.
- Submitting for approval is refused until the user clicks **Recalculate**.
- If sources change while the payroll is pending approval, the final approval step detects it (see below) and fails to complete. The approval instance stays open; the submitter withdraws, recalculates, then resubmits.
- **Recalculate** can only be done while the payroll is `draft`, and is an explicit user action. Every recalculation is audited. Never recalculate automatically.
- Payroll calculation and recalculation run as a **job on behalf of the user**, because they must decrypt and compute for each employee. The screen tracks progress with `useJobStatus` ([frontend.md](../docs/frontend.md#refreshing-after-writes)).

### Payroll lock

The payroll lock guarantees two things: sources cannot change between the check and the commit that closes the payroll; and each period has at most one payroll in effect.

**Affected date range.** A source document can affect several pay periods, so the check is by date range rather than by document date:

| Source | Affected date range |
| --- | --- |
| Leave request | From the leave start date to the end date (a request from 30/03 to 02/04 affects both March and April) |
| Overtime request | The overtime date |
| Timesheet | The timesheet's period |
| Contract, addendum | From the effective date to the end date; with no end date, to infinity |
| Legal parameter | From the effective date to the effective date of the next version; with no next version, to infinity |

So a contract that was used to compute pay for a closed period cannot be cancelled. To change terms from an unclosed period onward, create an addendum effective from that period.

**Lock per legal entity.** Follows the same pattern as `record.period_locks`, and sits at the third step of the common lock order ([documents.md](../docs/documents.md#record-functions)), before any other row HRM locks. Within that step HRM's own order is `hrm.payroll_locks`, then the employee row (contracts and leave requests take it `FOR UPDATE` to serialize one employee's approvals), then leave balances:

- `hrm.payroll_locks (legal_entity_id)`: one row per legal entity, used only for locking. The lock is at legal-entity level, not period level: pay periods are created on demand, and a contract without an end date affects infinitely many periods. If locking per period row, a source transaction that started before the period row existed could not lock it.
- **Source documents** (in `OnTransition` of contracts, leave requests, overtime requests, timesheets; and when editing legal parameters): take `FOR SHARE` on the legal entity's row, then look in `hrm.payroll_periods` for periods with a non-null `posted_payroll_id` that overlap the affected date range. If any exist, reject with code `payroll_period_closed`, listing those periods.
- **Closing a payroll** (`OnTransition` to `posted`): take `FOR UPDATE` on the legal entity's row. `FOR UPDATE` waits for all running source changes of that legal entity and blocks new ones, so the set of sources cannot change until commit. Then, in the same transaction:
  1. Create the period row if missing. If the period's `posted_payroll_id` is already non-null, reject with code `payroll_already_posted`.
  2. Recompute the input hash and compare it with `inputs_hash`. Reject if they differ.
  3. Write the business lines, then set `posted_payroll_id` to this payroll's id.
- **Cancelling a payroll:** take `FOR UPDATE` on the legal entity's row, check that the period's `posted_payroll_id` **equals** the id of the payroll being cancelled, then reset it to null.
- DB constraint: `hrm.payroll_periods` has a unique key `(legal_entity_id, period_start)`, so each legal entity and period has only one row, and therefore only one `posted_payroll_id`.
- There can be several draft payrolls for the same period (e.g. for trial runs), but only one payroll in effect.
- Moving a legal entity's lock date has to wait for all running source changes of that legal entity, and vice versa. This is acceptable, because payroll is closed only once a month.

### After closing

A `posted` payroll is immutable, and a closed pay period blocks every source change. To fix a source, cancel the payroll first.

Errors found after the period is closed do not cancel that period's payroll; they are adjusted in the open period (see [Back pay and clawback](#back-pay-and-clawback)). The one exception is an operating rule, not enforced by the system, which does not know when pay is paid out or filed: a closed payroll may be cancelled to reopen its period only while that period's pay has not been paid out and its insurance and tax have not been filed.

## Payroll calculation

How the first release calculates. Every figure below is checked by the **payroll samples** in [docs/payroll-samples](../docs/payroll-samples/README.md); the samples are M6's acceptance test. The samples were prepared by Claude from public regulations and have not been reviewed by an accountant; open points are listed at the end of this section.

### Workdays and pay by workdays

- **Standard workdays** of a month are the normal working days under the legal entity's work calendar, including public holidays that fall on working days. At most 26 (Article 54, Decree 145/2020 (Điều 54 NĐ 145/2020)).
- **Paid workdays** are working days, plus public holidays and substitute days off, plus leave days of leave types with the company-paid flag. Public holidays before the hire date or after the termination date do not count.
- Pay by workdays = (salary + salary allowances + benefits) × paid workdays / standard workdays. A mid-period salary change through an addendum splits the period into segments by effective date; each segment is computed from its own workdays, then they are summed.
- Hire and termination dates come from the employee profile (`hire_date`, `termination_date`).

### Overtime

- Hourly rate = (salary + salary allowances) / standard workdays / 8 (Article 55, Decree 145/2020).
- Multipliers: working day 150%, weekly rest day 200%, public holiday 300% (not including holiday pay, which is already in the monthly salary). Night overtime adds 30% plus 20% × the daytime rate for that day: on a working day without daytime overtime it is 200%, with daytime overtime 210% (Article 57, Decree 145/2020).
- Overtime requests therefore record hours by day kind and by day or night shift.
- Overtime pay is not subject to insurance.

### Insurance

- Insurance salary = salary + salary allowances under the contract, not prorated by workdays. With a mid-month salary change, take the amount in effect on the last day worked in the period (the last day of the month, or the termination date if terminated during the month).
- Caps: social insurance (BHXH), health insurance (BHYT) and the trade union fee are 20 × the base salary (lương cơ sở); unemployment insurance (BHTN) is 20 × the legal entity's regional minimum wage (lương tối thiểu vùng).
- An employee who does not work and is not paid for 14 or more working days in a month (counting days before hire, after termination, sick leave, unpaid leave) pays no BHXH, BHYT, BHTN or trade union fee for that month.
- The employer's share (BHXH including the occupational accident and disease fund (TNLĐ-BNN), BHYT, BHTN, 2% trade union fee) is computed with the payroll, because it is needed for the `employer_insurance_expense` line.

### Personal income tax

- Everyone is withheld monthly under the progressive schedule, with the full personal deduction even in the month of hire or termination. Non-residents and contracts under 3 months are not in the first release.
- Tax-exempt income: overtime and night-work pay within the limit (40 hours/month, 200 hours/year, or 300 hours for permitted industries); pay for untaken annual leave paid on termination; benefits with the tax-exempt flag. Overtime beyond the limit is fully taxable, so the payroll accumulates overtime hours over the year and warns when the limit is exceeded.
- Taxable income is recognised in the period it is paid.

### Termination

- Untaken annual leave is paid at the contract salary of the month before the termination month / that month's standard workdays × the days left in the leave balance, plus, under monthly accrual, the days earned up to the termination date that the job has not credited yet (the termination month itself) (clause 3, Article 113, Labour Code 2019 (khoản 3 Điều 113 BLLĐ 2019); Article 67, Decree 145/2020). Not subject to insurance, tax-exempt.
- The severance or job-loss allowance of a posted termination dated in the period is paid by that period's payroll; see [Termination](#termination-1).

### Back pay and clawback

- Errors in a closed period are corrected with an addition (back pay, truy lĩnh) or a deduction (clawback, truy thu) on the payroll of the open period, with a reason and a reference to the original period. The closed period's payroll is not cancelled, and the old period's insurance and tax are not recalculated.
- An adjustment increases or decreases the taxable income of the open period.
- Because a clawback can make an aggregated line negative (e.g. a group of people whose recovered money exceeds their pay), amounts in `posting.lines` are **signed**; a negative amount is a reduction.

### Rounding

Per the [Rounding](#rounding-1) section: pay by workdays, each overtime amount, each insurance type and each person's tax are rounded to the dong in `line` mode.

### Still open

To be settled before loading real data (the production-readiness gate in the roadmap):

- Employees hired after the 15th: use the 14-day rule above, or start contributions the following month. Ask the social insurance (BHXH) office that manages the company.
- Recovering an amount from a previous tax year (e.g. in January recovering an overpayment from December): deducting it in the current year means the previous year's tax finalisation does not balance on its own. No rule yet.
- An accountant reviews all the samples and the assumptions in the "Giả định" (Assumptions) sheet.
- Annual leave accrual: Decree 145/2020, Article 66 prescribes no rounding for a partial year (we round the cumulative figure down to a half day) and Article 114 does not say whether the five-year step counts in full or pro rata in the year it falls (we pro-rate it by month); unpaid leave and long sick leave inside an employed month are not subtracted (Article 65). Both decide the termination payout.

## Termination

A termination (`hrm.termination`, [ADR-0046](../docs/adr/0046-termination-document.md)) records that an employee leaves. Only posting one sets the employee's `termination_date`: editing a profile or importing profiles refuses a changed date (`termination_via_document`); creating a profile may still carry one, for former employees loaded without documents, and such legacy dates stay until a termination replaces them. Cancelling restores the date a termination replaced (`previous_date`).

- One termination per employee that is not cancelled (`termination_exists`). Its legal entity and org unit are the employee's when drafted; posting or cancelling after the employee moved to another legal entity is refused (`termination_legal_entity_changed`).
- Lock order: record row → `hrm.payroll_locks` `FOR SHARE` with the payroll period check from the earlier of the new and the replaced date, to infinity → employee row `FOR NO KEY UPDATE`.
- **Late entry.** An employee leaves 31/03, March's payroll closes, then HR enters the termination: posting is refused (`payroll_period_closed`, naming March). While March's pay is not paid out and not filed, HR cancels March's payroll, posts the termination, recalculates and closes again. Once paid out, the allowance does not go through mmerp yet (Still open).
- Warnings on the termination, refusing nothing: `contract_in_force_after`, `leave_after_termination`, `overtime_after_termination`, `leave_balance_to_pay`. Posted contracts are not changed.
- The profile's "Cho nghỉ việc" opens a new termination (`create_termination` in the employee's actions); the profile links to the termination that stands.

### Reasons

Stable codes in `hrm.terminations.reason`; reports may read them.

| Code | Labour Code 2019 | Allowance |
| --- | --- | --- |
| `contract_expired` | Art 34.1 | severance |
| `work_completed` | Art 34.2 | severance |
| `mutual_agreement` | Art 34.3 | severance |
| `imprisonment` | Art 34.4 | severance |
| `foreigner_expelled` | Art 34.5 | none |
| `death` | Art 34.6 | severance |
| `employer_ceased` | Art 34.7 | severance |
| `dismissal` | Art 34.8 | none |
| `employee_resignation` | Art 34.9 (Art 35) | severance |
| `employer_unilateral` | Art 34.10 (Art 36) | severance |
| `absent_without_reason` | Art 36.1(e) | none |
| `restructuring` | Art 34.11 (Art 42, 43) | job loss |
| `work_permit_expired` | Art 34.12 | none |
| `probation_failed` | Art 34.13 | none |

An employee eligible for a pension (`pension_eligible`) gets no severance allowance; the job-loss allowance is not affected. Retirement is `contract_expired` or `mutual_agreement` with that flag.

### Allowance

- Paid only after at least 12 months worked: whole months from the hire date to the termination date, leftover days dropped.
- Months counted = months worked − months in unemployment insurance − months already paid (both entered by HR from the social insurance book); years = whole years, a remainder of 1–6 months is ½, 7–11 is 1 (clause 3, Article 8, Decree 145/2020).
- Average salary = salary + salary allowances of the contract in force at the end of each of the six months before the termination month, averaged over the months with one (clause 5, Article 8, Decree 145/2020).
- Severance = years × ½ × average; job loss = max(years, 2) × average; rounded once to the dong.
- The payroll of the termination month pays it: the termination's figures go into the hashed basis (a change marks a draft payroll stale), the six monthly amounts are captured encrypted with the line. Gross and exempt income, never insured; the export and payslip show it. Article 48 (pay within 14 working days) is an operating rule, not enforced.
- The termination previews the allowance; the average and amount need `hrm.salary.view` at the termination's unit and at every contract read, and the read is audited.

### Still open

- Leftover days in months worked are dropped; some practice rounds a part month up.
- A month without a contract in force among the six is left out of the average; whether the six months include the termination month when it ends on its last day.
- The exact PIT provision for periods from 01/07/2026 (Law 109/2025/QH15, Decree 253/2026/NĐ-CP).
- A termination entered after its period's pay is paid out: a back-pay style allowance line on the open payroll is the follow-up.
- The two payroll samples C18 and C19 need the accountant's review like the others.

## Legal parameters

- Contribution rates for BHXH, BHYT, BHTN and the trade union fee (employee and employer shares), contribution caps, base salary, regional minimum wage, family deduction amounts, the progressive PIT schedule and the tax-exempt overtime hour limit are **not hard-coded**. They are rows in `hrm.legal_params`, with effective dates.
- Default values ship with each update, as data migrations. Tenants can edit them, and every edit is audited.
- A payroll picks parameters by pay period, not by creation date: each key takes the version in effect at `period_end`.
- Adding or editing a version needs tenant-wide `hrm.legal_param.manage`, and is rejected if the range from the effective date until before the next version overlaps a closed pay period of any legal entity.
- Overtime multipliers are fixed law and live in code.
- Each legal entity's minimum wage region is a setting, because the BHTN cap depends on the region.
- Parameter names are stored as translation keys ([techstack.md](../docs/techstack.md#i18n)).

## Rounding

Per [ADR-0006](../docs/adr/0006-money-rounding.md). The default is `line`: each employee × each item (salary, each insurance type, PIT) is rounded separately. When a legal entity chooses `total`, the difference is allocated down to each employee by the largest remainder method, so payslips, the payroll and business lines always agree.

## Business lines

Only payrolls produce business lines, on moving to `posted`:

| Line type | Party | Org unit |
| --- | --- | --- |
| `salary_expense` (salary and allowance expense) | — | The employee's department |
| `employer_insurance_expense` (employer's share of insurance) | — | Department |
| `salary_payable` (payable to employees, after deductions) | — (aggregated) | Department |
| `insurance_payable` (insurance payable, both shares) | — | Legal entity |
| `pit_payable` (PIT withheld) | — | Legal entity |

- Every line is aggregated by department or legal entity. Business lines and the ledger are not encrypted ([ADR-0003](../docs/adr/0003-business-lines-are-data.md)), so they must not carry per-person amounts.
- Payables detailed per employee live in `hrm.payroll_lines` (encrypted), and are reconciled when salaries are paid.
- A department with only one person still lets that person's salary be inferred from the ledger. Accepted, because viewing the ledger is already an accountant's permission.
- Mapping these line types to accounts is the job of `accounting`.

## Reports

Excel files of the statutory filings, one legal entity and one period each, through `dataio` exports ([design](../docs/superpowers/specs/2026-10-28-insurance-pit-reports-design.md)):

| Kind | Form | Period |
| --- | --- | --- |
| `hrm.report.d02lt` | D02-LT, Quyết định 366/QĐ-BHXH (from 05/2026) | Month |
| `hrm.report.pit_05kk` | 05/KK-TNCN, Thông tư 89/2026/TT-BTC (from Q3/2026; every employer files quarterly) | Quarter |
| `hrm.report.pit_05qtt` | 05/QTT-TNCN, Thông tư 89/2026/TT-BTC (from tax year 2026) | Year |
| `hrm.report.pit_05_1bk` | 05-1/BK-QTT-TNCN, table 1 | Year |
| `hrm.report.pit_05_3bk` | 05-3/BK-QTT-TNCN | Year |

- Figures come only from the posted payrolls of the period (D02-LT: the month and the one before) and the employee profiles; a month without a posted payroll refuses the report (`report_payroll_missing`, naming the months). Nothing is stored.
- Flat sheets: list forms have the official column headings, with their codes, as the first row; declaration forms are rows of indicator, code and value under the header fields. Each form version is its own layout in code; earlier periods are refused (`report_form_not_supported`).
- D02-LT: insured salary positive now and not before is an increase ("Đi làm lại" only for who was insured on an earlier posted payroll of the legal entity), the reverse a decrease (the termination date, a move to another legal entity, or the 14-day rule), a changed amount an adjustment.
- Finalisation: the authorising employees are those HR recorded for the legal entity and year (`hrm.pit_authorisations`, one legal entity per person per year, chosen among those on its posted payrolls of the year; the signed 08/UQ-QTT-TNCN is attached to the employee profile; [ADR-0048](../docs/adr/0048-pit-authorisation-recorded.md)); the year's tax uses the schedule in force on 31/12 with its monthly bounds × 12; at most 50,000 đồng still due is waived.
- Every report needs `hrm.salary.view` at the legal entity and `hrm.employee.sensitive` at each employee's current unit, when it runs and at every download; identifiers read are audited (`hrm.employee_sensitive_exported`, `hrm.dependents_viewed`).
- Not built: withholding certificates (electronic documents with a registered series and a digital signature, Decree 123/2020, Decree 70/2025), electronic filing.

### Still open (reports)

- Whether the BHXH office takes D02-LT as the QĐ 366 PDF layout or the eBHXH xlsx (with unit details), and the wording of its column 18.
- Whether a termination mid-month is a decrease from that month or the next.
- Whether [16] of 05-1/BK includes the personal deduction (assumed yes), and the formula of [36] of 05/KK.
- The employer's finalisation deadline for 2026 under Law 108/2025/QH15.
- The TT 89 forms were read from form files and secondary sources; the accountant checks the mapping against the official text.
- A company starting mid-year cannot finalise that year from mmerp: its earlier months are not there.

## Personal data

| Level | Fields | Storage |
| --- | --- | --- |
| Sensitive | ID card (CCCD) or passport number, social insurance book number, personal tax code, bank account, salary and allowances, every per-person amount in `payroll_lines` (including captured inputs) and `payroll_adjustments`, dependant information | Column encryption; viewing needs a dedicated permission; every read is audited |
| Ordinary personal | Full name, date of birth, gender, address, phone, email | Not encrypted; viewed per `Can(view)` |

The system does not collect health information. A leave request's `reason` is plain text, not encrypted, so it must not contain a diagnosis; the leave request form reminds users of this. A medical certificate attached to a leave request is an attachment: not encrypted, anyone who can view the request can download it, and every download is audited.

Dedicated view permissions: `hrm.employee.sensitive` (role `sensitive_viewer`) for sensitive fields of profiles and dependants; `hrm.salary.view` (role `payroll`) for every per-person amount: contract amounts, payroll lines and adjustments; calculating and exporting payrolls also need this permission. The `payroll_viewer` role only sees payroll totals by department, not anyone's amounts. An employee reads their own payslips (posted payrolls only) without these permissions; every such read is audited, and amounts never enter the audit data.

**Copies of salary data** are protected like the original:

| Where | Protection |
| --- | --- |
| `hrm.contracts`, `hrm.payroll_lines`, `hrm.payroll_adjustments`, captured inputs | Column encryption |
| `posting.lines`, ledger | Only amounts aggregated over at least 3 paid people by org unit (a smaller unit rolls up to its parent, up to the legal entity), or by legal entity ([ADR-0040](../docs/adr/0040-payroll-posting-minimum-group.md)) |
| `record.documents.amount` of a payroll | Only the total cost of the whole legal entity |
| Audit log | Sensitive field values are stored encrypted ([documents.md](../docs/documents.md#permissions)) |
| Export files, payslips | Need the salary view permission; temporary files expire; every export is audited |
| Contract attachments | Need the salary view permission; every download re-checks the permission and is audited |
| Logs, heartbeat, error reports | Never contain per-person amounts |

Because salaries are encrypted, SQL cannot sum them. Every aggregation (payroll totals by department, including department totals on a payroll, insurance reports, PIT finalisation) decrypts and computes in the service, and is not stored. At SMB scale (under a few thousand employees), this cost is acceptable.

## Frontend

The `web/src/hrm` area follows the pattern in [frontend.md](../docs/frontend.md). The manifest declares `recordTypes` for leave requests, overtime requests, contracts, timesheets, payrolls and terminations. When a leave request changes, including when it is approved from the `core` approval inbox, `invalidate` refreshes the request detail, the request list and the leave balance.

- The menu is in HR's words and grouped: overview; Nhân sự (employee records, org chart, employment contracts, terminations (nghỉ việc)); Chấm công & nghỉ phép (timesheets (bảng chấm công), leave requests, overtime requests); Tiền lương (payrolls); Thiết lập (collapsible, folded by default: contract types, leave types, work calendar & holidays, legal parameters (mức đóng bảo hiểm & thuế), approval rules (quy trình duyệt)). The home tile is "Nhân sự".
- The overview page is the area's entry page. A "Cần xử lý" block above the counts lists, each hidden when zero or out of the user's scope: requests waiting for the user's approval (links to the approval inbox), org units without a closed timesheet last month, legal entities without a closed payroll last month.
- The HRM area is for people with an HRM role; anyone else who opens it is sent home. The org chart is open to every signed-in user, inside HRM and in the personal space.
- Everyone else, direct managers included, uses the personal space (`web/src/hrm/me`, home tile "Cá nhân"): own timekeeping, payslips, leave and overtime requests, profile and org chart. A direct manager also sees "Nhóm của tôi": their reports' leave and overtime requests, past ones included (`GET /hrm/leaves?reports=true`, `GET /hrm/overtimes?reports=true`; shown when the self-service actions say `has_reports`). Their requests, and links to them from notifications and the approval inbox, open in the personal space, where the approver approves.
- The HRM approval rules screen lives in the HRM area (`/hrm/approval-rules`) and shares components with `core`.

## Proving flow

The sample flow "leave and payroll" in [documents.md](../docs/documents.md#sample-flow-leave-and-payroll) is the scenario set that proves the core with HRM. The first milestone when writing code is completing the leave request flow end to end across frontend and API.
