# Annual leave accrual and computed leave days — design

Feature C of the parallel HRM work. ADR: [0047](../../adr/0047-leave-accrual-and-computed-days.md).

## Why

HR grants every employee's leave balance by hand each January and corrects it for joiners, leavers and seniority. A leave request's day count is typed and only capped by the work calendar (ADR-0030), so the person entering it does arithmetic the system already knows how to do. HR wants the balance to build up by itself as the law says, and the request to count its own days.

## Goals

- Monthly accrual of annual leave (Labour Code 2019, Article 113), pro rata for partial years, plus one day per five years of service (Article 114).
- Carry-over of last year's unused days into the new year, expiring after a configurable number of months.
- A leave request's days computed from the work calendar; half days still possible on the first and last day.
- The payroll's payout of untaken leave on termination includes what has been earned but not yet credited.

## Non-goals

- 14 and 16-day entitlements (Article 113.1b, c: hazardous work, minors, disabled employees): HR adds the difference by hand, as today.
- Leave taken in advance of accrual: the balance stays non-negative.
- A morning/afternoon marker for scheduling; a cap on carried days; combining up to 3 years (Article 113.4) beyond what the carry-over months allow.
- Changing termination itself (feature B) or the payout formula (only its day count changes).

## Anti-goals

- A credit applied twice, or a month never credited because the server was off on the 1st.
- Approved days that change silently after approval.
- Moving any figure of the payroll samples.

## Legal defaults (assumptions)

- **Entitlement**: 12 days a year (Article 113.1a) + 1 day per 5 full years since `hire_date` (Article 114), years completed by the end of the month being counted.
- **Monthly rate**: (12 + seniority days) / 12 per counted month (Decree 145/2020, Article 66.1).
- **Counted month**: the employee is employed (between `hire_date` and `termination_date`) on at least half of the month's working days under the work calendar (Decree 145/2020, Article 66.2). Simplified: the decree counts days worked plus paid days off, so unpaid leave inside an employed month is not subtracted (Article 65 lists what counts as working time); HR adjusts if it matters.
- **Rounding**: Decree 145/2020, Article 66 is silent on rounding. Its full text: (1) "lấy số ngày nghỉ hằng năm cộng với số ngày được nghỉ tăng thêm theo thâm niên (nếu có), chia cho 12 tháng, nhân với số tháng làm việc thực tế trong năm để tính thành số ngày được nghỉ hằng năm"; (2) a partial month counts as one month when the days worked plus paid days off reach "từ 50% số ngày làm việc bình thường trong tháng"; (3) state-sector service counts towards the Article 114 step. (The rule "0.5 or more rounds up to a whole day" was Decree 45/2013, Article 7, repealed.) Assumption, on the accountant's list: the year's cumulative entitlement through a month is rounded down to a half day, and each month credits the difference from the previous month's rounded figure, so a full year without a seniority step lands exactly on 12 (13, …).
- **Seniority step in its year**: Article 114 says only "cứ đủ 05 năm … được tăng thêm tương ứng 01 ngày"; nothing says whether the year the fifth year completes gets the whole day or a share. Assumption, on the accountant's list: pro rata by month (completing it at the end of June gives 12 + 6/12 = 12.5).
- **Timing**: a month's days are credited on the 1st of the next month (earned, and the half-month rule is judged on facts). January starts with carried days only.
- **Carry-over**: no statutory expiry exists; it is company policy. Default: every unused day carries into the next year and expires at the end of March.

There is no production data yet (the production-readiness gate is on hold), so upgrading needs no conversion of hand grants: the default `monthly` applies to dev and demo data, which the seed and the tests account for.

## Settings (per legal entity, rendered by the existing settings screen)

- `hrm.leave_accrual` = `monthly` | `manual`, default `monthly`. `manual` is today's behaviour: HR grants by hand. Turning `monthly` on mid-year credits every counted month of the year so far (and, in January, of last year), which doubles any hand grant; turning it back to `manual` credits nothing more and removes nothing. The settings screen says both in one line under the option (an optional `<key>.hint` i18n text, shown by the core settings edit dialog when it exists), and the change is audited like every legal-entity setting.
- `hrm.leave_carry_over_months` = `0` | `3` | `6` | `12`, default `3`. `0` turns carry-over off; carried days expire at the end of that month of the new year.

The legal entity is the employee's current org unit's, read when the job runs.

## Data

One migration (`hrm_leave_accrual`):

- `hrm.leave_accruals (employee_id, year, month, days numeric(4,1), PRIMARY KEY (employee_id, year, month))`: the ledger of monthly credits. A row is written for every month the job has decided, including uncounted months (0 days), so no month is decided twice. The primary key is what makes a credit idempotent.
- `hrm.leave_balances` gains `carried numeric(5,1) NOT NULL DEFAULT 0` (carried days still unspent) and `carried_until date` (their expiry), with `CHECK (carried >= 0 AND carried <= days)`. `days` stays the total remaining, carried included.
- `hrm.leave_requests` gains `start_half`, `end_half boolean NOT NULL DEFAULT false`, `computed boolean NOT NULL DEFAULT false` (set on every save from now on; existing rows stay `false`) and `deducted_carried numeric(4,1) NOT NULL DEFAULT 0` (the carried part of `deducted`).

## The job

`hrm.leave_accrual`, a system job listed in the module's `Periodic` (hourly, run on start), written to catch up rather than to react. For each employee not terminated before last year, in its own transaction, in this order:

1. **Accrue last year** (in January, or later if the employee already has ledger rows for last year, so a server off all January still catches up while turning `monthly` on in, say, October never back-fills the previous year): every month not in the ledger, through December.
2. **Carry over** (once the year has turned): move last year's remaining `days` into this year's row, add them to `carried`, set `carried_until` to the end of the setting's month of this year; with `0` that is 31 December of last year, so step 4 lapses them at once, audited like any expiry (my call: unused days never sit stranded in a year no request can reach). Last year's row is left at 0, which makes the step idempotent; a later refund or adjustment to last year is carried by the next run.
3. **Accrue this year**: every month not in the ledger, through the last completed month (a `manual` legal entity skips 1 and 3).
4. **Expire**: once today is past `carried_until`, take the remaining `carried` out of `days`.

**Leavers.** Once an employee's `termination_date` has passed, the job only credits the months up to and including the termination month (step 1 or 3); it never carries over into a later year and never expires their carried days. Otherwise a run after the termination payroll closed would change a balance already paid out.

Each step locks the `leave_balances` row (`FOR UPDATE`, through `AddLeaveBalance`) like approval and adjustment, so the lock order and the non-negative rule are unchanged. Each credit, carry and expiry is audited on the employee (`hrm.leave_accrued`, `hrm.leave_carried_over`, `hrm.leave_carry_expired`) with the days and the resulting balance, so the employee's history explains the balance. A failing employee fails the run after the others are done; the next run retries it.

## Spending carried days

- Approving a request of a deducting type whose start date is on or before `carried_until` spends `min(carried, days)` of carried days first; `deducted_carried` records how many.
- Cancelling gives `deducted` back to `days`; the carried part goes back to `carried` while it has not expired, otherwise it is dropped (it would have expired).
- Any change to `days` caps `carried` at the new `days`, so a negative hand adjustment never breaks the check.
- Known limit: a December request approved after the new year finds last year's row carried away and is refused for insufficient days; approve before year end, or HR adjusts last year back (the next run carries any remainder).

## Computed days

- The request takes `start_half` and `end_half` instead of `days`. `days` = working days of the range (neither a weekly day off nor a holiday) − 0.5 for each half flag. The halves apply to the first and last working date of the range; a range with a single working date accepts one flag only (`leave_half_days`).
- Allocation: one day per working date, except 0.5 on the first working date with `start_half` and on the last with `end_half`. Old rows (no flags) keep the greedy allocation of their typed count: the same function with no halves is exactly that.
- A draft or pending request saved before this change (`computed = false`) is approved under the ADR-0030 rules (typed count, capped by the working days), so no existing request is refused for a count it could not express (one day over a three-day range). Editing it makes it computed. (My call.)
- `days` is computed and stored on every save. Approval recomputes it from the calendar in force; if it differs from the stored figure, approval is refused with `leave_days_changed` and the requester edits and resubmits. Approved days never change.
- The API keeps `days` as an optional input: the count the client expects. When given, it must equal the computed count, or the save is refused with `leave_days_changed` (the same check approval makes). Every existing client and test that sends a full-range count keeps working unchanged; the form stops sending it. (My call, so the existing leave tests run unchanged.)
- `leave_days_exceed_working_days` and the typed form field go away. `leave_no_working_days`, `max_days`, overlap ("a date holds at most one day of absence") stay, now on the explicit halves.
- The form shows the computed count read-only (working days from the existing endpoint minus the halves) and two checkboxes: "First day: afternoon only" and "Last day: morning only"; when start = end, one "Half day" checkbox.

## Payroll payout on termination

The untaken-leave payout uses the balance plus the days earned but not yet credited up to the termination date (the termination month itself, credited by the job only after the employee has left). One function computes those days for both the job and the payroll, so they cannot disagree. The payroll's inputs hash covers the sum, which does not change when the job later credits it. Carried days are already in the balance until they expire.

## Tests

Go, against Postgres, test-first:

- Accrual arithmetic: full year, joiner on the 10th and on the 20th, leaver mid-month, seniority crossing 5 years mid-year, December total exact.
- The job: a leaver terminated in March with carried days is not touched by the April run (no expiry), nor carried into the next year; idempotent on rerun; catches up three missed months in one run; `manual` entity untouched; carry-over and expiry; turning `monthly` on mid-year does not back-fill last year; balance history audited.
- Carried spending and refund before and after expiry; negative adjustment caps `carried`.
- Computed days with halves; single-date both flags refused; approval refused with `leave_days_changed` after a holiday is added; overlap with a morning and an afternoon on one date allowed, half + full refused.
- Payroll: a leaver with accrual on is paid the termination month's days; `TestPayrollSamples` stays on `manual` and unchanged.

Existing tests and e2e specs that grant balances by hand set their legal entity to `manual`. The demo seed relies on accrual (the lead runs it). An e2e spec covers the form's computed count and half days.

Termination is read from `hrm.employees.termination_date`, which feature B's termination document sets; nothing here depends on B's branch.

## Still open

- Rounding of a partial year's entitlement (Decree 145/2020, Article 66 is silent; we round the cumulative figure down to a half day) and the seniority step in the year it falls (we pro-rate it by month). Both decide the termination payout: an accountant confirms.

- Whether unpaid leave over a month and sick leave over the BHXH limit should stop accrual (Decree 145/2020, Article 65): needs an accountant.
- Whether the company's internal rules (nội quy) set a different carry-over cap or expiry than the default.
- Seniority counted from `hire_date` only; prior service with the same employer under an earlier contract chain, or state-sector time (Article 66.3), is a hand adjustment.

## Left to the builder

File layout inside `internal/modules/hrm` (an `accrual.go` area), the exact audit payloads, i18n wording.
