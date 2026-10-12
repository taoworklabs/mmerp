import { Alert, Button, Menu, Select } from '@mantine/core'
import { IconAlertCircle, IconArchive, IconArchiveOff, IconCircleDot, IconPencil, IconPlus } from '@tabler/icons-react'
import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { useCan, useProductOn } from '@/shared/auth/me'
import { api, unwrap, type Item, type ItemList, type ItemSort } from '@/shared/api/sales'
import { errorText, formatNumber } from '@/shared/i18n'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { CheckboxField, Form, FormActions, MoneyField, SearchInput, SelectField, TextField } from '@/shared/ui/form'
import { useListView } from '@/shared/ui/ListView'
import { notifySuccess, notifyUndo } from '@/shared/ui/notify'
import { FormModal, ListPage } from '@/shared/ui/page'
import { ContentSkeleton, EmptyState, ErrorState } from '@/shared/ui/states'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { filterWidth, icon } from '@/shared/ui/theme'
import { useListParams } from '@/shared/url/listParams'
import { useVatOptions } from '../components/vat'
import { salesKeys } from '../keys'

const sorts = ['code', '-code', 'name', '-name'] as const satisfies readonly ItemSort[]
const defaults = { filters: { q: '', active: '' }, sort: 'code', sorts: [...sorts], pageSize: 50 }

// ItemsPage: the tenant's catalogue of goods and services; catalogue admins edit it in a modal
// opened from the row, and retire or reactivate an item from its menu with a way back.
export function ItemsPage() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const vat = useVatOptions()
  // Catalogue writes have no allowed_actions; a disabled product is read-only.
  const canManage = useCan('sales.item.manage')
  const on = useProductOn('sales')
  const writable = canManage && on
  const [editing, setEditing] = useState<Item | 'new' | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [params, set] = useListParams(defaults)
  const { q, active } = params.filters as typeof defaults.filters
  const query = { q, active, sort: params.sort, page: params.page, pageSize: params.pageSize }
  const list = useQuery({
    queryKey: salesKeys.items.list(query),
    queryFn: () =>
      unwrap(
        api.GET('/sales/items', {
          params: {
            query: {
              q: q || undefined,
              active: (active || undefined) as 'true' | 'false' | undefined,
              sort: params.sort as ItemSort,
              page: params.page,
              page_size: params.pageSize as 50,
            },
          },
        }),
      ),
    placeholderData: keepPreviousData,
  })

  // Retiring is reversible and touches no document, so it applies at once and offers Undo.
  const toggle = useMutation({
    // undo marks the way back, which offers no Undo of its own.
    mutationFn: ({ item: x }: { item: Item; undo?: boolean }) => unwrap(api.PUT('/sales/items/{id}', { params: { path: { id: x.id } }, body: { ...fieldsOf(x), active: !x.active } })),
    onMutate: async ({ item: x }) => {
      setError(null)
      await qc.cancelQueries({ queryKey: salesKeys.items.all() })
      const before = qc.getQueriesData<ItemList>({ queryKey: salesKeys.items.all() })
      qc.setQueriesData<ItemList>({ queryKey: salesKeys.items.all() }, (old) =>
        old && { ...old, items: old.items.map((i) => (i.id === x.id ? { ...i, active: !x.active } : i)) },
      )
      return { before }
    },
    onError: (err, _x, ctx) => {
      ctx?.before.forEach(([key, data]) => qc.setQueryData(key, data))
      setError(errorText(t, err))
    },
    onSuccess: (_r, { item: x, undo }) => {
      if (undo) return
      notifyUndo(t(x.active ? 'sales.items.retired_done' : 'sales.items.reactivated_done', { name: x.name }), t('shared.undo'), () => toggle.mutate({ item: { ...x, active: !x.active }, undo: true }))
    },
    onSettled: () => qc.invalidateQueries({ queryKey: salesKeys.items.all() }),
  })

  const columns: Column<Item>[] = [
    { key: 'name', header: t('sales.item.name'), role: 'title', sortable: true, render: (x) => x.name },
    { key: 'code', header: t('sales.item.code'), role: 'meta', sortable: true, render: (x) => x.code },
    { key: 'unit', header: t('sales.item.unit'), role: 'hidden', render: (x) => x.unit },
    { key: 'price', header: t('sales.item.price'), role: 'meta', numeric: true, render: (x) => formatNumber(x.price) },
    { key: 'vat_rate', header: t('sales.item.vat_rate'), role: 'hidden', render: (x) => vat.label(x.vat_rate) },
    {
      key: 'active',
      header: t('sales.item.status'),
      role: 'status',
      render: (x) => <StatusBadge tone={x.active ? 'positive' : 'neutral'}>{t(x.active ? 'sales.item.in_use' : 'sales.item.retired')}</StatusBadge>,
    },
  ]
  const view = useListView('sales.items', columns)
  let body
  if (list.isPending) body = <ContentSkeleton />
  else if (list.isError) body = <ErrorState message={errorText(t, list.error)} onRetry={() => void list.refetch()} />
  else if (list.data.items.length === 0)
    body =
      q !== '' || active !== '' ? (
        <EmptyState
          title={t('sales.common.no_results')}
          action={
            <Button variant="default" onClick={() => set({ filters: defaults.filters })}>
              {t('sales.common.clear_filters')}
            </Button>
          }
        />
      ) : (
        <EmptyState title={t('sales.items.empty')} description={t('sales.items.empty_description')} />
      )
  else
    body = (
      <DataTable
        label={t('sales.items.title')}
        columns={view.columns}
        onResize={view.onResize}
        rows={list.data.items}
        rowKey={(x) => x.id}
        sort={params.sort}
        onSort={(sort) => set({ sort })}
        onRowClick={writable ? (x) => setEditing(x) : undefined}
        menu={
          writable
            ? (x) => (
                <>
                  <Menu.Item leftSection={<IconPencil {...icon.text} />} onClick={() => setEditing(x)}>
                    {t('sales.items.edit')}
                  </Menu.Item>
                  <Menu.Item leftSection={x.active ? <IconArchive {...icon.text} /> : <IconArchiveOff {...icon.text} />} onClick={() => toggle.mutate({ item: x })}>
                    {t(x.active ? 'sales.items.retire' : 'sales.items.reactivate')}
                  </Menu.Item>
                </>
              )
            : undefined
        }
      />
    )
  return (
    <ListPage
      title={t('sales.items.title')}
      description={t('sales.items.description')}
      action={
        writable && (
          <Button leftSection={<IconPlus {...icon.button} />} onClick={() => setEditing('new')}>
            {t('sales.items.create')}
          </Button>
        )
      }
      filters={
        <>
          <SearchInput label={t('sales.items.search')} placeholder={t('sales.items.search_hint')} value={q} onSearch={(v) => set({ filters: { q: v } })} />
          <Select
            aria-label={t('sales.item.status')}
            placeholder={t('sales.item.status')}
            leftSection={<IconCircleDot {...icon.text} />}
            w={{ base: '100%', md: filterWidth.select }}
            data={[
              { value: 'true', label: t('sales.item.in_use') },
              { value: 'false', label: t('sales.item.retired') },
            ]}
            value={active || null}
            onChange={(v) => set({ filters: { active: v ?? '' } })}
            clearable
          />
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
      {error && (
        <Alert color="danger" icon={<IconAlertCircle {...icon.button} />} role="alert">
          {error}
        </Alert>
      )}
      {body}
      {editing && <ItemModal item={editing === 'new' ? null : editing} onClose={() => setEditing(null)} />}
    </ListPage>
  )
}

// fieldsOf is what the API takes for an item: its editable fields.
const fieldsOf = (x: Item) => ({ code: x.code, name: x.name, unit: x.unit, price: x.price, vat_rate: x.vat_rate, active: x.active })

type Values = { code: string; name: string; unit: string; price: number | null; vat_rate: string; active: boolean }

function ItemModal({ item, onClose }: { item: Item | null; onClose: () => void }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const vat = useVatOptions()
  const form = useForm<Values>({ mode: 'onBlur', defaultValues: item ?? { code: '', name: '', unit: '', price: null, vat_rate: '10', active: true } })
  async function save(v: Values) {
    const body = {
      code: v.code.trim(),
      name: v.name.trim(),
      unit: v.unit.trim(),
      price: v.price ?? 0,
      vat_rate: v.vat_rate as Item['vat_rate'],
      active: v.active,
    }
    if (item) await unwrap(api.PUT('/sales/items/{id}', { params: { path: { id: item.id } }, body }))
    else await unwrap(api.POST('/sales/items', { body }))
    await qc.invalidateQueries({ queryKey: salesKeys.items.all() })
    notifySuccess(t('sales.items.saved'))
    onClose()
  }
  return (
    <FormModal opened title={t(item ? 'sales.items.edit' : 'sales.items.create')} onClose={onClose}>
      <Form form={form} onSubmit={save} fieldOf={(err) => (err.code === 'item_code_taken' ? 'code' : undefined)}>
        <TextField name="code" label={t('sales.item.code')} required maxLength={50} />
        <TextField name="name" label={t('sales.item.name')} required maxLength={300} />
        <TextField name="unit" label={t('sales.item.unit')} description={t('sales.item.unit_hint')} required maxLength={30} />
        <MoneyField name="price" label={t('sales.item.price')} description={t('sales.item.price_hint')} required />
        <SelectField name="vat_rate" label={t('sales.item.vat_rate')} required data={vat.options} />
        <CheckboxField name="active" label={t('sales.item.active')} description={t('sales.item.active_hint')} />
        <FormActions submitLabel={t('sales.common.save')} onCancel={onClose} cancelLabel={t('sales.common.cancel')} />
      </Form>
    </FormModal>
  )
}
