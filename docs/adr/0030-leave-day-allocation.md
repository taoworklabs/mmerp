# 0030. A leave request places days on dates: the calendar caps the entry and a date holds at most one day

Status: accepted · 2026-10-11 · amends [ADR-0016](./0016-leave-requests.md) (`days` "no more than the number of calendar days in the range")

## Context

Reviewing the leave flow against the work calendar and the payroll turned up one missing notion with two symptoms. Nothing says how many days of absence a request places on each individual date, so three places answer differently:

- The entry checks the total only against the count of calendar days in the range (`leaveHeader`).
- The leave balance deducts that total and never looks at dates or at the calendar (`leaveTransition`).
- The payroll spreads the total over the working days of the range, at most one per date, drops whatever does not fit, adds up across requests, and then caps worked plus leave at one day per date (`leaveByDay`, and the cap in `payrollInputs`).

Measured consequences, each one silent: four days entered over a Friday-to-Monday range take four days of balance and are paid two; two days entered on a Saturday and a Sunday take two and are paid none; two approved requests sharing one date take the balance for both and are paid for one. In every case the employee loses paid leave they are never paid for. The per-date figure existed only inside the payroll's own allocation, where neither the balance nor any overlap check could see it.

ADR-0016 already decided that `days` "will be computed once a work calendar exists" and deferred it; the deferral is what left the three answers unreconciled. Separately, nothing anywhere — not ADR-0016, not `products/hrm.md`, not the code — said whether two requests may cover the same days. Contracts check overlap (`ContractOverlaps`); leave did not.

## Decision

- **A leave request's day count is allocated to dates.** The allocation spreads the entered days over the days of `[start_date, end_date]` that are neither a weekly day off nor a holiday, in date order, at most one day per date. This is what the payroll already did; it becomes the single notion the balance, the payroll and the overlap check all read, and `leaveByDay` is expressed in terms of it.
- **The entry is capped by the calendar, not by the calendar days of the range.** `days` must not exceed the number of days in `[start_date, end_date]` that are neither a weekly day off nor a holiday, under the work-week version and holidays in force for those dates. A range holding no such day is refused outright (`leave_no_working_days`): a request that would take nothing and pay nothing is a mistake by whoever entered it. Exceeding the count is `leave_days_exceed_working_days`, carrying the count as a parameter. With this cap nothing is ever dropped from the allocation, so the figure the balance deducts and the figure the payroll pays are the same figure.
- **A date holds at most one day of absence.** Moving a request to `posted` is refused with `leave_overlaps` if any date in it would hold more than one day once every **approved** leave of that employee is counted. Drafts and requests still waiting for a decision block nothing, so two overlapping requests may both be submitted and whichever is approved second is refused. Half days therefore compose: a morning and an afternoon on one date make one day and are allowed, a half day plus a full day make 1.5 and are refused.
- **A half-day request needs no morning/afternoon marker.** The rule above needs only the sum per date, so overlap detection does not require knowing which half. A marker is a separate requirement, for scheduling, and stays out of scope.
- **The checks run in two places, for two reasons.** The entry check in `leaveHeader` gives the person entering the request immediate feedback; it reads the legal entity after `iam.ShareTree`, per [ADR-0029](./0029-withdraw-right-and-tree-lock.md). The authoritative check is in `leaveTransition` on the move to `posted`, because a holiday added to the range after a draft was saved would otherwise let a stale day count through. The transition order is the existing one extended the way contracts do it: period locks, then the employee row (`LockEmployee`), then the overlap check, then the leave balance. Locking the employee is what makes two simultaneous approvals of overlapping requests safe.

## Rejected alternatives

- **Bringing the deferred computation forward now** — deriving `days` from the calendar and making the field read-only. It needs a half-day marker to keep half-day requests expressible, and it needs the derived count frozen onto the request at approval, or a later calendar edit rewrites approved history. Both are real work and the roadmap has not reached it. The cap decided here is a strict subset: when the count is finally computed, the cap becomes redundant and is deleted.
- **Warning instead of refusing**, for either symptom. The loss is monetary and the person entering the request cannot act on a warning differently than on an error.
- **Refusing any shared date**, rather than any date over one day. It would refuse a legitimate morning-plus-afternoon split on one date.
- **Counting drafts and pending requests in the overlap check.** It would refuse the second approval of two requests the employee submitted as alternatives before either was decided, and it would make a draft block a colleague's approval.
- **Checking overlap only when the request is created.** Two approvals committing together would both pass. The check belongs where the lock is.

## Consequences

- An employee can no longer be charged leave for a day nobody pays them for, and cannot hold two approved requests over one date.
- The allocation is greedy by date: a request that takes fewer days than its range offers places them on the earliest working dates. For a request covering its whole range — the ordinary case — this is exact. For a partial one it is arbitrary but defined, and it is what the payroll has always done. Letting the employee place the days themselves is part of the deferred calendar work.
- The entry cap depends on the work calendar, so the leave form has to show the employee the number of working days in the range they picked; without it the ceiling is invisible until the error appears.
- Existing approved requests are not revisited. The cap applies to entries and to approvals from here on, so data captured before this ADR can still hold a count the calendar would now refuse.
- `leaveByDay` keeps skipping unpaid types, because the payroll has no use for them, but the overlap check counts every type: overlap is about absence, not about pay.
