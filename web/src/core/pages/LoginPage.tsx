import { Alert, Button, PasswordInput, Stack, TextInput } from '@mantine/core'
import { IconAlertCircle } from '@tabler/icons-react'
import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { api, ApiError, unwrap } from '@/shared/api/core'
import { announceSessionStarted } from '@/shared/auth/session'
import { errorText } from '@/shared/i18n'
import { AuthPage } from '@/shared/ui/page'
import { icon } from '@/shared/ui/theme'
import { nextFromLocation } from '../login'

// Plain state, not shared/ui/form: the login screen renders before the router exists, and Form needs it.
export function LoginPage() {
  const { t } = useTranslation()
  const [login, setLogin] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function submit(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      await unwrap(api.POST('/auth/login', { body: { login, password } }))
    } catch (err) {
      setError(err instanceof ApiError && err.code === 'invalid_credentials' ? t('core.login.invalid_credentials') : errorText(t, err))
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
      <form onSubmit={submit} noValidate>
        <Stack gap="md">
          {error && (
            <Alert color="danger" icon={<IconAlertCircle {...icon.button} />} role="alert">
              {error}
            </Alert>
          )}
          <TextInput
            label={t('core.login.login')}
            autoComplete="username"
            required
            value={login}
            onChange={(e) => setLogin(e.currentTarget.value)}
          />
          <PasswordInput
            label={t('core.login.password')}
            autoComplete="current-password"
            required
            value={password}
            onChange={(e) => setPassword(e.currentTarget.value)}
            visibilityToggleButtonProps={{ 'aria-label': t('core.login.toggle_password'), tabIndex: 0 }}
          />
          <Button type="submit" loading={busy} disabled={!login || !password}>
            {t('core.login.submit')}
          </Button>
        </Stack>
      </form>
    </AuthPage>
  )
}
