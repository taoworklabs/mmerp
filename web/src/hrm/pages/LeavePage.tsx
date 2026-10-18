import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router'
import { api, unwrap } from '@/shared/api/hrm'
import { DocumentScreen } from '@/shared/document'
import { EmployeeLink } from '../components/EmployeeLink'
import { LeaveDetails, LeaveImpact } from '../components/LeaveDetails'
import { loaded } from '@/shared/ui/loaded'
import { useLeave, useLeaveTypes, useOverlappingLeaves } from '../hooks/useLeave'
import { hrmKeys, leaveDocType } from '../keys'

export function LeavePage() {
  const { t } = useTranslation()
  const id = Number(useParams().id)
  const leave = useLeave(id)
  const types = useLeaveTypes()
  const employeeId = leave.data?.employee_id
  // For the department; a user who may see the request but not the profile goes without.
  const employee = useQuery({
    queryKey: hrmKeys.employees.detail(employeeId ?? 0),
    queryFn: () => unwrap(api.GET('/hrm/employees/{id}', { params: { path: { id: employeeId! } } })),
    enabled: !!employeeId,
    retry: false,
  })
  const overlapping = useOverlappingLeaves(leave.data, employee.data?.org_unit_id)

  const state = loaded(leave, t, { to: '/hrm/leaves', label: t('hrm.leaves.back') })
  if (!state.ok) return state.fallback
  const l = state.data
  const deducts = types.data?.find((x) => x.id === l.leave_type_id)?.deducts_balance
  return (
    <DocumentScreen
      docType={leaveDocType}
      doc={l}
      list={{ label: t('hrm.leaves.title'), to: '/hrm/leaves' }}
      facts={[
        <EmployeeLink key="employee" id={l.employee_id}>
          {l.employee_name}
        </EmployeeLink>,
        employee.data?.org_unit_name,
        l.leave_type_name,
      ]}
      subject={l.employee_name}
      attachments
    >
      <LeaveDetails leave={l} deducts={deducts} />
      <LeaveImpact overlapping={overlapping.data} />
    </DocumentScreen>
  )
}
