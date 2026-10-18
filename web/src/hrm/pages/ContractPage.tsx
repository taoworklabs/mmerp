import { Group } from '@mantine/core'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router'
import { ActivityFeed, ApprovalDecision, ApprovalPanel, DocumentActions, DocumentStatus, useAttachmentSection } from '@/shared/document'
import { DocumentPage } from '@/shared/ui/page'
import { ContractDetails } from '../components/ContractDetails'
import { EmployeeLink } from '../components/EmployeeLink'
import { loaded } from '@/shared/ui/loaded'
import { useContract } from '../hooks/useContract'
import { contractDocType } from '../keys'

export function ContractPage() {
  const { t } = useTranslation()
  const id = Number(useParams().id)
  const contract = useContract(id)
  const attachments = useAttachmentSection(contractDocType, id)
  const [error, setError] = useState<string | null>(null)

  const state = loaded(contract, t, { to: '/hrm/contracts', label: t('hrm.contracts.back') })
  if (!state.ok) return state.fallback
  const c = state.data
  return (
    <DocumentPage
      title={c.number}
      breadcrumbs={[{ label: t('hrm.contracts.title'), to: '/hrm/contracts' }]}
      description={
        <Group gap="xs" component="span">
          <EmployeeLink id={c.employee_id}>{c.employee_name}</EmployeeLink>
          <span aria-hidden>·</span>
          <span>{c.parent_number ? t('hrm.contract.appendix_of', { number: c.parent_number }) : c.contract_type_name}</span>
          <DocumentStatus docType={contractDocType} status={c.status} />
        </Group>
      }
      error={error}
      actions={
        <Group gap="xs" justify="flex-end">
          <ApprovalDecision docType={contractDocType} id={c.id} onError={setError} />
          {c.allowed_actions.length > 0 && (
            <DocumentActions
              docType={contractDocType}
              id={c.id}
              version={c.version}
              number={c.number}
              actions={c.allowed_actions}
              onError={setError}
              listPath="/hrm/contracts"
              printLabel={t('hrm.contract.print')}
            />
          )}
        </Group>
      }
      approval={<ApprovalPanel docType={contractDocType} id={c.id} subject={c.employee_name} decideElsewhere />}
      sections={[attachments]}
      activity={<ActivityFeed docType={contractDocType} id={c.id} />}
    >
      <ContractDetails contract={c} />
    </DocumentPage>
  )
}
