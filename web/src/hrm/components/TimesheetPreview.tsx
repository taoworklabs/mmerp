import { Stack } from '@mantine/core'
import { useTranslation } from 'react-i18next'
import { DocumentStatus } from '@/shared/document'
import { errorText, formatDecimal, formatMonth, formatNumber } from '@/shared/i18n'
import { FieldList } from '@/shared/ui/FieldList'
import { ContentSkeleton, ErrorState } from '@/shared/ui/states'
import { useTimesheet } from '../hooks/useTimesheet'
import { timesheetDocType } from '../keys'

// TimesheetPreview is the read-only summary of a timesheet in the approval inbox; its strings
// live in the meta translations, since it loads outside the hrm screens.
export default function TimesheetPreview({ id }: { id: number }) {
  const { t } = useTranslation()
  const timesheet = useTimesheet(id)
  if (timesheet.isPending) return <ContentSkeleton />
  if (timesheet.isError) return <ErrorState message={errorText(t, timesheet.error)} onRetry={() => void timesheet.refetch()} />
  const ts = timesheet.data
  // Half days, so the sum stays exact.
  const halves = ts.lines.reduce((sum, l) => sum + (l.days === '1' ? 2 : 1), 0)
  const rows: [string, string][] = [
    [t('hrm.timesheet.org_unit'), ts.org_unit_name],
    [t('hrm.timesheet.month'), formatMonth(ts.period_start.slice(0, 7))],
    [t('hrm.timesheet.employees'), formatNumber(ts.employees.length)],
    [t('hrm.timesheet.days_worked'), formatDecimal(String(halves / 2))],
  ]
  return (
    <Stack gap="sm">
      <DocumentStatus docType={timesheetDocType} status={ts.status} />
      <FieldList rows={rows} />
    </Stack>
  )
}
