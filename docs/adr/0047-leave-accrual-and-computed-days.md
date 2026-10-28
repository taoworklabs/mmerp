# 0047. Annual leave accrues monthly by a catch-up job, and a leave request computes its days from the calendar

Status: accepted · 2026-10-28 · supersedes, for requests saved from now on, the typed count and greedy allocation of [ADR-0030](./0030-leave-day-allocation.md); amends [ADR-0016](./0016-leave-requests.md) (balances granted by hand only)

## Context

HR grants every leave balance by hand each January and corrects it for joiners, leavers and seniority. The law already defines the figure: 12 days a year (Labour Code 2019, Article 113.1a), pro rata by months worked for a partial year (Decree 145/2020, Article 66), one more day per five years of service (Article 114). ADR-0030 kept the day count of a request typed, capped by the calendar, and deferred computing it, because computing it needed a half-day marker and a count frozen at approval.

## Decision

- **Accrual is a system job that catches up.** `hrm.leave_accrual` runs hourly and at start. For each employee, in its own transaction, it decides every month not yet in the ledger `hrm.leave_accruals (employee_id, year, month)` through the last completed month, and credits the difference between the year's cumulative entitlement through that month, rounded down to a half day, and through the month before. The primary key makes a credit idempotent; a server that was off on the 1st credits every missed month at its next run. A month counts when the employee is employed on at least half of its working days. A month is credited on the 1st of the next month.
- **A leaver's balance is final.** After `termination_date` the job only credits months up to and including the termination month; it neither carries their days into a later year nor expires them, so nothing changes a balance the termination payroll has paid out.
- **Rounding is an assumption.** Decree 145/2020, Article 66 prescribes the proportional formula and the half-month rule but no rounding; the cumulative figure is rounded down to a half day, and the seniority step is pro-rated in the year it falls, until an accountant says otherwise.
- **A legal-entity setting chooses the mode.** `hrm.leave_accrual` = `monthly` (default) or `manual`, which keeps hand grants as the only source.
- **Unused days carry over and expire.** On the new year, the job moves last year's remaining days into this year's balance as `carried`, expiring at the end of month `hrm.leave_carry_over_months` (`0`, `3` default, `6`, `12`) of the new year; `0` lapses them at once. Approved leave dated up to the expiry spends carried days first and records how many (`deducted_carried`), so a cancel returns them to where they came from, or drops them if they have expired by then.
- **The balance row is the single lock.** Credit, carry and expiry change `hrm.leave_balances` through the same `UPDATE … RETURNING` as approval and adjustment, so the lock order and the non-negative check are unchanged; `carried` never exceeds `days`. Every change is audited on the employee.
- **A request's days are computed.** The request carries `start_half` and `end_half` instead of a typed count; its days are the working days of the range less half a day per flag, the halves falling on the first and last working date. The allocation places exactly that: one day per working date, a half on a flagged end. The count is computed on save (a client may send the count it expects, refused if it differs); approval recomputes it and refuses with `leave_days_changed` if the calendar has changed it since, so approved days never move. Requests saved before this decision keep their typed count and the ADR-0030 rules until edited.
- **The payout of untaken leave counts what is earned but not credited.** Because a month is credited after it ends, a leaver's last month would reach the balance after the payroll that pays them out. The payroll adds the days the job would credit up to the termination date, computed by the same function.

## Rejected alternatives

- **Crediting at the start of the month.** Usable a month earlier, but it credits a month the employee may not work, and the half-month rule cannot be judged in advance.
- **Granting the whole year on 1 January.** That is what `manual` already allows; it is not accrual.
- **A run-date marker instead of a ledger.** A single "last run" date cannot tell which employees a failed run already credited; a row per month can.
- **Refunding carried days into the ordinary balance on cancel.** It would let a request filed before the expiry and cancelled after it extend carried days indefinitely.
- **Silently storing the recomputed count on approval.** The approver approved a figure; changing it after the fact is the silent loss ADR-0030 set out to remove.
- **Converting existing typed requests.** Some typed counts cannot be expressed with two half flags (one day over a three-day range); they keep the rules they were entered under.

## Consequences

- January starts with carried days only: the first month of a year is credited on 1 February.
- Turning `monthly` on mid-year credits every counted month of the year so far, on top of any hand grant, which HR takes back.
- 14 and 16-day entitlements, and pauses for long unpaid or sick leave (Decree 145/2020, Article 65), stay hand adjustments.
- A December leave request approved after the new year finds last year's balance carried away; it is approved before the year ends or the days are adjusted back.
