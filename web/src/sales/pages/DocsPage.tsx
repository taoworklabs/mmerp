import { Button } from '@mantine/core'
import { IconPlus } from '@tabler/icons-react'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { api, unwrap, type DocSort } from '@/shared/api/sales'
import { DocumentStatusFilter, type DocStatus } from '@/shared/document'
import { SearchInput } from '@/shared/ui/form'
import { ListPage } from '@/shared/ui/page'
import { useListView } from '@/shared/ui/ListView'
import { icon } from '@/shared/ui/theme'
import { useListParams } from '@/shared/url/listParams'
import { useDocColumns } from '../components/docColumns'
import { useDocActions } from '../hooks/useSales'
import { docTypeOf, pathOf, salesKeys, type Kind } from '../keys'

const sorts = ['-date', 'date', '-number', 'number', '-total', 'total'] as const satisfies readonly DocSort[]
const defaults = { filters: { q: '', status: '' }, sort: '-date', sorts: [...sorts], pageSize: 50 }

// DocsPage lists the quotations or the orders the user may see.
export function DocsPage({ kind: docKind }: { kind: Kind }) {
  const { t } = useTranslation()
  const docType = docTypeOf(docKind)
  const base = `/sales/${pathOf(docKind)}`
  const canCreate = useDocActions(docKind).data?.includes('create') ?? false
  const [params, set, filterBar] = useListParams(defaults)
  const { q, status } = params.filters as typeof defaults.filters
  const query = { q, status, sort: params.sort, page: params.page, pageSize: params.pageSize }
  const list = useQuery({
    queryKey: salesKeys.docs.list(docKind, query),
    queryFn: () => {
      const p = {
        params: {
          query: {
            q: q || undefined,
            status: (status || undefined) as DocStatus | undefined,
            sort: params.sort as DocSort,
            page: params.page,
            page_size: params.pageSize as 50,
          },
        },
      }
      return unwrap(docKind === 'quote' ? api.GET('/sales/quotes', p) : api.GET('/sales/orders', p))
    },
    placeholderData: keepPreviousData,
  })

  const columns = useDocColumns(docKind)
  const view = useListView(`sales.${pathOf(docKind)}`, columns)

  return (
    <ListPage
      title={t(`sales.${pathOf(docKind)}.title`)}
      description={t(`sales.${pathOf(docKind)}.description`)}
      action={
        canCreate && (
          <Button component={Link} to={`${base}/new`} leftSection={<IconPlus {...icon.button} />}>
            {t(`sales.${pathOf(docKind)}.create`)}
          </Button>
        )
      }
      onClearFilters={filterBar.active ? filterBar.clear : undefined}
      filters={
        <>
          <SearchInput label={t('sales.doc.search')} placeholder={t('sales.doc.search_hint')} value={q} onSearch={(v) => set({ filters: { q: v } })} />
          <DocumentStatusFilter docType={docType} {...filterBar.bind('status')} />
        </>
      }
      tools={view.tools}
      list={{
        query: list,
        table: {
          label: t(`sales.${pathOf(docKind)}.title`),
          columns: view.columns,
          onResize: view.onResize,
          rowKey: (d) => d.id,
          href: (d) => `${base}/${d.id}`,
          sort: params.sort,
          onSort: (sort) => set({ sort }),
        },
        empty: { title: t(`sales.${pathOf(docKind)}.empty`), description: t(`sales.${pathOf(docKind)}.empty_description`) },
        noResults: t('sales.common.no_results'),
        paging: { page: params.page, pageSize: params.pageSize, onPage: (page) => set({ page }), onPageSize: (pageSize) => set({ pageSize }) },
      }}
    >
    </ListPage>
  )
}
