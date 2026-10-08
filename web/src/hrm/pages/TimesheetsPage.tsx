import { Button, Select } from '@mantine/core'
import { IconCircleDot, IconPlus } from '@tabler/icons-react'
import { keepPreviousData, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router'
import { api, unwrap, type TimesheetListItem } from '@/shared/api/hrm'
import { DocumentStatus, useStatusLabel } from '@/shared/document'
import { errorText, formatMonth } from '@/shared/i18n'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { Form, FormActions, MonthField, MonthFilter, OrgUnitField, OrgUnitSelect } from '@/shared/ui/form'
import { FormModal, ListPage } from '@/shared/ui/page'
import { ContentSkeleton, EmptyState, ErrorState } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import { useListParams } from '@/shared/url/listParams'
import { useTimesheetActions } from '../hooks/useTimesheet'
import { hrmKeys, timesheetDocType } from '../keys'

const statuses = ['draft', 'pending_approval', 'posted', 'cancelled'] as const
const defaults = { filters: { month: '', org_unit_id: '', status: '' }, sort: '', sorts: [], pageSize: 50 }

export function TimesheetsPage() {
  const { t } = useTranslation()
  const statusLabel = useStatusLabel()
  const canCreate = useTimesheetActions().data?.includes('create') ?? false
  const [creating, setCreating] = useState(false)
  const [params, set] = useListParams(defaults)
  const { month, org_unit_id, status } = params.filters as typeof defaults.filters
  const query = { month, org_unit_id, status, page: params.page, pageSize: params.pageSize }
  const list = useQuery({
    queryKey: hrmKeys.timesheets.list(query),
    queryFn: () =>
      unwrap(
        api.GET('/hrm/timesheets', {
          params: {
            query: {
              month: month || undefined,
              org_unit_id: org_unit_id ? Number(org_unit_id) : undefined,
              status: (status || undefined) as (typeof statuses)[number] | undefined,
              page: params.page,
              page_size: params.pageSize as 50,
            },
          },
        }),
      ),
    placeholderData: keepPreviousData,
  })

  const columns: Column<TimesheetListItem>[] = [
    { key: 'number', header: t('hrm.timesheet.number'), role: 'title', render: (s) => s.number },
    { key: 'org_unit', header: t('hrm.timesheet.org_unit'), role: 'meta', render: (s) => s.org_unit_name },
    { key: 'month', header: t('hrm.timesheet.month'), role: 'meta', render: (s) => formatMonth(s.period_start.slice(0, 7)) },
    { key: 'status', header: t('hrm.timesheet.status'), role: 'status', render: (s) => <DocumentStatus docType={timesheetDocType} status={s.status} /> },
  ]

  const filtered = month !== '' || org_unit_id !== '' || status !== ''
  let body
  if (list.isPending) body = <ContentSkeleton />
  else if (list.isError) body = <ErrorState message={errorText(t, list.error)} onRetry={() => void list.refetch()} />
  else if (list.data.items.length === 0)
    body = filtered ? (
      <EmptyState
        title={t('hrm.timesheets.no_results')}
        action={
          <Button variant="default" onClick={() => set({ filters: defaults.filters })}>
            {t('hrm.common.clear_filters')}
          </Button>
        }
      />
    ) : (
      <EmptyState title={t('hrm.timesheets.empty')} description={canCreate ? t('hrm.timesheets.empty_description') : undefined} />
    )
  else
    body = (
      <DataTable label={t('hrm.timesheets.title')} columns={columns} rows={list.data.items} rowKey={(s) => s.id} href={(s) => `/hrm/timesheets/${s.id}`} />
    )

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
      filters={
        <>
          <MonthFilter label={t('hrm.timesheet.month')} value={month || null} onChange={(v) => set({ filters: { month: v ?? '' } })} />
          <OrgUnitSelect
            label={t('hrm.timesheet.org_unit')}
            product="hrm"
            permission="hrm.timesheet.view"
            value={org_unit_id || null}
            onChange={(v) => set({ filters: { org_unit_id: v ?? '' } })}
            clearable
          />
          <Select
            aria-label={t('hrm.timesheet.status')}
            leftSection={<IconCircleDot {...icon.text} />}
            w={{ base: '100%', md: 200 }}
            data={statuses.map((s) => ({ value: s, label: statusLabel(timesheetDocType, s) }))}
            value={status || null}
            onChange={(v) => set({ filters: { status: v ?? '' } })}
            placeholder={t('hrm.timesheet.status')}
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
      {creating && <NewTimesheet onClose={() => setCreating(false)} />}
    </ListPage>
  )
}

// NewTimesheet creates an empty draft for an org unit and month, then opens it.
function NewTimesheet({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const form = useForm<{ org_unit_id: string | null; month: string | null }>({ mode: 'onBlur', defaultValues: { org_unit_id: null, month: null } })

  async function create(v: { org_unit_id: string | null; month: string | null }) {
    const { id } = await unwrap(api.POST('/hrm/timesheets', { body: { org_unit_id: Number(v.org_unit_id), month: v.month ?? '' } }))
    await qc.invalidateQueries({ queryKey: hrmKeys.timesheets.lists() })
    navigate(`/hrm/timesheets/${id}`)
  }

  return (
    <FormModal opened title={t('hrm.timesheets.create')} onClose={onClose}>
      <Form form={form} onSubmit={create} fieldOf={(err) => (err.code === 'timesheet_exists' ? 'month' : undefined)}>
        <OrgUnitField name="org_unit_id" label={t('hrm.timesheet.org_unit')} product="hrm" permission="hrm.timesheet.edit" required />
        <MonthField name="month" label={t('hrm.timesheet.month')} description={t('hrm.timesheet.month_hint')} required />
        <FormActions submitLabel={t('hrm.common.create_draft')} onCancel={onClose} cancelLabel={t('hrm.common.cancel')} />
      </Form>
    </FormModal>
  )
}
