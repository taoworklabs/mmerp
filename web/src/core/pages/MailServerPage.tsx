import { Button, Group, Stack } from '@mantine/core'
import { IconMailForward, IconPencil, IconTrash } from '@tabler/icons-react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, ApiError, unwrap, type MailServer } from '@/shared/api/core'
import { errorText } from '@/shared/i18n'
import { useConfirm } from '@/shared/ui/confirm'
import { FieldList } from '@/shared/ui/FieldList'
import { Form, FormActions, PasswordField, SelectField, TextField } from '@/shared/ui/form'
import { notifySuccess } from '@/shared/ui/notify'
import { FormModal, ListPage } from '@/shared/ui/page'
import { ContentSkeleton, EmptyState, ErrorState } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import { coreKeys } from '../keys'

// MailServerPage: the company's SMTP server. Without one, notifications stay in the app.
// The password is never read back; a test message to oneself checks the settings.
export function MailServerPage() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [editing, setEditing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [ask, dialog] = useConfirm()
  const server = useQuery({
    queryKey: coreKeys.mailServer(),
    queryFn: async () => {
      try {
        return await unwrap(api.GET('/mail-server'))
      } catch (err) {
        if (err instanceof ApiError && err.status === 404) return null
        throw err
      }
    },
  })

  async function run(action: () => Promise<unknown>, done: string) {
    setError(null)
    try {
      await action()
    } catch (err) {
      setError(errorText(t, err))
      return
    }
    await qc.invalidateQueries({ queryKey: coreKeys.mailServer() })
    notifySuccess(done)
  }
  const test = () => run(() => unwrap(api.POST('/mail-server/test')), t('core.mail.test_sent'))
  async function remove() {
    if (!(await ask({ title: t('core.mail.delete_title'), message: t('core.mail.delete_confirm'), confirmLabel: t('core.mail.delete'), danger: true }))) return
    await run(() => unwrap(api.DELETE('/mail-server')), t('core.mail.deleted'))
  }

  let body
  if (server.isPending) body = <ContentSkeleton />
  else if (server.isError) body = <ErrorState message={errorText(t, server.error)} onRetry={() => void server.refetch()} />
  else if (!server.data)
    body = (
      <EmptyState
        title={t('core.mail.empty')}
        description={t('core.mail.empty_description')}
        action={<Button onClick={() => setEditing(true)}>{t('core.mail.configure')}</Button>}
      />
    )
  else
    body = (
      <Stack gap="md">
        <FieldList
          rows={[
            [t('core.mail.host'), `${server.data.host}:${server.data.port}`],
            [t('core.mail.security'), t(`core.mail.security.${server.data.security}`)],
            [t('core.mail.username'), server.data.username || '—'],
            [t('core.mail.password'), server.data.password_set ? t('core.mail.password_set') : '—'],
            [t('core.mail.from_address'), server.data.from_address],
            [t('core.mail.base_url'), server.data.base_url],
          ]}
        />
      </Stack>
    )

  return (
    <ListPage
      title={t('core.mail.title')}
      description={t('core.mail.description')}
      action={
        server.data && (
          <Group gap="xs">
            <Button variant="subtle" color="danger" leftSection={<IconTrash {...icon.button} />} onClick={() => void remove()}>
              {t('core.mail.delete')}
            </Button>
            <Button variant="default" leftSection={<IconMailForward {...icon.button} />} onClick={() => void test()}>
              {t('core.mail.test')}
            </Button>
            <Button leftSection={<IconPencil {...icon.button} />} onClick={() => setEditing(true)}>
              {t('core.mail.edit')}
            </Button>
          </Group>
        )
      }
    >
      {error && <ErrorState message={error} />}
      {body}
      {editing && <MailServerModal current={server.data ?? null} onClose={() => setEditing(false)} />}
      {dialog}
    </ListPage>
  )
}

type Values = { host: string; port: string; security: string; username: string; password: string; from_address: string; base_url: string }

function MailServerModal({ current, onClose }: { current: MailServer | null; onClose: () => void }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const form = useForm<Values>({
    mode: 'onBlur',
    defaultValues: {
      host: current?.host ?? '',
      port: String(current?.port ?? 587),
      security: current?.security ?? 'starttls',
      username: current?.username ?? '',
      password: '',
      from_address: current?.from_address ?? '',
      base_url: current?.base_url ?? window.location.origin,
    },
  })

  async function save(v: Values) {
    await unwrap(
      api.PUT('/mail-server', {
        body: {
          host: v.host.trim(),
          port: Number(v.port),
          security: v.security as MailServer['security'],
          username: v.username.trim(),
          // Blank keeps the stored password.
          password: v.password === '' && current?.password_set ? null : v.password,
          from_address: v.from_address.trim(),
          base_url: v.base_url.trim(),
        },
      }),
    )
    await qc.invalidateQueries({ queryKey: coreKeys.mailServer() })
    notifySuccess(t('core.mail.saved'))
    onClose()
  }

  return (
    <FormModal opened title={t('core.mail.title')} onClose={onClose}>
      <Form form={form} onSubmit={save}>
        <TextField name="host" label={t('core.mail.host')} required maxLength={255} />
        <TextField name="port" label={t('core.mail.port')} required rules={{ pattern: { value: /^\d{1,5}$/, message: t('core.mail.port_invalid') } }} />
        <SelectField
          name="security"
          label={t('core.mail.security')}
          required
          data={['starttls', 'tls', 'none'].map((v) => ({ value: v, label: t(`core.mail.security.${v}`) }))}
        />
        <TextField name="username" label={t('core.mail.username')} maxLength={255} autoComplete="off" />
        <PasswordField name="password" label={t('core.mail.password')} description={current?.password_set ? t('core.mail.password_keep') : undefined} />
        <TextField name="from_address" label={t('core.mail.from_address')} type="email" required maxLength={254} />
        <TextField name="base_url" label={t('core.mail.base_url')} description={t('core.mail.base_url_hint')} required maxLength={255} />
        <FormActions submitLabel={t('core.common.save')} onCancel={onClose} cancelLabel={t('core.common.cancel')} />
      </Form>
    </FormModal>
  )
}
