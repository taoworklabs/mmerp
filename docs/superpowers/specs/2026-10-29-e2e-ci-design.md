# E2E reorganisation and faster CI

## Intent

Two annoyances, one change.

- CI on GitHub Actions takes about 10 minutes per push. Every step runs in one job, one after another, and the `image` job waits for that job. The owner wants a ceiling of 5 minutes.
- `web/e2e` still names eleven files after Phase 0 milestones (`m2.spec.ts` … `m14.spec.ts`). The project no longer works by milestone, so the names say nothing about what a file tests. Several files mix unrelated topics.

Success: a push goes green in 5 minutes or less with the same checks as today, and every e2e file is named after the feature it tests.

## Measured today (runs of 2026-10-28)

| Step (job `check`) | Time |
| --- | --- |
| Setup (containers, Go, pnpm, Node) | ~45 s |
| `make gen` + diff check | 1:00–1:20 |
| `make lint` | 1:00–1:10 |
| `make test` | 2:10–2:30 |
| Playwright Chromium install | ~20 s |
| `make e2e` (includes `pnpm -C web build` and `go build -tags embedweb` in `scripts/e2e-server.sh`) | 2:30–3:00 |
| Job `image` (`make image` + `make smoke`), after `check` | ~1:30 |

No single step is the problem; the sum is.

## Goals

1. CI wall time ≤ 5 minutes, push to all green, with caches warm.
2. The same checks as today: `make gen` with no diff, `make lint`, `make test`, `make e2e`, `make image`, `make smoke`. Nothing is dropped or weakened to gain time.
3. No e2e file is named after a milestone; each file name says the feature it covers. Files that mix topics are split.
4. The e2e suite passes as reliably as today after the move, including the hidden ordering dependency described below.

## Non-goals (this change)

- Fixing dates that rot in 2027, timezone inconsistencies, tests that assert nothing, shared tenant-wide rules raced by parallel files, brittle selectors, fixed sleeps. Known, listed at the end, left for a later change.
- Writing new e2e tests for uncovered features (period locks, login policy, payroll adjustments, leave types, legal parameters).
- A separate Go e2e suite: `internal/app/*_test.go` already drives the whole app over HTTP on a real Postgres.
- Changing runners, adding paid services, or changing when CI triggers (stays `push` + `workflow_dispatch`, cancel-in-progress per ref). CI on `main` after a squash merge stays: the squash commit is new.

## Anti-goals

- Changing what a test checks while moving it. Tests move between files verbatim apart from imports and the shared helpers they now use.
- Editing the GitHub ruleset. It requires the status checks `check` and `image`; those two names must still exist and still mean "everything passed" and "image builds and starts".

## Constraints

- Stack rules from `AGENTS.md`: one Go binary + Postgres, no new services. No new npm or Go dependencies.
- Make targets stay the single entry point; CI calls `make <target>` as today. Local `make e2e` keeps working unchanged.
- `make gen` keeps failing CI on any diff.
- Comments never reference milestones, ADRs or docs.

## CI design (decided)

- Split job `check` into parallel jobs: `gen` (gen + diff check), `lint`, `test` (Postgres service), `e2e` (Postgres service). Lint does not wait for gen: generated code is committed, and a stale commit fails the `gen` job anyway.
- Keep a job named `check` with `needs: [gen, lint, test, e2e]` and a trivial step, so the ruleset's required check keeps working and means all of them passed. (Builder's call: if the aggregate job needs `if: always()` plus a result check to avoid a skipped-counts-as-success trap, do that.)
- `image` drops `needs: check` and runs in parallel from the start.
- Add caches that are missing: Playwright browsers (`~/.cache/ms-playwright`, keyed on the installed `@playwright/test` version) and golangci-lint's analysis cache. `actions/setup-go` already caches modules and the build cache; make sure each job gets it.
- Upload Playwright traces on failure from the `e2e` job, as today.
- Fallback, only if the measured `e2e` job alone exceeds ~4:30: shard it. Not built unless measurement says so.

## E2E reorganisation (decided)

Proposed names (the builder may adjust a name after reading the tests, keeping it feature-based):

| Today | Becomes |
| --- | --- |
| `m2` | `employee-access` (org scope, sensitive fields, show all, activity, manager label), `employee-list` (URL state, cards below 1024px), `employee-profile` (create/edit/dependent, tabs, leave guard), `admin-org-users` (admin builds the tree and grants a role) |
| `m3` | `leave-requests` |
| `m4` | `contracts` (contract kinds + contracts), `overtime` |
| `m5` | `timesheets` |
| `m6` | `payroll` |
| `m8` | `hrm-org-chart`, `approval-rules-menu` |
| `m9` | `hrm-overview` (tile counts; the only part that must run alone), `navigation` (home, menus, drawer, deep link, tile visibility), `product-disabled` (the :8091 tests) |
| `m11` | `attachments`, `discussion` |
| `m12` | `notifications`, `admin-mail-server` |
| `m13` | `printing` |
| `m14` | `sales-quote-to-order` |

- Files stay flat in `web/e2e/` (no `core/`, `hrm/`, `sales/` folders). My call: the suite is ~30 files and Playwright's config matches by name; folders add path churn for little gain.
- Test data carrying milestone ids (logins like `m4m_`, names like "Công ty M4…") is renamed to match the new file, since it shows in failure output. Uniqueness through `runId` stays.
- Helpers copied across files move into `helpers.ts` / `api.ts` when the move touches them: sign in as a given user (copied ~20 times), `status()`, `ok()`, `fixture()`, the tenant-today/date helpers, the contract + posted March timesheet builder shared by payroll and personal-space, and the approval-rule snapshot/restore. Behaviour of each helper stays as the copies have it; where copies differ, keep the variants the tests need.
- `playwright.config.ts`: the `exclusive` list is updated to the new names; its comment explains why each file runs alone without naming milestones.

### The ordering dependency

`personal-space.spec.ts` only passes because `m6.spec.ts` runs before it (alphabetical order inside the `exclusive` project) and deletes the tenant-wide `hrm.contract`, `hrm.timesheet` and `hrm.overtime_request` rules; `personal-space` itself deletes only `hrm.payroll`. `payroll` still sorts before `personal-space`, so the rename alone would not break it, but any other name or a new exclusive file could. Required: `personal-space` deletes the rules it needs gone itself. Order inside the whole `exclusive` set must not matter afterwards.

Splitting a file whose tests share one file-level `beforeAll` (`m2`, `m4`, `m8`, `m9`, `m11`, `m12`) gives each new file the part of that setup its tests use. That setup code is the one thing that grows; test bodies stay as they are.

## Docs to update

Current docs naming the old files: `docs/hrm-review-plan.md`, `docs/hrm-review-round1.md`, `docs/hrm-review-round2.md` (evidence columns pointing at `m3`, `m5`, `m12`). Point them at the new names. Plans and specs under `docs/superpowers/` are history; leave them.

## Left to the builder

- Exact cache keys and actions used for caching.
- Whether `--with-deps` is still needed on a Playwright cache hit.
- Exact file names within the feature-based scheme above, and which helper variants to keep.
- Order of the work and how it is split into commits.

## Verification

- `make e2e` passes locally after the move; tests are counted before and after and the numbers match (no test lost or duplicated).
- Each file of the `exclusive` set passes when run alone after setup, so its order no longer matters.
- Three CI runs on the branch, each ≤ 5:00 wall time with warm caches; report the timings.

## Known issues left for later

- Dates that break in 2027: `m5`/`m6` pick "Tháng 3" but expect `BC-2026`/`BL-2026`; `keyboard-inputs` expects `07/2026`; `guards` hardcodes 2026 quote dates; `/NP-2026-/` patterns.
- Mixed UTC / runner-local / tenant time zones in date helpers.
- Parallel files overwriting the tenant-wide `hrm.leave_request` rule (`m3`, `m8`, `m12`, `notification-links`); `code-series` changes the customer prefix without restoring it.
- Assertion-less or either-way tests: `notification-links:195-245`, `termination:55-61`, `m9:180-185`.
- Fixed sleeps and brittle selectors listed in the e2e review.
