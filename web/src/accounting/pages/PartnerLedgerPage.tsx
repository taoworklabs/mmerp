import { Anchor } from '@mmerp/ui'
import { IconListNumbers, IconUsers } from '@tabler/icons-react'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { api, unwrap, type LedgerLine } from '@/shared/api/accounting'
import { useRecordLinks } from '@/shared/area'
import { formatDate, formatMoney, formatNumber } from '@/shared/i18n'
import { type Column } from '@/shared/ui/DataTable'
import { DateFilter, FilterChip, LegalEntityFilter } from '@/shared/ui/form'
import { ListPage } from '@/shared/ui/page'
import { EmptyState } from '@/shared/ui/states'
import { kindLabel, splitBalance } from '../balance'
import { accountOptions, asList, useAccounts, useReportParams } from '../hooks/useAccounting'
import { accountingKeys } from '../keys'

type Row = LedgerLine & { key: string; edge?: 'opening' | 'closing' }

const edgeRow = (edge: 'opening' | 'closing', date: string, balance: number): Row => ({
  key: edge,
  edge,
  date,
  balance,
  entry_id: 0,
  doc_type: '',
  doc_id: 0,
  doc_number: '',
  kind: null,
  description: null,
  partner_id: null,
  partner_code: null,
  partner_name: null,
  org_unit_id: null,
  account_code: '',
  debit: 0,
  credit: 0,
})

// PartnerLedgerPage: one partner's lines on the accounts tracked by partner (or one of them)
// over a range, with running balances. The partners to pick are those with a balance in the range.
export function PartnerLedgerPage() {
  const { t } = useTranslation()
  const links = useRecordLinks()
  const accounts = useAccounts()
  const { p, put, entities } = useReportParams()
  const ready = !!p.legal_entity_id && !!p.partner_id
  const range = { legal_entity_id: p.legal_entity_id, from: p.from, to: p.to }
  const account = p.account_id ? { account_id: p.account_id } : {}

  const partners = useQuery({
    queryKey: accountingKeys.partnerBalances(range),
    queryFn: () => unwrap(api.GET('/accounting/partner-balances', { params: { query: range } })),
    enabled: !!range.legal_entity_id,
  })
  const partnerOptions = [...new Map((partners.data?.rows ?? []).map((r) => [r.partner_id, `${r.partner_code} · ${r.partner_name}`])).entries()].map(([id, label]) => ({
    value: String(id),
    label,
  }))
  const query = { ...range, ...account, partner_id: p.partner_id }
  const ledger = useQuery({
    queryKey: accountingKeys.partnerLedger(query),
    queryFn: () => unwrap(api.GET('/accounting/partner-ledger', { params: { query } })),
    enabled: ready,
    placeholderData: keepPreviousData,
  })

  const balance = (side: 'debit' | 'credit') => (r: Row) => formatNumber(splitBalance(r.balance)[side])
  const columns: Column<Row>[] = [
    { key: 'date', header: t('accounting.ledger.date'), role: 'title', kind: 'date', render: (r) => (r.edge ? t(`accounting.ledger.${r.edge}`) : formatDate(r.date)) },
    {
      key: 'document',
      header: t('accounting.ledger.document'),
      role: 'meta',
      kind: 'code',
      render: (r) => {
        const link = r.doc_id ? links(r.doc_type, r.doc_id) : null
        return link ? <Anchor render={<Link to={link} />}>{r.doc_number}</Anchor> : r.doc_number
      },
    },
    { key: 'description', header: t('accounting.ledger.description'), role: 'meta', render: (r) => r.description ?? (r.kind ? kindLabel(t, r.kind) : '') },
    { key: 'account', header: t('accounting.ledger.account'), role: 'meta', kind: 'code', render: (r) => r.account_code },
    { key: 'debit', header: t('accounting.side.debit_money'), band: t('accounting.ledger.movement'), role: 'meta', kind: 'number', render: (r) => formatNumber(r.debit) },
    { key: 'credit', header: t('accounting.side.credit_money'), band: t('accounting.ledger.movement'), role: 'meta', kind: 'number', render: (r) => formatNumber(r.credit) },
    { key: 'balance_debit', header: t('accounting.side.debit_money'), band: t('accounting.ledger.balance'), role: 'meta', kind: 'number', strong: true, render: balance('debit') },
    { key: 'balance_credit', header: t('accounting.side.credit_money'), band: t('accounting.ledger.balance'), role: 'meta', kind: 'number', strong: true, render: balance('credit') },
  ]
  const lines = ledger.data?.lines ?? []
  const rows = ledger.data && (lines.length ? [edgeRow('opening', p.from, ledger.data.opening), ...lines.map((l, i) => ({ ...l, key: String(i) }))] : [])
  const still = ledger.data ? splitBalance(ledger.data.closing) : { debit: 0, credit: 0 }

  return (
    <ListPage
      title={t('accounting.partner_ledger.title')}
      description={t('accounting.partner_ledger.description')}
      filters={
        <>
          <LegalEntityFilter label={t('accounting.common.legal_entity')} options={entities.options} value={p.legal_entity_id || undefined} onChange={put('legal_entity_id')} />
          <FilterChip
            label={t('accounting.partners.partner')}
            icon={IconUsers}
            data={partnerOptions}
            searchable
            clearable={false}
            value={p.partner_id ? String(p.partner_id) : null}
            onChange={put('partner_id')}
          />
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
        ready
          ? {
              query: asList(ledger, rows),
              table: { label: t('accounting.partner_ledger.title'), columns, rowKey: (r) => r.key, footer: ledger.data && edgeRow('closing', p.to, ledger.data.closing) },
              empty: {
                title: t('accounting.ledger.no_lines'),
                description: t(still.credit ? 'accounting.ledger.no_lines_credit' : 'accounting.ledger.no_lines_debit', { amount: formatMoney(still.debit || still.credit) }),
              },
              noResults: t('accounting.common.no_results'),
            }
          : undefined
      }
    >
      {!ready && <EmptyState icon={IconUsers} title={t('accounting.partner_ledger.pick')} description={t('accounting.partner_ledger.pick_description')} />}
    </ListPage>
  )
}
