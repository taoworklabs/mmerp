import { Group } from '@mantine/core'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useParams } from 'react-router'
import { api, unwrap } from '@/shared/api/hrm'
import { ActivityFeed, ApprovalDecision, ApprovalPanel, DocumentActions, DocumentStatus, useAttachmentSection } from '@/shared/document'
import { formatDate } from '@/shared/i18n'
import { DocumentPage } from '@/shared/ui/page'
import { OvertimeDetails } from '../components/OvertimeDetails'
import { EmployeeLink } from '../components/EmployeeLink'
import { loaded } from '@/shared/ui/loaded'
import { useOvertime } from '../hooks/useOvertime'
import { overtimeDocType } from '../keys'

export function OvertimePage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const id = Number(useParams().id)
  const attachments = useAttachmentSection(overtimeDocType, id)
  const overtime = useOvertime(id)
  const [error, setError] = useState<string | null>(null)

  const state = loaded(overtime, t, { to: '/hrm/overtimes', label: t('hrm.overtimes.back') })
  if (!state.ok) return state.fallback
  const o = state.data
  return (
    <DocumentPage
      title={o.number}
      breadcrumbs={[{ label: t('hrm.overtimes.title'), to: '/hrm/overtimes' }]}
      description={
        <Group gap="xs" component="span">
          <EmployeeLink id={o.employee_id}>{o.employee_name}</EmployeeLink>
          <span aria-hidden>·</span>
          <span>{formatDate(o.date)}</span>
          <DocumentStatus docType={overtimeDocType} status={o.status} />
        </Group>
      }
      error={error}
      actions={
        <Group gap="xs" justify="flex-end">
          <ApprovalDecision docType={overtimeDocType} id={o.id} onError={setError} />
          {o.allowed_actions.length > 0 && (
            <DocumentActions
              docType={overtimeDocType}
              id={o.id}
              version={o.version}
              number={o.number}
              actions={o.allowed_actions}
              onError={setError}
              onDelete={async (version) => {
                await unwrap(api.DELETE('/hrm/overtimes/{id}', { params: { path: { id: o.id }, query: { version } } }))
                navigate('/hrm/overtimes', { replace: true })
              }}
            />
          )}
        </Group>
      }
      approval={<ApprovalPanel docType={overtimeDocType} id={o.id} subject={o.employee_name} decideElsewhere />}
      sections={[attachments]}
      activity={<ActivityFeed docType={overtimeDocType} id={o.id} />}
    >
      <OvertimeDetails overtime={o} />
    </DocumentPage>
  )
}
