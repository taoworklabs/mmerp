# Tech stack

Decision and rationale: [ADR-0008](./docs/adr/0008-tech-stack.md). This document describes what is used, for what, and the conventions for using it.

## Constraints

Every choice below must satisfy these three constraints:

- **On-premise has only the app and Postgres.** No Redis, no message broker, no sidecar container. Sessions, jobs and caches all live in Postgres or in the app's memory.
- **One binary that migrates and backs up by itself.** The app image must contain `pg_dump`. Only self-migration exists so far; self-backup and `pg_dump` in the image are not built yet ([tenancy.md](./tenancy.md#update-procedure-on-premise)).
- **No Internet dependency** in business flows. No SaaS service is used while handling a request.

## Backend

| Area | Uses | Notes |
| --- | --- | --- |
| Language | Go, latest stable release | Upgrade Go with each app release |
| HTTP | [Huma v2](https://huma.rocks) on the [chi](https://go-chi.io) router | Huma generates OpenAPI 3.1 from Go types and validates requests; chi handles routing and middleware. Handlers only translate between HTTP and the service, with no logic |
| Database | Postgres, pinned to one major | At least version 15 (needs `NULLS NOT DISTINCT`). Pick the newest major that has had at least one patch release |
| DB access | [pgx v5](https://github.com/jackc/pgx) + [sqlc](https://sqlc.dev) | One `queries.sql` per module; generated code lives in the module's `internal/store`. Get a connection through `platform.DBFrom(ctx)` ([backend.md](./backend.md#transactions)) |
| Jobs | [River](https://riverqueue.com) | Queue in the tenant database, enqueued with `InsertTx` in the same transaction ([ADR-0004](./docs/adr/0004-job-queue-in-tenant-database.md)) |
| Migration | Custom runner in `platform` | SQL files embedded with `embed`; all pending migrations run in one transaction under an advisory lock, and River's migrations run in the same transaction ([tenancy.md](./tenancy.md#versions-and-migrations)) |
| Sign-in | Cookie + session stored in the DB (`iam`) | Cookie `HttpOnly`, `Secure`, `SameSite=Lax`. Revoking a session takes effect immediately. Third-party API access uses API keys |
| Password hashing | argon2id (`golang.org/x/crypto/argon2`) | |
| Column encryption | AES-256-GCM (standard library) | Encrypted data starts with a key version, so keys can be rotated gradually. Keys live outside the database ([platform.md](./platform.md#personal-data)) |
| Decimals | [`shopspring/decimal`](https://github.com/shopspring/decimal) | For rates, multipliers, fractional quantities. Money is always `int64`/`bigint`; rounding only through the `platform` function ([ADR-0006](./docs/adr/0006-money-rounding.md)). No `float` for money or rates |
| Excel | [excelize](https://github.com/qax-os/excelize) | Only used in `dataio` |
| File storage | Local disk | The `platform.Files` type (one directory, `FILES_DIR`), no interface. Add an S3-compatible adapter when cloud needs it |
| Configuration | Environment variables, read with the standard library | Read and validated in full at startup; on error, stop and report the variable name |
| Logging | `log/slog`, JSON to stdout | Every request log line carries the request id; never log personal data |
| Print templates / PDF | Not chosen yet | Chosen when building `printing`. Headless Chrome needs a third container, violating the constraints above |

## Frontend

| Area | Uses | Notes |
| --- | --- | --- |
| Build | [Vite](https://vite.dev) | A single app; each product is an area in `web/src/<product>` ([frontend.md](./frontend.md)) |
| Framework | React + TypeScript (strict) | Pin TypeScript 5.x until `typescript-eslint` and `openapi-typescript` support a newer version |
| API client | [`openapi-typescript`](https://openapi-ts.dev) + `openapi-fetch` | Generated from Huma's spec by `make gen`; API types are never hand-written |
| API calls | [TanStack Query](https://tanstack.com/query) | |
| Data tables | Mantine's `Table`, wrapped in `DataTable` | Enough for predeclared columns. Add [TanStack Table](https://tanstack.com/table) when dynamic columns for custom fields arrive |
| Router | [React Router](https://reactrouter.com), data mode | Builds the route tree from each area's manifest, lazily loaded ([frontend.md](./frontend.md#startup-flow)) |
| Forms | [React Hook Form](https://react-hook-form.com) | |
| UI kit | [Mantine](https://mantine.dev) (`core`, `dates`, `notifications`) | Areas do not use convention-bearing components directly; they use the wrappers in `shared/ui` ([ui.md](./ui.md#enforcement)). `@mantine/dates` needs `dayjs` (peer dependency), used only through `DateField`, `MonthField` and `MonthFilter` in `shared/ui/form` |
| Font | Inter, self-hosted via `@fontsource-variable/inter` | Full Vietnamese diacritics and tabular figures; not loaded from a CDN |
| Icons | [Tabler Icons](https://tabler.io/icons) (`@tabler/icons-react`) | |
| i18n | [`react-i18next`](https://react.i18next.com) | See the i18n section below |
| Package manager | pnpm | |

## i18n

**Vietnamese (`vi`) and English (`en`)** are supported from the start.

- Each user's language is stored in `iam.users.locale`. It defaults to the tenant setting, which defaults to `vi`.
- **The API returns error codes, not sentences.** E.g. `{"code": "period_locked", "params": {"date": "2026-03-31"}}`. The frontend translates error codes. Error codes are part of the API interface: once released, their meaning does not change.
- **Backend-generated text** (per-row Excel import errors, notifications, emails; later print templates) uses each module's JSON translation files, embedded in the binary. Translation keys are prefixed with the module name (`hrm.leave.insufficient_balance`). Looked up through a `platform` function.
- **Data seeded by core and modules** (role names, record type names, business line type names, legal parameter names) is stored as translation keys.
- **User-entered data** (department names, employee names, notes, custom field labels) is not translated.
- Numbers, money and dates are formatted in the frontend with `Intl`, in the user's language. VND has no decimal part.
- CI checks that every translation key exists in both `vi` and `en`.

## Time

- Instants (created, updated, audit, session) are stored as `timestamptz`.
- Business dates (document date, leave date, pay period, book closing date) are stored as `date`, without time.
- The tenant time zone is stored in a setting, default `Asia/Ho_Chi_Minh`. "Today" and day boundaries follow this time zone, not the server's or the browser's.

## Testing and quality

| Area | Uses | Notes |
| --- | --- | --- |
| Backend tests | Go's `testing` + real Postgres | Each test gets a database created from a migrated template (`CREATE DATABASE … TEMPLATE`). Never mock the database |
| Contract tests | `recordtest` | Mandatory for every document type ([documents.md](./documents.md#guarantees)) |
| Backend lint | golangci-lint + `depguard` + `forbidigo` | Layering and isolation rules ([backend.md](./backend.md#isolation)) |
| Frontend lint | [dependency-cruiser](https://github.com/sverweij/dependency-cruiser) + ESLint (`@eslint/js`, `typescript-eslint`, `eslint-plugin-i18next`, `globals` for browser and Node globals; `style` banned with the built-in `no-restricted-syntax`) + TypeScript strict | dependency-cruiser checks area boundaries on the resolved dependency graph: it catches relative paths, aliases, re-exports and dynamic imports. The configuration is verified by `web/lint-fixtures` ([frontend.md](./frontend.md#isolation)) |
| Frontend tests | Vitest | Logic does not live in components; no snapshot tests |
| Frontend–API end-to-end tests | Playwright, run against the real backend and Postgres | Only for flows where a bug has serious consequences: sign-in and per-unit permissions (users do not see data outside their scope); submitting, withdrawing, approving and rejecting requests; timesheet import from Excel; `version` conflicts when two tabs edit at once. Not used to test individual screens. Runs only in CI and on dev machines, not in the image |

## Development tools

Everything goes through `make`; the full list (`gen`, `lint`, `test`, `e2e`, `db`, `dev`, `reset-db`, `seed`, `image`, `smoke`) is in the command table of `CLAUDE.md`. `make lint` covers golangci-lint, `tsc`, ESLint, the colour-literal scan ([ui.md](./ui.md#enforcement)), dependency-cruiser (must report no errors on `web/src`; must report every violation on `web/lint-fixtures`), the translation key check, and the `queries.sql` scan.

CI runs `make gen` and fails if the generated code differs from the committed code.

## Packaging and deployment

- Multi-stage image build. The runtime image is Debian slim; the frontend is prebuilt and embedded in the binary with `embed`.
- `compose.yml` has only `app` and `postgres`.
- Not built yet, done in [M10](./roadmap.md#m10-on-premise-operations): `postgresql-client` of the same major as Postgres in the runtime image; offline distribution (`docker save` to a single file, with an install script); image signing with cosign.

## Adding a new library

In order, stop at the first option that solves it:

1. Go's standard library, or a built-in browser API.
2. A library already in the lists above.
3. Write it yourself, if it takes only a few dozen lines.
4. Add a new library. It must be recorded in this document with the reason. Server-side libraries must not call out to the Internet and must not need an extra container.
