import { Group, Text } from '@mantine/core'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useParams } from 'react-router'
import { api, unwrap } from '@/shared/api/hrm'
import { ApprovalPanel, DocumentActions, DocumentHistory, DocumentStatus, useAttachmentSection, useDiscussionSection } from '@/shared/document'
import { formatDecimal } from '@/shared/i18n'
import { DocumentPage } from '@/shared/ui/page'
import { LeaveForm } from '../components/LeaveForm'
import { loaded } from '../components/loaded'
import { useLeave } from '../hooks/useLeave'
import { leaveDocType } from '../keys'

export function LeavePage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const id = Number(useParams().id)
  const leave = useLeave(id)
  const attachments = useAttachmentSection(leaveDocType, id)
  const discussion = useDiscussionSection(leaveDocType, id)
  const [error, setError] = useState<string | null>(null)

  const state = loaded(leave, t, { to: '/hrm/leaves', label: t('hrm.leaves.back') })
  if (!state.ok) return state.fallback
  const l = state.data
  const year = l.start_date.slice(0, 4)
  return (
    <DocumentPage
      title={l.number}
      breadcrumbs={[{ label: t('hrm.leaves.title'), to: '/hrm/leaves' }]}
      description={
        <Group gap="xs" component="span">
          <span>{l.employee_name}</span>
          <span aria-hidden>·</span>
          <span>{l.leave_type_name}</span>
          <DocumentStatus docType={leaveDocType} status={l.status} />
        </Group>
      }
      error={error}
      actions={
        l.allowed_actions.length > 0 && (
          <DocumentActions
            docType={leaveDocType}
            id={l.id}
            version={l.version}
            number={l.number}
            actions={l.allowed_actions}
            onError={setError}
            onDelete={async (version) => {
              await unwrap(api.DELETE('/hrm/leaves/{id}', { params: { path: { id: l.id }, query: { version } } }))
              navigate('/hrm/leaves', { replace: true })
            }}
          />
        )
      }
      approval={<ApprovalPanel docType={leaveDocType} id={l.id} />}
      sections={[attachments, discussion]}
      history={<DocumentHistory docType={leaveDocType} id={l.id} />}
    >
      {l.balance !== null && l.balance !== undefined && (
        <Text size="sm" c="dimmed" mb="md">
          {t('hrm.leave.balance_line', { year, days: formatDecimal(l.balance) })}
        </Text>
      )}
      {/* Keyed by version: a reload after a conflict starts from the saved data. */}
      <LeaveForm key={l.version} leave={l} />
    </DocumentPage>
  )
}
