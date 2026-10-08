import { Stack } from '@mantine/core'
import { useTranslation } from 'react-i18next'
import { DocumentStatus } from '@/shared/document'
import { errorText, formatDate, formatDecimal } from '@/shared/i18n'
import { FieldList } from '@/shared/ui/FieldList'
import { ContentSkeleton, ErrorState } from '@/shared/ui/states'
import { useLeave } from '../hooks/useLeave'
import { leaveDocType } from '../keys'

// LeavePreview is the read-only view of a leave request in the approval inbox. It is
// loaded outside the hrm screens, so its strings live in the meta translations.
export default function LeavePreview({ id }: { id: number }) {
  const { t } = useTranslation()
  const leave = useLeave(id)
  if (leave.isPending) return <ContentSkeleton />
  if (leave.isError) return <ErrorState message={errorText(t, leave.error)} onRetry={() => void leave.refetch()} />
  const l = leave.data
  const rows: [string, string][] = [
    [t('hrm.leave.employee'), `${l.employee_code} · ${l.employee_name}`],
    [t('hrm.leave.type'), l.leave_type_name],
    [t('hrm.leave.period'), `${formatDate(l.start_date)} – ${formatDate(l.end_date)}`],
    [t('hrm.leave.days'), formatDecimal(l.days)],
    ...(l.balance !== null && l.balance !== undefined ? [[t('hrm.leave.balance'), formatDecimal(l.balance)] as [string, string]] : []),
    ...(l.reason ? [[t('hrm.leave.reason'), l.reason] as [string, string]] : []),
  ]
  return (
    <Stack gap="sm">
      <DocumentStatus docType={leaveDocType} status={l.status} />
      <FieldList rows={rows} />
    </Stack>
  )
}
