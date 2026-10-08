# 0018. Fixed-term contract types, contract list

Status: accepted · 2026-10-06

## Context

ADR-0017 left contract types with only a name, so the system does not know which contracts must have an end date. Contracts could also only be viewed in each employee's profile, while HR needs to track contracts across the company, especially those about to expire.

## Decision

- `hrm.contract_types` gets a **`fixed_term`** flag, default `false`. A base contract of a fixed-term type must have an end date (`contract_end_required`); a type without a fixed term must not have one (`contract_end_not_allowed`). Checked when creating and editing a draft, against the type's flag at that moment; changing a type's flag does not touch contracts already issued. Appendices have no end date of their own, so they are not checked.
- The type's "pays insurance" flag is **not added yet**; it is added in M6 with the insurance calculation, as a column with a default. (2026-10-07: not done; payroll treats every allowance as subject to insurance until a real case needs this flag.)
- **Contract list** for the whole tenant, `GET /hrm/contracts`: base contracts and appendices, scoped by `hrm.contract.view` on the document's org unit; filters by status, type, org unit, employee, and **expiring soon** (`posted` base contracts with an end date in the next 30 days, in the tenant time zone, today included). The list has no money columns.

## Consequences

- Types that existed before the flag became "not fixed-term"; there is only test data, so no conversion is needed.
- The 30-day window is fixed in code; it becomes a setting when there is a need.
