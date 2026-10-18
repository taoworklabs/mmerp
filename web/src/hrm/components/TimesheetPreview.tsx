import { useTranslation } from 'react-i18next'
import { DocumentPreview } from '@/shared/document'
import { formatDecimal, formatMonth, formatNumber } from '@/shared/i18n'
import { useTimesheet } from '../hooks/useTimesheet'
import { timesheetDocType } from '../keys'

// TimesheetPreview is the read-only summary of a timesheet in the approval inbox; its strings
// live in the meta translations, since it loads outside the hrm screens.
export default function TimesheetPreview({ id }: { id: number }) {
  const { t } = useTranslation()
  return (
    <DocumentPreview
      docType={timesheetDocType}
      query={useTimesheet(id)}
      rows={(ts) => {
        // Half days, so the sum stays exact.
        const halves = ts.lines.reduce((sum, l) => sum + (l.days === '1' ? 2 : 1), 0)
        return [
          [t('hrm.timesheet.org_unit'), ts.org_unit_name],
          [t('hrm.timesheet.month'), formatMonth(ts.period_start.slice(0, 7))],
          [t('hrm.timesheet.employees'), formatNumber(ts.employees.length)],
          [t('hrm.timesheet.days_worked'), formatDecimal(String(halves / 2))],
        ]
      }}
    />
  )
}
