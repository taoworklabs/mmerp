# HRM review, round two: observations

Created: 2026-10-12. Covers what round one left: its two unguarded rows (NP-01, NP-07), TB-02, and
section 5 of [the plan](hrm-review-plan.md) (profiles, contracts, addenda). Sections 6 and 7 are not
started yet.

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

## Next

Section 6 (timesheet, work calendar, overtime), then section 7 (payroll and period closing), using
the timeline above.
