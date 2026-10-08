import { Group } from '@mantine/core'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useParams } from 'react-router'
import { api, unwrap } from '@/shared/api/hrm'
import { ApprovalPanel, DocumentActions, DocumentHistory, DocumentStatus, useDiscussionSection } from '@/shared/document'
import { formatDate } from '@/shared/i18n'
import { DocumentPage } from '@/shared/ui/page'
import { OvertimeForm } from '../components/OvertimeForm'
import { loaded } from '../components/loaded'
import { useOvertime } from '../hooks/useOvertime'
import { overtimeDocType } from '../keys'

export function OvertimePage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const id = Number(useParams().id)
  const discussion = useDiscussionSection(overtimeDocType, id)
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
          <span>{o.employee_name}</span>
          <span aria-hidden>·</span>
          <span>{formatDate(o.date)}</span>
          <DocumentStatus docType={overtimeDocType} status={o.status} />
        </Group>
      }
      error={error}
      actions={
        o.allowed_actions.length > 0 && (
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
        )
      }
      approval={<ApprovalPanel docType={overtimeDocType} id={o.id} />}
      sections={[discussion]}
      history={<DocumentHistory docType={overtimeDocType} id={o.id} />}
    >
      {/* Keyed by version: a reload after a conflict starts from the saved data. */}
      <OvertimeForm key={o.version} overtime={o} />
    </DocumentPage>
  )
}
