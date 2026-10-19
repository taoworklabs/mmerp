import { useTranslation } from 'react-i18next'
import { DocumentPreview } from '@/shared/document'
import { formatDate, formatDecimal } from '@/shared/i18n'
import { useLeave } from '../hooks/useLeave'
import { leaveDocType } from '../keys'

// LeavePreview is the read-only view of a leave request in the approval inbox. It is
// loaded outside the hrm screens, so its strings live in the meta translations.
export default function LeavePreview({ id }: { id: number }) {
  const { t } = useTranslation()
  return (
    <DocumentPreview
      docType={leaveDocType}
      query={useLeave(id)}
      rows={(l) => [
        [t('hrm.leave.employee'), `${l.employee_code} · ${l.employee_name}`],
        [t('hrm.leave.type'), l.leave_type_name],
        [t('hrm.leave.period'), `${formatDate(l.start_date)} – ${formatDate(l.end_date)}`],
        [t('hrm.leave.days'), formatDecimal(l.days)],
        l.balance !== null && l.balance !== undefined && [t('hrm.leave.balance'), formatDecimal(l.balance)],
        l.reason && [t('hrm.leave.reason'), l.reason],
      ]}
    />
  )
}
