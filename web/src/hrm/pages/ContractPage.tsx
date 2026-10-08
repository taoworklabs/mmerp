import { Group } from '@mantine/core'
import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate, useParams } from 'react-router'
import { api, unwrap } from '@/shared/api/hrm'
import { ApprovalPanel, DocumentActions, DocumentHistory, DocumentStatus, useAttachmentSection, useDiscussionSection } from '@/shared/document'
import { DocumentPage } from '@/shared/ui/page'
import { ContractForm } from '../components/ContractForm'
import { loaded } from '../components/loaded'
import { useContract } from '../hooks/useContract'
import { contractDocType, hrmKeys } from '../keys'

export function ContractPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const id = Number(useParams().id)
  const contract = useContract(id)
  const attachments = useAttachmentSection(contractDocType, id)
  const discussion = useDiscussionSection(contractDocType, id)
  const [error, setError] = useState<string | null>(null)

  const state = loaded(contract, t, { to: '/hrm/employees', label: t('hrm.employees.back') })
  if (!state.ok) return state.fallback
  const c = state.data
  const employeePath = `/hrm/employees/${c.employee_id}`
  return (
    <DocumentPage
      title={c.number}
      breadcrumbs={[
        { label: t('hrm.employees.title'), to: '/hrm/employees' },
        { label: c.employee_name, to: employeePath },
      ]}
      description={
        <Group gap="xs" component="span">
          <span>{c.employee_name}</span>
          <span aria-hidden>·</span>
          <span>{c.parent_number ? t('hrm.contract.appendix_of', { number: c.parent_number }) : c.contract_type_name}</span>
          <DocumentStatus docType={contractDocType} status={c.status} />
        </Group>
      }
      error={error}
      actions={
        c.allowed_actions.length > 0 && (
          <DocumentActions
            docType={contractDocType}
            id={c.id}
            version={c.version}
            number={c.number}
            actions={c.allowed_actions}
            onError={setError}
            onDelete={async (version) => {
              await unwrap(api.DELETE('/hrm/contracts/{id}', { params: { path: { id: c.id }, query: { version } } }))
              await qc.invalidateQueries({ queryKey: hrmKeys.contracts.employee(c.employee_id) })
              navigate(`${employeePath}?tab=contracts`, { replace: true })
            }}
          />
        )
      }
      approval={<ApprovalPanel docType={contractDocType} id={c.id} />}
      sections={[attachments, discussion]}
      history={<DocumentHistory docType={contractDocType} id={c.id} />}
    >
      {/* Keyed by version: a reload after a conflict starts from the saved data. */}
      <ContractForm key={c.version} contract={c} />
    </DocumentPage>
  )
}
