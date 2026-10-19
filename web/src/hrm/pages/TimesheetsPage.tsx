import { Button } from '@mantine/core'
import { IconPlus } from '@tabler/icons-react'
import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router'
import { api, unwrap, type TimesheetListItem, type TimesheetSort } from '@/shared/api/hrm'
import { useInvalidateRecord } from '@/shared/area'
import { DocumentStatus, DocumentStatusFilter, type DocStatus } from '@/shared/document'
import { formatMonthShort } from '@/shared/i18n'
import { type Column } from '@/shared/ui/DataTable'
import { Form, FormActions, MonthField, MonthFilter, OrgUnitField, OrgUnitFilter } from '@/shared/ui/form'
import { useListView } from '@/shared/ui/ListView'
import { notifySuccess } from '@/shared/ui/notify'
import { FormModal, ListPage } from '@/shared/ui/page'
import { icon } from '@/shared/ui/theme'
import { useListParams } from '@/shared/url/listParams'
import { useTimesheetActions } from '../hooks/useTimesheet'
import { hrmKeys, timesheetDocType } from '../keys'

const sorts = ['-period', 'period', '-number', 'number'] as const satisfies readonly TimesheetSort[]
const defaults = { filters: { month: '', org_unit_id: '', status: '' }, sort: '-period', sorts: [...sorts], pageSize: 50 }

export function TimesheetsPage() {
  const { t } = useTranslation()
  const canCreate = useTimesheetActions().data?.includes('create') ?? false
  const [creating, setCreating] = useState(false)
  const [params, set, filterBar] = useListParams(defaults)
  const { month, org_unit_id, status } = params.filters as typeof defaults.filters
  const query = { month, org_unit_id, status, sort: params.sort, page: params.page, pageSize: params.pageSize }
  const list = useQuery({
    queryKey: hrmKeys.timesheets.list(query),
    queryFn: () =>
      unwrap(
        api.GET('/hrm/timesheets', {
          params: {
            query: {
              month: month || undefined,
              org_unit_id: org_unit_id ? Number(org_unit_id) : undefined,
              status: (status || undefined) as DocStatus | undefined,
              sort: params.sort as TimesheetSort,
              page: params.page,
              page_size: params.pageSize as 50,
            },
          },
        }),
      ),
    placeholderData: keepPreviousData,
  })

  const columns: Column<TimesheetListItem>[] = [
    { key: 'number', header: t('hrm.timesheet.number'), role: 'title', kind: 'code', sortable: true, render: (s) => s.number },
    { key: 'org_unit', header: t('hrm.timesheet.org_unit'), role: 'meta', render: (s) => s.org_unit_name },
    { key: 'period', header: t('hrm.timesheet.month'), role: 'meta', kind: 'date', sortable: true, render: (s) => formatMonthShort(s.period_start.slice(0, 7)) },
    { key: 'status', header: t('hrm.timesheet.status'), role: 'status', kind: 'status', render: (s) => <DocumentStatus docType={timesheetDocType} status={s.status} /> },
  ]
  const view = useListView('hrm.timesheets', columns)

  return (
    <ListPage
      title={t('hrm.timesheets.title')}
      description={t('hrm.timesheets.description')}
      action={
        canCreate && (
          <Button leftSection={<IconPlus {...icon.button} />} onClick={() => setCreating(true)}>
            {t('hrm.timesheets.create')}
          </Button>
        )
      }
      onClearFilters={filterBar.active ? filterBar.clear : undefined}
      filters={
        <>
          <OrgUnitFilter
            label={t('hrm.timesheet.org_unit')}
            product="hrm"
            permission="hrm.timesheet.view"
            {...filterBar.bind('org_unit_id')}
          />
          <DocumentStatusFilter docType={timesheetDocType} {...filterBar.bind('status')} />
          <MonthFilter label={t('hrm.timesheet.month')} {...filterBar.bind('month')} />
        </>
      }
      tools={view.tools}
      list={{
        query: list,
        table: {
          label: t('hrm.timesheets.title'),
          columns: view.columns,
          onResize: view.onResize,
          rowKey: (s) => s.id,
          href: (s) => `/hrm/timesheets/${s.id}`,
          sort: params.sort,
          onSort: (sort) => set({ sort }),
        },
        empty: { title: t('hrm.timesheets.empty'), description: t('hrm.timesheets.empty_description') },
        noResults: t('hrm.timesheets.no_results'),
        paging: { page: params.page, pageSize: params.pageSize, onPage: (page) => set({ page }), onPageSize: (pageSize) => set({ pageSize }) },
      }}
    >
      {creating && <NewTimesheet onClose={() => setCreating(false)} />}
    </ListPage>
  )
}

// NewTimesheet creates an empty draft for an org unit and month, then opens it.
function NewTimesheet({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation()
  // One id per opened form: sending it twice returns the document made the first time.
  const [requestId] = useState(() => crypto.randomUUID())
  const navigate = useNavigate()
  const invalidate = useInvalidateRecord()
  const form = useForm<{ org_unit_id: string | null; month: string | null }>({ mode: 'onBlur', defaultValues: { org_unit_id: null, month: null } })

  async function create(v: { org_unit_id: string | null; month: string | null }) {
    const { id } = await unwrap(api.POST('/hrm/timesheets', { body: { request_id: requestId, org_unit_id: Number(v.org_unit_id), month: v.month ?? '' } }))
    await invalidate(timesheetDocType, id)
    notifySuccess(t('hrm.timesheets.created'))
    navigate(`/hrm/timesheets/${id}`)
  }

  return (
    <FormModal opened title={t('hrm.timesheets.create')} onClose={onClose}>
      <Form form={form} onSubmit={create} errorFields={{ timesheet_exists: 'month' }}>
        <OrgUnitField name="org_unit_id" label={t('hrm.timesheet.org_unit')} product="hrm" permission="hrm.timesheet.edit" required />
        <MonthField name="month" label={t('hrm.timesheet.month')} description={t('hrm.timesheet.month_hint')} required />
        <FormActions submitLabel={t('hrm.common.create_draft')} onCancel={onClose} cancelLabel={t('hrm.common.cancel')} />
      </Form>
    </FormModal>
  )
}
