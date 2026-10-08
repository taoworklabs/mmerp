# 0021. Payroll

Status: accepted · 2026-10-06

## Context

M6 builds payroll that can be closed, gives correct figures and is immutable once closed. The calculation was settled at the gate before M4 ([products/hrm.md](../../products/hrm.md#payroll-calculation)) and is checked against the set of 28 sample cases. By M5 every source document exists (contracts, leave requests, overtime requests, timesheets), along with the payroll lock per legal entity and background jobs ([ADR-0017](./0017-contracts-overtime-payroll-lock.md), [ADR-0019](./0019-background-jobs-dataio-and-file-storage.md), [ADR-0020](./0020-timesheets.md)). Still missing: the rounding function ([ADR-0006](./0006-money-rounding.md)), per-legal-entity settings, the work calendar, legal parameters, business lines and payroll itself.

## Decision

### Rounding (`platform`)

- `platform.Round(decimal) int64` rounds half away from zero; `platform.Allocate([]decimal) []int64` rounds the total, then distributes the difference by largest remainder, exactly as in ADR-0006. A negative difference (a total with negative numbers) is taken from the lines with the smallest fractional parts.
- `platform.Rounding` is `line` or `total`; the module reads the legal entity's setting and passes it in.

### Per-legal-entity settings (`core/setting`)

- Table `setting.legal_entity_values (legal_entity_id, key, value)`. A key with no row takes the default declared in code.
- Modules register keys with `RegisterLegalEntityKey(key, default, valid values)`; `GetFor`, `SetFor` read and write. Writes check for valid values and write audit `setting.changed`.
- `setting` registers `setting.rounding` itself (`line`, `total`; default `line`). HRM registers `hrm.wage_region` (`1`–`4`; default `1`).
- Routes `GET /legal-entities/{id}/settings`, `PUT /legal-entities/{id}/settings/{key}`, need permission `core.setting.manage` (role `core.admin`). `iam` uses `setting`, so `setting` asks for permissions through an `Authz` hook in `hooks.go`, and `internal/app` wires `iam` in.

### `record`: `BeforeSubmit`

- `record.Type` gets an optional callback `BeforeSubmit(ctx, d)`, run in `Transition` from `draft` to `posted`, before the approval gate, even when no approval is needed. An error blocks the submit. Payroll uses it to block submitting when it has not been computed or its sources have changed.

### Business lines (`shared/posting`)

- Module `posting`, schema `posting`, table `posting.lines (doc_type, doc_id, legal_entity_id, date, kind, org_unit_id, amount, voided_at)`. `amount` is signed.
- `Record(ctx, ref, legal entity, date, lines)` and `Void(ctx, ref)`; `Void` sets `voided_at`, it does not delete. Both call the hook `OnChanged(ctx, ref)` in the document's transaction; implementers may only enqueue a job in the same transaction, so a job processing error does not roll back the document. There is no accounting yet, so `internal/app` wires a no-op.
- Payroll line kinds: `salary_expense`, `employer_insurance_expense`, `salary_payable` per department; `insurance_payable`, `pit_payable` per legal entity.

### Work calendar and leave types

- `hrm.work_weeks (legal_entity_id, effective_from, off_days)`: weekly days off, versioned by effective date. A legal entity with no row has Saturday and Sunday off.
- `hrm.holidays (legal_entity_id, date, name)`, including substitute days off.
- Editing the calendar edits a payroll source: check for closed pay periods over the affected range (holiday: that day; work week: from the effective date up to the next version), and write audit. Permission `hrm.calendar.manage`.
- The timesheet grid shades weekly days off and public holidays from the calendar.
- `hrm.leave_types.paid`: the "company pays" flag. The migration sets it equal to the deduct-balance flag for existing leave types.

### Legal parameters

- `hrm.legal_params (key, effective_from, value)`, one version per key and date. The value is a string (a decimal, or JSON for the tax schedule `[[threshold, rate], …]`, last threshold `null`). Payroll takes each key at the version in effect on `period_end`.
- The 2026 defaults are in the migration. Adding or editing a version needs `hrm.legal_param.manage`, checks closed pay periods (of every legal entity) over `[effective_from, next version)`, and writes audit `hrm.legal_param_changed`.
- Overtime rates (150/200/300%, night 200/210/270/390%) are fixed by law and live in code.

### Payroll (`hrm.payroll`)

- Document prefix `BL`, the date is `period_end`, the org unit is the legal entity, no approval fields; approval rules go by role. `amount` is the total payroll cost.
- Tables: `hrm.payrolls (id, period_start, period_end, computed_at, inputs_hash, totals)`; `hrm.payroll_lines (payroll_id, employee_id, org_unit_id, data)` where `data` is encrypted JSON holding the captured inputs and every amount; `hrm.payroll_sources (payroll_id, doc_type, doc_id, version)`; `hrm.payroll_adjustments (payroll_id, employee_id, source_period, data)` with encrypted amount and reason. `totals` holds only per-department totals.
- Permissions at the legal entity: viewing and exporting need `hrm.payroll.view`; editing, computing, submitting, cancelling need `hrm.payroll.edit`. Per-person amounts, computing and exporting additionally need `hrm.salary.view`; each response returning per-person amounts writes audit `hrm.payroll_lines_viewed`. The four new permissions belong to the `payroll` role. The `payroll_viewer` role has only `hrm.payroll.view`: sees per-department totals without seeing anyone's amounts (e.g. accountants). Creating a payroll needs both `hrm.payroll.edit` and `hrm.salary.view`, because creating means computing.
- **Computing and recomputing** is the job `hrm.payroll_compute`, run on behalf of the user (write class). Creating a payroll, clicking Recompute and saving an adjustment all enqueue the job in the same transaction. The job runs one transaction: `record.Edit` (permission, draft, version) → check `hrm.salary.view` → gather inputs → compute → rewrite lines, sources, hash, totals → audit `hrm.payroll_computed` → complete the job.
- `hrm.payroll_periods.posted_payroll_id` still references `record.documents` as since M4; only the payroll's `OnTransition` sets it.
- **The calculation is a pure function** that does not touch the database: per-person inputs and parameters in, amounts out. The table tests and the sample set run on the same function.
- **The input hash** (`inputs_hash`, sha256) covers the set of `posted` source documents with their versions, the legal parameters used, the work calendar, settings, and per employee: hire date, termination date, department, the encrypted dependants blob, leave balance on termination within the period. It contains no amounts. Sources have changed if and only if the hash differs; it can be computed without decrypting salaries.
- **Back pay and clawbacks** are entered on a draft payroll. Saving an adjustment clears `computed_at` and enqueues the compute job.
- `BeforeSubmit`: not computed → `payroll_not_computed`; hash differs → `payroll_sources_changed`.
- `OnTransition` to `posted`: lock `FOR UPDATE` the legal entity row in `hrm.payroll_locks` → create the period row if missing → the period already has another payroll → `payroll_already_posted` → compare the hash again → write business lines grouped by department → set `posted_payroll_id`. To `cancelled`: lock as above → `posted_payroll_id` must equal this payroll (otherwise `payroll_not_holding_period`) → clear it → `posting.Void`.
- Excel export `hrm.payroll` through `dataio`: one row per person with every amount, plus per-department total rows; needs `export` and `hrm.salary.view` when it runs.

## Consequences

- Salaries cannot be summed in SQL; every total (per-department totals, business lines) is decrypted and computed by the service.
- Opening a payroll recomputes the hash, so the "dữ liệu nguồn đã thay đổi" (source data has changed) warning needs no decryption and writes no salary-read audit.
- The overtime request's `day_kind` is still entered by hand; deriving it from the calendar is deferred. Cash rounding, severance allowance and the 300 hours/year limit for permitted industries are not there yet (organisations edit the `ot_exempt_hours_year` parameter).
