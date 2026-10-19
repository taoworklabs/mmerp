import { useTranslation } from 'react-i18next'
import { DocumentPreview } from '@/shared/document'
import { formatDate, formatDecimal } from '@/shared/i18n'
import { useOvertime } from '../hooks/useOvertime'
import { overtimeDocType } from '../keys'

// OvertimePreview is the read-only view of an overtime request in the approval inbox. It is
// loaded outside the hrm screens, so its strings live in the meta translations.
export default function OvertimePreview({ id }: { id: number }) {
  const { t } = useTranslation()
  return (
    <DocumentPreview
      docType={overtimeDocType}
      query={useOvertime(id)}
      rows={(o) => [
        [t('hrm.overtime.employee'), `${o.employee_code} · ${o.employee_name}`],
        [t('hrm.overtime.date'), formatDate(o.date)],
        [t('hrm.overtime.day_kind'), t(`hrm.overtime.day_kind.${o.day_kind}`)],
        [t('hrm.overtime.day_hours'), formatDecimal(o.day_hours)],
        [t('hrm.overtime.night_hours'), formatDecimal(o.night_hours)],
        o.reason && [t('hrm.overtime.reason'), o.reason],
      ]}
    />
  )
}
