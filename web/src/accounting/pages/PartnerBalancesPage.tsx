import { Anchor } from '@mmerp/ui'
import { IconListNumbers, IconUsers } from '@tabler/icons-react'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { api, unwrap, type PartnerBalanceRow } from '@/shared/api/accounting'
import { formatNumber } from '@/shared/i18n'
import { type Column } from '@/shared/ui/DataTable'
import { DateFilter, FilterChip, LegalEntityFilter } from '@/shared/ui/form'
import { ListPage } from '@/shared/ui/page'
import { EmptyState } from '@/shared/ui/states'
import { rowBalance } from '../balance'
import { accountOptions, asList, useAccounts, useReportParams } from '../hooks/useAccounting'
import { accountingKeys } from '../keys'

// The totals row carries only the period's debits and credits; balances on both sides do not add up.
type Row = PartnerBalanceRow & { total?: boolean }

// PartnerBalancesPage: per partner and tracked account, the opening balance, the period's debits
// and credits and the closing balance; a partner opens their ledger over the same range.
export function PartnerBalancesPage() {
  const { t } = useTranslation()
  const accounts = useAccounts()
  const { p, put, entities } = useReportParams()
  const range = { legal_entity_id: p.legal_entity_id, from: p.from, to: p.to, ...(p.account_id ? { account_id: p.account_id } : {}) }
  const balances = useQuery({
    queryKey: accountingKeys.partnerBalances(range),
    queryFn: () => unwrap(api.GET('/accounting/partner-balances', { params: { query: range } })),
    enabled: !!range.legal_entity_id,
    placeholderData: keepPreviousData,
  })

  const ledgerOf = (r: Row) =>
    `/accounting/partner-ledger?${new URLSearchParams({ legal_entity_id: String(p.legal_entity_id), partner_id: String(r.partner_id), account_id: String(r.account_id), from: p.from, to: p.to })}`
  const balance = (of: 'opening' | 'closing', side: 'debit' | 'credit') => (r: Row) => {
    const n = rowBalance(r, of, side)
    return n === null ? '' : formatNumber(n)
  }
  const opening = t('accounting.trial.opening')
  const movement = t('accounting.trial.movement')
  const closing = t('accounting.trial.closing')
  const columns: Column<Row>[] = [
    {
      key: 'partner',
      header: t('accounting.partners.partner'),
      role: 'title',
      render: (r) => (r.total ? r.partner_name : <Anchor render={<Link to={ledgerOf(r)} />}>{`${r.partner_code} · ${r.partner_name}`}</Anchor>),
    },
    { key: 'account', header: t('accounting.partners.account'), role: 'meta', kind: 'code', render: (r) => r.account_code },
    { key: 'opening_debit', header: t('accounting.side.debit_money'), band: opening, role: 'hidden', kind: 'number', render: balance('opening', 'debit') },
    { key: 'opening_credit', header: t('accounting.side.credit_money'), band: opening, role: 'hidden', kind: 'number', render: balance('opening', 'credit') },
    { key: 'debit', header: t('accounting.side.debit_money'), band: movement, role: 'meta', kind: 'number', render: (r) => formatNumber(r.debit) },
    { key: 'credit', header: t('accounting.side.credit_money'), band: movement, role: 'meta', kind: 'number', render: (r) => formatNumber(r.credit) },
    { key: 'closing_debit', header: t('accounting.side.debit_money'), band: closing, role: 'meta', kind: 'number', strong: true, render: balance('closing', 'debit') },
    { key: 'closing_credit', header: t('accounting.side.credit_money'), band: closing, role: 'meta', kind: 'number', strong: true, render: balance('closing', 'credit') },
  ]
  const d = balances.data
  const footer: Row | undefined = d && {
    total: true,
    account_id: 0,
    account_code: '',
    partner_id: 0,
    partner_code: '',
    partner_name: t('accounting.trial.total'),
    opening: d.total_opening,
    debit: d.total_debit,
    credit: d.total_credit,
    closing: d.total_closing,
  }

  return (
    <ListPage
      title={t('accounting.partners.title')}
      description={t('accounting.partners.description')}
      filters={
        <>
          <LegalEntityFilter label={t('accounting.common.legal_entity')} options={entities.options} value={p.legal_entity_id || undefined} onChange={put('legal_entity_id')} />
          <FilterChip
            label={t('accounting.partners.account')}
            icon={IconListNumbers}
            data={accountOptions(accounts.data, (a) => !!a.by_partner)}
            value={p.account_id ? String(p.account_id) : null}
            onChange={put('account_id')}
          />
          <DateFilter label={t('accounting.common.from')} value={p.from} onChange={put('from')} clearable={false} />
          <DateFilter label={t('accounting.common.to')} value={p.to} onChange={put('to')} clearable={false} />
        </>
      }
      list={
        range.legal_entity_id
          ? {
              query: asList(balances, d?.rows),
              table: { label: t('accounting.partners.title'), columns, rowKey: (r) => `${r.account_id}-${r.partner_id}`, footer },
              empty: { title: t('accounting.partners.empty'), description: t('accounting.partners.empty_description') },
              noResults: t('accounting.common.no_results'),
            }
          : undefined
      }
    >
      {!range.legal_entity_id && !entities.isPending && <EmptyState icon={IconUsers} title={t('accounting.common.no_legal_entity')} />}
    </ListPage>
  )
}
