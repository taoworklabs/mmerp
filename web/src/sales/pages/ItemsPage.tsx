import { Button, Menu } from '@mantine/core'
import { IconPencil, IconPlus } from '@tabler/icons-react'
import { keepPreviousData, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { useCan, useProductOn } from '@/shared/auth/me'
import { api, unwrap, type Item } from '@/shared/api/sales'
import { errorText, formatNumber } from '@/shared/i18n'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { CheckboxField, Form, FormActions, MoneyField, SearchInput, SelectField, TextField } from '@/shared/ui/form'
import { notifySuccess } from '@/shared/ui/notify'
import { FormModal, ListPage } from '@/shared/ui/page'
import { ContentSkeleton, EmptyState, ErrorState } from '@/shared/ui/states'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { icon } from '@/shared/ui/theme'
import { useListParams } from '@/shared/url/listParams'
import { useVatOptions } from '../components/vat'
import { salesKeys } from '../keys'

const defaults = { filters: { q: '', active: '' }, sort: '', sorts: [''], pageSize: 50 }

// ItemsPage: the tenant's catalogue of goods and services; catalogue admins edit it in a modal.
export function ItemsPage() {
  const { t } = useTranslation()
  const vat = useVatOptions()
  // Catalogue writes have no allowed_actions; a disabled product is read-only.
  const writable = useCan('sales.item.manage') && useProductOn('sales')
  const [editing, setEditing] = useState<Item | 'new' | null>(null)
  const [params, set] = useListParams(defaults)
  const { q } = params.filters as typeof defaults.filters
  const query = { q, active: '', page: params.page, pageSize: params.pageSize }
  const list = useQuery({
    queryKey: salesKeys.items.list(query),
    queryFn: () => unwrap(api.GET('/sales/items', { params: { query: { q: q || undefined, page: params.page, page_size: params.pageSize as 50 } } })),
    placeholderData: keepPreviousData,
  })
  const columns: Column<Item>[] = [
    { key: 'name', header: t('sales.item.name'), role: 'title', render: (x) => x.name },
    { key: 'code', header: t('sales.item.code'), role: 'meta', render: (x) => x.code },
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
  let body
  if (list.isPending) body = <ContentSkeleton />
  else if (list.isError) body = <ErrorState message={errorText(t, list.error)} onRetry={() => void list.refetch()} />
  else if (list.data.items.length === 0)
    body =
      q !== '' ? (
        <EmptyState
          title={t('sales.common.no_results')}
          action={
            <Button variant="default" onClick={() => set({ filters: defaults.filters })}>
              {t('sales.common.clear_filters')}
            </Button>
          }
        />
      ) : (
        <EmptyState title={t('sales.items.empty')} description={writable ? t('sales.items.empty_description') : undefined} />
      )
  else
    body = (
      <DataTable
        label={t('sales.items.title')}
        columns={columns}
        rows={list.data.items}
        rowKey={(x) => x.id}
        menu={
          writable
            ? (x) => (
                <Menu.Item leftSection={<IconPencil {...icon.text} />} onClick={() => setEditing(x)}>
                  {t('sales.items.edit')}
                </Menu.Item>
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
        <SearchInput label={t('sales.items.search')} placeholder={t('sales.items.search_hint')} value={q} onSearch={(v) => set({ filters: { q: v } })} />
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
      {editing && <ItemModal item={editing === 'new' ? null : editing} onClose={() => setEditing(null)} />}
    </ListPage>
  )
}

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
