import { Navigate, Route, Routes } from 'react-router'
import { useCan } from '@/shared/auth/me'
import { ApprovalRules } from '@/shared/document'
import { ForbiddenPage, NotFoundPage } from '@/shared/ui/states'
import { ContractPage } from './pages/ContractPage'
import { ContractsPage } from './pages/ContractsPage'
import { ContractTypesPage } from './pages/ContractTypesPage'
import { EmployeePage } from './pages/EmployeePage'
import { EmployeesPage } from './pages/EmployeesPage'
import { LeavePage } from './pages/LeavePage'
import { LeavesPage } from './pages/LeavesPage'
import { LeaveTypesPage } from './pages/LeaveTypesPage'
import { NewEmployeePage } from './pages/NewEmployeePage'
import { NewContractPage } from './pages/NewContractPage'
import { NewLeavePage } from './pages/NewLeavePage'
import { NewOvertimePage } from './pages/NewOvertimePage'
import { OverviewPage } from './pages/OverviewPage'
import { OrgStructurePage } from './pages/OrgStructurePage'
import { OvertimePage } from './pages/OvertimePage'
import { OvertimesPage } from './pages/OvertimesPage'
import { TimesheetPage } from './pages/TimesheetPage'
import { TimesheetsPage } from './pages/TimesheetsPage'
import { LegalParamsPage } from './pages/LegalParamsPage'
import { PayrollPage } from './pages/PayrollPage'
import { PayrollsPage } from './pages/PayrollsPage'
import { WorkCalendarPage } from './pages/WorkCalendarPage'

export default function HrmRoutes() {
  const canView = useCan('hrm.employee.view')
  const canTypes = useCan('hrm.leave_type.manage')
  const canContractTypes = useCan('hrm.contract_type.manage')
  const canContracts = useCan('hrm.contract.view')
  const canTimesheets = useCan('hrm.timesheet.view')
  const canCalendar = useCan('hrm.calendar.manage')
  const canPayrolls = useCan('hrm.payroll.view')
  const canParams = useCan('hrm.legal_param.manage')
  const canRules = useCan(['hrm.approval.manage', 'core.approval.manage'])
  return (
    <Routes>
      <Route path="employees" element={canView ? <EmployeesPage /> : <ForbiddenPage />} />
      <Route path="employees/new" element={canView ? <NewEmployeePage /> : <ForbiddenPage />} />
      <Route path="employees/:id" element={canView ? <EmployeePage /> : <ForbiddenPage />} />
      <Route index element={<Navigate to="overview" replace />} />
      <Route path="overview" element={<OverviewPage />} />
      <Route path="org" element={<OrgStructurePage />} />
      <Route path="leaves" element={<LeavesPage />} />
      <Route path="leaves/new" element={<NewLeavePage />} />
      <Route path="leaves/:id" element={<LeavePage />} />
      <Route path="leave-types" element={canTypes ? <LeaveTypesPage /> : <ForbiddenPage />} />
      <Route path="overtimes" element={<OvertimesPage />} />
      <Route path="overtimes/new" element={<NewOvertimePage />} />
      <Route path="overtimes/:id" element={<OvertimePage />} />
      {/* The API decides per contract; a contract outside the user's reach answers 404. */}
      <Route path="contracts" element={canContracts ? <ContractsPage /> : <ForbiddenPage />} />
      <Route path="contracts/new" element={<NewContractPage />} />
      <Route path="contracts/:id" element={<ContractPage />} />
      <Route path="contract-types" element={canContractTypes ? <ContractTypesPage /> : <ForbiddenPage />} />
      <Route path="timesheets" element={canTimesheets ? <TimesheetsPage /> : <ForbiddenPage />} />
      <Route path="timesheets/:id" element={canTimesheets ? <TimesheetPage /> : <ForbiddenPage />} />
      <Route path="payrolls" element={canPayrolls ? <PayrollsPage /> : <ForbiddenPage />} />
      <Route path="payrolls/:id" element={canPayrolls ? <PayrollPage /> : <ForbiddenPage />} />
      <Route path="work-calendar" element={canCalendar ? <WorkCalendarPage /> : <ForbiddenPage />} />
      <Route path="legal-params" element={canParams ? <LegalParamsPage /> : <ForbiddenPage />} />
      <Route path="approval-rules/*" element={canRules ? <ApprovalRules product="hrm" basePath="/hrm/approval-rules" /> : <ForbiddenPage />} />
      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  )
}
