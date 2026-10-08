import { Stack } from '@mantine/core'
import { useTranslation } from 'react-i18next'
import { DocumentStatus } from '@/shared/document'
import { errorText, formatDate, formatDecimal } from '@/shared/i18n'
import { FieldList } from '@/shared/ui/FieldList'
import { ContentSkeleton, ErrorState } from '@/shared/ui/states'
import { useOvertime } from '../hooks/useOvertime'
import { overtimeDocType } from '../keys'

// OvertimePreview is the read-only view of an overtime request in the approval inbox. It is
// loaded outside the hrm screens, so its strings live in the meta translations.
export default function OvertimePreview({ id }: { id: number }) {
  const { t } = useTranslation()
  const overtime = useOvertime(id)
  if (overtime.isPending) return <ContentSkeleton />
  if (overtime.isError) return <ErrorState message={errorText(t, overtime.error)} onRetry={() => void overtime.refetch()} />
  const o = overtime.data
  const rows: [string, string][] = [
    [t('hrm.overtime.employee'), `${o.employee_code} · ${o.employee_name}`],
    [t('hrm.overtime.date'), formatDate(o.date)],
    [t('hrm.overtime.day_kind'), t(`hrm.overtime.day_kind.${o.day_kind}`)],
    [t('hrm.overtime.day_hours'), formatDecimal(o.day_hours)],
    [t('hrm.overtime.night_hours'), formatDecimal(o.night_hours)],
    ...(o.reason ? [[t('hrm.overtime.reason'), o.reason] as [string, string]] : []),
  ]
  return (
    <Stack gap="sm">
      <DocumentStatus docType={overtimeDocType} status={o.status} />
      <FieldList rows={rows} />
    </Stack>
  )
}
