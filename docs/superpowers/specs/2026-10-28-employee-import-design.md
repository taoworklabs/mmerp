# Employee profiles from Excel: export, bulk create and bulk edit

Date: 2026-10-28 · Status: draft, awaiting approval

## Intent

A company moving to mmerp today types every employee profile by hand, one form at a time, and has no way to correct many profiles at once (e.g. filling in the personal tax code of 200 people). That is the first thing that keeps HR on a parallel Excel file. `products/hrm.md` says it plainly: "there is no profile import/export yet".

This work adds one Excel round trip for employee profiles: **export** the list, edit it in Excel, **import** it back. The same import creates new employees and edits existing ones. It is the first of three steps; contracts in force and dependants come later, each with its own design (contracts need an ADR, because importing a contract that is already in force means a `posted` document that never went through approval).

## Goals

### Export

- From the employee list, HR exports the employees the list currently shows (same filters: org unit, status, search), as one sheet whose layout is exactly the import's. An empty export is the template. The paged `Employees` list lacks most columns, so the export has its own unpaged query with the same filters and scope.
- Dates are written as `dd/mm/yyyy` text: `dataio.Sheet` cells are only strings, `int64` and decimals, and extending core for date cells is not worth it here *(my call)*.
- Needs `hrm.employee.view`; only employees in the actor's view scope appear. Export is an export operation for `ProductGate`, so it works while HRM is disabled.
- Sensitive columns (ID card, social insurance book number, personal tax code, bank account) hold **full values** only for employees on whom the actor has `hrm.employee.sensitive`; otherwise those cells are empty. Every sensitive value exported is audited like a reveal (`hrm.employee_sensitive_viewed`, or one audit row per employee listing the fields, *builder's call*); values never enter the audit data. The export file is protected like the other `dataio` exports: temporary, and every download re-checks the permission (`Check`).

### Import

- One sheet, one row per employee, a header that must match the export's (translated, like the other imports). Columns, in this order:
  - Mã NV, Họ tên, Ngày sinh, Giới tính, SĐT, Email, Địa chỉ
  - Phòng ban, Mã người quản lý trực tiếp, Ngày vào làm, Ngày nghỉ việc
  - CCCD/Hộ chiếu, Số sổ BHXH, Mã số thuế, Số tài khoản
- **Employee code is the key.** A code that exists edits that employee. A code that does not exist creates an employee with that code. An empty code creates an employee with a code from the `hrm.employee` series. A code cannot be changed through the file. Matching is case-insensitive, like the leave balance import. The same code twice in one file is a row error.
- **An empty cell keeps the current value; it never clears it.** This is what makes the round trip safe: a file exported by someone without the sensitive permission has empty sensitive columns, and importing it back must not wipe those values. Clearing a value is done on the profile screen. On a new employee an empty cell simply means no value; required fields (full name, org unit, hire date) must be filled.
- **Org unit** is written by name. If the name is ambiguous, the row writes the path from an ancestor that makes it unique, segments joined by ` / ` (e.g. `Chi nhánh HN / Kinh doanh`). Export always writes the shortest unique path *(my call)*. A name matching no unit, or still matching several, is a row error naming the value. Two sibling units with the same name have no unique path; their employees cannot be imported until one is renamed in the org chart. Adding a uniqueness rule to `iam` is out of scope *(my call: such siblings are a data mistake, rare at SMB scale)*.
- **Manager** is written as the manager's employee code. It may be an employee created by another row of the same file. The cycle rule is the same as on the form (`manager_self`, `manager_cycle`), checked against the state after the whole file is applied, so a file that swaps two people's managers is accepted. This needs one final-state check under `LockManagerTree`, not the per-call check of `UpdateEmployee`.
- **Gender** and dates use the same values the export writes (localised gender labels; dates as Excel dates or `dd/mm/yyyy`, *builder's call* on what is accepted beyond what export writes).
- **Unchanged rows are skipped before any permission check** on their plain fields: an exported file may hold employees the actor may view but not edit, and sending those rows back untouched is not an error. Sensitive cells are the exception: a non-empty sensitive cell without the permission is a row error even when it equals the stored value, so the file cannot be used to test guesses against a hidden value. Someone without the permission who imports a file exported by someone with it must empty those columns first *(my call)*.
- **The account link is kept.** `prepare` today builds the row without `old.UserID`, so a nil login unlinks the account; the import must carry the current link (or `prepare` keeps `old.UserID` when no login is given).
- **Same rules as the form**, applied per row: edit permission at the target unit (and at the old unit when moving), the sensitive permission at both units for a sensitive cell, termination not before hire, the sensitive-access-laundering rule on moves (`prepare` in `employee.go`). Writing a non-empty sensitive cell without the permission is a row error, not a silent skip.
- **Scope:** elsewhere an employee outside the actor's view scope does not exist, but here an unknown code means "create". So a code that exists outside the actor's scope is a row error `employee_code_taken` *(my call: it reveals only that the code is in use, which the form's code check already reveals)*.
- **All or nothing**, as every `dataio` import: one wrong row and nothing is written; the requester sees every row error with its Excel row number. The import is a job on behalf of the requester and re-checks permissions when it runs.
- **Audit:** each created or changed employee gets the same audit row the form writes (`hrm.employee_created` / `hrm.employee_updated`, with the changes). A row identical to the current profile writes nothing and no audit.
- The import button sits on the employee list next to the existing leave balance import, shown when the actor may create or edit employees (from the list's actions). The dialog explains the empty-cell rule in one line.

## Non-goals

- Contracts, dependants, leave balances on hire: later steps. Leave balances already have their own import.
- Linking accounts (`user_login`) and creating accounts: stays on the profile screen *(my call; accounts have their own lifecycle in `iam`)*.
- Clearing values through the file, renaming codes, deleting employees.
- Importing other files' layouts (e.g. a MISA export). The file must be mmerp's template.
- Several sheets in one file: `dataio` reads the first sheet, and that stays.

## Anti-goals

- Re-importing an export must never lose data, whoever exported it.
- No path through the file may let someone read or write a sensitive value, or reach an employee, that the form would refuse them.
- No half-applied file.

## Constraints

- Everything goes through the existing services' rules; the import must not grow a second copy of the profile validation. Reusing `prepare` (or splitting it so both form and import call the same checks) is expected.
- `UpdateEmployee` replaces every plain field; the import merges the file's non-empty cells over the current row before applying it.
- Profiles are master data, not documents, so one transaction for the whole file does not break the one-document-per-transaction rule.
- SMB scale: a file of a few thousand rows must import in one job without timing out *(builder measures; the timesheet import is the reference)*.
- Error codes are stable keys under `hrm.import.*`, in `vi` and `en`.
- Tests against a real Postgres, an e2e for the round trip (export, edit, import) and for the empty-sensitive-columns case.

## Left to the builder

- How rows are applied so that managers created in the same file resolve (e.g. create all, then set managers).
- Exact audit grouping for sensitive values in an export.
- Which date and gender spellings import accepts beyond the exported ones.
- Doc updates: `products/hrm.md` (import/export list), `CONTEXT.md` if a new term appears.
