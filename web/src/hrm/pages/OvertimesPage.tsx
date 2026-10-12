import { Button, Select } from '@mantine/core'
import { IconCircleDot, IconPlus } from '@tabler/icons-react'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { api, unwrap, type OvertimeListItem, type OvertimeSort } from '@/shared/api/hrm'
import { DocumentStatus, useStatusLabel } from '@/shared/document'
import { errorText, formatDate, formatDecimal } from '@/shared/i18n'
import { PersonName } from '@/shared/ui/avatar'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { MonthFilter, OrgUnitSelect, SearchInput } from '@/shared/ui/form'
import { useListView } from '@/shared/ui/ListView'
import { ListPage } from '@/shared/ui/page'
import { SidePanel, usePeek } from '@/shared/ui/SidePanel'
import { ContentSkeleton, EmptyState, ErrorState } from '@/shared/ui/states'
import { filterWidth, icon } from '@/shared/ui/theme'
import { useListParams } from '@/shared/url/listParams'
import OvertimePreview from '../components/OvertimePreview'
import { useOvertimeActions } from '../hooks/useOvertime'
import { hrmKeys, overtimeDocType } from '../keys'
import { monthBounds } from '../month'

const sorts = ['-date', 'date', '-number', 'number'] as const satisfies readonly OvertimeSort[]
const statuses = ['draft', 'pending_approval', 'posted', 'cancelled'] as const
const defaults = { filters: { q: '', status: '', org_unit: '', month: '' }, sort: '-date', sorts: [...sorts], pageSize: 50 }

// OvertimesPage lists the overtime requests the user may see; a row opens its preview beside the list.
export function OvertimesPage() {
  const { t } = useTranslation()
  const statusLabel = useStatusLabel()
  const canCreate = useOvertimeActions().data?.allowed_actions.includes('create') ?? false
  const [params, set] = useListParams(defaults)
  const peek = usePeek()
  const { q, status, org_unit, month } = params.filters as typeof defaults.filters
  const query = { q, status, orgUnit: org_unit, month, sort: params.sort, page: params.page, pageSize: params.pageSize }
  const list = useQuery({
    queryKey: hrmKeys.overtimes.list(query),
    queryFn: () =>
      unwrap(
        api.GET('/hrm/overtimes', {
          params: {
            query: {
              q: q || undefined,
              status: (status || undefined) as (typeof statuses)[number] | undefined,
              org_unit_id: org_unit ? Number(org_unit) : undefined,
              ...monthBounds(month),
              sort: params.sort as OvertimeSort,
              page: params.page,
              page_size: params.pageSize as 50,
            },
          },
        }),
      ),
    placeholderData: keepPreviousData,
  })

  const columns: Column<OvertimeListItem>[] = [
    { key: 'number', header: t('hrm.overtime.number'), role: 'title', sortable: true, render: (o) => o.number },
    { key: 'employee', header: t('hrm.overtime.employee'), role: 'meta', render: (o) => <PersonName name={o.employee_name} /> },
    { key: 'date', header: t('hrm.overtime.date'), role: 'meta', sortable: true, render: (o) => formatDate(o.date) },
    { key: 'day_kind', header: t('hrm.overtime.day_kind'), role: 'hidden', render: (o) => t(`hrm.overtime.day_kind.${o.day_kind}`) },
    { key: 'day_hours', header: t('hrm.overtime.day_hours'), role: 'hidden', numeric: true, render: (o) => formatDecimal(o.day_hours) },
    { key: 'night_hours', header: t('hrm.overtime.night_hours'), role: 'hidden', numeric: true, render: (o) => formatDecimal(o.night_hours) },
    { key: 'status', header: t('hrm.overtime.status'), role: 'status', render: (o) => <DocumentStatus docType={overtimeDocType} status={o.status} /> },
  ]
  const view = useListView('hrm.overtimes', columns)
  const open = list.data?.items.find((l) => String(l.id) === peek.id)

  let body
  if (list.isPending) body = <ContentSkeleton />
  else if (list.isError) body = <ErrorState message={errorText(t, list.error)} onRetry={() => void list.refetch()} />
  else if (list.data.items.length === 0)
    body =
      q !== '' || status !== '' || org_unit !== '' || month !== '' ? (
        <EmptyState
          title={t('hrm.overtimes.no_results')}
          action={
            <Button variant="default" onClick={() => set({ filters: defaults.filters })}>
              {t('hrm.common.clear_filters')}
            </Button>
          }
        />
      ) : (
        <EmptyState title={t('hrm.overtimes.empty')} description={t('hrm.overtimes.empty_description')} />
      )
  else
    body = (
      <DataTable
        label={t('hrm.overtimes.title')}
        columns={view.columns}
        onResize={view.onResize}
        rows={list.data.items}
        rowKey={(l) => l.id}
        href={(l) => `/hrm/overtimes/${l.id}`}
        onRowClick={(l) => peek.open(l.id)}
        sort={params.sort}
        onSort={(sort) => set({ sort })}
      />
    )

  return (
    <ListPage
      title={t('hrm.overtimes.title')}
      description={t('hrm.overtimes.description')}
      action={
        canCreate && (
          <Button component={Link} to="/hrm/overtimes/new" leftSection={<IconPlus {...icon.button} />}>
            {t('hrm.overtimes.create')}
          </Button>
        )
      }
      filters={
        <>
          <SearchInput label={t('hrm.overtimes.search')} placeholder={t('hrm.common.search_request_hint')} value={q} onSearch={(v) => set({ filters: { q: v } })} />
          <Select
            aria-label={t('hrm.overtime.status')}
            placeholder={t('hrm.overtime.status')}
            leftSection={<IconCircleDot {...icon.text} />}
            w={{ base: '100%', md: filterWidth.select }}
            data={statuses.map((s) => ({ value: s, label: statusLabel(overtimeDocType, s) }))}
            value={status || null}
            onChange={(v) => set({ filters: { status: v ?? '' } })}
            clearable
          />
          <OrgUnitSelect
            product="hrm"
            permission="hrm.overtime.view"
            label={t('hrm.common.org_unit')}
            value={org_unit || null}
            onChange={(v) => set({ filters: { org_unit: v ?? '' } })}
            clearable
          />
          <MonthFilter label={t('hrm.common.month')} value={month || null} onChange={(v) => set({ filters: { month: v ?? '' } })} />
        </>
      }
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
      <SidePanel title={open?.number ?? ''} href={`/hrm/overtimes/${open?.id ?? ''}`} opened={!!open} onClose={peek.close}>
        {open && <OvertimePreview id={open.id} />}
      </SidePanel>
    </ListPage>
  )
}
