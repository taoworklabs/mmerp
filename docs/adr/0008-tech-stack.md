# 0008. Tech stack

Status: accepted · 2026-10-05

## Context

Three constraints drive the choices: on-premise has only the app and Postgres; one binary that migrates and backs up by itself; no dependency on the Internet in business flows. The full list and usage conventions are in [techstack.md](../../techstack.md). This ADR only records the hard-to-reverse decisions.

## Decision

- **Go + Postgres + River.** No Redis, no broker. Jobs live in Postgres.
- **Huma v2 on chi.** The OpenAPI 3.1 spec generated from code is the single source for the frontend and third parties.
- **pgx + sqlc**, hand-written SQL. No ORM.
- **A home-grown migration runner**, because no popular tool can run all pending migrations in one transaction, together with an advisory lock and River's migrations.
- **Sessions stored in the DB, no JWT.** Revocation takes effect immediately, and there are no multiple services that need independent authentication.
- **React + TypeScript + Mantine**, one Vite app.
- **Vietnamese and English i18n from the start.** The API returns error codes, not sentences; seeded data is stored as translation keys; user-entered data is not translated.
- **Business dates are `date`; the time zone is per tenant**, default `Asia/Ho_Chi_Minh`.

## Consequences

- Every UI string and error code must have an English version from the start, even if the first users use only Vietnamese. Adding a third language only needs a new translation file.
- Error codes become part of the API interface: once released, their meaning does not change.
- A home-grown migration runner means maintaining it ourselves. Accepted, because it is small.
- The print template and PDF engine is not chosen yet. That choice must not break the app-plus-Postgres-only constraint.
