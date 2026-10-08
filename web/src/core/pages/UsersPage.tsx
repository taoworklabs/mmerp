import { Button } from '@mantine/core'
import { IconPlus } from '@tabler/icons-react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router'
import { api, unwrap, type User } from '@/shared/api/core'
import { errorText } from '@/shared/i18n'
import { PersonName } from '@/shared/ui/avatar'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { Form, FormActions, PasswordField, TextField } from '@/shared/ui/form'
import { notifySuccess } from '@/shared/ui/notify'
import { FormModal, ListPage } from '@/shared/ui/page'
import { ContentSkeleton, EmptyState, ErrorState } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import { coreKeys } from '../keys'

export function UsersPage() {
  const { t } = useTranslation()
  const [creating, setCreating] = useState(false)
  const users = useQuery({ queryKey: coreKeys.users.all(), queryFn: async () => (await unwrap(api.GET('/users'))) ?? [] })
  const columns: Column<User>[] = [
    { key: 'name', header: t('core.user.name'), role: 'title', render: (u) => <PersonName name={u.name} /> },
    { key: 'login', header: t('core.user.login'), role: 'meta', render: (u) => u.login },
  ]

  let body
  if (users.isPending) body = <ContentSkeleton />
  else if (users.isError) body = <ErrorState message={errorText(t, users.error)} onRetry={() => void users.refetch()} />
  else if (users.data.length === 0) body = <EmptyState title={t('core.user.empty')} />
  else body = <DataTable label={t('core.user.title')} columns={columns} rows={users.data} rowKey={(u) => u.id} href={(u) => `/admin/users/${u.id}`} />

  return (
    <ListPage
      title={t('core.user.title')}
      description={t('core.user.description')}
      action={
        <Button leftSection={<IconPlus {...icon.button} />} onClick={() => setCreating(true)}>
          {t('core.user.create')}
        </Button>
      }
    >
      {body}
      {creating && <CreateUserModal onClose={() => setCreating(false)} />}
    </ListPage>
  )
}

type Values = { login: string; name: string; password: string }

function CreateUserModal({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const form = useForm<Values>({ mode: 'onBlur', defaultValues: { login: '', name: '', password: '' } })

  async function save(body: Values) {
    const { id } = await unwrap(api.POST('/users', { body }))
    await qc.invalidateQueries({ queryKey: coreKeys.users.all() })
    notifySuccess(t('core.user.created'))
    onClose()
    navigate(`/admin/users/${id}`)
  }

  return (
    <FormModal opened title={t('core.user.create')} onClose={onClose}>
      <Form
        form={form}
        onSubmit={save}
        fieldOf={(err) => (err.code === 'login_taken' ? 'login' : err.code === 'password_too_short' ? 'password' : undefined)}
      >
        <TextField name="login" label={t('core.user.login')} required maxLength={200} autoComplete="off" />
        <TextField name="name" label={t('core.user.name')} required maxLength={200} />
        <PasswordField name="password" label={t('core.user.password')} required description={t('core.user.password_hint')} />
        <FormActions submitLabel={t('core.user.create')} onCancel={onClose} cancelLabel={t('core.common.cancel')} />
      </Form>
    </FormModal>
  )
}
