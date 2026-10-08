import { Button, Menu } from '@mantine/core'
import { IconPencil, IconPlus } from '@tabler/icons-react'
import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type OrgUnit, type OrgUnitInput } from '@/shared/api/core'
import { OrgTree } from '@/shared/org'
import { Form, FormActions, OrgUnitField, SelectField, TextField } from '@/shared/ui/form'
import { notifySuccess } from '@/shared/ui/notify'
import { FormModal, ListPage } from '@/shared/ui/page'
import { icon } from '@/shared/ui/theme'
import { coreKeys } from '../keys'

const kinds = ['group', 'company', 'branch', 'department'] as const

type Values = { kind: string; name: string; parent_id: string | null; tax_code: string; legal_name: string; address: string }

// OrgUnitsPage edits the permission-scope tree; company nodes are legal entities.
export function OrgUnitsPage() {
  const { t } = useTranslation()
  const [editing, setEditing] = useState<OrgUnit | 'new' | null>(null)
  return (
    <ListPage
      title={t('core.org.title')}
      description={t('core.org.description')}
      action={
        <Button leftSection={<IconPlus {...icon.button} />} onClick={() => setEditing('new')}>
          {t('core.org.create')}
        </Button>
      }
    >
      <OrgTree
        label={t('core.org.title')}
        emptyDescription={t('core.org.empty_description')}
        menu={(unit) => (
          <Menu.Item leftSection={<IconPencil {...icon.text} />} onClick={() => setEditing(unit)}>
            {t('core.org.edit')}
          </Menu.Item>
        )}
      />
      {editing && <OrgUnitModal unit={editing === 'new' ? null : editing} onClose={() => setEditing(null)} />}
    </ListPage>
  )
}

function OrgUnitModal({ unit, onClose }: { unit: OrgUnit | null; onClose: () => void }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const form = useForm<Values>({
    mode: 'onBlur',
    defaultValues: {
      kind: unit?.kind ?? 'department',
      name: unit?.name ?? '',
      parent_id: unit?.parent_id ? String(unit.parent_id) : null,
      tax_code: unit?.tax_code ?? '',
      legal_name: unit?.legal_name ?? '',
      address: unit?.address ?? '',
    },
  })
  const company = form.watch('kind') === 'company'

  async function save(v: Values) {
    const opt = (s: string) => (company && s ? s : undefined)
    const body: OrgUnitInput = {
      kind: v.kind as OrgUnitInput['kind'],
      name: v.name,
      parent_id: v.parent_id ? Number(v.parent_id) : null,
      tax_code: opt(v.tax_code),
      legal_name: opt(v.legal_name),
      address: opt(v.address),
    }
    if (unit) await unwrap(api.PUT('/org-units/{id}', { params: { path: { id: unit.id } }, body }))
    else await unwrap(api.POST('/org-units', { body }))
    await qc.invalidateQueries({ queryKey: coreKeys.orgUnits() })
    notifySuccess(t('core.org.saved'))
    onClose()
  }

  return (
    <FormModal opened title={t(unit ? 'core.org.edit' : 'core.org.create')} onClose={onClose}>
      <Form form={form} onSubmit={save}>
        <SelectField name="kind" label={t('shared.org.kind')} required data={kinds.map((k) => ({ value: k, label: t(`shared.org.kind.${k}`) }))} />
        <TextField name="name" label={t('shared.org.name')} required maxLength={200} />
        <OrgUnitField name="parent_id" label={t('core.org.parent')} clearable />
        {company && (
          <>
            <TextField name="tax_code" label={t('shared.org.tax_code')} maxLength={20} />
            <TextField name="legal_name" label={t('core.org.legal_name')} maxLength={300} />
            <TextField name="address" label={t('core.org.address')} maxLength={500} />
          </>
        )}
        <FormActions submitLabel={t('core.common.save')} onCancel={onClose} cancelLabel={t('core.common.cancel')} />
      </Form>
    </FormModal>
  )
}
