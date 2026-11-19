import { Button } from '@mmerp/ui'
import { IconArchive, IconArchiveOff, IconCircleDot, IconPencil, IconPlus } from '@tabler/icons-react'
import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type Item, type ItemList, type ItemSort } from '@/shared/api/core'
import { useCan } from '@/shared/auth/me'
import { errorText, formatNumber } from '@/shared/i18n'
import { type Column } from '@/shared/ui/DataTable'
import { CheckboxField, CodeField, FilterChip, Form, FormActions, MoneyField, SearchInput, SelectField, TextField } from '@/shared/ui/form'
import { useListView } from '@/shared/ui/ListView'
import { MenuAction } from '@/shared/ui/MenuAction'
import { notifySuccess, notifyUndo } from '@/shared/ui/notify'
import { FormModal, ListPage } from '@/shared/ui/page'
import { ErrorState } from '@/shared/ui/states'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { icon } from '@/shared/ui/theme'
import { useListParams } from '@/shared/url/listParams'
import { useVatOptions } from '../components/vat'
import { stockKeys } from '../keys'

const sorts = ['code', '-code', 'name', '-name'] as const satisfies readonly ItemSort[]
const trackings = ['none', 'quantity', 'lot', 'serial'] as const satisfies readonly Item['tracking'][]
const defaults = { filters: { q: '', active: '' }, sort: 'code', sorts: [...sorts], pageSize: 50 }

// ItemsPage: the tenant's catalogue of goods and services; catalogue admins edit it in a modal
// opened from the row, and retire or reactivate an item from its menu with a way back.
export function ItemsPage() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const vat = useVatOptions()
  // Catalogue writes have no allowed_actions; items belong to no product, so nothing gates them.
  const writable = useCan('item.manage')
  const [editing, setEditing] = useState<Item | 'new' | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [params, set, filterBar] = useListParams(defaults)
  const { q, active } = params.filters as typeof defaults.filters
  const query = { q, active, sort: params.sort, page: params.page, pageSize: params.pageSize }
  const list = useQuery({
    queryKey: stockKeys.items.list(query),
    queryFn: () =>
      unwrap(
        api.GET('/items', {
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
    mutationFn: ({ item: x }: { item: Item; undo?: boolean }) => unwrap(api.PUT('/items/{id}', { params: { path: { id: x.id } }, body: { ...fieldsOf(x), active: !x.active } })),
    onMutate: async ({ item: x }) => {
      setError(null)
      await qc.cancelQueries({ queryKey: stockKeys.items.all() })
      const before = qc.getQueriesData<ItemList>({ queryKey: stockKeys.items.all() })
      qc.setQueriesData<ItemList>({ queryKey: stockKeys.items.all() }, (old) =>
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
      notifyUndo(t(x.active ? 'stock.items.retired_done' : 'stock.items.reactivated_done', { name: x.name }), t('shared.undo'), () => toggle.mutate({ item: { ...x, active: !x.active }, undo: true }))
    },
    onSettled: () => qc.invalidateQueries({ queryKey: stockKeys.items.all() }),
  })

  const columns: Column<Item>[] = [
    { key: 'name', header: t('stock.item.name'), role: 'title', sortable: true, render: (x) => x.name },
    { key: 'code', header: t('stock.item.code'), role: 'meta', kind: 'code', sortable: true, render: (x) => x.code },
    { key: 'unit', header: t('stock.item.unit'), role: 'hidden', kind: 'code', render: (x) => x.unit },
    { key: 'price', header: t('stock.item.price'), role: 'meta', kind: 'number', render: (x) => formatNumber(x.price) },
    { key: 'tracking', header: t('stock.item.tracking'), role: 'meta', render: (x) => t(`stock.item.tracking_${x.tracking}`) },
    { key: 'vat_rate', header: t('stock.item.vat_rate'), role: 'hidden', kind: 'number', render: (x) => vat.label(x.vat_rate) },
    {
      key: 'active',
      header: t('stock.item.status'),
      role: 'status',
      kind: 'status',
      render: (x) => <StatusBadge tone={x.active ? 'success' : 'neutral'}>{t(x.active ? 'stock.item.in_use' : 'stock.item.retired')}</StatusBadge>,
    },
  ]
  const view = useListView('stock.items', columns)
  return (
    <ListPage
      title={t('stock.items.title')}
      description={t('stock.items.description')}
      action={
        writable && (
          <Button leftSection={<IconPlus {...icon.button} />} onClick={() => setEditing('new')}>
            {t('stock.items.create')}
          </Button>
        )
      }
      onClearFilters={filterBar.active ? filterBar.clear : undefined}
      filters={
        <>
          <SearchInput label={t('stock.items.search')} placeholder={t('stock.items.search_hint')} value={q} onSearch={(v) => set({ filters: { q: v } })} />
          <FilterChip
            label={t('stock.item.status')}
            icon={IconCircleDot}
            data={[
              { value: 'true', label: t('stock.item.in_use') },
              { value: 'false', label: t('stock.item.retired') },
            ]}
            {...filterBar.bind('active')}
          />
        </>
      }
      tools={view.tools}
      list={{
        query: list,
        table: {
          label: t('stock.items.title'),
          columns: view.columns,
          onResize: view.onResize,
          rowKey: (x) => x.id,
          sort: params.sort,
          onSort: (sort) => set({ sort }),
          onRowClick: writable ? (x) => setEditing(x) : undefined,
          menu: writable
            ? (x) => (
                <>
                  <MenuAction icon={IconPencil} onClick={() => setEditing(x)}>
                    {t('stock.items.edit')}
                  </MenuAction>
                  <MenuAction icon={x.active ? IconArchive : IconArchiveOff} onClick={() => toggle.mutate({ item: x })}>
                    {t(x.active ? 'stock.items.retire' : 'stock.items.reactivate')}
                  </MenuAction>
                </>
              )
            : undefined,
        },
        empty: { title: t('stock.items.empty'), description: t('stock.items.empty_description') },
        noResults: t('stock.common.no_results'),
        paging: { page: params.page, pageSize: params.pageSize, onPage: (page) => set({ page }), onPageSize: (pageSize) => set({ pageSize }) },
      }}
    >
      {error && <ErrorState message={error} />}
      {editing && <ItemModal item={editing === 'new' ? null : editing} onClose={() => setEditing(null)} />}
    </ListPage>
  )
}

// fieldsOf is what the API takes for an item: its editable fields; tracking left out stays as it is.
const fieldsOf = (x: Item) => ({ name: x.name, unit: x.unit, price: x.price, vat_rate: x.vat_rate, active: x.active })

type Values = { code: string; name: string; unit: string; price: number | null; vat_rate: string; tracking: Item['tracking']; allow_negative: boolean; active: boolean }

function ItemModal({ item, onClose }: { item: Item | null; onClose: () => void }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const vat = useVatOptions()
  const form = useForm<Values>({
    mode: 'onBlur',
    defaultValues: item ?? { code: '', name: '', unit: '', price: null, vat_rate: '10', tracking: 'none', allow_negative: false, active: true },
  })
  const tracking = useWatch({ control: form.control, name: 'tracking' })
  async function save(v: Values) {
    const body = {
      name: v.name.trim(),
      unit: v.unit.trim(),
      price: v.price ?? 0,
      vat_rate: v.vat_rate as Item['vat_rate'],
      tracking: v.tracking,
      // Only a quantity-tracked item may go below zero.
      allow_negative: v.tracking === 'quantity' && v.allow_negative,
      active: v.active,
    }
    if (item) await unwrap(api.PUT('/items/{id}', { params: { path: { id: item.id } }, body }))
    else await unwrap(api.POST('/items', { body: { ...body, code: v.code.trim() } }))
    // The pickers too: an item's name and price fill new lines.
    await qc.invalidateQueries({ queryKey: stockKeys.items.all() })
    notifySuccess(t('stock.items.saved'))
    onClose()
  }
  return (
    <FormModal open title={t(item ? 'stock.items.edit' : 'stock.items.create')} onOpenChange={onClose}>
      <Form form={form} onSubmit={save} errorFields={{ item_code_taken: 'code', item_tracking_locked: 'tracking', item_negative_not_quantity: 'allow_negative' }}>
        <CodeField name="code" label={t('stock.item.code')} existing={!!item} />
        <TextField name="name" label={t('stock.item.name')} required maxLength={300} />
        <TextField name="unit" label={t('stock.item.unit')} description={t('stock.item.unit_hint')} required maxLength={30} />
        <MoneyField name="price" label={t('stock.item.price')} description={t('stock.item.price_hint')} required />
        <SelectField name="vat_rate" label={t('stock.item.vat_rate')} required data={vat.options} />
        <SelectField
          name="tracking"
          label={t('stock.item.tracking')}
          description={t('stock.item.tracking_hint')}
          required
          data={trackings.map((x) => ({ value: x, label: t(`stock.item.tracking_${x}`) }))}
        />
        {tracking === 'quantity' && <CheckboxField name="allow_negative" label={t('stock.item.allow_negative')} description={t('stock.item.allow_negative_hint')} />}
        <CheckboxField name="active" label={t('stock.item.active')} description={t('stock.item.active_hint')} />
        <FormActions submitLabel={t('stock.common.save')} onCancel={onClose} cancelLabel={t('stock.common.cancel')} />
      </Form>
    </FormModal>
  )
}
