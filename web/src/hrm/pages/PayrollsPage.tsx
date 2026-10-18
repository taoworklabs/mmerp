import { Button } from '@mantine/core'
import { IconBuildingSkyscraper, IconPlus } from '@tabler/icons-react'
import { keepPreviousData, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router'
import { api, unwrap, type PayrollListItem } from '@/shared/api/hrm'
import { DocumentStatus, DocumentStatusFilter, type DocStatus } from '@/shared/document'
import { formatMonthShort } from '@/shared/i18n'
import { type Column } from '@/shared/ui/DataTable'
import { FilterChip, Form, FormActions, MonthField, MonthFilter, SelectField, useOrgUnits } from '@/shared/ui/form'
import { useListView } from '@/shared/ui/ListView'
import { notifySuccess } from '@/shared/ui/notify'
import { FormModal, ListPage } from '@/shared/ui/page'
import { icon } from '@/shared/ui/theme'
import { useListParams } from '@/shared/url/listParams'
import { usePayrollActions } from '../hooks/usePayroll'
import { hrmKeys, payrollDocType } from '../keys'

const defaults = { filters: { month: '', legal_entity_id: '', status: '' }, sort: '', sorts: [], pageSize: 50 }

// useLegalEntities lists the legal entities where the user holds a payroll permission.
function useLegalEntities(permission: string) {
  const units = useOrgUnits({ product: 'hrm', permission })
  return (units.data ?? []).filter((u) => u.kind === 'company').map((u) => ({ value: String(u.id), label: u.name }))
}

export function PayrollsPage() {
  const { t } = useTranslation()
  const canCreate = usePayrollActions().data?.includes('create') ?? false
  const entities = useLegalEntities('hrm.payroll.view')
  const [creating, setCreating] = useState(false)
  const [params, set, filterBar] = useListParams(defaults)
  const { month, legal_entity_id, status } = params.filters as typeof defaults.filters
  const query = { month, legal_entity_id, status, page: params.page, pageSize: params.pageSize }
  const list = useQuery({
    queryKey: hrmKeys.payrolls.list(query),
    queryFn: () =>
      unwrap(
        api.GET('/hrm/payrolls', {
          params: {
            query: {
              month: month || undefined,
              legal_entity_id: legal_entity_id ? Number(legal_entity_id) : undefined,
              status: (status || undefined) as DocStatus | undefined,
              page: params.page,
              page_size: params.pageSize as 50,
            },
          },
        }),
      ),
    placeholderData: keepPreviousData,
  })

  const columns: Column<PayrollListItem>[] = [
    { key: 'number', header: t('hrm.payroll.number'), role: 'title', kind: 'code', render: (p) => p.number },
    { key: 'legal_entity', header: t('hrm.payroll.legal_entity'), role: 'meta', render: (p) => p.legal_entity_name },
    { key: 'month', header: t('hrm.payroll.month'), role: 'meta', kind: 'date', render: (p) => formatMonthShort(p.period_start.slice(0, 7)) },
    { key: 'status', header: t('hrm.payroll.status'), role: 'status', kind: 'status', render: (p) => <DocumentStatus docType={payrollDocType} status={p.status} /> },
  ]
  const view = useListView('hrm.payrolls', columns)

  return (
    <ListPage
      title={t('hrm.payrolls.title')}
      description={t('hrm.payrolls.description')}
      action={
        canCreate && (
          <Button leftSection={<IconPlus {...icon.button} />} onClick={() => setCreating(true)}>
            {t('hrm.payrolls.create')}
          </Button>
        )
      }
      onClearFilters={filterBar.active ? filterBar.clear : undefined}
      filters={
        <>
          <FilterChip
            label={t('hrm.payroll.legal_entity')}
            icon={IconBuildingSkyscraper}
            data={entities}
            {...filterBar.bind('legal_entity_id')}
          />
          <DocumentStatusFilter docType={payrollDocType} {...filterBar.bind('status')} />
          <MonthFilter label={t('hrm.payroll.month')} {...filterBar.bind('month')} />
        </>
      }
      tools={view.tools}
      list={{
        query: list,
        table: {
          label: t('hrm.payrolls.title'),
          columns: view.columns,
          onResize: view.onResize,
          rowKey: (p) => p.id,
          href: (p) => `/hrm/payrolls/${p.id}`,
        },
        empty: { title: t('hrm.payrolls.empty'), description: t('hrm.payrolls.empty_description') },
        noResults: t('hrm.payrolls.no_results'),
        paging: { page: params.page, pageSize: params.pageSize, onPage: (page) => set({ page }), onPageSize: (pageSize) => set({ pageSize }) },
      }}
    >
      {creating && <NewPayroll onClose={() => setCreating(false)} />}
    </ListPage>
  )
}

// NewPayroll creates a draft for a legal entity and month, then opens it while the job computes it.
function NewPayroll({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const entities = useLegalEntities('hrm.payroll.edit')
  const form = useForm<{ legal_entity_id: string | null; month: string | null }>({
    mode: 'onBlur',
    defaultValues: { legal_entity_id: entities.length === 1 ? entities[0]!.value : null, month: null },
  })

  async function create(v: { legal_entity_id: string | null; month: string | null }) {
    const { id, job_id } = await unwrap(api.POST('/hrm/payrolls', { body: { legal_entity_id: Number(v.legal_entity_id), month: v.month ?? '' } }))
    await qc.invalidateQueries({ queryKey: hrmKeys.payrolls.lists() })
    notifySuccess(t('hrm.payrolls.created'))
    navigate(`/hrm/payrolls/${id}?job=${job_id}`)
  }

  return (
    <FormModal opened title={t('hrm.payrolls.create')} onClose={onClose}>
      <Form form={form} onSubmit={create} fieldOf={(err) => (err.code === 'payroll_timesheets_missing' ? 'month' : undefined)}>
        <SelectField name="legal_entity_id" label={t('hrm.payroll.legal_entity')} data={entities} required />
        <MonthField name="month" label={t('hrm.payroll.month')} description={t('hrm.payroll.month_hint')} required />
        <FormActions submitLabel={t('hrm.payrolls.create_compute')} onCancel={onClose} cancelLabel={t('hrm.common.cancel')} />
      </Form>
    </FormModal>
  )
}
