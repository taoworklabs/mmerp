import { Button, Group, Select } from '@mantine/core'
import { IconCircleDot, IconPlus } from '@tabler/icons-react'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { api, unwrap, type DocListItem, type DocSort } from '@/shared/api/sales'
import { DocumentStatus, useStatusLabel } from '@/shared/document'
import { errorText, formatDate, formatNumber } from '@/shared/i18n'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { SearchInput } from '@/shared/ui/form'
import { ListPage } from '@/shared/ui/page'
import { ContentSkeleton, EmptyState, ErrorState } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import { useListParams } from '@/shared/url/listParams'
import { DocProgress } from '../components/DocProgress'
import { useDocActions } from '../hooks/useSales'
import { docTypeOf, pathOf, salesKeys, type Kind } from '../keys'

const sorts = ['-date', 'date', '-number', 'number', '-total', 'total'] as const satisfies readonly DocSort[]
const statuses = ['draft', 'pending_approval', 'posted', 'cancelled'] as const
const defaults = { filters: { q: '', status: '' }, sort: '-date', sorts: [...sorts], pageSize: 50 }

// DocsPage lists the quotations or the orders the user may see.
export function DocsPage({ kind }: { kind: Kind }) {
  const { t } = useTranslation()
  const statusLabel = useStatusLabel()
  const docType = docTypeOf(kind)
  const base = `/sales/${pathOf(kind)}`
  const canCreate = useDocActions(kind).data?.includes('create') ?? false
  const [params, set] = useListParams(defaults)
  const { q, status } = params.filters as typeof defaults.filters
  const query = { q, status, sort: params.sort, page: params.page, pageSize: params.pageSize }
  const list = useQuery({
    queryKey: salesKeys.docs.list(kind, query),
    queryFn: () => {
      const p = {
        params: {
          query: {
            q: q || undefined,
            status: (status || undefined) as (typeof statuses)[number] | undefined,
            sort: params.sort as DocSort,
            page: params.page,
            page_size: params.pageSize as 50,
          },
        },
      }
      return unwrap(kind === 'quote' ? api.GET('/sales/quotes', p) : api.GET('/sales/orders', p))
    },
    placeholderData: keepPreviousData,
  })

  const columns: Column<DocListItem>[] = [
    { key: 'number', header: t('sales.doc.number'), role: 'title', sortable: true, render: (d) => d.number },
    { key: 'customer', header: t('sales.doc.customer'), role: 'meta', render: (d) => d.customer_name },
    { key: 'date', header: t(`sales.${kind}.date`), role: 'meta', sortable: true, render: (d) => formatDate(d.date) },
    ...(kind === 'quote'
      ? [{ key: 'valid_until', header: t('sales.quote.valid_until'), role: 'hidden' as const, render: (d: DocListItem) => formatDate(d.valid_until) }]
      : []),
    { key: 'org_unit', header: t('sales.doc.org_unit'), role: 'hidden', render: (d) => d.org_unit_name },
    { key: 'total', header: t('sales.doc.total_vnd'), role: 'meta', numeric: true, sortable: true, render: (d) => formatNumber(d.total) },
    {
      key: 'status',
      header: t('sales.doc.status'),
      role: 'status',
      render: (d) => (
        <Group gap="xs" wrap="nowrap">
          <DocumentStatus docType={docType} status={d.status} />
          <DocProgress doc={d} />
        </Group>
      ),
    },
  ]

  let body
  if (list.isPending) body = <ContentSkeleton />
  else if (list.isError) body = <ErrorState message={errorText(t, list.error)} onRetry={() => void list.refetch()} />
  else if (list.data.items.length === 0)
    body =
      q !== '' || status !== '' ? (
        <EmptyState
          title={t('sales.common.no_results')}
          action={
            <Button variant="default" onClick={() => set({ filters: defaults.filters })}>
              {t('sales.common.clear_filters')}
            </Button>
          }
        />
      ) : (
        <EmptyState title={t(`sales.${pathOf(kind)}.empty`)} description={canCreate ? t(`sales.${pathOf(kind)}.empty_description`) : undefined} />
      )
  else
    body = (
      <DataTable
        label={t(`sales.${pathOf(kind)}.title`)}
        columns={columns}
        rows={list.data.items}
        rowKey={(d) => d.id}
        href={(d) => `${base}/${d.id}`}
        sort={params.sort}
        onSort={(sort) => set({ sort })}
      />
    )

  return (
    <ListPage
      title={t(`sales.${pathOf(kind)}.title`)}
      description={t(`sales.${pathOf(kind)}.description`)}
      action={
        canCreate && (
          <Button component={Link} to={`${base}/new`} leftSection={<IconPlus {...icon.button} />}>
            {t(`sales.${pathOf(kind)}.create`)}
          </Button>
        )
      }
      filters={
        <>
          <SearchInput label={t('sales.doc.search')} placeholder={t('sales.doc.search_hint')} value={q} onSearch={(v) => set({ filters: { q: v } })} />
          <Select
            aria-label={t('sales.doc.status')}
            leftSection={<IconCircleDot {...icon.text} />}
            w={{ base: '100%', md: 200 }}
            data={statuses.map((s) => ({ value: s, label: statusLabel(docType, s) }))}
            value={status || null}
            onChange={(v) => set({ filters: { status: v ?? '' } })}
            placeholder={t('sales.doc.status')}
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
