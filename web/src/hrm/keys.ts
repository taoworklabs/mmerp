// The leave request's record type, as registered by the backend.
export const leaveDocType = 'hrm.leave_request'
export const overtimeDocType = 'hrm.overtime_request'
export const contractDocType = 'hrm.contract'
export const timesheetDocType = 'hrm.timesheet'
export const payrollDocType = 'hrm.payroll'
export const employeeDocType = 'hrm.employee'

// Query keys of the hrm area; every parameter that changes a result is part of its key.
export type EmployeeQuery = { q: string; status: string; org_unit_id: string; sort: string; page: number; pageSize: number }
export type LeaveQuery = { status: string; sort: string; page: number; pageSize: number }
export type OvertimeQuery = LeaveQuery
export type TimesheetQuery = { month: string; org_unit_id: string; status: string; page: number; pageSize: number }
export type PayrollQuery = { month: string; legal_entity_id: string; status: string; page: number; pageSize: number }
export type ContractQuery = { status: string; contract_type_id: string; org_unit_id: string; expiring: string; sort: string; page: number; pageSize: number }

export const hrmKeys = {
  employees: {
    all: () => ['hrm', 'employees'] as const,
    actions: () => ['hrm', 'employees', 'actions'] as const,
    lists: () => ['hrm', 'employees', 'list'] as const,
    list: (q: EmployeeQuery) => ['hrm', 'employees', 'list', q] as const,
    picker: (q: string) => ['hrm', 'employees', 'picker', q] as const,
    detail: (id: number) => ['hrm', 'employees', id] as const,
    dependents: (id: number) => ['hrm', 'employees', id, 'dependents'] as const,
  },
  leaves: {
    actions: () => ['hrm', 'leaves', 'actions'] as const,
    lists: () => ['hrm', 'leaves', 'list'] as const,
    list: (q: LeaveQuery) => ['hrm', 'leaves', 'list', q] as const,
    detail: (id: number) => ['hrm', 'leaves', 'detail', id] as const,
  },
  balances: {
    all: () => ['hrm', 'balances'] as const,
    employee: (id: number) => ['hrm', 'balances', id] as const,
  },
  overtimes: {
    actions: () => ['hrm', 'overtimes', 'actions'] as const,
    lists: () => ['hrm', 'overtimes', 'list'] as const,
    list: (q: OvertimeQuery) => ['hrm', 'overtimes', 'list', q] as const,
    detail: (id: number) => ['hrm', 'overtimes', 'detail', id] as const,
  },
  contracts: {
    all: () => ['hrm', 'contracts'] as const,
    list: (q: ContractQuery) => ['hrm', 'contracts', 'list', q] as const,
    employee: (id: number) => ['hrm', 'contracts', 'employee', id] as const,
    detail: (id: number) => ['hrm', 'contracts', 'detail', id] as const,
  },
  timesheets: {
    actions: () => ['hrm', 'timesheets', 'actions'] as const,
    lists: () => ['hrm', 'timesheets', 'list'] as const,
    list: (q: TimesheetQuery) => ['hrm', 'timesheets', 'list', q] as const,
    detail: (id: number) => ['hrm', 'timesheets', 'detail', id] as const,
  },
  payrolls: {
    actions: () => ['hrm', 'payrolls', 'actions'] as const,
    lists: () => ['hrm', 'payrolls', 'list'] as const,
    list: (q: PayrollQuery) => ['hrm', 'payrolls', 'list', q] as const,
    detail: (id: number) => ['hrm', 'payrolls', 'detail', id] as const,
  },
  leaveTypes: () => ['hrm', 'leave-types'] as const,
  calendar: (legalEntity: number, year: number) => ['hrm', 'calendar', legalEntity, year] as const,
  calendars: () => ['hrm', 'calendar'] as const,
  legalParams: () => ['hrm', 'legal-params'] as const,
  contractTypes: () => ['hrm', 'contract-types'] as const,
}
