import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type CodeSeries } from '@/shared/api/core'
import { errorText } from '@/shared/i18n'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { Form, FormActions, TextField } from '@/shared/ui/form'
import { notifySuccess } from '@/shared/ui/notify'
import { FormModal, ListPage } from '@/shared/ui/page'
import { ContentSkeleton, EmptyState, ErrorState } from '@/shared/ui/states'
import { coreKeys } from '../keys'

// CodeSeriesPage: the prefix of each catalogue's issued codes. Changing it keeps the count.
export function CodeSeriesPage() {
  const { t } = useTranslation()
  const [editing, setEditing] = useState<CodeSeries | null>(null)
  const series = useQuery({ queryKey: coreKeys.codeSeries(), queryFn: async () => (await unwrap(api.GET('/code-series'))) ?? [] })
  const columns: Column<CodeSeries>[] = [
    { key: 'name', header: t('core.code_series.name'), role: 'title', render: (s) => t(`numbering.series.${s.code}`) },
    { key: 'prefix', header: t('core.code_series.prefix'), role: 'meta', kind: 'code', render: (s) => s.prefix || t('core.code_series.no_prefix') },
    { key: 'next', header: t('core.code_series.next'), role: 'meta', kind: 'code', render: (s) => s.next },
  ]
  let body
  if (series.isPending) body = <ContentSkeleton />
  else if (series.isError) body = <ErrorState message={errorText(t, series.error)} onRetry={() => void series.refetch()} />
  else if (series.data.length === 0) body = <EmptyState title={t('core.code_series.empty')} />
  else
    body = (
      <DataTable
        label={t('core.code_series.title')}
        columns={columns}
        rows={series.data}
        rowKey={(s) => s.code}
        // One action per row: the row itself opens it. A disabled product's series only read.
        onRowClick={(s) => s.allowed_actions.includes('edit') && setEditing(s)}
      />
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

  return (
    <FormModal opened title={t('core.code_series.change_title', { name: t(`numbering.series.${series.code}`) })} onClose={onClose}>
      <Form form={form} onSubmit={save} fieldOf={(err) => (err.code === 'invalid_code_prefix' ? 'prefix' : undefined)}>
        <TextField name="prefix" label={t('core.code_series.prefix')} description={t('core.code_series.prefix_hint')} maxLength={10} />
        <FormActions submitLabel={t('core.common.save')} onCancel={onClose} cancelLabel={t('core.common.cancel')} />
      </Form>
    </FormModal>
  )
}
