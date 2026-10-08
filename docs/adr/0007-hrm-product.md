# 0007. The HRM product

Status: accepted · 2026-10-05

## Context

HRM is the first product, and the core is built to exactly what HRM needs. (2026-10-07: from M11 the core is built first, with HRM proving it; see [ADR-0023](./0023-build-core-first.md).) The first version covers employee profiles, contracts, leave and overtime, timekeeping, and payroll calculation. The full description is in [products/hrm.md](../../products/hrm.md). This ADR only records the hard-to-reverse decisions.

## Decision

- **A single module** `hrm`. Split only when another product needs one part on its own.
- **A payroll captures all of its inputs**, together with the version of each source document and of the legal parameters. When a source changes, a warning is shown, and recomputation happens only when the user explicitly clicks. Once finalised it is immutable.
- **Payroll lock at the legal-entity level**, following the same pattern as `record`'s period lock. Source documents take `FOR SHARE` and then check the **affected date range** (a leave request spanning two months, a contract with no end date) against every finalised period. Finalising and cancelling a payroll take `FOR UPDATE`. No lock per period row, because periods are created on demand and a source transaction may start before the period row exists.
- **Each legal entity and period has at most one valid payroll**, recorded in `posted_payroll_id`. Finalising is refused if the period already has another payroll; cancelling is only accepted for the payroll currently holding the period.
- **Payroll business lines are aggregated by department or legal entity**, carrying no per-person amounts. Per-person detail lives in the encrypted payroll.
- **Legal parameters are not hard-coded.** They are data with effective dates; each release ships the default set.
- **Pay periods are stored as date ranges**, even though the first version only supports calendar months.
- **Direct manager** is an HRM business relationship (`manager_id`), provided to `approval` through `Approvers`, independent of the permission-scope tree.

## Consequences

- HRM proves almost the whole core: the permission-scope tree, document lifecycle, approval, period lock, Excel import, business lines, sensitive fields.
- A legal-parameters update must be shipped every year. This is a long-term operational obligation.
- Because salaries are encrypted, every salary aggregation must be done in the service, not in SQL.
- The ledger has no per-employee salary payables. An accountant must confirm this approach before production use.
- Installations without accounting enabled still have payroll business lines, ready for when accounting is enabled later.
