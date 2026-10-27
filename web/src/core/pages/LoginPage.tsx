import { Button, Stack } from '@mmerp/ui'
import { useState } from 'react'
import { FormProvider, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, ApiError, unwrap } from '@/shared/api/core'
import { announceSessionStarted } from '@/shared/auth/session'
import { errorText } from '@/shared/i18n'
import { PasswordField, TextField } from '@/shared/ui/form'
import { AuthPage } from '@/shared/ui/page'
import { ErrorState } from '@/shared/ui/states'
import { nextFromLocation } from '../login'

// The fields without Form: the login screen renders before the router exists, and Form needs it.
export function LoginPage() {
  const { t } = useTranslation()
  const form = useForm({ defaultValues: { login: '', password: '' } })
  const [login, password] = form.watch(['login', 'password'])
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function submit(values: { login: string; password: string }) {
    setBusy(true)
    setError(null)
    try {
      await unwrap(api.POST('/auth/login', { body: values }))
    } catch (err) {
      const code = err instanceof ApiError ? err.code : undefined
      setError(
        code === 'invalid_credentials'
          ? t('core.login.invalid_credentials')
          : code === 'login_temporarily_locked'
            ? t('core.login.temporarily_locked')
            : errorText(t, err),
      )
      setBusy(false)
      return
    }
    announceSessionStarted()
    // A full load runs bootstrap again with the new session.
    const here = location.pathname === '/login' ? '/' : location.pathname + location.search
    location.assign(nextFromLocation() ?? here)
  }

  return (
    <AuthPage title={t('core.login.title')}>
      <FormProvider {...form}>
        <form onSubmit={(e) => void form.handleSubmit(submit)(e)} noValidate>
          <Stack gap="md">
            {error && <ErrorState message={error} />}
            <TextField name="login" label={t('core.login.login')} autoComplete="username" required />
            <PasswordField name="password" label={t('core.login.password')} autoComplete="current-password" required />
            <Button type="submit" loading={busy} disabled={!login || !password}>
              {t('core.login.submit')}
            </Button>
          </Stack>
        </form>
      </FormProvider>
    </AuthPage>
  )
}
