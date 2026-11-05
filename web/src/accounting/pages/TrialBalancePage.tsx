import { IconScale } from '@tabler/icons-react'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type TrialRow } from '@/shared/api/accounting'
import { formatNumber } from '@/shared/i18n'
import { type Column } from '@/shared/ui/DataTable'
import { DateFilter, LegalEntityFilter } from '@/shared/ui/form'
import { ListPage } from '@/shared/ui/page'
import { EmptyState } from '@/shared/ui/states'
import { splitBalance } from '../balance'
import { asList, useReportParams } from '../hooks/useAccounting'
import { accountingKeys } from '../keys'

// The totals row carries only what the server totals: the period's debits and credits.
type Row = TrialRow & { total?: boolean }

// TrialBalancePage: per account, the opening balance, the period's debits and credits and the
// closing balance over a range; parents roll up their children.
export function TrialBalancePage() {
  const { t } = useTranslation()
  const { p, put, entities } = useReportParams()
  const range = { legal_entity_id: p.legal_entity_id, from: p.from, to: p.to }
  const tb = useQuery({
    queryKey: accountingKeys.trialBalance(range),
    queryFn: () => unwrap(api.GET('/accounting/trial-balance', { params: { query: range } })),
    enabled: !!range.legal_entity_id,
    placeholderData: keepPreviousData,
  })

  const balance = (of: 'opening' | 'closing', side: 'debit' | 'credit') => (r: Row) => (r.total ? '' : formatNumber(splitBalance(r[of])[side]))
  const opening = t('accounting.trial.opening')
  const movement = t('accounting.trial.movement')
  const closing = t('accounting.trial.closing')
  const columns: Column<Row>[] = [
    { key: 'code', header: t('accounting.trial.code'), role: 'title', kind: 'code', render: (r) => r.code },
    { key: 'name', header: t('accounting.trial.name'), role: 'meta', render: (r) => r.name },
    { key: 'opening_debit', header: t('accounting.side.debit_money'), band: opening, role: 'hidden', kind: 'number', render: balance('opening', 'debit') },
    { key: 'opening_credit', header: t('accounting.side.credit_money'), band: opening, role: 'hidden', kind: 'number', render: balance('opening', 'credit') },
    { key: 'debit', header: t('accounting.side.debit_money'), band: movement, role: 'meta', kind: 'number', render: (r) => formatNumber(r.debit) },
    { key: 'credit', header: t('accounting.side.credit_money'), band: movement, role: 'meta', kind: 'number', render: (r) => formatNumber(r.credit) },
    { key: 'closing_debit', header: t('accounting.side.debit_money'), band: closing, role: 'meta', kind: 'number', strong: true, render: balance('closing', 'debit') },
    { key: 'closing_credit', header: t('accounting.side.credit_money'), band: closing, role: 'meta', kind: 'number', strong: true, render: balance('closing', 'credit') },
  ]
  const footer: Row | undefined = tb.data && {
    total: true,
    account_id: 0,
    code: t('accounting.trial.total'),
    name: '',
    parent_id: null,
    opening: 0,
    debit: tb.data.total_debit,
    credit: tb.data.total_credit,
    closing: 0,
  }

  return (
    <ListPage
      title={t('accounting.trial.title')}
      description={t('accounting.trial.description')}
      filters={
        <>
          <LegalEntityFilter label={t('accounting.common.legal_entity')} options={entities.options} value={p.legal_entity_id || undefined} onChange={put('legal_entity_id')} />
          <DateFilter label={t('accounting.common.from')} value={p.from} onChange={put('from')} clearable={false} />
          <DateFilter label={t('accounting.common.to')} value={p.to} onChange={put('to')} clearable={false} />
        </>
      }
      list={
        range.legal_entity_id
          ? {
              query: asList(tb, tb.data?.rows ?? (tb.data && [])),
              table: { label: t('accounting.trial.title'), columns, rowKey: (r) => r.account_id, footer },
              empty: { title: t('accounting.trial.empty'), description: t('accounting.trial.empty_description') },
              noResults: t('accounting.common.no_results'),
            }
          : undefined
      }
    >
      {!range.legal_entity_id && !entities.isPending && <EmptyState icon={IconScale} title={t('accounting.common.no_legal_entity')} />}
    </ListPage>
  )
}
