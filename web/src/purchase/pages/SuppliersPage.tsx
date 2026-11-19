import { Button } from '@mmerp/ui'
import { IconCircleDot, IconPlus } from '@tabler/icons-react'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { api, unwrap, type SupplierListItem, type SupplierSort } from '@/shared/api/purchase'
import { type Column } from '@/shared/ui/DataTable'
import { FilterChip, SearchInput } from '@/shared/ui/form'
import { ListPage } from '@/shared/ui/page'
import { useListView } from '@/shared/ui/ListView'
import { icon } from '@/shared/ui/theme'
import { useListParams } from '@/shared/url/listParams'
import { SupplierStatus } from '../components/SupplierStatus'
import { useSupplierActions } from '../hooks/usePurchase'
import { purchaseKeys } from '../keys'

// Every sort the API offers; `satisfies` fails tsc if the API drops one.
const sorts = ['code', '-code', 'name', '-name'] as const satisfies readonly SupplierSort[]
const defaults = { filters: { q: '', active: '' }, sort: 'code', sorts: [...sorts], pageSize: 50 }

export function SuppliersPage() {
  const { t } = useTranslation()
  const canCreate = useSupplierActions().data?.includes('create') ?? false
  const [params, set, filterBar] = useListParams(defaults)
  const { q, active } = params.filters as typeof defaults.filters
  const query = { q, active, sort: params.sort, page: params.page, pageSize: params.pageSize }
  const list = useQuery({
    queryKey: purchaseKeys.suppliers.list(query),
    queryFn: () =>
      unwrap(
        api.GET('/purchase/suppliers', {
          params: {
            query: {
              q: q || undefined,
              active: (active || undefined) as 'true' | 'false' | undefined,
              sort: params.sort as SupplierSort,
              page: params.page,
              page_size: params.pageSize as 50,
            },
          },
        }),
      ),
    placeholderData: keepPreviousData,
  })

  const columns: Column<SupplierListItem>[] = [
    { key: 'name', header: t('purchase.supplier.name'), role: 'title', sortable: true, render: (c) => c.name },
    { key: 'code', header: t('purchase.supplier.code'), role: 'meta', kind: 'code', sortable: true, render: (c) => c.code },
    { key: 'tax_code', header: t('purchase.supplier.tax_code'), role: 'hidden', kind: 'code', render: (c) => c.tax_code ?? '' },
    { key: 'phone', header: t('purchase.supplier.phone'), role: 'hidden', kind: 'code', render: (c) => c.phone ?? '' },
    { key: 'org_unit', header: t('purchase.supplier.org_unit'), role: 'meta', render: (c) => c.org_unit_name },
    { key: 'active', header: t('purchase.supplier.status'), role: 'status', kind: 'status', render: (c) => <SupplierStatus active={c.active} /> },
  ]
  const view = useListView('purchase.suppliers', columns)

  return (
    <ListPage
      title={t('purchase.suppliers.title')}
      description={t('purchase.suppliers.description')}
      action={
        canCreate && (
          <Button render={<Link to="/purchase/suppliers/new" />} leftSection={<IconPlus {...icon.button} />}>
            {t('purchase.suppliers.create')}
          </Button>
        )
      }
      onClearFilters={filterBar.active ? filterBar.clear : undefined}
      filters={
        <>
          <SearchInput
            label={t('purchase.suppliers.search')}
            placeholder={t('purchase.suppliers.search_hint')}
            value={q}
            onSearch={(v) => set({ filters: { q: v } })}
          />
          <FilterChip
            label={t('purchase.supplier.status')}
            icon={IconCircleDot}
            data={[
              { value: 'true', label: t('purchase.supplier.status.active') },
              { value: 'false', label: t('purchase.supplier.status.inactive') },
            ]}
            {...filterBar.bind('active')}
          />
        </>
      }
      tools={view.tools}
      list={{
        query: list,
        table: {
          label: t('purchase.suppliers.title'),
          columns: view.columns,
          onResize: view.onResize,
          rowKey: (c) => c.id,
          href: (c) => `/purchase/suppliers/${c.id}`,
          sort: params.sort,
          onSort: (sort) => set({ sort }),
        },
        empty: { title: t('purchase.suppliers.empty'), description: t('purchase.suppliers.empty_description') },
        noResults: t('purchase.common.no_results'),
        paging: { page: params.page, pageSize: params.pageSize, onPage: (page) => set({ page }), onPageSize: (pageSize) => set({ pageSize }) },
      }}
    >
    </ListPage>
  )
}
