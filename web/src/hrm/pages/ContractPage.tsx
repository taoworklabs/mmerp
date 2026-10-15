import { Anchor, Group } from '@mantine/core'
import { IconPrinter } from '@tabler/icons-react'
import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, useNavigate, useParams } from 'react-router'
import { api, unwrap } from '@/shared/api/hrm'
import { ActivityFeed, ApprovalPanel, DocumentActions, DocumentStatus, useAttachmentSection } from '@/shared/document'
import { ExportButton } from '@/shared/jobs'
import { DocumentPage } from '@/shared/ui/page'
import { ContractForm } from '../components/ContractForm'
import { loaded } from '@/shared/ui/loaded'
import { useContract } from '../hooks/useContract'
import { contractDocType, hrmKeys } from '../keys'

export function ContractPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const id = Number(useParams().id)
  const contract = useContract(id)
  const attachments = useAttachmentSection(contractDocType, id)
  const [error, setError] = useState<string | null>(null)

  const state = loaded(contract, t, { to: '/hrm/contracts', label: t('hrm.contracts.back') })
  if (!state.ok) return state.fallback
  const c = state.data
  const employeePath = `/hrm/employees/${c.employee_id}`
  return (
    <DocumentPage
      title={c.number}
      breadcrumbs={[{ label: t('hrm.contracts.title'), to: '/hrm/contracts' }]}
      description={
        <Group gap="xs" component="span">
          <Anchor component={Link} to={employeePath} size="sm">
            {c.employee_name}
          </Anchor>
          <span aria-hidden>·</span>
          <span>{c.parent_number ? t('hrm.contract.appendix_of', { number: c.parent_number }) : c.contract_type_name}</span>
          <DocumentStatus docType={contractDocType} status={c.status} />
        </Group>
      }
      error={error}
      actions={
        c.allowed_actions.length > 0 && (
          <Group gap="xs">
            {c.allowed_actions.includes('print') && (
              <ExportButton kind={`printing.${contractDocType}`} params={{ id: c.id }} label={t('hrm.contract.print')} icon={IconPrinter} />
            )}
            <DocumentActions
              docType={contractDocType}
              id={c.id}
              version={c.version}
              number={c.number}
              actions={c.allowed_actions}
              onError={setError}
              onDelete={async (version) => {
                await unwrap(api.DELETE('/hrm/contracts/{id}', { params: { path: { id: c.id }, query: { version } } }))
                await qc.invalidateQueries({ queryKey: hrmKeys.contracts.all() })
                navigate('/hrm/contracts', { replace: true })
              }}
            />
          </Group>
        )
      }
      approval={<ApprovalPanel docType={contractDocType} id={c.id} />}
      sections={[attachments]}
      activity={<ActivityFeed docType={contractDocType} id={c.id} />}
    >
      {/* Keyed by version: a reload after a conflict starts from the saved data. */}
      <ContractForm key={c.version} contract={c} />
    </DocumentPage>
  )
}
