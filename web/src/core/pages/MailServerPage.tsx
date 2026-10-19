import { Button, Group } from '@mantine/core'
import { IconLogin, IconMailForward, IconSend, IconServer, IconTrash } from '@tabler/icons-react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, ApiError, unwrap, type MailServer } from '@/shared/api/core'
import { Form, FormActions, FormSection, IntegerField, PasswordField, SelectField, TextField } from '@/shared/ui/form'
import { notifySuccess } from '@/shared/ui/notify'
import { Page } from '@/shared/ui/page'
import { QueryView } from '@/shared/ui/QueryView'
import { EmptyState, ErrorState } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import { useAction } from '@/shared/ui/useAction'
import { coreKeys } from '../keys'

// MailServerPage: the company's SMTP server, viewed and edited on the page. Without one,
// notifications stay in the app. The password is never read back; a test message to oneself
// checks the settings.
export function MailServerPage() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [configuring, setConfiguring] = useState(false)
  const action = useAction()
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

  const refresh = () => qc.invalidateQueries({ queryKey: coreKeys.mailServer() })
  const test = () =>
    action.run(
      async () => {
        await unwrap(api.POST('/mail-server/test'))
        await refresh()
      },
      { success: t('core.mail.test_sent') },
    )
  async function remove() {
    const done = await action.run(
      async () => {
        await unwrap(api.DELETE('/mail-server'))
        await refresh()
      },
      { confirm: { title: t('core.mail.delete_title'), message: t('core.mail.delete_confirm'), confirmLabel: t('core.mail.delete'), danger: true }, success: t('core.mail.deleted') },
    )
    if (done) setConfiguring(false)
  }

  const body = (
    <QueryView
      query={server}
      // While configuring, the empty server shows as an empty form.
      empty={
        configuring ? undefined : (
          <EmptyState
            title={t('core.mail.empty')}
            description={t('core.mail.empty_description')}
            action={<Button onClick={() => setConfiguring(true)}>{t('core.mail.configure')}</Button>}
          />
        )
      }
    >
      {(data) => <MailServerForm key={JSON.stringify(data)} current={data ?? null} />}
    </QueryView>
  )

  return (
    <Page
      centered
      title={t('core.mail.title')}
      description={t('core.mail.description')}
      actions={
        server.data && (
          <Group gap="xs">
            <Button variant="subtle" color="danger" leftSection={<IconTrash {...icon.button} />} onClick={() => void remove()}>
              {t('core.mail.delete')}
            </Button>
            <Button variant="default" leftSection={<IconMailForward {...icon.button} />} onClick={() => void test()}>
              {t('core.mail.test')}
            </Button>
          </Group>
        )
      }
    >
      {action.error && <ErrorState message={action.error} />}
      {body}
      {action.dialog}
    </Page>
  )
}

type Values = { host: string; port: number | null; security: string; username: string; password: string; from_address: string; base_url: string }

// MailServerForm is the server's settings in three groups: where, who logs in, what mail says.
function MailServerForm({ current }: { current: MailServer | null }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const form = useForm<Values>({
    mode: 'onBlur',
    defaultValues: {
      host: current?.host ?? '',
      port: current?.port ?? 587,
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
          port: v.port ?? 0,
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
  }

  return (
    <Form form={form} onSubmit={save}>
      <FormSection title={t('core.mail.section.server')} description={t('core.mail.section.server_hint')} icon={IconServer}>
        <TextField name="host" label={t('core.mail.host')} required maxLength={255} />
        <IntegerField name="port" label={t('core.mail.port')} required min={1} max={65535} />
        <SelectField
          name="security"
          label={t('core.mail.security')}
          required
          data={['starttls', 'tls', 'none'].map((v) => ({ value: v, label: t(`core.mail.security.${v}`) }))}
        />
      </FormSection>
      <FormSection title={t('core.mail.section.login')} description={t('core.mail.section.login_hint')} icon={IconLogin}>
        <TextField name="username" label={t('core.mail.username')} maxLength={255} autoComplete="off" />
        <PasswordField name="password" label={t('core.mail.password')} description={current?.password_set ? t('core.mail.password_keep') : undefined} />
      </FormSection>
      <FormSection title={t('core.mail.section.sender')} description={t('core.mail.section.sender_hint')} icon={IconSend}>
        <TextField name="from_address" label={t('core.mail.from_address')} type="email" required maxLength={254} />
        <TextField name="base_url" label={t('core.mail.base_url')} description={t('core.mail.base_url_hint')} required maxLength={255} />
      </FormSection>
      <FormActions submitLabel={t('core.common.save')} />
    </Form>
  )
}
