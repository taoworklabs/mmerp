import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router'
import { DocumentScreen } from '@/shared/document'
import { ContractDetails } from '../components/ContractDetails'
import { EmployeeLink } from '../components/EmployeeLink'
import { loaded } from '@/shared/ui/loaded'
import { useContract } from '../hooks/useContract'
import { contractDocType } from '../keys'

export function ContractPage() {
  const { t } = useTranslation()
  const id = Number(useParams().id)
  const contract = useContract(id)

  const state = loaded(contract, t, { to: '/hrm/contracts', label: t('hrm.contracts.back') })
  if (!state.ok) return state.fallback
  const c = state.data
  return (
    <DocumentScreen
      docType={contractDocType}
      doc={c}
      list={{ label: t('hrm.contracts.title'), to: '/hrm/contracts' }}
      facts={[
        <EmployeeLink key="employee" id={c.employee_id}>
          {c.employee_name}
        </EmployeeLink>,
        c.parent_number ? t('hrm.contract.appendix_of', { number: c.parent_number }) : c.contract_type_name,
      ]}
      subject={c.employee_name}
      attachments
      printLabel={t('hrm.contract.print')}
    >
      <ContractDetails contract={c} />
    </DocumentScreen>
  )
}
