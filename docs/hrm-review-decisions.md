# HRM review: the decisions NP-04 and NP-06 need

Created: 2026-10-11. Status: proposal awaiting the business owner; nothing here is implemented.

Fills the decision table of the [review plan](hrm-review-plan.md) section 10.2 for the two rows that
round one found to have no rule. The evidence is in the [round-one observations](hrm-review-round1.md).
Every figure marked *measured* was produced by a probe; figures marked *proposed* describe what a
rule would do and have not been run.

## 1. The two questions are one question

NP-04 and NP-06 look like separate problems — leave over a weekend, and two requests over the same
days. They are the same gap: **nothing in the system says how many days of absence a leave request
places on each individual date.** Three places need that number and three places answer differently.

| Place | How it answers today | Code |
| --- | --- | --- |
| Entry | One total, in half days, checked only against the count of *calendar* days in the range | `service_leave.go:300` |
| Leave balance | Deducts the total and never looks at dates or at the calendar | `service_leave.go:111` |
| Payroll | Spreads the total over the *working* days of the range, at most 1 per date, drops the remainder, adds up across requests, then caps worked + leave at 1 per date | `service_payroll_inputs.go:379`, cap at `:297` |

The per-date figure exists in exactly one place, `leaveByDay`, is used by exactly one caller, payroll,
and is invisible to both the balance and any overlap check. Both findings follow from that.

So the two rows below share one decision: **make the per-date allocation a single agreed notion, and
make the balance, the payroll and any overlap rule all read it.** The alternatives differ only in how
that notion is produced.

## 2. What it costs today, measured

| Situation | Balance loses | Payroll pays | Employee's loss |
| --- | --- | --- | --- |
| 4 days entered over Fri 04-03 → Mon 04-06, Sat/Sun off | 4 | 2 | 2 days |
| 2 days entered on Sat 04-04 + Sun 04-05 | 2 | 0 | 2 days |
| 1 day entered on the holiday 04-30 | 1 | 0 | 1 day |
| 6 days entered over 04-29 → 05-04, holiday 04-30 | 6 | 3 | 3 days |
| Two 3-day requests sharing 05-06, all working days (NP-06) | 6 | 5 | 1 day |
| Two identical 2-day requests on two working days (NP-06) | 4 | 2 | 2 days |
| 1.5 days entered over Fri 04-03 → Mon 04-06 | 1.5 | 1.5 | none |

The payroll column is measured in every row, and so is the balance column in rows 1, 2, 5 and 6 —
the four the database probe actually approved. In rows 3, 4 and 7 the balance figure follows from the
deduction rule, which subtracts the entered figure unconditionally without consulting dates or the
calendar (`service_leave.go:111-115`), as those four measured rows confirm.

The loss is silent in every row: no warning, no error, nothing in the document.

This is what makes the two rows urgent rather than merely undefined. The system takes paid leave from
an employee and does not pay it. The last row shows the mechanism is not always wrong — when the
entered figure happens to fit the working days available, the three places agree.

## 3. NP-04 — what the entered day count means

**The direction is already decided and only the timing is open.** ADR-0016 says `days` is "entered by
hand and no more than the number of calendar days in the range; it will be computed once a work
calendar exists", and notes on 2026-10-07 that the calendar has existed since M6 while the
computation stays deferred. `CONTEXT.md` repeats it. So nobody has to decide whether to derive the
count from the calendar; that is settled. What is open is what happens in the meantime, because the
deferral is what produces the losses in section 2.

### Option B — keep manual entry, validate it against the calendar (recommended now)

Change one check: the entered days must not exceed the **working** days of the range, instead of its
calendar days. Nothing else moves. Payroll already allocates at most one day per working day, so once
the entry cannot exceed that, the balance figure and the allocation agree by construction.

| Case | Today, measured | Under option B, proposed |
| --- | --- | --- |
| Fri → Mon, 4 entered | accepted; balance −4, paid 2 | refused: only 2 working days in the range |
| Fri → Mon, 2 entered | accepted; balance −2, paid 2 | unchanged, accepted |
| Fri → Mon, 1.5 entered | accepted; balance −1.5, paid 1.5 | unchanged, accepted |
| Sat + Sun, 2 entered | accepted; balance −2, paid 0 | refused: no working day in the range |
| Holiday only, 1 entered | accepted; balance −1, paid 0 | refused: no working day in the range |
| 04-29 → 05-04, 6 entered | accepted; balance −6, paid 3 | refused: 3 working days; entering 3 is accepted |

- Cost: one validation, one error code, and the working-day count shown on the form so the employee
  knows the ceiling. Fully inside the scope the documents already describe.
- It does not pre-empt the deferred work: when the count is finally computed, this validation becomes
  redundant and is deleted. Option B is strictly a subset of option A.
- Open sub-question for the owner: should leave that falls entirely on rest days be **refused**, as
  proposed above, or **accepted and free** (no balance taken, nothing paid)? Refusing is proposed,
  because a request that takes nothing and gives nothing is almost certainly a mistake by the person
  entering it, and silently accepting it today is what costs them 2 days.

### Option A — bring the deferred computation forward

Derive the count from the calendar and make the field read-only.

- Needs a half-day marker to keep half-day requests expressible, which is a schema and API change.
- Needs the derived count frozen onto the request when it is approved, or a later calendar edit
  (scenario CC-07) silently rewrites history. This is a real design question, not a detail.
- Larger than option B and the roadmap has not reached it.

Recommend option B now and option A when the roadmap reaches the calendar-derived count.

### Option C — leave it, warn only

Rejected in the recommendation. The harm is monetary and silent; a warning on a form the employee
cannot act on differently does not undo the deduction.

## 4. NP-06 — two requests over the same days

Nothing covers this: not ADR-0016, not `products/hrm.md`, not the code. Contracts check overlap
(`ContractOverlaps`), leave does not.

### Option 1 — refuse when a date would exceed one day of absence (recommended)

At the transition to `posted`, with the employee row locked, refuse if any date in the request would
hold more than 1 day of absence once every non-cancelled leave of that employee is counted. Error
code `leave_overlaps`, naming the date and the other document.

| Case | Date in question holds | Proposed outcome |
| --- | --- | --- |
| A 04-06 → 04-07 (2 days) posted, then B 04-07 → 04-08 (2 days) | 04-07 would hold 1 + 1 = 2 | B refused |
| A 03-10 → 03-11 (2 days) posted, then an identical B | both dates would hold 2 | B refused |
| A 04-07 half a day, then B 04-07 half a day | 04-07 would hold 0.5 + 0.5 = 1 | both accepted |
| A 04-07 half a day, then B 04-07 a full day | 04-07 would hold 1.5 | B refused |
| A posted and then cancelled, then B over the same days | 04-07 would hold 1 | B accepted |
| A still pending approval, B submitted over the same days | nothing posted yet | both submit; whichever is approved second is refused |

This settles an open question from the plan's section 4.3 as a side effect: **a half-day request does
not need a morning/afternoon marker for overlap detection.** Half days compose to at most 1 and the
rule needs no more than that. A marker is only needed if the organisation wants to know *which* half
for its own scheduling, which is a different requirement and not this one.

Two implementation points the owner does not need to decide but which must not be missed:

- The allocation used for the overlap check has to count **every** leave type, not only the paid ones.
  `leaveByDay` skips unpaid leave (`service_payroll_inputs.go:382`) because payroll has no use for it.
  Reusing it unchanged would let unpaid leave overlap anything invisibly.
- Checking at the transition under `LockEmployee` is the pattern `contractTransition` already uses
  (`service_contract.go:139`), and it is what makes two simultaneous approvals safe. Checking only at
  creation time would not.

### Option 2 — refuse any shared date

Simpler to describe and wrong: it refuses a legitimate morning-plus-afternoon split on one date, which
option 1 accepts. Only choose this if the organisation decides two half-day requests on one date are
themselves an error.

### Option 3 — warn only, or allow

Leaves the double deduction in place. Allowing it silently is today's behaviour and costs the employee
up to a full day per overlapping date.

## 5. The decision table, filled

| Question | Existing rule | Evidence | Proposed decision | Confirmed by | Acceptance criterion |
| --- | --- | --- | --- | --- | --- |
| What the entered day count means against the calendar (NP-04) | Direction settled in ADR-0016, computation deferred; today's check is against calendar days | Measured: up to 3 days of balance taken and not paid, section 2 | Fix within the current scope: option B, entered days must not exceed the working days of the range | HR owner | The six rows of the option B table hold, and no case remains where the balance figure differs from the allocated figure |
| Leave that falls entirely on rest days (NP-04, sub-question) | None | Measured: balance −2, paid 0 | Refuse it | HR owner | A request whose range contains no working day is refused with a named error |
| Two requests over the same days (NP-06) | None, in the documents or the code | Measured: balance −4, paid 3 | Fix within the current scope: option 1, refuse a date exceeding one day of absence | HR owner | The six rows of the option 1 table hold, including both half-day rows |
| Morning/afternoon marker for half days (NP-06, from plan 4.3) | None | Option 1 needs none | Not needed for overlap; widen the scope later only if scheduling needs it | HR owner | Two half-day requests on one date are accepted; a half day plus a full day is refused |

Decision values available, per the plan: keep as is, fix within the current scope, widen the scope
later, or needs more confirmation.

## 6. If this is approved, what follows

1. An ADR amending ADR-0016 with the per-date allocation, the tightened validation and the overlap
   rule — the plan requires the ADR before the code, and ADR-0016 is the decision being changed.
2. `CONTEXT.md` gains the per-date allocation as a term, and its work-calendar and leave-type entries
   are corrected where they describe the old check.
3. `products/hrm.md` gains the two rules in its leave section.
4. Implementation, test-first, with the tables in sections 3 and 4 as the test cases, plus a
   concurrency test for two simultaneous approvals of overlapping requests, following the existing
   `TestTwoRequestsApprovedAtOnce`.
5. The probes keep their place: once a rule exists, the probes for NP-04 and NP-06 stop being
   observations and become assertions.
