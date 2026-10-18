import { Group } from '@mantine/core'
import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router'
import { api, unwrap } from '@/shared/api/hrm'
import { ActivityFeed, ApprovalDecision, ApprovalPanel, DocumentActions, DocumentStatus, useAttachmentSection } from '@/shared/document'
import { DocumentPage } from '@/shared/ui/page'
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
  const attachments = useAttachmentSection(leaveDocType, id)
  const [error, setError] = useState<string | null>(null)

  const state = loaded(leave, t, { to: '/hrm/leaves', label: t('hrm.leaves.back') })
  if (!state.ok) return state.fallback
  const l = state.data
  const deducts = types.data?.find((x) => x.id === l.leave_type_id)?.deducts_balance
  return (
    <DocumentPage
      title={l.number}
      breadcrumbs={[{ label: t('hrm.leaves.title'), to: '/hrm/leaves' }]}
      description={
        <Group gap="xs" component="span">
          <EmployeeLink id={l.employee_id}>{l.employee_name}</EmployeeLink>
          {employee.data && (
            <>
              <span aria-hidden>·</span>
              <span>{employee.data.org_unit_name}</span>
            </>
          )}
          <span aria-hidden>·</span>
          <span>{l.leave_type_name}</span>
          <DocumentStatus docType={leaveDocType} status={l.status} />
        </Group>
      }
      error={error}
      actions={
        <Group gap="xs" justify="flex-end">
          <ApprovalDecision docType={leaveDocType} id={l.id} onError={setError} />
          {l.allowed_actions.length > 0 && (
            <DocumentActions
              docType={leaveDocType}
              id={l.id}
              version={l.version}
              number={l.number}
              actions={l.allowed_actions}
              onError={setError}
              listPath="/hrm/leaves"
            />
          )}
        </Group>
      }
      approval={<ApprovalPanel docType={leaveDocType} id={l.id} subject={l.employee_name} decideElsewhere />}
      sections={[attachments]}
      activity={<ActivityFeed docType={leaveDocType} id={l.id} />}
    >
      <LeaveDetails leave={l} deducts={deducts} />
      <LeaveImpact overlapping={overlapping.data} />
    </DocumentPage>
  )
}
