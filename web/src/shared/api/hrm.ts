// The HRM API as the hrm area sees it.
import type { components, operations } from './schema.gen'

import { productClient } from './client'

export { unwrap } from './client'
export const api = productClient<'/hrm/'>()
export { ApiError } from './error'

export type Employee = components['schemas']['Employee']
export type EmployeeInput = components['schemas']['EmployeeInput']
export type EmployeeListItem = components['schemas']['EmployeeListItem']
export type Dependent = components['schemas']['Dependent']
export type DependentInput = components['schemas']['DependentInput']
export type EmployeeSort = NonNullable<NonNullable<operations['list-employees']['parameters']['query']>['sort']>
export type Leave = components['schemas']['Leave']
export type LeaveListItem = components['schemas']['LeaveListItem']
export type NewLeave = components['schemas']['NewLeave']
export type LeaveType = components['schemas']['LeaveType']
export type LeaveTypeInput = components['schemas']['LeaveTypeInput']
export type LeaveBalance = components['schemas']['LeaveBalance']
export type LeaveSort = NonNullable<NonNullable<operations['list-leaves']['parameters']['query']>['sort']>
export type Overtime = components['schemas']['Overtime']
export type OvertimeListItem = components['schemas']['OvertimeListItem']
export type OvertimeSort = NonNullable<NonNullable<operations['list-overtimes']['parameters']['query']>['sort']>
export type Contract = components['schemas']['Contract']
export type ContractListItem = components['schemas']['ContractListItem']
export type ContractSort = NonNullable<NonNullable<operations['list-contracts']['parameters']['query']>['sort']>
// The editable fields of a contract, as both create and update send them.
export type ContractFields = Omit<components['schemas']['NewContract'], '$schema' | 'employee_id' | 'parent_id'>
export type ContractTerms = components['schemas']['ContractTerms']
export type ContractType = components['schemas']['ContractType']
export type ContractTypeInput = components['schemas']['ContractTypeInput']
export type Timesheet = components['schemas']['Timesheet']
export type TimesheetListItem = components['schemas']['TimesheetListItem']
export type WorkCalendar = components['schemas']['WorkCalendar']
export type WorkWeek = components['schemas']['WorkWeek']
export type Holiday = components['schemas']['Holiday']
export type LegalParam = components['schemas']['LegalParam']
export type Payroll = components['schemas']['Payroll']
export type PayrollListItem = components['schemas']['PayrollListItem']
export type PayrollLine = components['schemas']['PayrollLine']
export type PayrollTotal = components['schemas']['PayrollTotal']
export type PayrollAdjustmentInput = components['schemas']['PayrollAdjustmentInput']
