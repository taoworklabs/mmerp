---
name: new-module
description: Add a backend module under internal/ (core, shared, or a product module). Use when creating a new Go module package or Postgres schema.
---

# New backend module

The rules live in `backend.md`; this skill is the order to apply them in. Each step ends on its completion criterion.

1. **Decide and record.** Read `backend.md` (Layers, Rules between modules, Data ownership) and the existing ADRs in `docs/adr/`. Write an ADR covering every item in `backend.md` → "Adding a module", plus the personal-data fields required by `platform.md` → "Personal data".
   Done when: the ADR names the layer, owned tables, `deps.go` interfaces, `hooks.go` hooks (sync or job), record types, posting line types, and personal/sensitive fields.

2. **Name things.** Add every new domain noun from the ADR to `CONTEXT.md` (use the `mattpocock-skills:domain-modeling` skill).
   Done when: each noun in the ADR has a `CONTEXT.md` entry, and code identifiers use the English form of those terms.

3. **Schema and queries.** Write the SQL migration creating a schema named after the module, the module's `queries.sql`, and its sqlc entry. Writes target only this module's schema; reads may join only its own tier, lower tiers, and the schemas of products its product declares as dependencies. Apply `roadmap.md` → "Criteria common to every milestone": every status or kind column has a `CHECK` with exactly the values the service accepts; columns are `NOT NULL` unless empty means something; money is `bigint` in the minor unit and business dates are `date`; no dynamic SQL.
   Done when: `make gen` runs clean, the `queries.sql` scan in `make lint` passes, and every status or kind column has its `CHECK`.

4. **Module shape.** Create the files of the module shape in `backend.md` (`module.go`, `service.go`, `types.go`, `deps.go`, `hooks.go`, `handler.go`, `internal/store/`), leaving out any that would be empty (no routes: no `handler.go` or `module.go`; no hooks: no `hooks.go`). Services take the DB from `platform.DBFrom(ctx)` and open transactions with `platform.InTx`. Register record types and roles through the core registration functions received in `Deps`, and declare the permissions that guard sensitive data with `iam.RegisterSensitive` (otherwise tenant administrators hold them); `platform.Module` carries only platform and library types. An ID that is easy to confuse (user, employee, org unit, legal entity, document) and crosses the module boundary (`types.go`, `deps.go`, hooks) uses the owning module's ID type; declare it there if missing. Stores stay `int64`.
   Done when: exported identifiers live only in `service.go`, `service_<part>.go` and `types.go`, plus `Deps` in `deps.go`, `Module` in `module.go` and hook types in `hooks.go`, and the module compiles.

5. **Wire it.** In `internal/app`, pass real implementations for every `deps.go` interface; attach no-op adapters only for hooks nobody implements. Add the product's `depguard` rule (`.golangci.yml`), the same shape as the others: lax mode, allow the product's own package, deny `internal/modules`.
   Done when: `make lint` is green.

6. **Document types.** For each record type that is a document, run the `new-document-type` skill.
   Done when: every document type of the module passes `recordtest`.

7. **Test.** Write tests through the service interface against real Postgres (use the `mattpocock-skills:tdd` skill).
   Done when: `make test` is green.

8. **Frontend.** For a product module, add or extend its area following `frontend.md` → "Adding an area", and build screens with the `new-screen` skill.
