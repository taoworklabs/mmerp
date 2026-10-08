import { Button, Select } from '@mantine/core'
import { IconBuildingSkyscraper, IconCircleDot, IconPlus } from '@tabler/icons-react'
import { keepPreviousData, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router'
import { api, unwrap, type PayrollListItem } from '@/shared/api/hrm'
import { DocumentStatus, useStatusLabel } from '@/shared/document'
import { errorText, formatMonth } from '@/shared/i18n'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { Form, FormActions, MonthField, MonthFilter, SelectField, useOrgUnits } from '@/shared/ui/form'
import { FormModal, ListPage } from '@/shared/ui/page'
import { ContentSkeleton, EmptyState, ErrorState } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import { useListParams } from '@/shared/url/listParams'
import { usePayrollActions } from '../hooks/usePayroll'
import { hrmKeys, payrollDocType } from '../keys'

const statuses = ['draft', 'pending_approval', 'posted', 'cancelled'] as const
const defaults = { filters: { month: '', legal_entity_id: '', status: '' }, sort: '', sorts: [], pageSize: 50 }

// useLegalEntities lists the legal entities where the user holds a payroll permission.
function useLegalEntities(permission: string) {
  const units = useOrgUnits({ product: 'hrm', permission })
  return (units.data ?? []).filter((u) => u.kind === 'company').map((u) => ({ value: String(u.id), label: u.name }))
}

export function PayrollsPage() {
  const { t } = useTranslation()
  const statusLabel = useStatusLabel()
  const canCreate = usePayrollActions().data?.includes('create') ?? false
  const entities = useLegalEntities('hrm.payroll.view')
  const [creating, setCreating] = useState(false)
  const [params, set] = useListParams(defaults)
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
              status: (status || undefined) as (typeof statuses)[number] | undefined,
              page: params.page,
              page_size: params.pageSize as 50,
            },
          },
        }),
      ),
    placeholderData: keepPreviousData,
  })

  const columns: Column<PayrollListItem>[] = [
    { key: 'number', header: t('hrm.payroll.number'), role: 'title', render: (p) => p.number },
    { key: 'legal_entity', header: t('hrm.payroll.legal_entity'), role: 'meta', render: (p) => p.legal_entity_name },
    { key: 'month', header: t('hrm.payroll.month'), role: 'meta', render: (p) => formatMonth(p.period_start.slice(0, 7)) },
    { key: 'status', header: t('hrm.payroll.status'), role: 'status', render: (p) => <DocumentStatus docType={payrollDocType} status={p.status} /> },
  ]

  const filtered = month !== '' || legal_entity_id !== '' || status !== ''
  let body
  if (list.isPending) body = <ContentSkeleton />
  else if (list.isError) body = <ErrorState message={errorText(t, list.error)} onRetry={() => void list.refetch()} />
  else if (list.data.items.length === 0)
    body = filtered ? (
      <EmptyState
        title={t('hrm.payrolls.no_results')}
        action={
          <Button variant="default" onClick={() => set({ filters: defaults.filters })}>
            {t('hrm.common.clear_filters')}
          </Button>
        }
      />
    ) : (
      <EmptyState title={t('hrm.payrolls.empty')} description={canCreate ? t('hrm.payrolls.empty_description') : undefined} />
    )
  else body = <DataTable label={t('hrm.payrolls.title')} columns={columns} rows={list.data.items} rowKey={(p) => p.id} href={(p) => `/hrm/payrolls/${p.id}`} />

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
      filters={
        <>
          <MonthFilter label={t('hrm.payroll.month')} value={month || null} onChange={(v) => set({ filters: { month: v ?? '' } })} />
          <Select
            aria-label={t('hrm.payroll.legal_entity')}
            placeholder={t('hrm.payroll.legal_entity')}
            leftSection={<IconBuildingSkyscraper {...icon.text} />}
            w={{ base: '100%', md: 240 }}
            data={entities}
            value={legal_entity_id || null}
            onChange={(v) => set({ filters: { legal_entity_id: v ?? '' } })}
            clearable
          />
          <Select
            aria-label={t('hrm.payroll.status')}
            leftSection={<IconCircleDot {...icon.text} />}
            w={{ base: '100%', md: 200 }}
            data={statuses.map((s) => ({ value: s, label: statusLabel(payrollDocType, s) }))}
            value={status || null}
            onChange={(v) => set({ filters: { status: v ?? '' } })}
            placeholder={t('hrm.payroll.status')}
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
