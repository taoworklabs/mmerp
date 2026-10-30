import { Group, Text } from '@mmerp/ui'
import { IconBuildingWarehouse } from '@tabler/icons-react'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useEffect } from 'react'
import { FormProvider, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type OnHandRow } from '@/shared/api/core'
import { useMe } from '@/shared/auth/me'
import { formatDate, formatDecimal, today as todayIn } from '@/shared/i18n'
import { type Column } from '@/shared/ui/DataTable'
import { DateField, FilterChip, SearchInput, type Option } from '@/shared/ui/form'
import { useListView } from '@/shared/ui/ListView'
import { ListPage } from '@/shared/ui/page'
import { useListParams } from '@/shared/url/listParams'
import { salesKeys } from '../keys'
import { isExpired, stockCardPath } from '../stock'

const defaults = { filters: { q: '', warehouse: '', expiring: '' }, sort: '', sorts: [], pageSize: 50 }

// useWarehouseOptions lists the active warehouses, "code · name", for a filter or a picker.
export function useWarehouseOptions(): Option[] {
  // ponytail: one page of 100 warehouses; a picker with search when tenants have more.
  const query = { q: '', active: 'true', sort: 'code', page: 1, pageSize: 100 }
  const list = useQuery({
    queryKey: salesKeys.stock.warehouses.list(query),
    queryFn: () => unwrap(api.GET('/stock/warehouses', { params: { query: { active: 'true', sort: 'code', page_size: 100 } } })),
  })
  return (list.data?.items ?? []).map((w) => ({ value: String(w.id), label: `${w.code} · ${w.name}` }))
}

// OnHandPage: what each warehouse holds now, per item, lot and serial; a row opens its stock card.
export function OnHandPage() {
  const { t } = useTranslation()
  const today = todayIn(useMe().timezone)
  const warehouses = useWarehouseOptions()
  const [params, set, filterBar] = useListParams(defaults)
  const { q, warehouse, expiring } = params.filters as typeof defaults.filters
  const query = { warehouse, item: '', q, expiring, page: params.page, pageSize: params.pageSize }
  const list = useQuery({
    queryKey: salesKeys.stock.onHand.list(query),
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

  // The date filter is a form field; the URL stays its source, so Back and Clear filters reach it.
  const form = useForm<{ expiring: string | null }>({ values: { expiring: expiring || null } })
  useEffect(() => {
    const sub = form.watch((v, { name }) => name && set({ filters: { expiring: v.expiring ?? '' } }))
    return () => sub.unsubscribe()
  }, [form, set])

  const columns: Column<OnHandRow>[] = [
    { key: 'item_code', header: t('sales.on_hand.item_code'), role: 'title', kind: 'code', render: (r) => r.item_code },
    { key: 'item_name', header: t('sales.on_hand.item_name'), role: 'meta', render: (r) => r.item_name },
    { key: 'warehouse', header: t('sales.on_hand.warehouse'), role: 'meta', kind: 'code', render: (r) => r.warehouse_code },
    { key: 'unit', header: t('sales.on_hand.unit'), role: 'meta', render: (r) => r.unit },
    { key: 'lot', header: t('sales.on_hand.lot'), role: 'meta', kind: 'code', render: (r) => r.lot_code ?? '' },
    {
      key: 'expiry',
      header: t('sales.on_hand.expiry'),
      role: 'meta',
      kind: 'date',
      render: (r) =>
        isExpired(r.expiry, today) ? (
          <Group gap="xs" wrap={false}>
            {formatDate(r.expiry)}
            <Text render={<span />} tone="danger">
              {t('sales.on_hand.expired')}
            </Text>
          </Group>
        ) : (
          formatDate(r.expiry)
        ),
    },
    { key: 'serial', header: t('sales.on_hand.serial'), role: 'meta', kind: 'code', render: (r) => r.serial_code ?? '' },
    { key: 'quantity', header: t('sales.on_hand.quantity'), role: 'meta', kind: 'number', strong: true, render: (r) => formatDecimal(r.quantity, 3) },
  ]
  const view = useListView('sales.on_hand', columns)
  return (
    <ListPage
      title={t('sales.on_hand.title')}
      description={t('sales.on_hand.description')}
      onClearFilters={filterBar.active ? filterBar.clear : undefined}
      filters={
        <>
          <SearchInput label={t('sales.on_hand.search')} placeholder={t('sales.on_hand.search_hint')} value={q} onSearch={(v) => set({ filters: { q: v } })} />
          <FilterChip label={t('sales.on_hand.warehouse')} icon={IconBuildingWarehouse} data={warehouses} searchable {...filterBar.bind('warehouse')} />
          <FormProvider {...form}>
            <DateField name="expiring" label={t('sales.on_hand.expiring_before')} clearable />
          </FormProvider>
        </>
      }
      tools={view.tools}
      list={{
        query: list,
        table: {
          label: t('sales.on_hand.title'),
          columns: view.columns,
          onResize: view.onResize,
          rowKey: (r) => `${r.warehouse_id}:${r.item_id}:${r.lot_code ?? ''}:${r.serial_code ?? ''}`,
          href: (r) => stockCardPath(r.warehouse_id, r.item_id, today),
        },
        empty: { title: t('sales.on_hand.empty'), description: t('sales.on_hand.empty_description') },
        noResults: t('sales.common.no_results'),
        paging: { page: params.page, pageSize: params.pageSize, onPage: (page) => set({ page }), onPageSize: (pageSize) => set({ pageSize }) },
      }}
    />
  )
}
