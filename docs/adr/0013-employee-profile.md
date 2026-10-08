# 0013. Employee profile and dependants

Status: accepted · 2026-10-05

## Context

The `hrm` module (ADR-0007) starts with the employee profile, the first catalog with per-org-unit permissions and sensitive fields.

## Decision

- `hrm.employees`: `code` (employee code, entered by hand, unique, case-insensitive), `full_name`, `date_of_birth`, `gender`, `phone`, `email`, `address`, `org_unit_id` (department, a node of the org tree), `manager_id` (direct manager), `user_id` (account, unique, may be empty), `hire_date`, `termination_date`. An employee is active when there is no termination date or the termination date has not yet passed in the tenant's time zone.
- Sensitive, encrypted columns (ADR-0011): `national_id` (citizen ID (CCCD) or passport), `social_insurance_no`, `tax_code`, `bank_account`. The detail API only returns a flag saying whether a value exists; values are fetched one field at a time, each read writing an audit row.
- `hrm.dependents (id, employee_id, data)`: all dependant information is sensitive, so the whole record is one encrypted JSON (full name, relationship, date of birth, ID document number, deduction from month, to month).
- `hrm.employee` is a catalog record type. Its scope is the employee's `org_unit_id`; an employee outside scope returns `not_found`.
- HRM roles:

  | Role | Permissions |
  | --- | --- |
  | `hr` | `hrm.employee.view`, `hrm.employee.edit` |
  | `viewer` | `hrm.employee.view` |
  | `sensitive_viewer` | `hrm.employee.sensitive` |

  Viewing and editing sensitive fields and dependants needs `hrm.employee.sensitive` at the employee's org unit, plus the view or edit permission. This permission is its own role, not bundled with any other role. Moving an employee to an org unit where the editor has this permission requires the editor to have it at the old org unit too, so that a move cannot become a way to grant oneself view access.
- Before any employee is selected, `GET /hrm/employees/actions` returns list-level actions: `create` (edit permission at at least one org unit and the product enabled) and `view_sensitive` (sensitive permission at at least one org unit). The frontend shows the create button and the sensitive fields of the create form from this list, never deriving them from `/me`.
- Account linking is entered by login name, so HR staff do not need permission to view the user list.
- **Direct manager:** nobody manages themselves (CHECK) and there are no management cycles. Changing `manager_id` takes, after the employee row lock, the advisory lock `hrm.manager_tree` before checking the management chain, so two concurrent changes cannot close a cycle. The manager must be within the editor's view scope; outside scope they are treated as non-existent (`manager_not_found`), so ids cannot be probed.
- **Status** (`active`, `terminated`) is computed by the API from today's date in the tenant's time zone; the frontend does not compute it.
- An employee's `allowed_actions` contains `record`'s actions (`view`, `edit`) plus `view_sensitive` when the user has `hrm.employee.sensitive` at the employee's org unit. `view_sensitive` is a read action, so it does not need to go through `ProductGate`.

## Personal data

| Table | Field | Kind |
| --- | --- | --- |
| `hrm.employees` | `full_name`, `date_of_birth`, `gender`, `phone`, `email`, `address` | Personal data |
| `hrm.employees` | `national_id`, `social_insurance_no`, `tax_code`, `bank_account` | Sensitive, encrypted |
| `hrm.dependents` | `data` | Sensitive, encrypted |

## Consequences

- Employees cannot be looked up by citizen ID number in SQL.
- Employee codes are not generated automatically yet; add it when there is a need.
- Hire date and termination date are reviewed again at the payroll-method sign-off gate.
