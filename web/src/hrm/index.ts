import { IconUsersGroup, IconLayoutDashboard, IconCalendarCog, IconCash, IconCalendarEvent, IconCategory, IconScale, IconSitemap, IconClockHour4, IconFileCertificate, IconFileText, IconGitBranch, IconTable, IconUsers } from '@tabler/icons-react'
import type { AreaManifest } from '@/shared/area'
import { contractDocType, hrmKeys, leaveDocType, overtimeDocType, payrollDocType, timesheetDocType } from './keys'

export const hrm: AreaManifest = {
  product: 'hrm',
  label: 'hrm.nav.group',
  basePath: '/hrm',
  routes: () => import('./routes'),
  nav: [
    { label: 'hrm.nav.overview', path: 'overview', icon: IconLayoutDashboard },
    { label: 'hrm.nav.employees', path: 'employees', icon: IconUsers, permission: 'hrm.employee.view', group: 'hrm.nav.group_people' },
    // Every signed-in user may read the tree.
    { label: 'hrm.nav.org', path: 'org', icon: IconSitemap, group: 'hrm.nav.group_people' },
    { label: 'hrm.nav.contracts', path: 'contracts', icon: IconFileText, permission: 'hrm.contract.view', group: 'hrm.nav.group_people' },
    { label: 'hrm.nav.timesheets', path: 'timesheets', icon: IconTable, permission: 'hrm.timesheet.view', group: 'hrm.nav.group_time' },
    // Everyone files their own leave; the list shows what each user may see.
    { label: 'hrm.nav.leaves', path: 'leaves', icon: IconCalendarEvent, group: 'hrm.nav.group_time' },
    { label: 'hrm.nav.overtimes', path: 'overtimes', icon: IconClockHour4, group: 'hrm.nav.group_time' },
    { label: 'hrm.nav.payrolls', path: 'payrolls', icon: IconCash, permission: 'hrm.payroll.view', group: 'hrm.nav.group_pay' },
    { label: 'hrm.nav.contract_types', path: 'contract-types', icon: IconFileCertificate, permission: 'hrm.contract_type.manage', group: 'hrm.nav.group_config' },
    { label: 'hrm.nav.leave_types', path: 'leave-types', icon: IconCategory, permission: 'hrm.leave_type.manage', group: 'hrm.nav.group_config' },
    { label: 'hrm.nav.work_calendar', path: 'work-calendar', icon: IconCalendarCog, permission: 'hrm.calendar.manage', group: 'hrm.nav.group_config' },
    { label: 'hrm.nav.legal_params', path: 'legal-params', icon: IconScale, permission: 'hrm.legal_param.manage', group: 'hrm.nav.group_config' },
    { label: 'hrm.nav.approval_rules', path: 'approval-rules', icon: IconGitBranch, permission: ['hrm.approval.manage', 'core.approval.manage'], group: 'hrm.nav.group_config' },
  ],
  collapsibleGroups: ['hrm.nav.group_config'],
  homeIcon: IconUsersGroup,
  i18n: {
    meta: { vi: () => import('./i18n/vi/meta.json'), en: () => import('./i18n/en/meta.json') },
    main: { vi: () => import('./i18n/vi/main.json'), en: () => import('./i18n/en/main.json') },
  },
  recordTypes: {
    [leaveDocType]: {
      path: (id) => `leaves/${id}`,
      preview: () => import('./components/LeavePreview'),
      invalidate: (id) => [hrmKeys.leaves.detail(id), hrmKeys.leaves.lists(), hrmKeys.balances.all()],
    },
    [overtimeDocType]: {
      path: (id) => `overtimes/${id}`,
      preview: () => import('./components/OvertimePreview'),
      invalidate: (id) => [hrmKeys.overtimes.detail(id), hrmKeys.overtimes.lists()],
    },
    [contractDocType]: {
      path: (id) => `contracts/${id}`,
      preview: () => import('./components/ContractPreview'),
      // Contracts are listed per employee; the employee is not known from the id alone.
      invalidate: () => [hrmKeys.contracts.all()],
    },
    [timesheetDocType]: {
      path: (id) => `timesheets/${id}`,
      preview: () => import('./components/TimesheetPreview'),
      invalidate: (id) => [hrmKeys.timesheets.detail(id), hrmKeys.timesheets.lists()],
    },
    [payrollDocType]: {
      path: (id) => `payrolls/${id}`,
      preview: () => import('./components/PayrollPreview'),
      invalidate: (id) => [hrmKeys.payrolls.detail(id), hrmKeys.payrolls.lists()],
    },
  },
}
