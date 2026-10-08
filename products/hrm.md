# HRM product

Human resources management: employee profiles, contracts, leave and overtime, timekeeping, payroll. HRM is the first product, and every core capability is proven by an HRM flow ([backend.md](../backend.md#layers), [ADR-0023](../docs/adr/0023-build-core-first.md)). Hard-to-reverse decisions are in [ADR-0007](../docs/adr/0007-hrm-product.md); terminology is in the HRM section of [CONTEXT.md](../CONTEXT.md#hrm).

## Scope

| In the first release | Not yet |
| --- | --- |
| Employee profiles, dependants | Recruitment, performance review, training |
| Employment contracts and addenda | Salary payment through banks |
| Leave requests, leave balances, overtime requests | Time clock integration |
| Timesheets: manual entry and Excel import | Electronic social insurance (BHXH) filing |
| Payroll: salary, social insurance (BHXH), health insurance (BHYT), unemployment insurance (BHTN), personal income tax (PIT, thuế TNCN) | Pay periods other than the calendar month |

## Module

A single module `internal/modules/hrm`, schema `hrm`, belonging to the `hrm` product. It is not split into several modules: all four parts share employees and pay periods. If it were split, then, because modules in `modules/` may not import each other, payroll would have to call timekeeping through an interface. Split only when another product needs one part on its own. Then apply the "promote an entity to shared" rule ([backend.md](../backend.md#data-ownership)), e.g. employees move to `shared/`.

### Dependencies

- **Dependency on other products:** none.
- **Used directly:** `iam` (users, org tree, `Scope`), `record`, `setting`, `audit`, `dataio`, `shared/posting`. `approval` and `numbering` are used only indirectly through `record`.
- **Hooks emitted:** none yet. The first hook may be "employee terminated", but it is added only when another module needs to react.

## Entities

| Table | Meaning |
| --- | --- |
| `hrm.employees` | Employee profile. Optional link to `iam.users.id`: an employee with an account submits their own requests; one without has HR enter them on their behalf. `org_unit_id` is the department; `manager_id` is the direct manager |
| `hrm.dependents` | Dependants for the family deduction (giảm trừ gia cảnh) |
| `hrm.contract_types` | Contract types, with a fixed-term flag |
| `hrm.contracts` | Employment contracts and addenda: type, term, salary, allowances |
| `hrm.leave_types` | Leave types, with a deducts-from-balance flag and a company-paid flag |
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

- Every document type uses approval and audit. Employee profiles use audit; there is no profile import/export yet. Excel import/export through `dataio` currently covers: timesheet import and export, payroll export, leave balance adjustment import.
- Discussion is available on every record type. Attachments are only on contracts, leave requests and employee profiles ([ADR-0024](../docs/adr/0024-attachments-and-discussion.md)):

  | Type | View attachments | Add attachments |
  | --- | --- | --- |
  | Contract | View the contract, plus `hrm.salary.view` (scans show the salary) | Edit the contract (already includes `hrm.salary.view`), even when `posted` |
  | Leave request | Same as viewing the request | The submitter, and HR who can edit the request |
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

`day_kind` is still entered manually; deriving it from the work calendar comes later. Overtime requests record day-shift and night-shift hours separately; `hours` is their sum.

This lets tenants configure rules such as "leave over 3 days needs director approval" or "public-holiday overtime needs an extra approval step" ([documents.md](../documents.md#shared-features)). HRM approval rules are configured inside the HRM area and need `hrm.approval.manage` (role `approval_admin`, grantable only tenant-wide) or `core.approval.manage`, both at tenant-wide scope.

### Direct manager

- The direct manager may not form a cycle: you cannot choose the employee themselves, or anyone under that employee (directly or indirectly). The manager picker does not list these people (`GET /hrm/employees?manager_of=<id>`); the backend still checks on save and returns `manager_cycle` with the name of the chosen person.
- `hrm` implements `Approvers` ([documents.md](../documents.md#record-functions)) for leave requests and overtime requests (not for contracts and timesheets: approval rules for those two types go by role): `level = 1` returns the user of the direct manager, `level = 2` returns that person's manager, and so on up through `manager_id`.
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
- The number of leave days is entered manually in half days, and is only checked not to exceed the calendar days from the start date to the end date. Computing it from the work calendar comes later.
- **Granting and adjusting leave balances:** HR grants the opening balance at the start of the year and adjusts it when needed, per employee in the UI, or in bulk from Excel. This needs a dedicated permission, a reason is mandatory, and every adjustment is audited. An adjustment locks the same `leave_balances` row (`FOR UPDATE`) as a deduction, so it never interleaves with approving a leave request; an adjustment that would make the balance negative is rejected. The first release has no automatic accrual rules.
- A `posted` leave request deducts from the leave balance in `OnTransition`; cancelling refunds it.
- Deducting locks the `leave_balances` row (`FOR UPDATE`), and the table has a non-negative constraint. Two requests approved at the same time cannot make the balance negative: the later one fails, stays pending approval, and the approver gets an "insufficient leave days" error.
- Only leave types that deduct from the balance (e.g. annual leave) touch the leave balance.
- Leave types also have a **company-paid** flag: annual leave has it; sick leave (paid by social insurance (BHXH), outside payroll) and unpaid leave do not. This flag decides whether a leave day counts as paid workdays.

## Timekeeping

- Timesheets are per pay period and org unit; each unit has one non-cancelled timesheet per period. A daily grid: each line is the actual days worked (0.5 or 1) by one employee on one day ([ADR-0020](../docs/adr/0020-timesheets.md)). Leave days come from leave requests, public holidays from the work calendar; the grid shades weekly rest days and public holidays.
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
- Payroll calculation and recalculation run as a **job on behalf of the user**, because they must decrypt and compute for each employee. The screen tracks progress with `useJobStatus` ([frontend.md](../frontend.md#refreshing-after-writes)).

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

**Lock per legal entity.** Follows the same pattern as `record.period_locks`, and sits at the third step of the common lock order ([documents.md](../documents.md#record-functions)), before any other row HRM locks (e.g. leave balances):

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

Errors found after the period is closed do not cancel that period's payroll; they are adjusted in the open period (see [Back pay and clawback](#back-pay-and-clawback)).

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

- Untaken annual leave is paid at the contract salary of the month before the termination month / that month's standard workdays × the days left in the leave balance (clause 3, Article 113, Labour Code 2019 (khoản 3 Điều 113 BLLĐ 2019); Article 67, Decree 145/2020). Not subject to insurance, tax-exempt.
- Severance allowance (trợ cấp thôi việc) is not in the first release.

### Back pay and clawback

- Errors in a closed period are corrected with an addition (back pay, truy lĩnh) or a deduction (clawback, truy thu) on the payroll of the open period, with a reason and a reference to the original period. The closed period's payroll is not cancelled, and the old period's insurance and tax are not recalculated.
- An adjustment increases or decreases the taxable income of the open period.
- Because a clawback can make an aggregated line negative (e.g. a department with only one person, already terminated, whose money is recovered), amounts in `posting.lines` are **signed**; a negative amount is a reduction.

### Rounding

Per the [Rounding](#rounding-1) section: pay by workdays, each overtime amount, each insurance type and each person's tax are rounded to the dong in `line` mode.

### Still open

To be settled before loading real data (the production-readiness gate in the roadmap):

- Employees hired after the 15th: use the 14-day rule above, or start contributions the following month. Ask the social insurance (BHXH) office that manages the company.
- Recovering an amount from a previous tax year (e.g. in January recovering an overpayment from December): deducting it in the current year means the previous year's tax finalisation does not balance on its own. No rule yet.
- An accountant reviews all the samples and the assumptions in the "Giả định" (Assumptions) sheet.

## Legal parameters

- Contribution rates for BHXH, BHYT, BHTN and the trade union fee (employee and employer shares), contribution caps, base salary, regional minimum wage, family deduction amounts, the progressive PIT schedule and the tax-exempt overtime hour limit are **not hard-coded**. They are rows in `hrm.legal_params`, with effective dates.
- Default values ship with each update, as data migrations. Tenants can edit them, and every edit is audited.
- A payroll picks parameters by pay period, not by creation date: each key takes the version in effect at `period_end`.
- Adding or editing a version needs tenant-wide `hrm.legal_param.manage`, and is rejected if the range from the effective date until before the next version overlaps a closed pay period of any legal entity.
- Overtime multipliers are fixed law and live in code.
- Each legal entity's minimum wage region is a setting, because the BHTN cap depends on the region.
- Parameter names are stored as translation keys ([techstack.md](../techstack.md#i18n)).

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

## Personal data

| Level | Fields | Storage |
| --- | --- | --- |
| Sensitive | ID card (CCCD) or passport number, social insurance book number, personal tax code, bank account, salary and allowances, every per-person amount in `payroll_lines` (including captured inputs) and `payroll_adjustments`, dependant information | Column encryption; viewing needs a dedicated permission; every read is audited |
| Ordinary personal | Full name, date of birth, gender, address, phone, email | Not encrypted; viewed per `Can(view)` |

The system does not collect health information. A leave request's `reason` is plain text, not encrypted, so it must not contain a diagnosis; the leave request form reminds users of this. A medical certificate attached to a leave request is an attachment: not encrypted, anyone who can view the request can download it, and every download is audited.

Dedicated view permissions: `hrm.employee.sensitive` (role `sensitive_viewer`) for sensitive fields of profiles and dependants; `hrm.salary.view` (role `payroll`) for every per-person amount: contract amounts, payroll lines and adjustments; calculating and exporting payrolls also need this permission. The `payroll_viewer` role only sees payroll totals by department, not anyone's amounts.

**Copies of salary data** are protected like the original:

| Where | Protection |
| --- | --- |
| `hrm.contracts`, `hrm.payroll_lines`, `hrm.payroll_adjustments`, captured inputs | Column encryption |
| `posting.lines`, ledger | Only amounts aggregated by department or legal entity |
| `record.documents.amount` of a payroll | Only the total cost of the whole legal entity |
| Audit log | Sensitive field values are stored encrypted ([documents.md](../documents.md#permissions)) |
| Export files, payslips | Need the salary view permission; temporary files expire; every export is audited |
| Contract attachments | Need the salary view permission; every download re-checks the permission and is audited |
| Logs, heartbeat, error reports | Never contain per-person amounts |

Because salaries are encrypted, SQL cannot sum them. Every aggregation (payroll totals by department, including department totals on a payroll, insurance reports, PIT finalisation) decrypts and computes in the service, and is not stored. At SMB scale (under a few thousand employees), this cost is acceptable.

## Frontend

The `web/src/hrm` area follows the pattern in [frontend.md](../frontend.md). The manifest declares `recordTypes` for leave requests, overtime requests, contracts, timesheets and payrolls. When a leave request changes, including when it is approved from the `core` approval inbox, `invalidate` refreshes the request detail, the request list and the leave balance.

- The menu is grouped: overview; people (employees, org structure, contracts); time (timesheets, leave requests, overtime requests); payroll; configuration (collapsible: contract types, leave types, work calendar, legal parameters, approval rules).
- The overview page is the area's entry page and shows counts.
- The org structure page is open to every signed-in user.
- The HRM approval rules screen lives in the HRM area (`/hrm/approval-rules`) and shares components with `core`.

## Proving flow

The sample flow "leave and payroll" in [documents.md](../documents.md#sample-flow-leave-and-payroll) is the scenario set that proves the core with HRM. The first milestone when writing code is completing the leave request flow end to end across frontend and API.
