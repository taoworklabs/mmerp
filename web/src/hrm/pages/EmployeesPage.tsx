import { Button, Group } from '@mantine/core'
import { IconCircleDot, IconFileImport, IconPlus } from '@tabler/icons-react'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { api, unwrap, type EmployeeListItem, type EmployeeSort } from '@/shared/api/hrm'
import { errorText, formatDate } from '@/shared/i18n'
import { ImportDialog } from '@/shared/jobs'
import { PersonName } from '@/shared/ui/avatar'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { FilterChip, OrgUnitFilter, SearchInput } from '@/shared/ui/form'
import { useListView } from '@/shared/ui/ListView'
import { ListPage } from '@/shared/ui/page'
import { ContentSkeleton, EmptyState, ErrorState } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import { useListParams } from '@/shared/url/listParams'
import { EmployeeStatus } from '../components/EmployeeStatus'
import { useEmployeeActions } from '../hooks/useEmployeeActions'
import { hrmKeys } from '../keys'

// Every sort the API offers; `satisfies` fails tsc if the API drops one.
const sorts = ['code', '-code', 'full_name', '-full_name', 'hire_date', '-hire_date'] as const satisfies readonly EmployeeSort[]
const defaults = { filters: { q: '', status: '', org_unit_id: '' }, sort: 'code', sorts: [...sorts], pageSize: 50 }

export function EmployeesPage() {
  const { t } = useTranslation()
  const actions = useEmployeeActions().data ?? []
  const canCreate = actions.includes('create')
  const [importing, setImporting] = useState(false)
  const [params, set, filterBar] = useListParams(defaults)
  const { q, status, org_unit_id } = params.filters as typeof defaults.filters
  const query = { q, status, org_unit_id, sort: params.sort, page: params.page, pageSize: params.pageSize }
  const list = useQuery({
    queryKey: hrmKeys.employees.list(query),
    queryFn: () =>
      unwrap(
        api.GET('/hrm/employees', {
          params: {
            query: {
              q: q || undefined,
              status: (status || undefined) as 'active' | 'terminated' | undefined,
              org_unit_id: org_unit_id ? Number(org_unit_id) : undefined,
              sort: params.sort as EmployeeSort,
              page: params.page,
              page_size: params.pageSize as 50,
            },
          },
        }),
      ),
    placeholderData: keepPreviousData,
  })

  const columns: Column<EmployeeListItem>[] = [
    { key: 'full_name', header: t('hrm.employee.full_name'), role: 'title', kind: 'person', sortable: true, render: (e) => <PersonName name={e.full_name} /> },
    { key: 'code', header: t('hrm.employee.code'), role: 'meta', kind: 'code', sortable: true, render: (e) => e.code },
    { key: 'org_unit', header: t('hrm.employee.org_unit'), role: 'meta', render: (e) => e.org_unit_name },
    { key: 'hire_date', header: t('hrm.employee.hire_date'), role: 'hidden', kind: 'date', sortable: true, render: (e) => formatDate(e.hire_date) },
    { key: 'status', header: t('hrm.employee.status'), role: 'status', kind: 'status', render: (e) => <EmployeeStatus status={e.status} /> },
  ]

  const view = useListView('hrm.employees', columns)
  let body
  if (list.isPending) body = <ContentSkeleton />
  else if (list.isError) body = <ErrorState message={errorText(t, list.error)} onRetry={() => void list.refetch()} />
  else if (list.data.items.length === 0)
    body = filterBar.active ? (
      <EmptyState
        title={t('hrm.employees.no_results')}
        action={
          <Button variant="default" onClick={filterBar.clear}>
            {t('hrm.common.clear_filters')}
          </Button>
        }
      />
    ) : (
      <EmptyState title={t('hrm.employees.empty')} description={t('hrm.employees.empty_description')} />
    )
  else
    body = (
      <DataTable
        label={t('hrm.employees.title')}
        columns={view.columns}
        onResize={view.onResize}
        rows={list.data.items}
        rowKey={(e) => e.id}
        href={(e) => `/hrm/employees/${e.id}`}
        sort={params.sort}
        onSort={(sort) => set({ sort })}
      />
    )

  return (
    <ListPage
      title={t('hrm.employees.title')}
      description={t('hrm.employees.description')}
      action={
        <Group gap="xs">
          {actions.includes('import_leave_balances') && (
            <Button variant="default" leftSection={<IconFileImport {...icon.button} />} onClick={() => setImporting(true)}>
              {t('hrm.employees.import_balances')}
            </Button>
          )}
          {canCreate && (
            <Button component={Link} to="/hrm/employees/new" leftSection={<IconPlus {...icon.button} />}>
              {t('hrm.employees.create')}
            </Button>
          )}
        </Group>
      }
      filters={
        <>
          <SearchInput label={t('hrm.employees.search')} placeholder={t('hrm.employees.search_hint')} value={q} onSearch={(v) => set({ filters: { q: v } })} />
          <FilterChip
            label={t('hrm.employee.status')}
            icon={IconCircleDot}
            data={[
              { value: 'active', label: t('hrm.employee.status.active') },
              { value: 'terminated', label: t('hrm.employee.status.terminated') },
            ]}
            {...filterBar.bind('status')}
          />
          <OrgUnitFilter
            label={t('hrm.employee.org_unit')}
            product="hrm"
            permission="hrm.employee.view"
            {...filterBar.bind('org_unit_id')}
          />
        </>
      }
      onClearFilters={filterBar.active ? filterBar.clear : undefined}
      tools={view.tools}
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
      {importing && (
        <ImportDialog
          kind="hrm.leave_balance"
          title={t('hrm.employees.import_balances')}
          description={t('hrm.employees.import_balances_hint')}
          invalidate={[hrmKeys.balances.all()]}
          onClose={() => setImporting(false)}
        />
      )}
    </ListPage>
  )
}

