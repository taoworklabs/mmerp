import { Text } from '@mmerp/ui'
import { IconFileSpreadsheet } from '@tabler/icons-react'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type StatementLine } from '@/shared/api/accounting'
import { formatNumber } from '@/shared/i18n'
import { ExportButton } from '@/shared/jobs/ExportButton'
import { type Column } from '@/shared/ui/DataTable'
import { DateFilter, LegalEntityFilter } from '@/shared/ui/form'
import { Notice } from '@/shared/ui/Notice'
import { ListPage } from '@/shared/ui/page'
import { EmptyState } from '@/shared/ui/states'
import { asList, useReportParams } from '../hooks/useAccounting'
import { accountingKeys, type StatementFormName } from '../keys'

// StatementPage computes a financial statement from the ledger through its form: the balance
// sheet at a date against the end of the previous year, the income statement over a range against
// the same range a year earlier.
export function StatementPage({ form }: { form: StatementFormName }) {
  const { t } = useTranslation()
  const { p, put, entities } = useReportParams()
  const sheet = form === 'balance_sheet'
  const le = p.legal_entity_id
  const st = useQuery({
    queryKey: sheet ? accountingKeys.balanceSheet({ legal_entity_id: le, date: p.to }) : accountingKeys.incomeStatement({ legal_entity_id: le, from: p.from, to: p.to }),
    queryFn: () =>
      sheet
        ? unwrap(api.GET('/accounting/statements/balance-sheet', { params: { query: { legal_entity_id: le, date: p.to } } }))
        : unwrap(api.GET('/accounting/statements/income-statement', { params: { query: { legal_entity_id: le, from: p.from, to: p.to } } })),
    enabled: !!le,
    placeholderData: keepPreviousData,
  })
  const k = `accounting.statement.${form}`
  const strong = (l: StatementLine, v: string) =>
    l.bold ? (
      <Text size="sm" weight={600}>
        {v}
      </Text>
    ) : (
      v
    )
  const columns: Column<StatementLine>[] = [
    { key: 'label', header: t('accounting.statement.label'), role: 'title', render: (l) => strong(l, l.label) },
    { key: 'code', header: t('accounting.statement.code'), role: 'meta', kind: 'code', render: (l) => l.code },
    { key: 'current', header: t(`${k}.current`), role: 'meta', kind: 'number', render: (l) => strong(l, formatNumber(l.current)) },
    { key: 'previous', header: t(`${k}.previous`), role: 'meta', kind: 'number', render: (l) => strong(l, formatNumber(l.previous)) },
  ]
  const params = sheet ? { legal_entity_id: le, date: p.to } : { legal_entity_id: le, from: p.from, to: p.to }

  return (
    <ListPage
      title={t(`${k}.title`)}
      description={t(`${k}.description`)}
      action={le ? <ExportButton kind={`accounting.report.${form}`} params={params} label={t('accounting.statement.export')} icon={IconFileSpreadsheet} /> : undefined}
      filters={
        <>
          <LegalEntityFilter label={t('accounting.common.legal_entity')} options={entities.options} value={le || undefined} onChange={put('legal_entity_id')} />
          {!sheet && <DateFilter label={t('accounting.common.from')} value={p.from} onChange={put('from')} clearable={false} />}
          <DateFilter label={t(sheet ? 'accounting.statement.date' : 'accounting.common.to')} value={p.to} onChange={put('to')} clearable={false} />
        </>
      }
      list={
        le
          ? {
              query: asList(st, st.data?.lines),
              table: { label: t(`${k}.title`), columns, rowKey: (l) => l.code },
              empty: { title: t('accounting.statement.empty'), description: t('accounting.statement.empty_description') },
              noResults: t('accounting.common.no_results'),
            }
          : undefined
      }
    >
      {st.data?.not_closed && <Notice tone="warning">{t('accounting.statement.not_closed')}</Notice>}
      {!le && !entities.isPending && <EmptyState icon={IconFileSpreadsheet} title={t('accounting.common.no_legal_entity')} />}
    </ListPage>
  )
}
