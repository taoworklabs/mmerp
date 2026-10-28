import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type LegalEntitySetting } from '@/shared/api/core'
import { errorText } from '@/shared/i18n'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { Form, FormActions, LegalEntityFilter, SelectField, useLegalEntities } from '@/shared/ui/form'
import { notifySuccess } from '@/shared/ui/notify'
import { FormModal, ListPage } from '@/shared/ui/page'
import { ContentSkeleton, EmptyState, ErrorState } from '@/shared/ui/states'
import { useListParams } from '@/shared/url/listParams'
import { coreKeys } from '../keys'

const defaults = { filters: { legal_entity: '' }, sort: '', sorts: [], pageSize: 50 }

// SettingsPage: the settings of one legal entity; each key and value is translated as <key> and <key>.<value>.
export function SettingsPage() {
  const { t } = useTranslation()
  const [params, set] = useListParams(defaults)
  const [editing, setEditing] = useState<LegalEntitySetting | null>(null)
  const units = useLegalEntities({ product: 'core', permission: 'core.setting.manage' }, params.filters.legal_entity)
  const chosen = units.chosen
  const settings = useQuery({
    queryKey: coreKeys.settings(chosen ?? 0),
    queryFn: async () => (await unwrap(api.GET('/legal-entities/{id}/settings', { params: { path: { id: chosen! } } }))) ?? [],
    enabled: chosen !== undefined,
  })
  const columns: Column<LegalEntitySetting>[] = [
    { key: 'key', header: t('core.settings.key'), role: 'title', render: (s) => t(s.key) },
    { key: 'value', header: t('core.settings.value'), role: 'meta', render: (s) => t(`${s.key}.${s.value}`) },
  ]
  let body
  if (units.isPending || (chosen !== undefined && settings.isPending)) body = <ContentSkeleton />
  else if (units.isError || settings.isError) body = <ErrorState message={errorText(t, units.error ?? settings.error)} onRetry={() => void settings.refetch()} />
  else if (chosen === undefined) body = <EmptyState title={t('core.period.empty')} />
  else
    body = (
      <DataTable
        label={t('core.settings.title')}
        columns={columns}
        rows={settings.data ?? []}
        rowKey={(s) => s.key}
        // One action per row: the row itself opens it.
        onRowClick={(s) => setEditing(s)}
      />
    )
  return (
    <ListPage
      title={t('core.settings.title')}
      description={t('core.settings.description')}
      filters={
        <LegalEntityFilter label={t('core.period.legal_entity')} options={units.options} value={chosen} onChange={(v) => set({ filters: { legal_entity: v } })} />
      }
    >
      {body}
      {editing && chosen && <SettingModal entity={chosen} setting={editing} onClose={() => setEditing(null)} />}
    </ListPage>
  )
}

function SettingModal({ entity, setting, onClose }: { entity: number; setting: LegalEntitySetting; onClose: () => void }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const form = useForm<{ value: string | null }>({ mode: 'onBlur', defaultValues: { value: setting.value } })

  async function save(v: { value: string | null }) {
    await unwrap(api.PUT('/legal-entities/{id}/settings/{key}', { params: { path: { id: entity, key: setting.key } }, body: { value: v.value ?? '' } }))
    await qc.invalidateQueries({ queryKey: coreKeys.settings(entity) })
    notifySuccess(t('core.settings.saved'))
    onClose()
  }

  return (
    <FormModal opened title={t(setting.key)} onClose={onClose}>
      <Form form={form} onSubmit={save}>
        <SelectField
          name="value"
          label={t('core.settings.value')}
          // A setting whose change has side effects explains them in its own hint.
          description={t(`${setting.key}.hint`, { defaultValue: '' }) || undefined}
          data={setting.values.map((v) => ({ value: v, label: t(`${setting.key}.${v}`) }))}
          required
        />
        <FormActions submitLabel={t('core.common.save')} onCancel={onClose} cancelLabel={t('core.common.cancel')} />
      </Form>
    </FormModal>
  )
}
