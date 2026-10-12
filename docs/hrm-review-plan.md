# HRM business review and core verification plan

Created: 2026-10-10. Status: review plan, not a test report.

## 1. Purpose and how to use this document

This document guides the review of the HR business flows through concrete scenarios: employee profiles, contracts, leave, overtime, timesheets and payroll. The results are used to complete the HRM requirements and to decide which core capabilities need fixing or adding.

Each flow must answer six questions:

1. Who starts it, who handles it, who may see it?
2. Which data and configuration must exist before it can start?
3. When is a status transition allowed, and when must it be refused?
4. Which data does a successful operation change?
5. What happens on a wrong entry, a rejection, a withdrawal or a cancellation?
6. How does the user get notified, open the document and return to their work?

Tell three kinds of information apart throughout the review:

| Kind | Meaning |
| --- | --- |
| Existing rule | A requirement written in the project documents; this does not yet mean the code implements it correctly |
| Observation | Behaviour that was actually run, with data, accounts, version and evidence |
| Proposal or open question | Not settled; it does not automatically become a requirement to implement |

Internal sources: [glossary](../CONTEXT.md), [HRM scope](../products/hrm.md), [document lifecycle](../documents.md), [architecture decisions](adr/). If a proposal changes an accepted decision, name the decision it affects and update the ADR before implementing the architectural change.

## 2. Scope and order

| Order | Flow | What to achieve |
| --- | --- | --- |
| 1 | Grant balance → request leave → approve → cancel | Understand leave balances, approvers, refunds, history and notifications |
| 2 | Profile → contract → addendum | Understand effective dates, the salary view permission and the payroll inputs |
| 3 | Actual days worked + leave + overtime → close the timesheet | Tell the data sources apart and find duplicates, gaps and wrong periods |
| 4 | Create payroll → compute → check → approve → close | Understand source data, recalculation, period locks and adjustments |
| 5 | Checks across every flow | Permissions, notifications, approval inbox, attachments, history and concurrency |

The first round finishes the leave flow before widening. Recruitment, performance reviews, training, time clocks and salary payment through banks are out of this review.

## 3. Test data and accounts

The dataset already exists: `make reset-db` followed by `make seed` builds the Demo company through the API — org chart, accounts, employees, contracts, leave, overtime, timesheets and payrolls (`web/e2e/demo.seed.ts`). `make reset-db` is also the "restore to the same starting state" this review needs for each scenario. Extend the seed when a scenario needs more data; never build a second dataset by hand beside it.

What the dataset has to contain for this review:

| Part | Data needed |
| --- | --- |
| Organisation | One legal entity, two departments |
| Employees | Two with an account; one without an account, so HR enters on their behalf |
| Managers | Two, one per department; the direct-manager relation configured |
| Business accounts | Employee, manager, HR, balance granter, payroll preparer, and someone who only sees the totals |
| Administrator account | Used to set the environment up; run business scenarios with the matching business role instead |
| Leave types | One that deducts the balance and is company-paid; one that neither deducts nor is paid |
| Leave balance | For example 12 days per person per year, for testing; this is not a legal entitlement |
| Work calendar | Working days, weekly rest days and simulated public holidays |
| Contracts | One in effect, plus an addendum changing the salary mid-period |
| Pay periods | One open and one closed, on simulated data |

Each run records the application version, the time zone, the business date, the approval configuration, the permissions and the opening balances. Date-dependent scenarios use fixed points in time.

Use development or simulated data only.

## 4. Leave review

### 4.1. Existing rules to verify

Per the [HRM scope](../products/hrm.md#leave):

- Leave balances are per employee and year; a leave request starts and ends within the same year.
- The number of leave days is entered manually in half days, and is only checked not to exceed the calendar days of the leave range. Computing it from the work calendar comes later.
- A leave type has two separate flags: deducts the balance, and company-paid.
- Only a request in effect deducts the balance; cancelling a request in effect refunds it.
- Granting or adjusting a balance needs a dedicated permission, a mandatory reason and an audit entry; it may not make the balance negative.
- When two requests are approved at the same time and the balance does not cover both, the one processed later must not make the balance negative and stays pending approval.
- A source change affecting a closed pay period must be refused by the affected date range.

### 4.2. Scenarios

The "Automated" column names the test that already covers the scenario, so manual effort goes where there is none. It was first written from test names alone; every row below has since been read against the test body, and the column now says what the test actually asserts. Where the two disagreed the row was corrected, and what the reading turned up is in the [round-one observations](hrm-review-round1.md#what-reading-the-automated-rows-changed).

| Code | Scenario | What to verify | Automated |
| --- | --- | --- | --- |
| NP-01 | Grant a balance, adjust it up and down | Permission, mandatory reason, balance before/after, history | Yes: `TestAdjustBalanceNeedsReason` (reason), `TestBalanceAdjustmentsRaceApproval` (adjust up, a negative adjustment refused), `TestAdjustBalanceIsGuardedAndAudited` (HR without `leave_admin` refused, a downward adjustment succeeds, one audit entry per adjustment with the actor, delta, resulting balance and reason) |
| NP-02 | Request one day of leave, then approve it | When the balance is deducted; the right person is asked and handles it | Partly `TestLeaveSampleFlow`: it covers the whole flow and the deduction, but with a role-based approver. The direct-manager routing is covered by `TestLeaveOverlapRefusedOnApproval` and the probes, not here |
| NP-03 | Request half a day | Days and balance stay consistent; effect on paid workdays | `TestLeaveDaysCappedByWorkingDays` (half days accepted up to the cap) and `TestLeaveAllocation` (1.5 days lands as 1 + 0.5 on the paid days). `TestLeaveDaysAndDates` only creates one and was the earlier, overstated mapping |
| NP-04 | Leave spanning a weekend or a public holiday | The entered count is capped by the working days of the range; how it is spread when payroll runs | `TestLeaveDaysCappedByWorkingDays`, `TestLeaveDaysRecheckedOnApproval`, `TestWorkingDays`, `TestLeaveAllocation` ([ADR-0030](adr/0030-leave-day-allocation.md)) |
| NP-05 | Request more than the remaining balance | Where it is refused, what it reports, and the state left behind | `TestLeaveSampleFlow` step 7: `insufficient_leave_balance`, the request stays pending, the balance is untouched |
| NP-06 | Two requests covering the same days, fully or partly | A date holds at most one day of absence across approved requests | `TestLeaveOverlapRefusedOnApproval`, `TestLeaveOverlapCountsHalfDays`, `TestLeaveOverlapFreedByCancelling`, `TestLeaveOverlapCountsEveryType`, `TestLeaveOverlapIsPerEmployee`, `TestOverlappingRequestsApprovedAtOnce` ([ADR-0030](adr/0030-leave-day-allocation.md)) |
| NP-07 | Reject → edit → resubmit | The old approval instance closes, a new one is created, history is kept | Yes: `TestRejectedLeaveReturnsToDraft` (only the approver rejects, back to draft with the balance untouched, the old ticket closed, edit and resubmit through a new ticket), and `web/e2e/m3.spec.ts` shows the reason in the UI |
| NP-08 | Withdraw while pending approval | The old approval instance accepts no action; no balance is deducted | Covered, not partly: `TestLeaveSampleFlow` steps 3–5 withdraw, edit, resubmit and then refuse the stale ticket with `approval_closed`, and the balance only moves at step 6. The `recordtest` row "approving a withdrawn submission is refused" holds it for every document type |
| NP-09 | Cancel an approved request | The exact number of days is refunded once; list, detail and balance all update | `TestCancelRefundsWhatWasDeducted`, including the subtle part: the refund follows what was deducted even after the type's deduct flag changed. The list and detail refresh is not asserted |
| NP-10 | Leave that neither deducts the balance nor is paid | The balance is untouched; paid workdays reflect the leave type | `TestProbeNP10UnpaidBalance` (observation) and `TestLeaveAllocationSkipsUnpaidInPayroll` |
| NP-11 | A request crossing a year boundary | Refused per the current scope, with clear guidance to split the request | `TestLeaveDaysAndDates` (`leave_spans_years`), and `TestLeaveWorkingDaysRejectsUnboundedRanges` for the same bound on the working-days lookup |
| NP-12 | No manager, or a manager without an account | Falls back to the fallback role per the approval rule | `TestProbeNP12NoManager` (observation: both cases fall back to `hrm.hr`) |
| NP-13 | The employee on leave is also an approver | No self-approval; that person is excluded and a replacement is found | `TestTheEmployeeNeverApprovesTheirOwnLeave`: the employee cannot approve, reassigning to them gives `self_approval`, and linking another account to the employee afterwards still blocks. **The scenario title used to say "submitter"**, which is not the rule: `Subjects` is the employee on the request, so HR filing for someone else may approve it |
| NP-14 | Two approvers at once, or a double click on approve | The balance is not deducted twice; it never goes negative; an error leaves no half-written data | `TestTwoRequestsApprovedAtOnce`, `TestNegativeAdjustmentRacesApproval`, `TestLeaveSampleFlow` step 6 (two approvers on one submission, deducted once) |
| NP-15 | HR creates a request for an employee without an account | The employee on leave is told apart from the user performing the operation | `TestProbeNP15HRForEmployeeWithoutAccount` (observation) |
| NP-16 | Cancel or approve a request affecting a closed period | Refused for the right period; no balance or source change when refused | `TestLeaveSampleFlow` step 12 (the cancel-versus-lock race both ways) and the `recordtest` rows "documents in a locked period are frozen" and "a write racing a period lock never lands in the locked period". **`TestPendingLeaveBlocksPeriodLock` was the earlier mapping and tests the opposite direction**: a pending document refusing the lock (`period_has_pending_documents`) |
| Permissions | Who may see and act on a request | Checked on every access path | `TestLeavePermissions`, for the request itself: view, create, list, the allowed actions per status, withdraw only by the sender, cancel only by HR. The leave-balance permission is NP-01's `TestAdjustBalanceIsGuardedAndAudited` |

The first round therefore spends its manual effort on NP-10, NP-12, NP-15 and the checks in section 8; NP-04 and NP-06 were settled and are now asserted. For the automated rows, read the test instead of repeating it by hand, and only run the scenario when the test turns out to cover less than the row asks for.

**NP-06 had no rule anywhere, and NP-04 had two that disagreed.** Contracts check for overlap (`ContractOverlaps` in `internal/modules/hrm/queries.sql`, which is what HD-01 covers), but nothing checked whether two leave requests cover the same days, neither in the code nor in [the HRM scope](../products/hrm.md#leave). Both entered the decision table from the start rather than as "record the current behaviour", and both were settled on 2026-10-11 by [ADR-0030](adr/0030-leave-day-allocation.md): a request's days are allocated per date, the entered count is capped by the working days of the range, and a date holds at most one day of approved absence. The reasoning and the worked examples are in [the decision proposal](hrm-review-decisions.md).

### 4.3. Questions for HR to confirm

- Which leave types does the organisation really need to keep apart? Which ones deduct the balance, and which are company-paid?
- Does the policy allow leave in advance, carrying leave over to the next year, or accruing it monthly? If so, write it up as a proposal to widen the scope.
- Does a half-day request need to distinguish morning from afternoon in order to detect overlaps?
- When a manager is away, who handles the request instead, and for how long?
- Must an approver give a reason when rejecting? Who may cancel an approved request?

## 5. Employee profile, contract and addendum review

An employee profile and a sign-in account are two different things. The direct-manager relation is likewise not the same as the permission scope over the org tree.

| Code | Scenario | What to verify |
| --- | --- | --- |
| HS-01 | An employee with no account | HR still manages the profile and performs the operations it is allowed |
| HS-02 | Making yourself or a subordinate the manager | A manager relation that forms a cycle is refused |
| HS-03 | An employee joining or leaving mid-month | The effective date reaches the timesheet and the payroll correctly |
| HD-01 | Two original contracts whose terms overlap | Two contracts in effect for one employee over the overlapping range are refused |
| HD-02 | A fixed-term contract with no end date | Refused per the contract type |
| HD-03 | An addendum changing the salary mid-month | The right amount is taken for each effective stretch |
| HD-04 | Editing or cancelling a contract in effect | No direct edit; cancelling has to consider addenda and the pay periods involved |
| HD-05 | Someone with the profile permission but not the salary permission | Cannot see money items, nor contract files that contain salary |

The output of this section is a timeline showing which contract and which addendum is in effect on each date. Use it as the reference data for the payroll checks.

## 6. Timesheet, work calendar and overtime review

In the current scope a timesheet belongs to an org unit and a pay period, and records actual days worked of 0.5 or 1. Leave requests, the work calendar and overtime requests are separate sources. Clock-in and clock-out data is not treated as a confirmed timesheet.

| Code | Scenario | What to verify |
| --- | --- | --- |
| CC-01 | Create two timesheets for the same unit and period | There are never two non-cancelled timesheets for the same scope |
| CC-02 | Enter half a day or a whole day | Stored and displayed consistently |
| CC-03 | A day with both work and company-paid leave | Verify the paid-workdays cap in the existing rules; state how duplicate data is warned about |
| CC-04 | Work recorded on a rest day or a public holiday | Tell normal work from overtime; check how the current payroll treats it |
| CC-05 | An Excel import with one bad row | Nothing is written partially; the row and the reason are reported |
| CC-06 | Re-import Excel into a draft timesheet | Confirm the new data replaces every old line; the action has to be clear enough to the user |
| CC-07 | Change the work calendar with an effective date | Days before and after the change are computed per the right version |
| CC-08 | A calendar or timesheet change affecting a closed period | A source change to a closed period is refused |
| TC-01 | Overtime on a working day, a rest day and a public holiday | The day kind is entered and checked explicitly; do not assume the system derives it from the calendar |
| TC-02 | Daytime and night overtime | The two groups of hours are stored separately and reach the payroll inputs correctly |
| TC-03 | A rejected or cancelled overtime request | It can no longer be used as a source in effect |

Rules about shifts, late arrival, early leaving and missing clock entries are written up separately if they come up; do not pull them into the current scope.

## 7. Payroll and period-closing review

Before testing, prepare a reference spreadsheet confirmed by whoever is responsible for payroll. Every amount must be traceable to its source data, effective date, formula and rounding rule.

This document does not set tax rates, insurance rates or legal coefficients. Those parameters have to be confirmed as of the date they apply. The [payroll samples](payroll-samples/README.md) are an input to the review; whether a specialist has confirmed them is recorded separately.

| Code | Scenario | What to verify |
| --- | --- | --- |
| LG-01 | Create a payroll while a posted timesheet is missing | Refused, naming the units still missing one |
| LG-02 | A full month, joining mid-month, leaving mid-month | The employee list and the paid workdays are right per person |
| LG-03 | An addendum changing amounts mid-period | Each effective stretch is split correctly; one amount is not used for the whole period |
| LG-04 | Unpaid leave, overtime and adjustments together | Every item has a basis and can be reconciled |
| LG-05 | Sources change after the first computation | The "source data has changed" warning appears; recalculation is required before submitting |
| LG-06 | Sources change while pending approval | The final approval step does not close on stale data; it has to be withdrawn, recalculated and resubmitted |
| LG-07 | The user asks for a recalculation | Only on a draft; with history, progress and the result of the background job |
| LG-08 | Two payrolls for the same period are both closed | Only one payroll is in effect per legal entity and period |
| LG-09 | A source is edited while the payroll is being closed | A closed payroll never ends up using inconsistent source data |
| LG-10 | An error found after closing | Walk the back-pay and clawback flow in the open period; tell it apart from cancelling to reopen the period |
| LG-11 | Reopen or print a closed payslip | Frozen data does not change with a newer profile or configuration |
| LG-12 | Someone who may only see totals opens a detail, a file or a print | Permission is checked on every access path; nobody's individual amount leaks |

Whoever is responsible for payroll has to state clearly which cases allow cancelling a payroll to reopen the period, and which have to be adjusted in the open period. This is an operating decision to be recorded, not something to infer from the system having a cancel button.

## 8. Notifications, approval inbox and working context

The first item below is existing behaviour and needs a regression check; the rest are proposals to be settled and tested, not conclusions about the current UI.

- Existing behaviour: opening a notification, the approval inbox or background jobs stays inside the area the user is working in (shipped in `238c82f`).
- Picking an item leads to the right document and to the action the user is allowed to perform.
- Going back restores the list, its filters and the previous position where that makes sense.
- Configuring approval rules is clearly separated from the work waiting for the user.
- Reading a notification does not mean the document has been approved.

| Code | Scenario | Expected result |
| --- | --- | --- |
| TB-01 | In HRM, open a notification or the approval inbox | Does not jump to the administration area; regression check on `238c82f` |
| TB-02 | Open a request from a notification | The right document; view permission re-checked at the moment it opens |
| TB-03 | Approve successfully from the inbox | The pending item updates; detail, list and leave balance all reflect the result |
| TB-04 | Someone else already handled it, or the submitter withdrew it | Approving again is refused; the current state is explained |
| TB-05 | Permission revoked after the notification arrived | No document content leaks; there is a way back |
| TB-06 | No mail server configured | In-app notifications still work |
| TB-07 | Go back after opening and handling a document | The list context that was in use is restored |

`web/e2e/m12.spec.ts` already touches TB-03 and TB-06. Run the rest with separate submitter and approver accounts, and check both the permission to view the document and the right to decide at the current approval step.

## 9. HRM and core responsibilities

| HRM responsibility | Core responsibility |
| --- | --- |
| Leave types, balances, deductions and refunds | Document lifecycle, approval instances and decision history |
| The direct-manager relation | The callback that resolves approvers, the fallback and the permission checks |
| Work calendar, days worked, overtime hours | Notifications, approval inbox and the path that opens a document |
| Contract and addendum effective dates | Attachments, discussion, numbering and printing |
| Payroll source data, locking sources per pay period | The shared period lock and coordination with the document lifecycle |
| Business formulas, balance checks | Permissions, audit and background jobs |

Every proposal to add something to the core needs a real scenario that uses it and a criterion to verify it. Rules for computing leave, workdays or pay stay with HRM.

Check the core column against Sales as well, not only HRM: `internal/modules/sales` gives the core a second product to be proven against. The HRM-specific assumptions already found in the core are filed as issues #1 to #15; the shared period lock is #5.

## 10. Recording evidence and settling requirements

### 10.1. Scenario sheet

```text
Scenario code:
Business goal:
Application version and run date:
Role, permissions and org unit scope:
Starting data, configuration and business date:
Steps:
Expected result and its basis:
Actual result:
Data before/after:
Evidence: screenshot, document number, API response or operation history
Conclusion: pass / fail / not enough to conclude
Points to confirm, and who owns them:
```

### 10.2. Decision table

| Question | Existing rule | Evidence | Decision | Confirmed by | Acceptance criterion |
| --- | --- | --- | --- | --- | --- |
| Two leave requests covering the same days (NP-06) | None, in the documents or the code | Measured: the balance charged both requests while the payroll paid one day | Settled 2026-10-11, fix within the current scope: a date holds at most one day of approved absence ([ADR-0030](adr/0030-leave-day-allocation.md)) | HR owner | Met: the examples of [the proposal](hrm-review-decisions.md) section 4 are the tests of NP-06 |
| What the entered day count means against the calendar (NP-04) | Entered manually, checked against the calendar days of the range | Measured: up to three days of balance taken and never paid | Settled 2026-10-11, fix within the current scope: capped by the working days of the range, and allocated per date ([ADR-0030](adr/0030-leave-day-allocation.md)) | HR owner | Met: the examples of [the proposal](hrm-review-decisions.md) section 3 are the tests of NP-04 |
| Computing leave days from the calendar instead of entering them | Entered manually today | The cap above removes the harm; deriving the count is still deferred | Widen the scope later; ADR-0016 already intends it | HR owner | Agree a set of examples across rest days and public holidays, and how an approved count is frozen against later calendar edits, before implementing |
| Keeping the area context when opening the inbox | Existing behaviour, shipped in `238c82f` | TB-01, TB-02 and TB-07 re-run and asserted | Keep as is | Product owner | TB-01, TB-02 and TB-07 pass |
| Explaining a refused decision in the inbox (TB-04) | The plan states the expected result: refused and the current state explained | Measured: refused correctly, explained only outside the inbox | Fixed within the current scope: the chosen submission stays on screen after it leaves the list | Product owner | Met: the TB-04 rows of `web/e2e/review-probe.spec.ts` assert the explanation and that it stays |

Decision values: keep as is, fix within the current scope, widen the scope later, or needs more confirmation. Do not mark a difference as a defect without a basis for the expected result.

### 10.3. How to handle a finding

1. Reproduce it with a fixed scenario and fixed data.
2. Compare it against the glossary, the business documents and the relevant ADRs.
3. Read the matching tests and code path to find the cause.
4. Settle the behaviour with the business owner when the rule is missing.
5. Write the acceptance criterion and decide whether it belongs to HRM or to the core.
6. Update the documents or ADRs needed, then implement and verify.

## 11. Exit criteria for one review round

- Every scenario in scope has a result, or a reason why it cannot be concluded yet.
- The normal path, rejection, withdrawal, cancellation, missing permission and concurrency have all been considered.
- Every change to a balance, a workday count or an amount has before/after data to reconcile.
- Open business questions have an owner to confirm them; they do not count as passed.
- Findings are classified as an implementation defect, a missing rule, a UI problem or a need outside the scope.
- Every change picked for implementation has an acceptance criterion.

The first round produces the leave dataset, sheets NP-04, NP-06, NP-10, NP-12 and NP-15, a read-through of the automated rows in section 4.2, the notification checks in section 8, and the decision table. Use those results to decide the order of HRM and core fixes before starting the timesheet and payroll rounds.

**The first round is done** (2026-10-11). Its results are in [the observations](hrm-review-round1.md), its two business decisions in [the proposal](hrm-review-decisions.md) and [ADR-0030](adr/0030-leave-day-allocation.md), and section 4.2's automated column has been read against the test bodies. Round two continues in [its observations](hrm-review-round2.md): round one's leftovers (NP-01, NP-07, TB-02) and section 5 are done, sections 6 and 7 are next.
