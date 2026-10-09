import { Button, Select } from '@mantine/core'
import { IconCircleDot, IconPlus } from '@tabler/icons-react'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { api, unwrap, type CustomerListItem, type CustomerSort } from '@/shared/api/sales'
import { errorText } from '@/shared/i18n'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { SearchInput } from '@/shared/ui/form'
import { ListPage } from '@/shared/ui/page'
import { ContentSkeleton, EmptyState, ErrorState } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import { useListParams } from '@/shared/url/listParams'
import { CustomerStatus } from '../components/CustomerStatus'
import { useCustomerActions } from '../hooks/useSales'
import { salesKeys } from '../keys'

// Every sort the API offers; `satisfies` fails tsc if the API drops one.
const sorts = ['code', '-code', 'name', '-name'] as const satisfies readonly CustomerSort[]
const defaults = { filters: { q: '', active: '' }, sort: 'code', sorts: [...sorts], pageSize: 50 }

export function CustomersPage() {
  const { t } = useTranslation()
  const canCreate = useCustomerActions().data?.includes('create') ?? false
  const [params, set] = useListParams(defaults)
  const { q, active } = params.filters as typeof defaults.filters
  const query = { q, active, sort: params.sort, page: params.page, pageSize: params.pageSize }
  const list = useQuery({
    queryKey: salesKeys.customers.list(query),
    queryFn: () =>
      unwrap(
        api.GET('/sales/customers', {
          params: {
            query: {
              q: q || undefined,
              active: (active || undefined) as 'true' | 'false' | undefined,
              sort: params.sort as CustomerSort,
              page: params.page,
              page_size: params.pageSize as 50,
            },
          },
        }),
      ),
    placeholderData: keepPreviousData,
  })

  const columns: Column<CustomerListItem>[] = [
    { key: 'name', header: t('sales.customer.name'), role: 'title', sortable: true, render: (c) => c.name },
    { key: 'code', header: t('sales.customer.code'), role: 'meta', sortable: true, render: (c) => c.code },
    { key: 'tax_code', header: t('sales.customer.tax_code'), role: 'hidden', render: (c) => c.tax_code ?? '' },
    { key: 'phone', header: t('sales.customer.phone'), role: 'hidden', render: (c) => c.phone ?? '' },
    { key: 'org_unit', header: t('sales.customer.org_unit'), role: 'meta', render: (c) => c.org_unit_name },
    { key: 'active', header: t('sales.customer.status'), role: 'status', render: (c) => <CustomerStatus active={c.active} /> },
  ]

  let body
  if (list.isPending) body = <ContentSkeleton />
  else if (list.isError) body = <ErrorState message={errorText(t, list.error)} onRetry={() => void list.refetch()} />
  else if (list.data.items.length === 0)
    body =
      q !== '' || active !== '' ? (
        <EmptyState
          title={t('sales.common.no_results')}
          action={
            <Button variant="default" onClick={() => set({ filters: defaults.filters })}>
              {t('sales.common.clear_filters')}
            </Button>
          }
        />
      ) : (
        <EmptyState title={t('sales.customers.empty')} description={canCreate ? t('sales.customers.empty_description') : undefined} />
      )
  else
    body = (
      <DataTable
        label={t('sales.customers.title')}
        columns={columns}
        rows={list.data.items}
        rowKey={(c) => c.id}
        href={(c) => `/sales/customers/${c.id}`}
        sort={params.sort}
        onSort={(sort) => set({ sort })}
      />
    )

  return (
    <ListPage
      title={t('sales.customers.title')}
      description={t('sales.customers.description')}
      action={
        canCreate && (
          <Button component={Link} to="/sales/customers/new" leftSection={<IconPlus {...icon.button} />}>
            {t('sales.customers.create')}
          </Button>
        )
      }
      filters={
        <>
          <SearchInput
            label={t('sales.customers.search')}
            placeholder={t('sales.customers.search_hint')}
            value={q}
            onSearch={(v) => set({ filters: { q: v } })}
          />
          <Select
            aria-label={t('sales.customer.status')}
            leftSection={<IconCircleDot {...icon.text} />}
            w={{ base: '100%', md: 180 }}
            data={[
              { value: 'true', label: t('sales.customer.status.active') },
              { value: 'false', label: t('sales.customer.status.inactive') },
            ]}
            value={active || null}
            onChange={(v) => set({ filters: { active: v ?? '' } })}
            placeholder={t('sales.customer.status')}
            clearable
          />
        </>
      }
      paging={
        list.data && {
          page: params.page,
          pageSize: params.pageSize,
          total: list.data.total,
          onPage: (page) => set({ page }),
          onPageSize: (pageSize) => set({ pageSize }),
        }
      }
    >
      {body}
    </ListPage>
  )
}
