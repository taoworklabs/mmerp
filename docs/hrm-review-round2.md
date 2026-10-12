# HRM review, round two: observations

Created: 2026-10-12. Covers what round one left: its two unguarded rows (NP-01, NP-07), TB-02, and
sections 5 and 6 of [the plan](hrm-review-plan.md) (profiles, contracts, addenda; timesheets, work
calendar, overtime). Section 7 is not started yet.

## Run environment

Go tests against Postgres 18 (`local.mk`, `localhost:5433`); Playwright through `pnpm -C web exec
playwright test`, which starts its own servers. Branch `hrm-review-round2`.

## Round one's leftovers

| Row | Result | Evidence |
| --- | --- | --- |
| NP-01 | Pass. HR without `leave_admin` is refused; a cut within the balance goes through; each adjustment writes one audit entry with actor, delta, resulting balance and reason | `TestAdjustBalanceIsGuardedAndAudited` |
| NP-07 | Pass. Only the approver rejects; the request returns to draft, balance untouched, old ticket closed; edit and resend goes through a new ticket | `TestRejectedLeaveReturnsToDraft` |
| TB-02 | Pass. A notification opens the document it names. Revoking the reader's roles while the list still shows a row in full leaks nothing: the changed authz version reloads the page on the next API answer, landing either on the list ("bạn không còn quyền xem") or on the document ("không tìm thấy") | `review-probe.spec.ts` TB-02 |

Each Go test was checked by breaking the code it guards (the permission check, the audit call) and
watching it fail.

## Section 5: profile, contract, addendum

Read against the test bodies, not their names.

| Code | Result | Evidence |
| --- | --- | --- |
| HS-01 | Covered. A profile needs no account; HR files and the approval routes for an employee without one | `TestManagerAndAccountLinks`, round one's NP-15 |
| HS-02 | Covered, now including concurrency. Self and cycle refused with the chosen person's name; the picker hides them. **New:** two edits that close a four-person cycle at once are serialized by the manager tree lock | `TestManagerAndAccountLinks`, `TestManagerCycleFormedAtOnce` |
| HS-03 | Covered. Mid-month hire and termination reach the timesheet and the payroll | `timesheet_test.go` (hire 10/03, leave 27/02), payroll samples C01, C02 and the quit samples |
| HD-01 | Covered, now including concurrency. **New:** two overlapping originals sent into force at once: the employee row lock makes the second wait and refuse | `TestContractRules`, `TestOverlappingContractsPostedAtOnce` |
| HD-02 | Covered. A fixed-term type needs an end date, an indefinite one refuses it, on create and on edit | `TestEndDateFollowsTheKind` |
| HD-03 | Covered. Payroll splits the month at the addendum's date and sums the segments | payroll sample C04 (20 → 25 million from 16/04) |
| HD-04 | Covered. A posted contract is not edited (the `recordtest` suite); an original with a posted addendum cannot be cancelled; a closed pay period inside the affected range blocks posting and cancelling | `TestContractRules`, `TestContractAffectedRange`, `TestContractRecordContract` |
| HD-05 | Covered. Without `hrm.salary.view` the contract shows no terms, cannot be written, and its files are hidden, re-checked on every download | `TestContractPermissions`, `TestContractAttachmentsNeedSalaryView` |

### Contract timeline

The section's output, used as reference data for the payroll round. `TestContractAt` asserts it:
one original from 2026-01-01 to 2026-12-31, addenda from 03-01 (posted), 05-01 (draft), 06-01
(posted) and 07-01 (posted then cancelled).

| Date | Salary in effect | Why |
| --- | --- | --- |
| 2025-12-31 | none | before the original |
| 2026-01-01 to 02-28 | 10,000,000 | the original |
| 2026-03-01 to 05-31 | 11,000,000 | the 03-01 addendum; the 05-01 draft does not count |
| 2026-06-01 to 12-31 | 12,000,000 | the 06-01 addendum; the cancelled 07-01 one does not count |
| 2027-01-01 | none | after the original |

### A trap for concurrency tests

The first versions of both new tests passed with their lock removed. Two things serialized the
transactions instead:

- `checkPayrollPeriods` inserts the legal entity's `hrm.payroll_locks` row on first use, so on a fresh
  test database the second transaction waits on the first one's uncommitted insert.
- Setting a manager takes a key-share lock on the manager's row through the foreign key. That alone
  stops a two-person cycle (X under Y while Y under X), but not a longer one where the two edits touch
  four different rows.

So a concurrency test has to post something beforehand to create the payroll lock row, and has to
build a case that no incidental lock covers. Then it is worth checking that the test fails with the
intended lock removed. The older race tests (`TestTwoRequestsApprovedAtOnce`,
`TestNegativeAdjustmentRacesApproval`, `TestBalanceAdjustmentsRaceApproval`) have not been checked
this way.

## Section 6: timesheet, work calendar, overtime

April 2026 with the payroll fixture: 22 working days, of which 27/04 and 30/04 are public holidays;
a salary of 22,000,000, so one hour of overtime is 125,000 before the multiplier. Probes are in
`timesheet_probe_test.go`.

| Code | Result | Evidence |
| --- | --- | --- |
| CC-01 | Covered. A second timesheet for the same unit and period is refused; a partial unique index holds it under concurrency too | `TestTimesheet`, `timesheets_org_unit_period_key` |
| CC-02 | Covered. 0.5 and 1 stored and read back; other values refused | `TestTimesheet` |
| CC-03 | **Open question, see below** | `TestProbeCC03WorkAndPaidLeaveSameDay` |
| CC-04 | **Open question, see below** | `TestProbeCC04WorkOnDaysOff` |
| CC-05 | Covered. Bad rows are named with their Excel row number and reason, and nothing is written | `TestImportRowErrors` |
| CC-06 | Covered. A save or import replaces every line of the draft (**new** test with a shorter set of lines); the import dialog says so ("File thay toàn bộ ngày công của bảng công nháp này") | `TestTimesheetSaveReplacesEveryLine`, `hrm.timesheet.import_hint` |
| CC-07 | Covered. The grid and the working days use the week version in force on each date | `TestWorkCalendar`, `TestWorkingDaysUsesTheWeekVersionInForce` |
| CC-08 | Covered. Holidays, work-week versions, legal parameters and timesheets touching a closed period are refused | `TestWorkCalendar`, `TestLegalParams`, `TestTimesheet` |
| TC-01 | **Open question, see below** | `TestProbeTC01DayKindAgainstCalendar` |
| TC-02 | Covered. Day and night hours are stored apart and priced apart | `TestOvertime`, payroll samples C11 and C17 |
| TC-03 | Covered (**new** test). A cancelled overtime request drops out of the next computation; the test fails when cancelled requests are let into the source query | `TestCancelledOvertimeLeavesThePayroll` |

### Three questions for the HR owner

All three are entries the system accepts and then quietly treats differently from what the person
entering them probably meant. None breaks a written rule; each lacks one.

**CC-04, work entered on a day off.** E's timesheet has 19 weekdays, plus a line on Sunday 05/04 and
one on the 30/04 holiday. The timesheet saves without a word. Paid days come out at 21 of 22: the
19 weekdays plus the two holidays, which are paid whether worked or not. The Sunday line counts for
nothing. Work on a day off is paid only through an overtime request, and nothing tells the person
filling in the timesheet that.
Options: (a) keep accepting it and warn on the grid that such lines are not paid and need an
overtime request; (b) refuse lines on days off; (c) keep as is. Recommended: (a). Imported files
often carry a whole calendar, so refusing would fail imports for harmless cells.

**CC-03, work and paid leave on the same day.** E worked every working day of April and also has a
1-day paid annual leave on 06/04, posted. The leave took a day off the balance (12 → 11), but the
day is capped at 1, so paid days stay 22 of 22. The employee loses a leave day and gains nothing,
and nothing warns about the clash. This is the same kind of conflict ADR-0030 settled between two
leave requests, but between a leave request and the timesheet.
Options: (a) warn on the timesheet grid and on the payroll line when a day has both work and
approved leave; (b) refuse the timesheet line, or the leave, when they clash; (c) keep as is.
Recommended: (a) for now. The timesheet and the leave come from different people at different
times, so refusing either one blocks someone who cannot fix the other.

**TC-01, an overtime day kind that disagrees with the calendar.** Two hours on the 30/04 holiday
are accepted both as `weekday` (paid 375,000 at 150%) and as `holiday` (paid 750,000 at 300%). One
wrong pick halves the pay. Today the documents say the kind is entered by hand and deriving it from
the calendar comes later. But the calendar already knows the answer for every date: weekly days off
by version, and the holidays.
Options: (a) refuse a kind that disagrees with the calendar, naming the right one; (b) derive the
kind and drop the field from entry; (c) keep as is. Recommended: (a). It is one check where the
request is saved, it keeps the field the approval rules filter on, and it is the first step toward
(b).

## Next

Answers to the three questions above, then section 7 (payroll and period closing), using the
contract timeline above.
