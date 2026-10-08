# 0001. One database per tenant

Status: accepted · 2026-10-05

## Context

The primary deployment is on-premise on the organisation's server, with no guaranteed Internet access; cloud runs the same binary. Three options were considered: one deployment per tenant (A), one app with many databases (B), and a shared schema with a `tenant_id` column and row-level security (C).

## Decision

Every tenant always has its own database. On-premise and early cloud use A; cloud moves to B when operating A becomes too heavy. C is not used.

## Consequences

- Business code is tenant-unaware: no tenant columns, services take no tenant parameter, the DB always comes from the context.
- Backup, restore and data deletion per tenant are operations on a single database.
- Under B, migrations run N times and the connection pool grows with the number of tenants (handled as described in [tenancy.md](../../tenancy.md)).
- C is rejected because forgetting one filter condition leaks data between tenants, and because it does not fit on-premise.
