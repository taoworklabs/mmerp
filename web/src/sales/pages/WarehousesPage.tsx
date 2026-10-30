import { Button, Group } from '@mmerp/ui'
import { IconCircleDot, IconPlus } from '@tabler/icons-react'
import { keepPreviousData, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type Warehouse, type WarehouseSort } from '@/shared/api/core'
import { useCan } from '@/shared/auth/me'
import { type Column } from '@/shared/ui/DataTable'
import { CheckboxField, CodeField, FilterChip, Form, FormActions, LegalEntityField, SearchInput, SelectField, TextField, useLegalEntities, type Option } from '@/shared/ui/form'
import { useListView } from '@/shared/ui/ListView'
import { notifySuccess } from '@/shared/ui/notify'
import { FormModal, ListPage } from '@/shared/ui/page'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { icon } from '@/shared/ui/theme'
import { useListParams } from '@/shared/url/listParams'
import { salesKeys } from '../keys'

const sorts = ['code', '-code', 'name', '-name'] as const satisfies readonly WarehouseSort[]
const defaults = { filters: { q: '', active: '' }, sort: 'code', sorts: [...sorts], pageSize: 50 }

// WarehousesPage: the warehouses of every legal entity; stock managers add and edit them in a modal.
export function WarehousesPage() {
  const { t } = useTranslation()
  const writable = useCan('stock.manage')
  // Every company, to name a warehouse's legal entity.
  const entities = useLegalEntities({}).options
  const entityName = (id: number) => entities.find((o) => o.value === String(id))?.label ?? ''
  const [editing, setEditing] = useState<Warehouse | 'new' | null>(null)
  const [params, set, filterBar] = useListParams(defaults)
  const { q, active } = params.filters as typeof defaults.filters
  const query = { q, active, sort: params.sort, page: params.page, pageSize: params.pageSize }
  const list = useQuery({
    queryKey: salesKeys.stock.warehouses.list(query),
    queryFn: () =>
      unwrap(
        api.GET('/stock/warehouses', {
          params: {
            query: {
              q: q || undefined,
              active: (active || undefined) as 'true' | 'false' | undefined,
              sort: params.sort as WarehouseSort,
              page: params.page,
              page_size: params.pageSize as 50,
            },
          },
        }),
      ),
    placeholderData: keepPreviousData,
  })

  const columns: Column<Warehouse>[] = [
    {
      key: 'name',
      header: t('sales.warehouse.name'),
      role: 'title',
      sortable: true,
      render: (x) => (
        <Group gap="xs" wrap={false}>
          {x.name}
          {x.is_default && <StatusBadge tone="neutral">{t('sales.warehouse.default')}</StatusBadge>}
        </Group>
      ),
    },
    { key: 'code', header: t('sales.warehouse.code'), role: 'meta', kind: 'code', sortable: true, render: (x) => x.code },
    { key: 'legal_entity', header: t('sales.warehouse.legal_entity'), role: 'meta', render: (x) => entityName(x.legal_entity_id) },
    {
      key: 'active',
      header: t('sales.warehouse.status'),
      role: 'status',
      kind: 'status',
      render: (x) => <StatusBadge tone={x.active ? 'positive' : 'neutral'}>{t(x.active ? 'sales.warehouse.in_use' : 'sales.warehouse.retired')}</StatusBadge>,
    },
  ]
  const view = useListView('sales.warehouses', columns)
  return (
    <ListPage
      title={t('sales.warehouses.title')}
      description={t('sales.warehouses.description')}
      action={
        writable && (
          <Button leftSection={<IconPlus {...icon.button} />} onClick={() => setEditing('new')}>
            {t('sales.warehouses.create')}
          </Button>
        )
      }
      onClearFilters={filterBar.active ? filterBar.clear : undefined}
      filters={
        <>
          <SearchInput label={t('sales.warehouses.search')} placeholder={t('sales.warehouses.search_hint')} value={q} onSearch={(v) => set({ filters: { q: v } })} />
          <FilterChip
            label={t('sales.warehouse.status')}
            icon={IconCircleDot}
            data={[
              { value: 'true', label: t('sales.warehouse.in_use') },
              { value: 'false', label: t('sales.warehouse.retired') },
            ]}
            {...filterBar.bind('active')}
          />
        </>
      }
      tools={view.tools}
      list={{
        query: list,
        table: {
          label: t('sales.warehouses.title'),
          columns: view.columns,
          onResize: view.onResize,
          rowKey: (x) => x.id,
          sort: params.sort,
          onSort: (sort) => set({ sort }),
          onRowClick: writable ? (x) => setEditing(x) : undefined,
        },
        empty: { title: t('sales.warehouses.empty'), description: t('sales.warehouses.empty_description') },
        noResults: t('sales.common.no_results'),
        paging: { page: params.page, pageSize: params.pageSize, onPage: (page) => set({ page }), onPageSize: (pageSize) => set({ pageSize }) },
      }}
    >
      {editing && <WarehouseModal warehouse={editing === 'new' ? null : editing} entities={entities} onClose={() => setEditing(null)} />}
    </ListPage>
  )
}

type Values = { code: string; name: string; legal_entity_id: string; active: boolean }

function WarehouseModal({ warehouse: w, entities, onClose }: { warehouse: Warehouse | null; entities: Option[]; onClose: () => void }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const form = useForm<Values>({
    mode: 'onBlur',
    defaultValues: w ? { ...w, legal_entity_id: String(w.legal_entity_id) } : { code: '', name: '', legal_entity_id: '', active: true },
  })
  async function save(v: Values) {
    const body = { name: v.name.trim(), active: v.active }
    if (w) await unwrap(api.PUT('/stock/warehouses/{id}', { params: { path: { id: w.id } }, body }))
    else await unwrap(api.POST('/stock/warehouses', { body: { ...body, code: v.code.trim(), legal_entity_id: Number(v.legal_entity_id) } }))
    await qc.invalidateQueries({ queryKey: salesKeys.stock.warehouses.all() })
    notifySuccess(t('sales.warehouses.saved'))
    onClose()
  }
  return (
    <FormModal opened title={t(w ? 'sales.warehouses.edit' : 'sales.warehouses.create')} onClose={onClose}>
      <Form form={form} onSubmit={save} errorFields={{ warehouse_code_taken: 'code' }}>
        <CodeField name="code" label={t('sales.warehouse.code')} existing={!!w} />
        <TextField name="name" label={t('sales.warehouse.name')} required maxLength={300} />
        {/* A warehouse stays with the legal entity it was created for. */}
        {w ? (
          <SelectField name="legal_entity_id" label={t('sales.warehouse.legal_entity')} data={entities} readOnly />
        ) : (
          <LegalEntityField name="legal_entity_id" label={t('sales.warehouse.legal_entity')} product="stock" permission="stock.manage" required />
        )}
        <CheckboxField name="active" label={t('sales.warehouse.active')} description={t('sales.warehouse.active_hint')} />
        <FormActions submitLabel={t('sales.common.save')} onCancel={onClose} cancelLabel={t('sales.common.cancel')} />
      </Form>
    </FormModal>
  )
}
