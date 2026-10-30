import { Anchor } from '@mmerp/ui'
import { IconListDetails } from '@tabler/icons-react'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useEffect } from 'react'
import { FormProvider, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { Link, useSearchParams } from 'react-router'
import { api, ApiError, unwrap, type StockCardRow } from '@/shared/api/core'
import { useRecordLinks } from '@/shared/area'
import { useCan, useMe } from '@/shared/auth/me'
import { errorText, formatDate, formatDecimal, today as todayIn } from '@/shared/i18n'
import { type Column } from '@/shared/ui/DataTable'
import { DateField, RemoteSelectField, SelectField } from '@/shared/ui/form'
import { ListPage } from '@/shared/ui/page'
import { EmptyState } from '@/shared/ui/states'
import { salesKeys, type StockCardQuery } from '../keys'
import { docTypeLabelKey, monthToDate } from '../stock'
import { useWarehouseOptions } from './OnHandPage'

type Values = { [K in keyof StockCardQuery]: string | null }
// The opening balance heads the moves; the closing one is the table's footer.
type Row = StockCardRow & { key: string; edge?: 'opening' | 'closing' }

const edgeRow = (edge: 'opening' | 'closing', date: string, balance: string): Row => ({ key: edge, edge, date, balance, doc_type: '', doc_id: 0, number: '', quantity: '' })

// StockCardPage: the moves of one item in one warehouse over a period, with the running balance.
export function StockCardPage() {
  const { t } = useTranslation()
  const today = todayIn(useMe().timezone)
  const canItems = useCan('item.view')
  const links = useRecordLinks()
  const warehouses = useWarehouseOptions()
  const [search, setSearch] = useSearchParams()
  const range = monthToDate(today)
  const p: StockCardQuery = {
    warehouse_id: search.get('warehouse_id') ?? '',
    item_id: search.get('item_id') ?? '',
    from: search.get('from') || range.from,
    to: search.get('to') || range.to,
  }
  const ready = !!p.warehouse_id && !!p.item_id

  const card = useQuery({
    queryKey: salesKeys.stock.card(p),
    queryFn: () =>
      unwrap(api.GET('/stock/card', { params: { query: { warehouse_id: Number(p.warehouse_id), item_id: Number(p.item_id), from: p.from, to: p.to } } })),
    enabled: ready,
    placeholderData: keepPreviousData,
  })
  // A range the server refuses is shown at the dates, not in place of the table.
  const rangeError = card.error instanceof ApiError && card.error.code.startsWith('stock_card_range_') ? errorText(t, card.error) : undefined

  // The picker only knows the item's id from the URL; its on-hand rows name it.
  // ponytail: an item with no stock left anywhere shows no name in the picker; add GET /items/{id} if that matters.
  const named = useQuery({
    queryKey: salesKeys.stock.onHand.list({ warehouse: '', item: p.item_id, q: '', expiring: '', page: 1, pageSize: 20 }),
    queryFn: () => unwrap(api.GET('/stock/on-hand', { params: { query: { item_id: Number(p.item_id), page_size: 20 } } })),
    enabled: !!p.item_id,
  })
  const first = named.data?.items[0]
  const current = first && { id: first.item_id, code: first.item_code, name: first.item_name }

  // The fields edit the URL, which stays the source, so Back reaches them.
  const form = useForm<Values>({ values: p })
  useEffect(() => {
    const sub = form.watch((v, { name }) => {
      if (!name) return
      const next = Object.entries(v).filter((e): e is [string, string] => !!e[1])
      setSearch(new URLSearchParams(next))
    })
    return () => sub.unsubscribe()
  }, [form, setSearch])

  const columns: Column<Row>[] = [
    {
      key: 'date',
      header: t('sales.stock_card.date'),
      role: 'title',
      kind: 'date',
      render: (r) => (r.edge ? t(`sales.stock_card.${r.edge}`) : formatDate(r.date)),
    },
    {
      key: 'document',
      header: t('sales.stock_card.document'),
      role: 'meta',
      kind: 'code',
      render: (r) => {
        if (r.edge) return ''
        const link = r.doc_id ? links(r.doc_type, r.doc_id) : null
        if (link)
          return (
            <Anchor render={<Link to={link} />}>
              {r.number}
            </Anchor>
          )
        if (r.doc_id) return r.number
        const key = docTypeLabelKey(r.doc_type)
        return key ? t(key) : r.doc_type
      },
    },
    { key: 'lot', header: t('sales.stock_card.lot'), role: 'meta', kind: 'code', render: (r) => r.lot_code ?? '' },
    { key: 'serial', header: t('sales.stock_card.serial'), role: 'meta', kind: 'code', render: (r) => r.serial_code ?? '' },
    { key: 'quantity', header: t('sales.stock_card.quantity'), role: 'meta', kind: 'number', render: (r) => formatDecimal(r.quantity, 3) },
    { key: 'balance', header: t('sales.stock_card.balance'), role: 'meta', kind: 'number', strong: true, render: (r) => formatDecimal(r.balance, 3) },
  ]
  const rows = card.data && [edgeRow('opening', p.from, card.data.opening), ...card.data.rows.map((r, i) => ({ ...r, key: String(i) }))]

  return (
    <ListPage
      title={t('sales.stock_card.title')}
      description={t('sales.stock_card.description')}
      filters={
        <FormProvider {...form}>
          <SelectField name="warehouse_id" label={t('sales.stock_card.warehouse')} data={warehouses} required />
          <RemoteSelectField
            name="item_id"
            label={t('sales.stock_card.item')}
            current={current}
            // Without the item role the card shows only the item it was opened for.
            readOnly={!canItems}
            required
            queryKey={salesKeys.stock.items}
            search={async (q) => (await unwrap(api.GET('/items', { params: { query: { q: q || undefined, active: 'true', page_size: 20 } } }))).items}
          />
          <DateField name="from" label={t('sales.stock_card.from')} required />
          <DateField name="to" label={t('sales.stock_card.to')} required error={rangeError} />
        </FormProvider>
      }
      list={
        ready && !rangeError
          ? {
              query: { isPending: card.isPending, isError: card.isError, error: card.error, refetch: card.refetch, data: rows && { items: rows, total: rows.length } },
              table: {
                label: t('sales.stock_card.title'),
                columns,
                rowKey: (r) => r.key,
                footer: card.data && edgeRow('closing', p.to, card.data.closing),
              },
              empty: { title: t('sales.stock_card.no_moves') },
              noResults: t('sales.common.no_results'),
            }
          : undefined
      }
    >
      {!ready && <EmptyState icon={IconListDetails} title={t('sales.stock_card.pick')} description={t('sales.stock_card.pick_description')} />}
    </ListPage>
  )
}
