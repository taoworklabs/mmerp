import { Button } from '@mantine/core'
import { IconPlus, IconTrash } from '@tabler/icons-react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm, type Path } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, ApiError, unwrap, type Delegation } from '@/shared/api/core'
import { useMe } from '@/shared/auth/me'
import { errorText, formatDate, today } from '@/shared/i18n'
import { type Column } from '@/shared/ui/DataTable'
import { useConfirm } from '@/shared/ui/confirm'
import { DateField, Form, FormActions, SelectField } from '@/shared/ui/form'
import { MenuAction } from '@/shared/ui/MenuAction'
import { Notice } from '@/shared/ui/Notice'
import { notifySuccess } from '@/shared/ui/notify'
import { FormModal, ListPage } from '@/shared/ui/page'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { icon } from '@/shared/ui/theme'
import { coreKeys } from '../keys'

// Delegations not yet ended: the user's own, or everyone's for whoever manages them all.
// Its strings are in meta, like the inbox: it lives at the root, outside the admin screens.
export default function DelegationsPage() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [ask, dialog] = useConfirm()
  const meta = useQuery({ queryKey: coreKeys.delegations.meta(), queryFn: () => unwrap(api.GET('/approval-delegations/meta')) })
  const manageAll = meta.data?.manage_all ?? false
  const list = useQuery({
    queryKey: coreKeys.delegations.list(),
    queryFn: () => unwrap(api.GET('/approval-delegations')),
    select: (d) => ({ items: d ?? [], total: d?.length ?? 0 }),
  })

  async function remove(d: Delegation) {
    const ok = await ask({
      title: t('core.delegations.delete'),
      // A manager may delete anyone's, so the message names whose it is.
      message: t(manageAll ? 'core.delegations.delete_confirm_of' : 'core.delegations.delete_confirm', {
        user: d.user_name,
        name: d.delegate_name,
        from: formatDate(d.starts_on),
        to: formatDate(d.ends_on),
      }),
      confirmLabel: t('core.delegations.delete'),
      danger: true,
    })
    if (!ok) return
    setError(null)
    try {
      await unwrap(api.DELETE('/approval-delegations/{id}', { params: { path: { id: d.id } } }))
      notifySuccess(t('core.delegations.deleted'))
    } catch (err) {
      setError(errorText(t, err))
    }
    await qc.invalidateQueries({ queryKey: coreKeys.delegations.list() })
  }

  const columns: Column<Delegation>[] = [
    ...(manageAll ? [{ key: 'user', header: t('core.delegations.user'), role: 'title', render: (d: Delegation) => d.user_name } as const] : []),
    { key: 'delegate', header: t('core.delegations.delegate'), role: manageAll ? 'meta' : 'title', render: (d) => d.delegate_name },
    { key: 'starts_on', header: t('core.delegations.starts_on'), role: 'meta', kind: 'date', render: (d) => formatDate(d.starts_on) },
    { key: 'ends_on', header: t('core.delegations.ends_on'), role: 'meta', kind: 'date', render: (d) => formatDate(d.ends_on) },
    {
      key: 'status',
      header: t('core.delegations.status'),
      role: 'status',
      kind: 'status',
      render: (d) => <StatusBadge tone={d.active ? 'positive' : 'info'}>{t(d.active ? 'core.delegations.active' : 'core.delegations.upcoming')}</StatusBadge>,
    },
  ]

  return (
    <ListPage
      title={t('core.delegations.title')}
      description={t('core.delegations.description')}
      action={
        <Button leftSection={<IconPlus {...icon.button} />} onClick={() => setCreating(true)}>
          {t('core.delegations.create')}
        </Button>
      }
      list={{
        query: list,
        table: {
          label: t('core.delegations.title'),
          columns,
          rowKey: (d) => d.id,
          menu: (d) =>
            d.allowed_actions.includes('delete') ? (
              <MenuAction icon={IconTrash} danger onClick={() => void remove(d)}>
                {t('core.delegations.delete')}
              </MenuAction>
            ) : null,
        },
        empty: { title: t('core.delegations.empty'), description: t('core.delegations.empty_description') },
        noResults: t('core.delegations.empty'),
      }}
    >
      {error && (
        <Notice tone="danger">{error}</Notice>
      )}
      {creating && <CreateModal manageAll={manageAll} onClose={() => setCreating(false)} />}
      {dialog}
    </ListPage>
  )
}

type Values = { user: string | null; delegate: string | null; starts_on: string | null; ends_on: string | null }

// The field each reason of invalid_delegation is about.
const reasonField: Record<string, Path<Values>> = { self: 'delegate', creator: 'delegate', dates: 'ends_on', past: 'ends_on', overlap: 'starts_on' }

function CreateModal({ manageAll, onClose }: { manageAll: boolean; onClose: () => void }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const me = useMe()
  const users = useQuery({ queryKey: coreKeys.delegations.users(), queryFn: () => unwrap(api.GET('/approval-delegations/users')) })
  // Logins come only to managers; they tell apart people of the same name.
  const options = (users.data ?? []).map((u) => ({ value: String(u.id), label: u.login ? `${u.name} (${u.login})` : u.name }))
  const form = useForm<Values>({ mode: 'onBlur', defaultValues: { user: null, delegate: null, starts_on: today(me.timezone), ends_on: null } })

  async function save(v: Values) {
    try {
      await unwrap(
        api.POST('/approval-delegations', {
          body: { user_id: v.user ? Number(v.user) : undefined, delegate_id: Number(v.delegate), starts_on: v.starts_on ?? '', ends_on: v.ends_on ?? '' },
        }),
      )
    } catch (err) {
      const reason = err instanceof ApiError && err.code === 'invalid_delegation' ? String(err.params.reason) : ''
      const field = reasonField[reason]
      if (!field) throw err
      form.setError(field, { message: t(`core.delegations.reason.${reason}`) }, { shouldFocus: true })
      return false
    }
    await qc.invalidateQueries({ queryKey: coreKeys.delegations.list() })
    notifySuccess(t('core.delegations.created'))
    onClose()
  }

  return (
    <FormModal opened title={t('core.delegations.create')} onClose={onClose}>
      <Form form={form} onSubmit={save}>
        {manageAll && <SelectField name="user" label={t('core.delegations.user')} description={t('core.delegations.user_hint')} clearable data={options} />}
        <SelectField name="delegate" label={t('core.delegations.delegate')} description={t('core.delegations.delegate_hint')} required data={options} />
        <DateField name="starts_on" label={t('core.delegations.starts_on')} required />
        <DateField name="ends_on" label={t('core.delegations.ends_on')} required />
        {users.isError && (
          <Notice tone="danger">{errorText(t, users.error)}</Notice>
        )}
        <FormActions submitLabel={t('core.delegations.create')} onCancel={onClose} cancelLabel={t('core.delegations.cancel')} />
      </Form>
    </FormModal>
  )
}
