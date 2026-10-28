import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { api, unwrap } from '@/shared/api/hrm'
import { formatMoney, formatMonth } from '@/shared/i18n'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { Page } from '@/shared/ui/page'
import { QueryView } from '@/shared/ui/QueryView'
import { EmptyState } from '@/shared/ui/states'
import { hrmKeys } from '../keys'

type Row = { payroll_id: number; period_start: string; net: number }

// PayslipsPage lists the user's payslips, newest month first; a row opens the payslip.
export function PayslipsPage() {
  const { t } = useTranslation()
  const list = useQuery({ queryKey: hrmKeys.me.payslips(), queryFn: async () => (await unwrap(api.GET('/hrm/me/payslips'))) ?? [] })
  const columns: Column<Row>[] = [
    { key: 'month', header: t('hrm.me.payslips.month'), role: 'title', render: (r) => formatMonth(r.period_start.slice(0, 7)) },
    { key: 'net', header: t('hrm.me.payslips.net'), role: 'meta', kind: 'number', render: (r) => formatMoney(r.net) },
  ]
  return (
    <Page title={t('hrm.me.payslips.title')} description={t('hrm.me.payslips.description')}>
      <QueryView
        query={list}
        select={(rows) => [...rows].sort((a, b) => b.period_start.localeCompare(a.period_start))}
        empty={<EmptyState title={t('hrm.me.payslips.empty')} description={t('hrm.me.payslips.empty_description')} />}
      >
        {(rows) => <DataTable label={t('hrm.me.payslips.title')} columns={columns} rows={rows} rowKey={(r) => r.payroll_id} href={(r) => `/me/payslips/${r.payroll_id}`} />}
      </QueryView>
    </Page>
  )
}
