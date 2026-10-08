import { Button, Select } from '@mantine/core'
import { IconCalendarDue, IconCircleDot, IconFileCertificate } from '@tabler/icons-react'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type ContractListItem, type ContractSort } from '@/shared/api/hrm'
import { DocumentStatus, useStatusLabel } from '@/shared/document'
import { errorText, formatDate } from '@/shared/i18n'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { OrgUnitSelect } from '@/shared/ui/form'
import { ListPage } from '@/shared/ui/page'
import { ContentSkeleton, EmptyState, ErrorState } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import { useListParams } from '@/shared/url/listParams'
import { useContractTypes } from '../hooks/useContract'
import { contractDocType, hrmKeys } from '../keys'

// Every sort the API offers; `satisfies` fails tsc if the API drops one.
const sorts = ['-start_date', 'start_date', 'end_date', '-end_date', '-number', 'number'] as const satisfies readonly ContractSort[]
const statuses = ['draft', 'pending_approval', 'posted', 'cancelled'] as const
const defaults = { filters: { status: '', contract_type_id: '', org_unit_id: '', expiring: '' }, sort: '-start_date', sorts: [...sorts], pageSize: 50 }

// ContractsPage: every contract in the user's scope, without amounts. New contracts start from the employee.
export function ContractsPage() {
  const { t } = useTranslation()
  const statusLabel = useStatusLabel()
  const types = useContractTypes()
  const [params, set] = useListParams(defaults)
  const filters = params.filters as typeof defaults.filters
  const { status, contract_type_id, org_unit_id, expiring } = filters
  const query = { ...filters, sort: params.sort, page: params.page, pageSize: params.pageSize }
  const list = useQuery({
    queryKey: hrmKeys.contracts.list(query),
    queryFn: () =>
      unwrap(
        api.GET('/hrm/contracts', {
          params: {
            query: {
              status: (status || undefined) as (typeof statuses)[number] | undefined,
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
    { key: 'number', header: t('hrm.contract.number'), role: 'title', sortable: true, render: (c) => c.number },
    { key: 'employee', header: t('hrm.contract.employee'), role: 'meta', render: (c) => `${c.employee_code} · ${c.employee_name}` },
    {
      key: 'kind',
      header: t('hrm.contract.kind'),
      role: 'hidden',
      render: (c) => (c.parent_number ? t('hrm.contract.appendix_of', { number: c.parent_number }) : c.contract_type_name),
    },
    { key: 'start_date', header: t('hrm.contract.start_date'), role: 'meta', sortable: true, render: (c) => formatDate(c.start_date) },
    {
      key: 'end_date',
      header: t('hrm.contract.end_date'),
      role: 'hidden',
      sortable: true,
      render: (c) => (c.parent_id ? '' : c.end_date ? formatDate(c.end_date) : t('hrm.contract.no_end')),
    },
    { key: 'status', header: t('hrm.contract.status'), role: 'status', render: (c) => <DocumentStatus docType={contractDocType} status={c.status} /> },
  ]

  const filtered = Object.values(filters).some((v) => v !== '')
  let body
  if (list.isPending) body = <ContentSkeleton />
  else if (list.isError) body = <ErrorState message={errorText(t, list.error)} onRetry={() => void list.refetch()} />
  else if (list.data.items.length === 0)
    body = filtered ? (
      <EmptyState
        title={t('hrm.contracts.no_results')}
        action={
          <Button variant="default" onClick={() => set({ filters: defaults.filters })}>
            {t('hrm.common.clear_filters')}
          </Button>
        }
      />
    ) : (
      <EmptyState title={t('hrm.contract.empty')} description={t('hrm.contracts.empty_description')} />
    )
  else
    body = (
      <DataTable
        label={t('hrm.contracts.title')}
        columns={columns}
        rows={list.data.items}
        rowKey={(c) => c.id}
        href={(c) => `/hrm/contracts/${c.id}`}
        sort={params.sort}
        onSort={(sort) => set({ sort })}
      />
    )

  return (
    <ListPage
      title={t('hrm.contracts.title')}
      description={t('hrm.contracts.description')}
      filters={
        <>
          <Button
            variant={expiring ? 'light' : 'default'}
            aria-pressed={expiring === '1'}
            leftSection={<IconCalendarDue {...icon.button} />}
            onClick={() => set({ filters: { expiring: expiring ? '' : '1' } })}
          >
            {t('hrm.contracts.expiring')}
          </Button>
          <Select
            aria-label={t('hrm.contract.status')}
            leftSection={<IconCircleDot {...icon.text} />}
            w={{ base: '100%', md: 180 }}
            data={statuses.map((s) => ({ value: s, label: statusLabel(contractDocType, s) }))}
            value={status || null}
            onChange={(v) => set({ filters: { status: v ?? '' } })}
            placeholder={t('hrm.contract.status')}
            clearable
          />
          <Select
            aria-label={t('hrm.contract.kind')}
            leftSection={<IconFileCertificate {...icon.text} />}
            w={{ base: '100%', md: 200 }}
            data={(types.data ?? []).map((x) => ({ value: String(x.id), label: x.name }))}
            value={contract_type_id || null}
            onChange={(v) => set({ filters: { contract_type_id: v ?? '' } })}
            placeholder={t('hrm.contract.kind')}
            clearable
          />
          <OrgUnitSelect
            label={t('hrm.contracts.org_unit')}
            product="hrm"
            permission="hrm.contract.view"
            value={org_unit_id || null}
            onChange={(v) => set({ filters: { org_unit_id: v ?? '' } })}
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
