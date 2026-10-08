import { Alert, Stack } from '@mantine/core'
import { IconAlertTriangle } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'
import { DocumentStatus } from '@/shared/document'
import { errorText, formatMoney, formatMonth, formatNumber } from '@/shared/i18n'
import { FieldList } from '@/shared/ui/FieldList'
import { ContentSkeleton, ErrorState } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import { usePayroll } from '../hooks/usePayroll'
import { payrollDocType } from '../keys'

// PayrollPreview is the read-only summary of a payroll in the approval inbox: department totals
// only; its strings live in the meta translations, since it loads outside the hrm screens.
export default function PayrollPreview({ id }: { id: number }) {
  const { t } = useTranslation()
  const payroll = usePayroll(id)
  if (payroll.isPending) return <ContentSkeleton />
  if (payroll.isError) return <ErrorState message={errorText(t, payroll.error)} onRetry={() => void payroll.refetch()} />
  const p = payroll.data
  const sum = (k: 'gross' | 'net' | 'cost' | 'employees') => p.totals.reduce((n, x) => n + x[k], 0)
  const rows: [string, string][] = [
    [t('hrm.payroll.legal_entity'), p.legal_entity_name],
    [t('hrm.payroll.month'), formatMonth(p.period_start.slice(0, 7))],
    [t('hrm.payroll.employees'), formatNumber(sum('employees'))],
    [t('hrm.payroll.gross'), formatMoney(sum('gross'))],
    [t('hrm.payroll.net'), formatMoney(sum('net'))],
    [t('hrm.payroll.cost'), formatMoney(sum('cost'))],
  ]
  return (
    <Stack gap="sm">
      <DocumentStatus docType={payrollDocType} status={p.status} />
      {p.sources_changed && (
        <Alert color="warning" icon={<IconAlertTriangle {...icon.button} />}>
          {t('hrm.payroll.sources_changed')}
        </Alert>
      )}
      <FieldList rows={rows} />
    </Stack>
  )
}
