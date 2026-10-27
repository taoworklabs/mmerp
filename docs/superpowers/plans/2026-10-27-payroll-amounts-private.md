# Payroll amounts stay private Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A payroll's posting lines never single out a small group's pay, and every payroll export says which payroll and how many rows left the system.

**Architecture:** One branch `fix/payroll-amounts-private`, one commit per ticket, one PR closing #52, #53, #54. The roll-up lives in HRM's pure `postingLines` function, fed with the legal entity's org-unit parents read by a new HRM query. The export audit gains a row count in `dataio`, so it covers every export.

**Tech Stack:** Go, sqlc, Postgres.

**Spec:** GitHub issue #52 and tickets #53, #54 (`gh issue view <n>`). Written 2026-10-16; checked against `main` on 2026-10-27 (see below).

## Changes to the spec (planner's calls)

- **ADR number:** the issue names ADR-0034, which is now taken. The ADR is `docs/adr/0040-payroll-posting-minimum-group.md`.
- **"Legal-entity level with no org unit":** `posting.lines.org_unit_id` is `NOT NULL`, and the code already posts legal-entity-level lines (`insurance_payable`, `pit_payable`) with `OrgUnitID = legal entity`. The roll-up ends at the legal-entity node the same way. No migration.
- **#54, payslip prints:** already covered. Every print writes `printing.printed` on the payroll (`audit.Ref` = payroll), with `parts` = the employee ids printed and no amounts (`internal/core/printing/service.go:157`, `printedData`). No `hrm.payslip_printed` is added; Task 2 adds a test pinning that event for payslips.
- **#54, payroll exports:** instead of a second event `hrm.payroll_exported`, `dataio.exported` gains `"rows": <data row count>` for every export. It already carries `kind` and `params` (the payroll id) and is written in the same transaction as the kept file, so it never claims an export whose file was not saved. A separate HRM event would have to be written outside that transaction.

## Global Constraints

- Money is `int64` in đồng; sums only, no rounding needed here.
- Posting lines are written once via `posting.Record`; existing lines are never rewritten (no backfill, no migration).
- Audit data never contains amounts.
- Comments short, why not what, never naming ADRs, milestones or tickets. Docs in English.
- Commits: Conventional Commits with scope, ending with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`. Stage explicit paths only.

## Review Focus

- A unit with exactly 3 payroll lines keeps its own posting lines (threshold is inclusive). Test in Task 1.
- Two small sibling units whose sum reaches 3 together in the parent: the parent line carries both, and neither child has a line. Test in Task 1.
- Amounts are conserved: for each kind, the sum over all posting lines equals the sum over all payroll lines, before and after roll-up. Test in Task 1.
- Employees whose org unit is the legal entity itself count at the legal-entity node. Test in Task 1.
- An export that fails before its file is kept writes no `dataio.exported`. Already true by construction; Task 2 checks the row count only.

---

### Task 1: Posting lines need at least 3 payroll lines per unit (#53)

**Files:**
- Create: `docs/adr/0040-payroll-posting-minimum-group.md`
- Modify: `internal/modules/hrm/payroll.go:304-342` (`postingLines` and its caller)
- Modify: `internal/modules/hrm/queries/payroll.sql` (new query; `make gen`)
- Modify: `products/hrm.md` (posting section near line 234, and the table row at line 295)
- Modify: `CONTEXT.md:62` (Business line entry, one sentence)
- Test: `internal/modules/hrm/payroll_calc_test.go`

**Interfaces:**
- `const postingMinLines = 3`
- `func postingLines(le int64, parent map[int64]int64, lines []PayrollLine) []posting.Line` — `parent[unit]` is the unit's parent within the legal entity; the legal entity itself has no entry.
- New sqlc query `OrgUnitParents :many` in `payroll.sql`: `id, parent_id` of every org unit in the legal entity's subtree (recursive CTE from `@legal_entity_id` over `iam.org_units.parent_id`, excluding the root row). The caller builds the map.

**Roll-up rule (the algorithm the tests pin):** count payroll lines per unit (the employee's `OrgUnitID`). Visit units deepest first (depth = steps to `le` via `parent`; ties by unit id, so output order is deterministic). A unit other than `le` with fewer than `postingMinLines` lines moves its count and all its per-kind sums into its parent. Units that keep ≥ 3, and `le` whatever its count, emit their non-zero sums. `insurance_payable` and `pit_payable` stay on `le` as today. A unit missing from `parent` (data drift) rolls straight to `le`.

- [ ] **Step 1: ADR.** Write ADR-0040 in the style of `docs/adr/0039-shared-master-data.md` (Status accepted · 2026-10-27; Context: a unit with one or two people exposes their pay in unencrypted `posting.lines`; Decision: the rule above, from the next posting on, existing lines unchanged, legal-entity node as the end of the roll-up; Rejected: encrypting posting lines (accounting must read them), a per-tenant threshold (no one asked), rewriting past lines (posted data is never rewritten); Consequences: a department's expense may appear under its parent; a legal entity with fewer than 3 people still shows its total, which its legal declarations already disclose).
- [ ] **Step 2: Failing tests** in `payroll_calc_test.go`, table-driven over a tree `le=1 → A=2 → {A1=3, A2=4}`, `B=5`:
  - `TestPostingLinesKeepUnitAtThreshold`: 3 lines in A1 → `salary_expense` line on unit 3.
  - `TestPostingLinesRollSmallUnitIntoParent`: 2 lines in A1, 3 in A → one line on A carrying all 5; none on A1.
  - `TestPostingLinesRollSiblingsTogether`: 1 in A1, 2 in A2, 0 in A → one line on A with all 3.
  - `TestPostingLinesRollUpToLegalEntity`: 1 in B, 1 in A1 → lines on unit 1 only.
  - `TestPostingLinesConserveAmounts`: random-ish fixed data across all units; per kind, sum of posting amounts == sum of payroll amounts.
  - `TestPostingLinesEmployeesOnLegalEntity`: 2 lines with `OrgUnitID = 1` → lines on unit 1.
  Update the existing call at `payroll_calc_test.go:88` to the new signature with a `parent` map that keeps today's behaviour for that test's data, or adjust its expectation if its units are below 3, noting why.
- [ ] **Step 3: Run** `go test ./internal/modules/hrm -run TestPostingLines` → FAIL (signature).
- [ ] **Step 4: Implement** the query (`make gen`), the new `postingLines`, and in the posting transition read `OrgUnitParents` for `le` and pass the map.
- [ ] **Step 5: Run** the tests → PASS; `go test ./internal/modules/hrm` → PASS (the payroll samples test must still pass; if a sample asserts posting lines per department, update the expectation and say so in the report).
- [ ] **Step 6: Docs.** `products/hrm.md`: one bullet under posting (the rule, from ADR-0040) and the table row at line 295 ("aggregated by org unit of at least 3 people, rolled up otherwise, or by legal entity"). `CONTEXT.md` Business line entry: add "Payroll lines roll small units up the org tree so no line covers fewer than 3 people."
- [ ] **Step 7: Commit** `feat(hrm): payroll posting lines never cover fewer than 3 people`.

### Task 2: Exports count their rows; payslip prints pinned (#54)

**Files:**
- Modify: `internal/core/dataio/jobs.go:236`
- Test: `internal/core/dataio/` export test file (grep `dataio.exported` in `*_test.go`), `internal/modules/hrm/` print test (grep `hrm.payslip` in `*_test.go`)

- [ ] **Step 1: Failing tests.**
  - `TestExportAuditCountsRows` (dataio): an export of a sheet with N data rows writes `dataio.exported` with `"rows": N` (header not counted) and still `kind`, `params`, `file_id`.
  - `TestPayslipPrintAudited` (hrm): printing two employees' payslips of a payroll writes one `printing.printed` row with `doc_type = hrm.payroll`, `doc_id` = the payroll id, `data.parts` = those two employee ids, and no key holding an amount (assert `data` keys are exactly `template`, `layout`, `parts`).
- [ ] **Step 2: Run** both → `TestExportAuditCountsRows` FAILS (no `rows`); `TestPayslipPrintAudited` may already PASS — that is expected, it pins existing behaviour; say so in the report.
- [ ] **Step 3: Implement** `"rows": len(sheet.Rows)` in the `dataio.exported` data (use the variable the job already holds).
- [ ] **Step 4: Run** → PASS; `go test ./internal/core/dataio ./internal/modules/hrm` → PASS.
- [ ] **Step 5: Commit** `feat(dataio): export audit records the row count`.

### Task 3: Whole branch

- [ ] **Step 1:** `make gen` → no diff; `make lint` → PASS; `make test` → PASS.
- [ ] **Step 2:** Push and open the PR: title `fix(hrm): payroll amounts stay private`, plain-text body per ticket with commits, the planner's calls above (ADR number, legal-entity node, prints already audited, row count on `dataio.exported` instead of a new event), no migrations, then `Closes #52`, `Closes #53`, `Closes #54`. No "Generated with" line. Do not merge.
