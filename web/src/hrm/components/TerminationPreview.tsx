import { useTranslation } from 'react-i18next'
import { DocumentPreview } from '@/shared/document'
import { formatDate, formatMoney } from '@/shared/i18n'
import { useTermination } from '../hooks/useTermination'
import { terminationDocType } from '../keys'

// TerminationPreview is the read-only view of a termination in the approval inbox. It is
// loaded outside the hrm screens, so its strings live in the meta translations.
export default function TerminationPreview({ id }: { id: number }) {
  const { t } = useTranslation()
  return (
    <DocumentPreview
      docType={terminationDocType}
      query={useTermination(id)}
      rows={(x) => [
        [t('hrm.termination.employee'), `${x.employee_code} · ${x.employee_name}`],
        [t('hrm.termination.date'), formatDate(x.termination_date)],
        [t('hrm.termination.reason'), t(`hrm.termination.reason.${x.reason}`)],
        [t('hrm.termination.allowance_kind'), t(`hrm.termination.allowance.${x.allowance.kind}`)],
        // The amount comes only with the salary permission.
        x.allowance.amount != null && [t('hrm.termination.amount'), formatMoney(x.allowance.amount)],
      ]}
    />
  )
}
