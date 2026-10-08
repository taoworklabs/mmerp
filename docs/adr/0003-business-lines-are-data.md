# 0003. Business lines are data; the posting job reconciles against the current state

Status: accepted · 2026-10-05

## Context

The first design put a "get posting lines" callback on `record`'s record type, so `core` had to depend upward on `shared`, and every module had to be able to rebuild past figures when accounting is enabled later. Moreover, requiring only "idempotent per document" is not enough. If the cancel job runs before the post job the ledger is wrong, and a locked period could be silently changed when posting rules are edited.

## Decision

- `shared/posting` owns `posting.lines`. In `OnTransition`, the module calls `posting.Record` on moving to `posted` and `posting.Void` on moving to `cancelled` (marked as voided, not deleted). Business lines are written even without accounting.
- Business lines contain no sensitive data tied to an individual. `posting.lines` and the ledger are not encrypted, so the module must aggregate such lines (e.g. salaries payable aggregated by department). Per-person detail lives in the owning module's encrypted tables.
- `posting` exposes an `OnChanged(ref)` hook; `accounting` implements it with a job carrying only `(doc_type, doc_id)`.
- The worker takes `FOR SHARE` on the legal entity's period-lock row, then locks per document, reads the current state of `posting.lines` and brings the ledger in line with that state. It does not replay in event order.
- `accounting` implements `record`'s `BeforeLock` hook, and refuses to lock a period while some documents do not match the ledger.
- A locked period is never re-posted. Applying new posting rules is a manual operation, and only on open periods. The only exception is the first posting run when accounting is enabled, because the ledger is still empty then.

## Consequences

- `record` knows nothing about posting; layering points the right way.
- Job order and retry count do not affect the result.
- `shared/posting` has tables. Installations that do not use accounting still pay storage for business lines, and that is exactly the data needed when accounting is enabled.
- A job that keeps failing blocks the period lock. This is intended, and the admin screen must show which documents are stuck.
