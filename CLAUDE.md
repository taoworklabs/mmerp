# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project status

mmerp is a modular business-management suite for Vietnamese SMBs. Implementation follows `roadmap.md` (Phase 1). Since 2026-10-07 the core is built out first (M11 attachments and discussion, M12 notifications, M13 PDF printing, all done), each capability proven by an HRM screen; since 2026-10-09 Sales is the second product (M14: customers, items, quotations, sales orders; ADR-0027); M10 (on-premise operations) and the production-readiness gate are on hold (ADR-0023). Phase 0 ended at M9 (home page of product tiles, a header title per area, grouped HRM menu, HRM overview counts; a disabled product with data stays readable), after M8 (every signed-in user reads the org tree; tenant-wide-only roles; approval rules configured per product, inside each product's area) and M6 (payroll computed by a job from captured inputs, checked against the 28 payroll samples; work calendar, legal parameters, legal-entity settings, money rounding, `shared/posting`). M7 is obsolete. HRM is the first product and Sales the second; accounting, custom fields and cloud model B are out of scope until the roadmap says otherwise.

The design docs are written in English; keep them in English when you edit them. `docs/archive/` holds documents of finished phases (the Phase 0 roadmap, decided proposals): do not read them unless the user asks; current docs and ADRs already carry everything still in force.

## Read before working

| Before you… | Read |
| --- | --- |
| Need the big picture | `architecture.md` (diagrams; detailed docs win on conflict) |
| Touch any backend module | `backend.md`, `documents.md` |
| Build or change any screen | `ui.md` (finish with its checklist), `frontend.md` |
| Add a library or tool | `techstack.md` ("Adding a new library" section) |
| Work on HRM | `products/hrm.md` |
| Work on Sales | `products/sales.md` |
| Name anything | `CONTEXT.md` (domain glossary; use its terms exactly) |
| Make a hard-to-reverse decision | Existing ADRs in `docs/adr/` — do not re-litigate them |

`roadmap.md` defines what each milestone builds and its exit criteria. Build only what the current milestone needs.

## Hard constraints

- On-premise is the primary deployment: the stack is **one Go binary + Postgres**, nothing else (no Redis, broker, or sidecar container). Cloud runs the same binary.
- No business operation may depend on the Internet or on external services. External calls go through River jobs enqueued in the same transaction.
- Business code is tenant-unaware: no tenant columns, no tenant parameters, no package-level DB handles.
- Money is `int64`/`bigint` in the currency's minor unit; every rounding goes through the single `platform` function. Rates and fractional quantities use `shopspring/decimal`. Never `float`.

## Architecture

### Backend layers (`internal/`)

`modules → shared → core → platform`, dependencies only point down. `internal/app` is the composition root.

- `platform`: infrastructure, no tables (db/tx, config, errors, i18n lookup, money rounding, jobs, file storage, `ProductGate`).
- `core`: always-on, business-agnostic modules (`iam`, `setting`, `numbering`, `audit`, `record`, `approval`, `dataio`, …).
- `shared`: master data and cross-product modules (e.g. `posting`).
- `modules`: product modules (e.g. `hrm`). They **never import each other**; cross-product calls go through interfaces in `deps.go` (required, never no-op) or `hooks.go` (optional reactions, no-op allowed), wired in `internal/app`. No events or event bus.
- Every module has the same shape: `module.go` (manifest), `service.go`/`types.go` (public surface; a large service splits into `service_<part>.go`), `deps.go`, `hooks.go`, `handler.go`, `queries.sql`, `internal/store` (sqlc). A file that would be empty is left out (a module without routes has no `handler.go` or `module.go`). `NewService(d Deps)` registers roles and record types; `Module(svc)` returns the manifest. Each module owns one Postgres schema named after it and only it writes there. SQL reads may join its own schema, its tier and lower tiers, and the schemas of products its product declares as dependencies; nothing else.
- `platform.Module` must never reference `core` types; modules register record types and roles by calling `core` registration functions received through deps.

### Transactions and locking

- Services get a connection only via `platform.DBFrom(ctx)` and open transactions with `platform.InTx(ctx, fn)`. Never pass `pgx.Tx`, or a store bound to one, as a parameter: helpers take the tx `ctx`, so audit rows land in the same transaction.
- Document writes follow one lock order: legal-entity row in `record.period_locks` (`FOR SHARE`; `FOR UPDATE` only to move the lock date) → document row in `record.documents` (`FOR UPDATE`; in `Create`, the `numbering` counter row, then the insert) → module-owned rows (e.g. `hrm.payroll_locks`, then leave balances).
- One transaction writes exactly one document. Bulk operations are jobs that process each document in its own transaction.

### Documents (`record`, `approval`, `posting`)

- `record.documents` is the single source of document status (`draft`, `pending_approval`, `posted`, `cancelled`), `version`, date and legal entity. Modules call `record.Create/Edit/Delete/Transition` **before** writing their own tables, in the same transaction; module side effects run in the record type's `OnTransition`. Module-specific progress (delivery, payment) lives in the module's own status columns.
- `approval` plugs into `record` through a gate interface; each submission is an approval instance with its own id and submitted `version`.
- Every document type must pass the `recordtest` contract suite.
- Posting lines are data written once via `posting.Record`/`posting.Void`; they never carry per-person sensitive amounts.
- APIs return `allowed_actions` for every document; the frontend renders actions from it and never derives permissions itself.

### Products, permissions, jobs

- `PRODUCTS` lists enabled products. Every operation is classified as read, export or write, and `ProductGate` decides all of them (module routes, core routes, `allowed_actions`, job creation and execution). Disabled products stay readable and exportable.
- System jobs are listed explicitly in code and run as the system actor. All other jobs run on behalf of the requesting user and re-check permissions when they execute.
- Sensitive fields are column-encrypted, need a dedicated permission, and every read is audited. They never appear in logs.

### Frontend (`web/src/`)

`app → core, <product areas> → shared`. Each product is an **area** entered only through its `index.ts` manifest (routes, nav, i18n, `recordTypes` with `path`/`preview`/`invalidate`). Areas never import each other; each area calls only `shared/api/core` and its own `shared/api/<product>` client, typed to that product's `/<product>/` routes (core gets the rest), so calling another area's endpoint fails `tsc`. Business components stay in their area, built from `shared/ui` primitives. Page layout always comes from a `shared/ui/page` template.

## Commands

Tools: Go, Node 24+ with pnpm, Docker. The server reads `DATABASE_URL`, `PRODUCTS`, `ENCRYPTION_KEYS`, `HTTP_ADDR` (`:8080`), `FILES_DIR` (`data/files`), `RUN_JOBS` (`true`) and `LOG_LEVEL` (`info`) (`platform.LoadConfig`). Postgres from compose listens on `localhost:5433`. To use a Postgres 18 already running on the machine instead, set `TEST_DATABASE_URL` (and `DATABASE_URL` if it differs from the default) in a gitignored `local.mk`; make then never starts compose.

| Command | Does |
| --- | --- |
| `make gen` | sqlc (`sqlc.yaml`, one entry per module; pinned via `go run`, needs cgo), OpenAPI spec from Huma (`api/openapi.json`), TypeScript types (`web/src/shared/api/schema.gen.ts`); the per-product clients in `shared/api/<product>.ts` are typed views of it |
| `make lint` | golangci-lint (pinned via `go run`; depguard, forbidigo), `tsc`, ESLint, dependency-cruiser on `web/src` and `web/lint-fixtures` (must report exactly `web/lint-fixtures/expected.json`), colour-literal scan, i18n key check, `queries.sql` schema scan (`scripts/check-queries.mjs`, which must also report exactly `scripts/query-fixtures/expected.json`) |
| `make test` | `go test ./...` and Vitest; starts Postgres via compose unless `TEST_DATABASE_URL` is set |
| `make e2e` | Playwright (`web/e2e`): builds the app, recreates the `mmerp_e2e` database (psql through the `postgres:18` image), creates the admin, serves `:8090` with `PRODUCTS=hrm,sales` and `:8091` with none (`scripts/e2e-server.sh`) |
| `make db` | Postgres via compose unless `TEST_DATABASE_URL` is set; `make` exports a dev `ENCRYPTION_KEYS` |
| `make dev` | Postgres via compose, backend (`go run ./cmd/server`, `:8080`) and Vite dev server (proxies `/api`, `/healthz`) |
| `make reset-db` | Drops and recreates the dev database, then creates `admin` (password `e2e password`); a running `make dev` reconnects |
| `make seed` | Builds the Demo demo company (org chart, accounts, employees, contracts, leave, overtime, timesheets, payrolls) through the API of a running `make dev` on an empty database (`make reset-db` first; Playwright project `seed`, `web/e2e/demo.seed.ts`); accounts other than `admin` use password `Demo@2026`; `E2E_ADMIN_LOGIN`/`E2E_ADMIN_PASSWORD` name a tenant administrator |
| `make image` | Runtime image `mmerp:dev`; the frontend is embedded with `-tags embedweb` |
| `mmerp create-admin <login> <name>` | Creates the first user at install time, with the tenant-wide `core.admin` role; password on stdin (`echo pw \| ENCRYPTION_KEYS=… docker compose run --rm -T app create-admin admin "Quản trị"`: compose refuses to start without `ENCRYPTION_KEYS`) |
| `make smoke` | Starts the image with Postgres in a throwaway compose project, waits for `/healthz` and the frontend, then cleans up |

CI (GitHub Actions, `.github/workflows/ci.yml`) runs `make gen` (fails on diff), `make lint`, `make test`, `make e2e`, `make image`, `make smoke`. Backend tests use a real Postgres: each test gets its own database cloned from a migrated template (`internal/platform/pgtest`). Never mock the database. Run a single Go test with `go test ./internal/<path> -run <TestName>` after `make db` (plain `docker compose` needs `ENCRYPTION_KEYS` set; `make` exports a dev key).

## Agent setup

`.claude/settings.json` enables the team's plugins from `claude-plugins-official`: `mattpocock-skills`, `commit-commands`, `gopls-lsp`, `typescript-lsp`, `security-guidance`, `pr-review-toolkit`. Claude Code offers to install them on first open. The LSP plugins need their servers on `PATH`:

- `go install golang.org/x/tools/gopls@latest`
- `npm install -g typescript-language-server typescript`

`.mcp.json` adds the Mantine MCP server (`@mantine/mcp-server`: search docs, component props); Claude Code asks to enable it on first open. Mantine's own skills (`mantine-combobox`, `mantine-custom-components`, `mantine-form`) are vendored in `.claude/skills`, pinned in `skills-lock.json`; update with `npx skills update -p`. `mantine-form` covers `@mantine/form`, but forms here use React Hook Form (`techstack.md`): take only its Mantine input patterns.

`ui-ux-pro-max` is a user-level skill, not part of the repo; install it separately to use it.

## Code rules

- **All code is in English**: identifiers, error codes, log messages, test names, commit messages. Only code comments may be in Vietnamese. User-facing text never lives in code; it goes through i18n files in both `vi` and `en`.
- **Commit messages follow Conventional Commits**: `<type>(<scope>): <summary>`. Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `ci`, `build`, `perf`. Scope is the module or area (`record`, `hrm`, `platform`, `web`); omit it for repo-wide changes.
- **Comments are short.** Explain why, not what, in a line or two. Never reference repository documents, ADRs or roadmap milestones in comments.
- APIs return stable error codes with parameters, never sentences.
- Business dates are `date`; instants are `timestamptz`; "today" uses the tenant time zone.
- Migrations are SQL only, never edited after release, and never down-migrated.

## Workflow

- **Remote: GitHub** (`https://github.com/taoworklabs/mmerp`). Each spec gets one branch off `main` holding all its tickets (a commit per ticket) and one PR into `main` that closes the spec and its tickets; never commit directly to `main`. Only merge when asked. Push to `github`, using the `gh` CLI for GitHub operations.
- **Use the relevant skills when applying code**:
  - `mattpocock-skills:tdd` when building a feature or fixing a bug test-first;
  - `mattpocock-skills:diagnosing-bugs` when debugging;
  - `mattpocock-skills:codebase-design` when shaping a module interface or seam;
  - `ui-ux-pro-max` when building or reviewing a screen, together with `ui.md`;
  - `mattpocock-skills:domain-modeling` when adding a term to `CONTEXT.md` or writing an ADR;
  - `code-review` or `simplify` before finishing a change, and the `pr-review-toolkit` agents when reviewing a PR.

  If a listed skill is not installed, follow the same practice without it.
- Project skills in `.claude/skills/` give the step order for adding a backend module (`new-module`), a document type (`new-document-type`) and a screen (`new-screen`).
- A hard-to-reverse decision (module boundaries, shared entities, core changes) needs an ADR in `docs/adr/` before code.
- New domain terms go into `CONTEXT.md` in the same change that introduces them.

## Agent skills

### Issue tracker

GitHub Issues on `taoworklabs/mmerp` via `gh`; bodies in English. See `docs/agents/issue-tracker.md`.

### Triage labels

The five default labels (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`). See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: `CONTEXT.md` and `docs/adr/` at the repo root. See `docs/agents/domain.md`.
