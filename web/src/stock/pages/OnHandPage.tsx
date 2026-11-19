import { Group, Text } from '@mmerp/ui'
import { IconBuildingWarehouse } from '@tabler/icons-react'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type OnHandRow } from '@/shared/api/core'
import { useMe } from '@/shared/auth/me'
import { formatDate, formatDecimal, today as todayIn } from '@/shared/i18n'
import { type Column } from '@/shared/ui/DataTable'
import { DateFilter, FilterChip, SearchInput } from '@/shared/ui/form'
import { useListView } from '@/shared/ui/ListView'
import { ListPage } from '@/shared/ui/page'
import { useListParams } from '@/shared/url/listParams'
import { useWarehouseOptions } from '../hooks/useStock'
import { stockKeys } from '../keys'
import { isExpired, stockCardPath } from '../stock'

const defaults = { filters: { q: '', warehouse: '', expiring: '' }, sort: '', sorts: [], pageSize: 50 }

// OnHandPage: what each warehouse holds now, per item, lot and serial; a row opens its stock card.
export function OnHandPage() {
  const { t } = useTranslation()
  const today = todayIn(useMe().timezone)
  const warehouses = useWarehouseOptions()
  const [params, set, filterBar] = useListParams(defaults)
  const { q, warehouse, expiring } = params.filters as typeof defaults.filters
  const query = { warehouse, item: '', q, expiring, page: params.page, pageSize: params.pageSize }
  const list = useQuery({
    queryKey: stockKeys.onHand.list(query),
    // Sales and Purchasing documents move stock without refreshing this area's queries.
    staleTime: 0,
    queryFn: () =>
      unwrap(
        api.GET('/stock/on-hand', {
          params: {
            query: {
              warehouse_id: warehouse ? Number(warehouse) : undefined,
              q: q || undefined,
              expiring_before: expiring || undefined,
              page: params.page,
              page_size: params.pageSize as 50,
            },
          },
        }),
      ),
    placeholderData: keepPreviousData,
  })

  const columns: Column<OnHandRow>[] = [
    { key: 'item_code', header: t('stock.on_hand.item_code'), role: 'title', kind: 'code', render: (r) => r.item_code },
    { key: 'item_name', header: t('stock.on_hand.item_name'), role: 'meta', render: (r) => r.item_name },
    { key: 'warehouse', header: t('stock.on_hand.warehouse'), role: 'meta', kind: 'code', render: (r) => r.warehouse_code },
    { key: 'unit', header: t('stock.on_hand.unit'), role: 'meta', render: (r) => r.unit },
    { key: 'lot', header: t('stock.on_hand.lot'), role: 'meta', kind: 'code', render: (r) => r.lot_code ?? '' },
    {
      key: 'expiry',
      header: t('stock.on_hand.expiry'),
      role: 'meta',
      kind: 'date',
      render: (r) =>
        isExpired(r.expiry, today) ? (
          <Group gap="xs" wrap={false}>
            {formatDate(r.expiry)}
            <Text render={<span />} tone="danger">
              {t('stock.on_hand.expired')}
            </Text>
          </Group>
        ) : (
          formatDate(r.expiry)
        ),
    },
    { key: 'serial', header: t('stock.on_hand.serial'), role: 'meta', kind: 'code', render: (r) => r.serial_code ?? '' },
    { key: 'quantity', header: t('stock.on_hand.quantity'), role: 'meta', kind: 'number', strong: true, render: (r) => formatDecimal(r.quantity, 3) },
  ]
  const view = useListView('stock.on_hand', columns)
  return (
    <ListPage
      title={t('stock.on_hand.title')}
      description={t('stock.on_hand.description')}
      onClearFilters={filterBar.active ? filterBar.clear : undefined}
      filters={
        <>
          <SearchInput label={t('stock.on_hand.search')} placeholder={t('stock.on_hand.search_hint')} value={q} onSearch={(v) => set({ filters: { q: v } })} />
          <FilterChip label={t('stock.on_hand.warehouse')} icon={IconBuildingWarehouse} data={warehouses} searchable {...filterBar.bind('warehouse')} />
          <DateFilter label={t('stock.on_hand.expiring_before')} {...filterBar.bind('expiring')} />
        </>
      }
      tools={view.tools}
      list={{
        query: list,
        table: {
          label: t('stock.on_hand.title'),
          columns: view.columns,
          onResize: view.onResize,
          rowKey: (r) => `${r.warehouse_id}:${r.item_id}:${r.lot_code ?? ''}:${r.serial_code ?? ''}`,
          href: (r) => stockCardPath(r.warehouse_id, r.item_id, today),
        },
        empty: { title: t('stock.on_hand.empty'), description: t('stock.on_hand.empty_description') },
        noResults: t('stock.common.no_results'),
        paging: { page: params.page, pageSize: params.pageSize, onPage: (page) => set({ page }), onPageSize: (pageSize) => set({ pageSize }) },
      }}
    />
  )
}
