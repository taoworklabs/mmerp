# HRM review, round one: observations

Created: 2026-10-11. Status: observations, not settled requirements.

Follows the [HRM business review and core verification plan](hrm-review-plan.md). This file holds
only what was actually run, per section 10.1 of the plan. A difference from what someone expected is
not called a defect here unless the plan or the product documents already state the expected result.

## Run environment

| Item | Value |
| --- | --- |
| Application version | `25f1098` (branch `docs/hrm-review-plan`) |
| Run date | 2026-10-11 |
| Postgres | 17.11 on `localhost:5433`, via `TEST_DATABASE_URL` in a gitignored `local.mk` |
| Backend probes | `internal/modules/hrm/leave_probe_test.go`, `payroll_inputs_probe_test.go` |
| Frontend probes | `web/e2e/review-probe.spec.ts`, Playwright on the built app (`:8090`, `PRODUCTS=hrm,sales`) |
| Business date in the data | 2026; leave balances granted for year 2026 |

The probes observe and log; they assert almost nothing, because what they measure is exactly what the
review has to settle. They carry `probe` in their name so they stay recognisable as observations
rather than as expectations the suite enforces.

Two deviations from the documented commands, both forced by the sandbox and neither affecting what was
measured: Docker cannot run here (the daemon cannot create its bridge network), so `make db` and
`make e2e` were replaced by the already-running Postgres 17 and by the steps of `scripts/e2e-server.sh`
run by hand with the local `psql` instead of the `postgres:18` image. The repo targets Postgres 18;
every migration and probe applied cleanly on 17, but **the figures below were not produced on the
version the project ships against**.

## Summary

| Code | Observation | Matches an existing rule? |
| --- | --- | --- |
| NP-04 | The balance is cut by the days entered; payroll pays only the working days inside the range, and silently drops the rest | No rule covers the gap between the two |
| NP-06 | Two requests covering the same days are both approved and both deducted | Confirmed: no rule exists |
| NP-10 | A type that neither deducts nor is paid leaves the balance alone, and cancelling refunds nothing | Yes, as the plan expects |
| NP-12 | No manager, and a manager without an account, both fall back to the `hrm.hr` role | Yes, as the plan expects |
| NP-15 | HR filing for an employee without an account keeps the two people apart and routes to the employee's manager | Yes, as the plan expects |
| TB-01 | Notifications and background jobs stay inside the area; from home they stay at the root | Yes, `238c82f` holds |
| TB-04 | The server refuses with `409 approval_closed`; the document page explains it, the inbox shows nothing | Half: refused, explained only outside the inbox |
| TB-05 | A notification whose permission was revoked degrades to "bạn không còn quyền xem" and leaks nothing | Yes |
| TB-07 | Going back restores the URL, the filter control and the rows | Yes |

Two of these need a decision before anything is implemented: NP-04 and NP-06. One is a defect against
an expected result the plan already states: TB-04.

## NP-06 — two requests covering the same days

```text
Scenario code: NP-06
Business goal: find out what happens when one employee has two leave requests over the same days
Application version and run date: 25f1098, 2026-10-11
Role, permissions and org unit scope: employee E submits; manager M approves; HR grants the balance
Starting data, configuration and business date: E reports to M; leave type "Phép năm"
  (deducts_balance = true); approval rule = one module step, fallback hrm.hr, max 3 levels;
  balance for 2026 granted as 12 days
Steps:
  1. Two identical requests, 2026-03-10 to 2026-03-11, 2 days each. Submit both, approve both.
  2. Two partly overlapping requests, 2026-05-04 to 06 and 2026-05-06 to 08, 3 days each
     (2026-05-06 is in both). Submit both, approve both.
Expected result and its basis: none. Neither products/hrm.md nor the code states a rule;
  the plan records this at section 4.3 as a question with no answer yet.
Actual result:
  1. Both approved with no error: NP-2026-00001 and NP-2026-00002, both posted.
  2. Both approved with no error, both posted.
Data before/after:
  balance 12 → 8 after step 1 (2 + 2 deducted)
  balance 8 → 2 after step 2 (3 + 3 deducted)
  10 days deducted in total, for 7 distinct calendar days of absence
Evidence: TestProbeNP06Overlap, leave_probe_test.go:52-65
Conclusion: pass, in the sense that it behaves as the code is written; there is no rule to fail against
Points to confirm, and who owns them: HR owner, per the plan's decision table. Whether an overlap is
  refused, warned about, or allowed; and if a half day needs morning/afternoon to detect an overlap
  at all, since the overlap would otherwise be undetectable for half-day requests
```

## NP-04 — leave over a weekend or a public holiday

Two probes, because the two halves live in different places: the balance is cut in
`service_leave.go:111`, the paid days are computed in `leaveByDay`, `service_payroll_inputs.go:379`.

```text
Scenario code: NP-04
Business goal: find out how a request spanning non-working days reaches the balance and the payroll
Application version and run date: 25f1098, 2026-10-11
Role, permissions and org unit scope: as NP-06
Starting data, configuration and business date: balance 12 days for 2026; for the payroll half, a
  calendar with Saturday and Sunday off and 2026-04-30 a public holiday
Steps and actual result, balance half (TestProbeNP04Balance):
  2 days entered on Sat 2026-03-14 + Sun 03-15, approved → balance 12 → 10
  4 days entered over Fri 2026-03-13 → Mon 03-16, approved → balance 10 → 6
Steps and actual result, paid-days half (TestProbeNP04, unit level):
  entered 4 over Fri 04-03 → Mon 04-06  → paid 2    (04-03 = 1, 04-06 = 1)
  entered 2 over Fri 04-03 → Mon 04-06  → paid 2    (04-03 = 1, 04-06 = 1)
  entered 6 over 04-29 → 05-04          → paid 3    (04-29, 05-01, 05-04)
  entered 1.5 over Fri 04-03 → Mon 04-06 → paid 1.5 (04-03 = 1, 04-06 = 0.5)
  entered 2 on Sat 04-04 + Sun 04-05    → paid 0    (no day at all)
  entered 1 on the holiday 04-30        → paid 0    (no day at all)
Expected result and its basis: products/hrm.md#leave says the day count is entered manually in half
  days and is only checked against the calendar days of the range. So the entry side is as documented.
  What the documents do not state is what the two numbers mean together.
Data before/after: see above; the balance never consults the calendar, the payroll never consults the
  entered total beyond using it as a ceiling
Evidence: TestProbeNP04Balance (leave_probe_test.go:79-86); TestProbeNP04 (payroll_inputs_probe_test.go:55)
Conclusion: fail to conclude without a business rule. The behaviour is self-consistent per file and
  undefined as a pair.
Points to confirm, and who owns them: HR owner. The observation to put in front of them is that
  `leaveByDay` spreads the entered days over the working days of the range at most one per day and
  drops the remainder with no warning, while the balance is cut by the full entered figure. An
  employee who enters 4 days across a Friday-to-Monday range therefore loses 4 days of balance and is
  paid for 2. Entering leave that falls only on rest days costs balance and buys nothing.
```

This is the same decision the plan's table already lists as "computing leave days from the calendar".
The observation sharpens it: the question is not only whether to automate the count, but what the
entered figure is supposed to mean when it disagrees with the calendar.

## NP-10 — a type that neither deducts nor is paid

```text
Scenario code: NP-10
Business goal: check that a non-deducting, unpaid type leaves the balance untouched both ways
Application version and run date: 25f1098, 2026-10-11
Role, permissions and org unit scope: as NP-06; HR cancels
Starting data, configuration and business date: balance 12 for 2026; a second leave type
  "Nghỉ không lương" with deducts_balance = false, paid = false
Steps: request 2 days 2026-03-10 to 03-11 of the unpaid type; submit; approve; then cancel it
Expected result and its basis: plan section 4.2 NP-10 — the balance is untouched and the paid
  workdays reflect the leave type
Actual result: approved, status posted, balance stayed 12. Cancelling succeeded and the balance
  stayed 12: no days were handed back that were never taken. At unit level, 2 unpaid days on two
  working days produced 0 paid days.
Data before/after: 12 → 12 → 12
Evidence: TestProbeNP10UnpaidBalance (leave_probe_test.go:100-106); TestProbeNP10UnpaidType
Conclusion: pass
Points to confirm, and who owns them: none
```

The refund path is safe because it refunds the stored `deducted` figure rather than recomputing from
the day count (`service_leave.go:116-119`), so a type that deducted nothing refunds nothing.

## NP-12 — no manager, or a manager without an account

```text
Scenario code: NP-12
Business goal: check who is asked when the direct-manager relation cannot yield an approver
Application version and run date: 25f1098, 2026-10-11
Role, permissions and org unit scope: employees X and Y each with an account; HR holds hrm.hr
Starting data, configuration and business date: approval rule = one module step, fallback hrm.hr,
  max 3 levels; balance 12 for both
Steps:
  1. X has an account and no manager at all. X submits 1 day, 2026-03-10.
  2. Y reports to an employee record NOACC that has no sign-in account. Y submits 1 day, 2026-03-12.
Expected result and its basis: plan section 4.2 NP-12 — falls back to the fallback role per the
  approval rule
Actual result: both submissions succeeded and opened an instance whose step 1 went to the fallback:
  status open, step 1, approvers = [hr], fallback = true, in both cases
Data before/after: no balance movement; nothing is deducted until approval
Evidence: TestProbeNP12NoManager (leave_probe_test.go:122-133)
Conclusion: pass
Points to confirm, and who owns them: HR owner, for the plan's open question of who covers for an
  absent manager and for how long. The fallback answers "nobody at all exists", not "the manager is
  away this week"; the two are different needs and only the first is built.
```

## NP-15 — HR files a request for an employee without an account

```text
Scenario code: NP-15
Business goal: check that the employee on leave is told apart from the user performing the operation
Application version and run date: 25f1098, 2026-10-11
Role, permissions and org unit scope: HR submits on behalf of Z; manager M2 approves
Starting data, configuration and business date: Z has no account and reports to M2, who has one;
  Z's balance 12 for 2026; approval rule as NP-12
Steps: HR creates a 2-day request for Z (2026-03-10 to 03-11), submits it, M2 approves
Expected result and its basis: plan section 4.2 NP-15
Actual result: the document was created against employee Z, numbered NP-2026-00001, status draft.
  HR's submission opened an instance whose step 1 went to M2 — Z's manager, not HR's — with
  fallback = false, so the routing followed the employee on the request and not the submitting user.
  M2 approved; status posted.
Data before/after: Z's balance 12 → 10
Evidence: TestProbeNP15HRForEmployeeWithoutAccount (leave_probe_test.go:153-159)
Conclusion: pass
Points to confirm, and who owns them: none for the routing. Not covered by this probe: whether HR
  should be able to approve what HR filed when HR is also the fallback approver; NP-13 covers
  self-approval for the employee, not for a filing third party.
```

## TB-01, TB-05, TB-07 — notifications and working context

```text
Scenario codes: TB-01, TB-05, TB-07
Application version and run date: 25f1098, 2026-10-11
Role, permissions and org unit scope: a staff account, a manager account, and a peer holding
  hrm.hr on the department until it is revoked mid-scenario
Evidence: web/e2e/review-probe.spec.ts, all passing

TB-01 — the notification and background-job links from inside HRM went to /hrm/notifications and
  /hrm/jobs, kept the "Quản lý nhân sự" header title and kept exactly one area menu. The same two
  links from the home page went to /notifications and /jobs with no area menu. The regression check
  on 238c82f holds. Asserted, not merely logged.

TB-05 — the peer was mentioned in a discussion and, while holding hrm.hr, saw the notification in
  full: "Nhắc tên bạn trong trao đổi · Đơn nghỉ phép NP-2026-00001 · tbstaff_…". After every role
  grant was deleted, the same row read "Đơn nghỉ phép · bạn không còn quyền xem": the document
  number and the submitter's login were both gone, so no content leaked. Clicking it kept the user
  on /notifications rather than opening a forbidden document.

TB-07 — opening a document from /hrm/leaves?status=draft and going back returned to that exact URL,
  with the status filter still showing "Nháp" and the same 2 rows as before. Asserted, not logged.
```

## TB-04 — approving a request the submitter already withdrew

This is the one finding that fails against an expected result the plan already states.

```text
Scenario code: TB-04
Business goal: an approver acts on an inbox item the submitter has withdrawn in the meantime
Application version and run date: 25f1098, 2026-10-11
Role, permissions and org unit scope: staff submits; their manager approves from /hrm/inbox
Starting data, configuration and business date: a 1-day request of a non-deducting, paid type;
  approval rule = one module step, fallback hrm.hr
Steps:
  1. The staff member submits. The manager opens /hrm/inbox and selects the item, so the approve
     button is on screen.
  2. The staff member withdraws the request (transition to draft) behind the approver's back.
  3. The manager presses "Duyệt".
Expected result and its basis: plan section 8, TB-04 — "Approving again is refused; the current
  state is explained"
Actual result, server side (deterministic, through the API): POST /api/approvals/{id}/approve
  returns 409 with code `approval_closed`. The inbox no longer lists the request and the document
  stays draft. The refusal itself is correct.
Actual result, user side in the inbox (9 runs): the click always reaches the server and always gets
  the 409 — captured from the browser as "POST /api/approvals/13/approve → 409" and the same in
  every other run. What the approver sees is a race. In 6 of 9 runs no message appeared at all: the
  detail pane simply emptied and the page read "Không có việc nào đang chờ duyệt". In 3 of 9 a
  message was briefly visible, and in each of those it had already disappeared before its text could
  be read — one run logged "(gone before it could be read: locator.innerText: Timeout 2000ms
  exceeded)" and "still visible a second later: false", and an earlier run of the whole file failed
  outright on the same read. So the approver is either told nothing, or told something they cannot
  finish reading.
Actual result, user side on the document page (2 runs, consistent): the same withdrawal, the same
  409, but the approver is told plainly. The panel shows "Người gửi đã rút lại lần duyệt này." and
  "Bước duyệt này đã được xử lý hoặc lần duyệt đã đóng", and the document reads Nháp. So the
  message exists and is correctly worded; only the inbox fails to show it.
Data before/after: document draft before and after; no balance movement; no decision recorded
Evidence: review-probe.spec.ts — "TB-04 approving a request the submitter already withdrew" (inbox,
  with the network call captured), "TB-04 the API refusal when approving a withdrawn request" (API),
  "TB-04 the same refusal seen from the document page" (the contrast)
Conclusion: fail in the inbox, on the "explained" half of the expected result. The refusal passes,
  and the document page passes in full.
Cause, read from the code and then confirmed by the contrast above:
  ApprovalPanel.tsx:58-60 stores the message in the panel's own useState and, for `approval_closed`
  and `approval_stale`, also invalidates the record. In the inbox that refetch removes the item from
  inbox.items, so InboxPage.tsx:22 can no longer find it, line 68 renders detail as null, and the
  panel unmounts — discarding the very message it just set. On the document page nothing unmounts
  the panel, which is why the identical error path is readable there. The defect is therefore in how
  the inbox disposes of the item, not in the error handling, and it bites exactly where the plan's
  section 8 says the user does their approving.
Points to confirm, and who owns them: none for whether this is wrong — the plan states the expected
  result. What needs deciding is only where the message belongs once the item is gone: a toast
  outside the unmounting subtree, or keeping the item on screen in a closed state. Product owner.
```

The 3-in-9 runs where something flashed up are not evidence against the finding; they are the
clearest evidence for it. The message is set before the refetch unmounts the panel, so it can paint
for a frame and then go. The probe caught this directly: it saw the element, then timed out trying to
read it, then found it gone. A message whose delivery depends on winning a repaint race is not a
message the approver can be said to have been given, so the row is recorded as a fail.

## What this round did not cover

- Nothing in sections 5, 6 or 7 of the plan: profiles, contracts, timesheets, payroll. Round one is
  the leave flow plus section 8, per the plan's scope.
- TB-02, TB-03, TB-06 were not re-run here; the plan records `web/e2e/m12.spec.ts` as already
  touching TB-03 and TB-06, and that mapping is still to be checked against the test bodies.
- The rows of plan section 4.2 marked as automated were not read line by line against the scenario
  text yet. The plan warns the mapping was made from test names alone; that check is the next step
  and may move rows from "covered" to "needs a scenario".
- Every figure here comes from Postgres 17, not the Postgres 18 the project targets.

## Suggested next steps

1. Put NP-04 and NP-06 in front of the HR owner as the plan's decision table entries. They are the
   two places where the system currently has no rule, and NP-04 costs an employee balance they are
   not paid for, so it is the more urgent of the two.
2. Fix TB-04's missing explanation in the inbox. It needs no business decision and no new wording —
   the text already exists and already reads correctly on the document page — only a place to put it
   that survives the item leaving the inbox.
3. Read the automated rows of section 4.2 against their test bodies, then decide the order of HRM
   and core fixes before opening the timesheet and payroll rounds.
