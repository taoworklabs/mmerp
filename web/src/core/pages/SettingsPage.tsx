import { Menu, Select } from '@mantine/core'
import { IconBuildingSkyscraper, IconPencil } from '@tabler/icons-react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type LegalEntitySetting } from '@/shared/api/core'
import { errorText } from '@/shared/i18n'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { Form, FormActions, SelectField, useOrgUnits } from '@/shared/ui/form'
import { notifySuccess } from '@/shared/ui/notify'
import { FormModal, ListPage } from '@/shared/ui/page'
import { ContentSkeleton, EmptyState, ErrorState } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import { useListParams } from '@/shared/url/listParams'
import { coreKeys } from '../keys'

const defaults = { filters: { legal_entity: '' }, sort: '', sorts: [], pageSize: 50 }

// SettingsPage: the settings of one legal entity; each key and value is translated as <key> and <key>.<value>.
export function SettingsPage() {
  const { t } = useTranslation()
  const [params, set] = useListParams(defaults)
  const [editing, setEditing] = useState<LegalEntitySetting | null>(null)
  const units = useOrgUnits({ product: 'core', permission: 'core.setting.manage' })
  const entities = (units.data ?? []).filter((u) => u.kind === 'company')
  const chosen = Number(params.filters.legal_entity) || entities[0]?.id
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
        menu={(s) => (
          <Menu.Item leftSection={<IconPencil {...icon.text} />} onClick={() => setEditing(s)}>
            {t('core.settings.change')}
          </Menu.Item>
        )}
      />
    )
  return (
    <ListPage
      title={t('core.settings.title')}
      description={t('core.settings.description')}
      filters={
        <Select
          aria-label={t('core.period.legal_entity')}
          placeholder={t('core.period.legal_entity')}
          leftSection={<IconBuildingSkyscraper {...icon.text} />}
          w={{ base: '100%', md: 240 }}
          data={entities.map((u) => ({ value: String(u.id), label: u.name }))}
          value={chosen ? String(chosen) : null}
          onChange={(v) => set({ filters: { legal_entity: v ?? '' } })}
          allowDeselect={false}
        />
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
        <SelectField name="value" label={t('core.settings.value')} data={setting.values.map((v) => ({ value: v, label: t(`${setting.key}.${v}`) }))} required />
        <FormActions submitLabel={t('core.common.save')} onCancel={onClose} cancelLabel={t('core.common.cancel')} />
      </Form>
    </FormModal>
  )
}
