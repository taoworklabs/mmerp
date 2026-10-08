import { Button, Menu } from '@mantine/core'
import { IconPencil, IconPlus } from '@tabler/icons-react'
import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { useProductOn } from '@/shared/auth/me'
import { api, unwrap, type ContractType, type ContractTypeInput } from '@/shared/api/hrm'
import { errorText } from '@/shared/i18n'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { CheckboxField, Form, FormActions, TextField } from '@/shared/ui/form'
import { notifySuccess } from '@/shared/ui/notify'
import { FormModal, ListPage } from '@/shared/ui/page'
import { ContentSkeleton, EmptyState, ErrorState } from '@/shared/ui/states'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { icon } from '@/shared/ui/theme'
import { useContractTypes } from '../hooks/useContract'
import { hrmKeys } from '../keys'

// ContractTypesPage: the kinds of contract the customer uses, the approval field contract_kind.
export function ContractTypesPage() {
  const { t } = useTranslation()
  const types = useContractTypes()
  const [editing, setEditing] = useState<ContractType | 'new' | null>(null)
  // Catalog writes have no allowed_actions; a disabled product is read-only.
  const writable = useProductOn('hrm')
  const columns: Column<ContractType>[] = [
    { key: 'name', header: t('hrm.contract_type.name'), role: 'title', render: (x) => x.name },
    { key: 'fixed_term', header: t('hrm.contract_type.term'), role: 'meta', render: (x) => t(x.fixed_term ? 'hrm.contract_type.fixed_term' : 'hrm.contract_type.no_term') },
    {
      key: 'active',
      header: t('hrm.contract_type.active'),
      role: 'status',
      render: (x) => <StatusBadge tone={x.active ? 'positive' : 'neutral'}>{t(x.active ? 'hrm.contract_type.in_use' : 'hrm.contract_type.retired')}</StatusBadge>,
    },
  ]
  let body
  if (types.isPending) body = <ContentSkeleton />
  else if (types.isError) body = <ErrorState message={errorText(t, types.error)} onRetry={() => void types.refetch()} />
  else if (types.data.length === 0) body = <EmptyState title={t('hrm.contract_type.empty')} description={t('hrm.contract_type.empty_description')} />
  else
    body = (
      <DataTable
        label={t('hrm.contract_type.title')}
        columns={columns}
        rows={types.data}
        rowKey={(x) => x.id}
        menu={
          writable
            ? (x) => (
                <Menu.Item leftSection={<IconPencil {...icon.text} />} onClick={() => setEditing(x)}>
                  {t('hrm.contract_type.edit')}
                </Menu.Item>
              )
            : undefined
        }
      />
    )
  return (
    <ListPage
      title={t('hrm.contract_type.title')}
      description={t('hrm.contract_type.description')}
      action={
        writable && (
          <Button leftSection={<IconPlus {...icon.button} />} onClick={() => setEditing('new')}>
            {t('hrm.contract_type.create')}
          </Button>
        )
      }
    >
      {body}
      {editing && <ContractTypeModal type={editing === 'new' ? null : editing} onClose={() => setEditing(null)} />}
    </ListPage>
  )
}

function ContractTypeModal({ type, onClose }: { type: ContractType | null; onClose: () => void }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const form = useForm<ContractTypeInput>({ mode: 'onBlur', defaultValues: type ?? { name: '', fixed_term: false, active: true } })
  async function save(v: ContractTypeInput) {
    const body = { name: v.name.trim(), fixed_term: v.fixed_term, active: v.active }
    if (type) await unwrap(api.PUT('/hrm/contract-types/{id}', { params: { path: { id: type.id } }, body }))
    else await unwrap(api.POST('/hrm/contract-types', { body }))
    await qc.invalidateQueries({ queryKey: hrmKeys.contractTypes() })
    notifySuccess(t('hrm.contract_type.saved'))
    onClose()
  }
  return (
    <FormModal opened title={t(type ? 'hrm.contract_type.edit' : 'hrm.contract_type.create')} onClose={onClose}>
      <Form form={form} onSubmit={save} fieldOf={(err) => (err.code === 'contract_type_taken' ? 'name' : undefined)}>
        <TextField name="name" label={t('hrm.contract_type.name')} required maxLength={100} />
        <CheckboxField name="fixed_term" label={t('hrm.contract_type.fixed_term')} description={t('hrm.contract_type.fixed_term_hint')} />
        <CheckboxField name="active" label={t('hrm.contract_type.active')} description={t('hrm.contract_type.active_hint')} />
        <FormActions submitLabel={t('hrm.common.save')} onCancel={onClose} cancelLabel={t('hrm.common.cancel')} />
      </Form>
    </FormModal>
  )
}
