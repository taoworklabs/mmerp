import { useTranslation } from 'react-i18next'
import { DocumentPreview, type PreviewRow } from '@/shared/document'
import { formatDate, formatMoney } from '@/shared/i18n'
import { useContract } from '../hooks/useContract'
import { contractDocType } from '../keys'

// ContractPreview is the read-only view of a contract in the approval inbox. It is
// loaded outside the hrm screens, so its strings live in the meta translations.
export default function ContractPreview({ id }: { id: number }) {
  const { t } = useTranslation()
  return (
    <DocumentPreview
      docType={contractDocType}
      query={useContract(id)}
      rows={(c) => [
        [t('hrm.contract.employee'), `${c.employee_code} · ${c.employee_name}`],
        [t('hrm.contract.kind'), c.parent_number ? `${c.contract_type_name} · ${t('hrm.contract.appendix_of', { number: c.parent_number })}` : c.contract_type_name],
        [t('hrm.contract.start_date'), formatDate(c.start_date)],
        !c.parent_number && [t('hrm.contract.end_date'), c.end_date ? formatDate(c.end_date) : t('hrm.contract.no_end')],
        // Terms come only with the salary permission.
        c.terms && [t('hrm.contract.salary'), formatMoney(c.terms.salary)],
        ...(c.terms?.lines.map((l): PreviewRow => [l.name, formatMoney(l.amount)]) ?? []),
      ]}
    />
  )
}
