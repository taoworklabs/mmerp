import { Button } from '@mantine/core'
import { IconCalendarDue, IconFileCertificate } from '@tabler/icons-react'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type ContractListItem, type ContractSort } from '@/shared/api/hrm'
import { DocumentStatus, DocumentStatusFilter, type DocStatus } from '@/shared/document'
import { formatDate } from '@/shared/i18n'
import { type Column } from '@/shared/ui/DataTable'
import { FilterChip, OrgUnitFilter, SearchInput } from '@/shared/ui/form'
import { useListView } from '@/shared/ui/ListView'
import { ListPage } from '@/shared/ui/page'
import { SidePanel, usePeek } from '@/shared/ui/SidePanel'
import { icon } from '@/shared/ui/theme'
import { useListParams } from '@/shared/url/listParams'
import ContractPreview from '../components/ContractPreview'
import { useContractTypes } from '../hooks/useContract'
import { contractDocType, hrmKeys } from '../keys'

// Every sort the API offers; `satisfies` fails tsc if the API drops one.
const sorts = ['-start_date', 'start_date', 'end_date', '-end_date', '-number', 'number'] as const satisfies readonly ContractSort[]
const defaults = { filters: { q: '', status: '', contract_type_id: '', org_unit_id: '', expiring: '' }, sort: '-start_date', sorts: [...sorts], pageSize: 50 }

// ContractsPage: every contract in the user's scope, without amounts. New contracts start from the employee.
export function ContractsPage() {
  const { t } = useTranslation()
  const types = useContractTypes()
  const [params, set, filterBar] = useListParams(defaults)
  const peek = usePeek()
  const filters = params.filters as typeof defaults.filters
  const { q, status, contract_type_id, org_unit_id, expiring } = filters
  const query = { ...filters, sort: params.sort, page: params.page, pageSize: params.pageSize }
  const list = useQuery({
    queryKey: hrmKeys.contracts.list(query),
    queryFn: () =>
      unwrap(
        api.GET('/hrm/contracts', {
          params: {
            query: {
              q: q || undefined,
              status: (status || undefined) as DocStatus | undefined,
              contract_type_id: contract_type_id ? Number(contract_type_id) : undefined,
              org_unit_id: org_unit_id ? Number(org_unit_id) : undefined,
              expiring: expiring === '1' || undefined,
              sort: params.sort as ContractSort,
              page: params.page,
              page_size: params.pageSize as 50,
            },
          },
        }),
      ),
    placeholderData: keepPreviousData,
  })

  const columns: Column<ContractListItem>[] = [
    { key: 'number', header: t('hrm.contract.number'), role: 'title', kind: 'code', sortable: true, render: (c) => c.number },
    { key: 'employee', header: t('hrm.contract.employee'), role: 'meta', render: (c) => `${c.employee_code} · ${c.employee_name}` },
    {
      key: 'kind',
      header: t('hrm.contract.kind'),
      role: 'hidden',
      render: (c) => (c.parent_number ? t('hrm.contract.appendix_of', { number: c.parent_number }) : c.contract_type_name),
    },
    { key: 'start_date', header: t('hrm.contract.start_date'), role: 'meta', kind: 'date', sortable: true, render: (c) => formatDate(c.start_date) },
    {
      key: 'end_date',
      header: t('hrm.contract.end_date'),
      role: 'hidden',
      kind: 'date',
      sortable: true,
      render: (c) => (c.parent_id ? '' : c.end_date ? formatDate(c.end_date) : t('hrm.contract.no_end')),
    },
    { key: 'status', header: t('hrm.contract.status'), role: 'status', kind: 'status', render: (c) => <DocumentStatus docType={contractDocType} status={c.status} /> },
  ]
  const view = useListView('hrm.contracts', columns)
  const open = list.data?.items.find((c) => String(c.id) === peek.id)

  return (
    <ListPage
      title={t('hrm.contracts.title')}
      description={t('hrm.contracts.description')}
      onClearFilters={filterBar.active ? filterBar.clear : undefined}
      filters={
        <>
          <SearchInput label={t('hrm.contracts.search')} placeholder={t('hrm.common.search_request_hint')} value={q} onSearch={(v) => set({ filters: { q: v } })} />
          <DocumentStatusFilter docType={contractDocType} {...filterBar.bind('status')} />
          <FilterChip
            label={t('hrm.contract.kind')}
            icon={IconFileCertificate}
            data={(types.data ?? []).map((x) => ({ value: String(x.id), label: x.name }))}
            {...filterBar.bind('contract_type_id')}
          />
          <OrgUnitFilter
            label={t('hrm.contracts.org_unit')}
            product="hrm"
            permission="hrm.contract.view"
            {...filterBar.bind('org_unit_id')}
          />
          <Button
            variant={expiring ? 'light' : 'default'}
            aria-pressed={expiring === '1'}
            leftSection={<IconCalendarDue {...icon.button} />}
            onClick={() => set({ filters: { expiring: expiring ? '' : '1' } })}
          >
            {t('hrm.contracts.expiring')}
          </Button>
        </>
      }
      tools={view.tools}
      list={{
        query: list,
        table: {
          label: t('hrm.contracts.title'),
          columns: view.columns,
          onResize: view.onResize,
          rowKey: (c) => c.id,
          href: (c) => `/hrm/contracts/${c.id}`,
          onRowClick: (c) => peek.open(c.id),
          sort: params.sort,
          onSort: (sort) => set({ sort }),
        },
        empty: { title: t('hrm.contract.empty'), description: t('hrm.contracts.empty_description') },
        noResults: t('hrm.contracts.no_results'),
        paging: { page: params.page, pageSize: params.pageSize, onPage: (page) => set({ page }), onPageSize: (pageSize) => set({ pageSize }) },
      }}
    >
      <SidePanel title={open?.number ?? ''} href={`/hrm/contracts/${open?.id ?? ''}`} opened={!!open} onClose={peek.close}>
        {open && <ContractPreview id={open.id} />}
      </SidePanel>
    </ListPage>
  )
}
