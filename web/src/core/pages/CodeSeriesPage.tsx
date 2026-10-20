import { Text } from '@mantine/core'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type CodeSeries } from '@/shared/api/core'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { FieldList } from '@/shared/ui/FieldList'
import { Form, FormActions, TextField } from '@/shared/ui/form'
import { notifySuccess } from '@/shared/ui/notify'
import { FormModal, ListPage } from '@/shared/ui/page'
import { QueryView } from '@/shared/ui/QueryView'
import { EmptyState } from '@/shared/ui/states'
import { coreKeys } from '../keys'

// CodeSeriesPage: the prefix of each catalogue's issued codes. Changing it keeps the count.
export function CodeSeriesPage() {
  const { t } = useTranslation()
  const [editing, setEditing] = useState<CodeSeries | null>(null)
  const series = useQuery({ queryKey: coreKeys.codeSeries(), queryFn: async () => (await unwrap(api.GET('/code-series'))) ?? [] })
  const columns: Column<CodeSeries>[] = [
    { key: 'name', header: t('core.code_series.name'), role: 'title', render: (s) => t(`${s.code}.code_series`) },
    { key: 'prefix', header: t('core.code_series.prefix'), role: 'meta', kind: 'code', render: (s) => s.prefix || t('core.code_series.no_prefix') },
  ]
  const body = (
    <QueryView query={series} empty={<EmptyState title={t('core.code_series.empty')} />}>
      {(data) => (
        <DataTable
          label={t('core.code_series.title')}
          columns={columns}
          rows={data}
          rowKey={(s) => s.code}
          // One action per row: the row itself opens it.
          onRowClick={(s) => setEditing(s)}
        />
      )}
    </QueryView>
  )
  return (
    <ListPage title={t('core.code_series.title')} description={t('core.code_series.description')}>
      {body}
      {editing && <PrefixModal series={editing} onClose={() => setEditing(null)} />}
    </ListPage>
  )
}

function PrefixModal({ series, onClose }: { series: CodeSeries; onClose: () => void }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const form = useForm<{ prefix: string }>({ mode: 'onBlur', defaultValues: { prefix: series.prefix } })

  async function save(v: { prefix: string }) {
    await unwrap(api.PUT('/code-series/{code}/prefix', { params: { path: { code: series.code } }, body: { prefix: v.prefix.trim() } }))
    await qc.invalidateQueries({ queryKey: coreKeys.codeSeries() })
    notifySuccess(t('core.code_series.saved'))
    onClose()
  }

  const title = t('core.code_series.change_title', { name: t(`${series.code}.code_series`) })
  // A disabled product's series only reads.
  if (!series.allowed_actions.includes('edit'))
    return (
      <FormModal opened title={title} onClose={onClose}>
        <Text size="sm">{t('core.code_series.read_only')}</Text>
        <FieldList rows={[[t('core.code_series.prefix'), series.prefix || t('core.code_series.no_prefix')]]} />
      </FormModal>
    )
  return (
    <FormModal opened title={title} onClose={onClose}>
      <Form form={form} onSubmit={save} errorFields={{ invalid_code_prefix: 'prefix' }}>
        <TextField name="prefix" label={t('core.code_series.prefix')} description={t('core.code_series.prefix_hint')} maxLength={10} />
        <FormActions submitLabel={t('core.common.save')} onCancel={onClose} cancelLabel={t('core.common.cancel')} />
      </Form>
    </FormModal>
  )
}
