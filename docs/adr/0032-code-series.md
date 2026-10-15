# 0032. Code series: issued codes for catalogue entries

Status: accepted · 2026-10-15 · extends [ADR-0014](./0014-documents-numbering-and-period-lock.md)

## Context

Customers, items and employees carry a code that is unique in the tenant, case-insensitive, and typed by the user. Users who do not care about codes still have to invent one, and a code that already sits on documents and prints can be edited at any time. Document numbers are already issued by the core (`numbering`, through `record`), but a module has no shared way to issue codes for its own entities; building a counter in each module repeats the locking and gap rules each time.

## Decision

- **`numbering` issues codes through code series.** A module registers a series at construction: its code (e.g. `sales.customer`), its product and a default prefix (`KH`, `HH`, `NV`). `numbering.series (code, prefix, last)` holds one row per series, created on first use; the document counters are untouched.
- **A code is issued only when the user leaves it empty**, inside the transaction that creates the entry: the counter row is incremented with `INSERT … ON CONFLICT DO UPDATE … RETURNING`, so it stays locked until commit and a rollback gives the number back. The form is `<prefix><number on 5 digits>` (`KH00001`); past 99999 the number grows. A typed code keeps its own validation and `*_code_taken` errors.
- **An issued code skips codes already typed by hand.** The module passes a check of its own table; `numbering` asks for the next number until the code is free. `numbering` stays unaware of module tables; the unique index stays the final guard.
- **One counter per series for the whole tenant**: no legal entity, no year. Gaps (rollbacks aside, deleted entries) are accepted; numbers are never reused.
- **The prefix is editable** by `core.admin` (`core.code_series.manage`) on one core screen listing every series. 0 to 10 characters, no whitespace, otherwise `invalid_code_prefix`. Changing it keeps the counter (`KH00012` then `CUS00013`). Each change is audited. A disabled product's series stay readable, not writable.
- **A code never changes after creation**, issued or typed. Edit bodies have no `code` field. A mistyped code is fixed by deactivating the entry and creating a new one.
- Document numbers are unchanged: format, scope and prefixes fixed in code.

## Rejected alternatives

- **A counter in each module.** Same locking and gap rules written three times, and three places to configure prefixes.
- **Reusing `numbering.counters`** with a neutral legal entity and year. It needs a fake org unit for the foreign key and has no place for the prefix.
- **Pushing the counter past typed codes** (parse every typed code that matches the prefix). Breaks when the prefix changes, and adds parsing for a rare case the skip loop already covers.
- **Free patterns or configurable digits** (`{PREFIX}-{YYYY}-{#####}`). No one has asked; a pattern language is code to maintain forever.
- **Codes editable until first used on a document.** Needs a usage check across modules and products for every catalogue; locking at creation needs none.

## Consequences

- Catalogue entries cannot be deleted today, so a mistyped code stays on a deactivated entry; employees have no active flag, so a wrong employee stays in the list.
- A future catalogue (suppliers, warehouses) gets issued codes by registering a series and calling `numbering` when the code is empty.
