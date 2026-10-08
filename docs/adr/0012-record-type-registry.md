# 0012. `record`: record type registration and `Can`

Status: accepted · 2026-10-05

## Context

The employee profile is the first record type. There are no documents yet, so `record` only needs registration and the permission question ([documents.md](../../documents.md#record-types)). The document lifecycle (ADR-0002) is added to this module in a later milestone.

## Decision

- `record` is a `core` module with no tables yet. The registry lives in `record.Service`, created once in `internal/app`.
- Modules register `record.Type{Code, Product, Kind, Can}` at initialisation. `Kind` currently has only `Catalog`. Registering a duplicate code panics at startup.
- `record.Can(ctx, type, id, action)`, with `action` one of `view`, `edit`, `post`, `cancel`, `export`: asks the owning module's `Can(ctx, id, action)`, then goes through `ProductGate` by the action's class (`view` is read, `export` is export, the rest are write). `AllowedActions` returns the list of permitted actions, so the API can return `allowed_actions`.

## Consequences

- When a product is disabled, `allowed_actions` loses the write actions automatically, without the module doing anything.
- Change history in the UI does not exist yet; it will go through `record.Can` when built in the documents milestone.
