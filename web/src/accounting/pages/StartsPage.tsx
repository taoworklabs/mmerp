import { Button } from '@mmerp/ui'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type Start } from '@/shared/api/accounting'
import { useCan, useProductOn } from '@/shared/auth/me'
import { formatDate } from '@/shared/i18n'
import { type Column } from '@/shared/ui/DataTable'
import { DateField, Form, FormActions, useLegalEntities } from '@/shared/ui/form'
import { ModalFooter } from '@/shared/ui/ModalFooter'
import { Notice } from '@/shared/ui/Notice'
import { notifySuccess } from '@/shared/ui/notify'
import { FormModal, ListPage } from '@/shared/ui/page'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { asList } from '../hooks/useAccounting'
import { accountingKeys } from '../keys'

// StartsPage: the day each legal entity starts booking; fixed once it has entries.
export function StartsPage() {
  const { t } = useTranslation()
  const on = useProductOn('accounting')
  const writable = useCan('accounting.manage') && on
  const entities = useLegalEntities({}).options
  const entityName = (id: number) => entities.find((o) => o.value === String(id))?.label ?? ''
  const [editing, setEditing] = useState<Start | null>(null)
  const starts = useQuery({
    queryKey: accountingKeys.starts(),
    queryFn: async () => (await unwrap(api.GET('/accounting/starts'))).items,
  })

  const columns: Column<Start>[] = [
    { key: 'legal_entity', header: t('accounting.common.legal_entity'), role: 'title', render: (s) => entityName(s.legal_entity_id) },
    { key: 'start_date', header: t('accounting.start.date'), role: 'meta', kind: 'date', render: (s) => formatDate(s.start_date) },
    {
      key: 'fixed',
      header: t('accounting.start.state'),
      role: 'status',
      kind: 'status',
      render: (s) =>
        s.fixed ? (
          <StatusBadge tone="success">{t('accounting.start.fixed')}</StatusBadge>
        ) : s.start_date ? (
          <StatusBadge tone="neutral">{t('accounting.start.open')}</StatusBadge>
        ) : (
          <StatusBadge tone="neutral">{t('accounting.start.unset')}</StatusBadge>
        ),
    },
  ]

  return (
    <ListPage
      title={t('accounting.starts.title')}
      description={t('accounting.starts.description')}
      list={{
        query: asList(starts, starts.data),
        table: {
          label: t('accounting.starts.title'),
          columns,
          rowKey: (s) => s.legal_entity_id,
          onRowClick: (s) => setEditing(s),
        },
        empty: { title: t('accounting.starts.empty'), description: t('accounting.starts.empty_description') },
        noResults: t('accounting.common.no_results'),
      }}
    >
      {editing && <StartModal start={editing} writable={writable} title={entityName(editing.legal_entity_id)} onClose={() => setEditing(null)} />}
    </ListPage>
  )
}

// StartModal edits a start date; it only shows it once fixed, or to whom may not change it.
function StartModal({ start, writable, title, onClose }: { start: Start; writable: boolean; title: string; onClose: () => void }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const form = useForm<{ start_date: string | null }>({ mode: 'onBlur', defaultValues: { start_date: start.start_date ?? null } })
  async function save(v: { start_date: string | null }) {
    await unwrap(api.PUT('/accounting/starts/{legal_entity_id}', { params: { path: { legal_entity_id: start.legal_entity_id } }, body: { start_date: v.start_date } }))
    // What is pending depends on the start date.
    await Promise.all([qc.invalidateQueries({ queryKey: accountingKeys.starts() }), qc.invalidateQueries({ queryKey: accountingKeys.books() })])
    notifySuccess(t('accounting.starts.saved'))
    onClose()
  }
  if (start.fixed || !writable)
    return (
      <FormModal opened title={title} onClose={onClose}>
        <Form form={form} onSubmit={async () => {}}>
          <DateField name="start_date" label={t('accounting.start.date')} readOnly />
          {start.fixed && <Notice tone="info">{t('accounting.start.fixed_hint')}</Notice>}
          <ModalFooter>
            <Button variant="subtle" tone="neutral" onClick={onClose}>
              {t('accounting.common.close')}
            </Button>
          </ModalFooter>
        </Form>
      </FormModal>
    )
  return (
    <FormModal opened title={title} onClose={onClose}>
      <Form form={form} onSubmit={save} errorFields={{ start_date_fixed: 'start_date', start_date_in_locked_period: 'start_date' }}>
        <DateField
          name="start_date"
          label={t('accounting.start.date')}
          description={t('accounting.start.date_hint')}
          clearable
        />
        <FormActions submitLabel={t('accounting.common.save')} onCancel={onClose} cancelLabel={t('accounting.common.cancel')} />
      </Form>
    </FormModal>
  )
}
