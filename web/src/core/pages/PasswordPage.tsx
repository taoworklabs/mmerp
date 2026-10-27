import { Button } from '@mmerp/ui'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router'
import { api, ApiError, unwrap } from '@/shared/api/core'
import { useMe } from '@/shared/auth/me'
import { Form, FormActions, FormSection, PasswordField, passwordErrorText } from '@/shared/ui/form'
import { notifySuccess } from '@/shared/ui/notify'
import { Page } from '@/shared/ui/page'
import { PasswordRules } from '../components/PasswordRules'

type Values = { current: string; next: string; confirm: string }

// Change the signed-in user's own password. Forced (an administrator set a temporary one), it is the only page.
export default function PasswordPage() {
  const { t } = useTranslation()
  const me = useMe()
  const navigate = useNavigate()
  const forced = me.must_change_password
  const form = useForm<Values>({ mode: 'onBlur', defaultValues: { current: '', next: '', confirm: '' } })
  const [rules, setRules] = useState<string[]>([])

  async function save(v: Values) {
    setRules([])
    if (v.next !== v.confirm) {
      form.setError('confirm', { message: t('core.password.mismatch') }, { shouldFocus: true })
      return false
    }
    try {
      await unwrap(api.POST('/me/password', { body: { current: v.current, next: v.next } }))
    } catch (err) {
      const code = err instanceof ApiError ? err.code : ''
      if (code === 'wrong_password') {
        form.setError('current', { message: passwordErrorText(t, err)[0] }, { shouldFocus: true })
        return false
      }
      if (code === 'password_weak') {
        setRules(passwordErrorText(t, err))
        form.setFocus('next')
        return false
      }
      throw err
    }
    notifySuccess(t('core.password.changed'))
    // A full load fetches /me again, so the app opens normally.
    if (forced) location.assign('/')
    else void navigate('/')
  }

  return (
    <Page centered title={t('core.password.title')} description={forced ? t('core.password.forced') : undefined}>
      <Form form={form} onSubmit={save}>
        <FormSection title={t('core.password.title')}>
          <PasswordField name="current" label={t('core.password.current')} autoComplete="current-password" required />
          <PasswordField name="next" label={t('core.password.next')} required />
          <PasswordRules rules={rules} />
          <PasswordField name="confirm" label={t('core.password.confirm')} required />
        </FormSection>
        {forced ? <Button type="submit">{t('core.password.submit')}</Button> : <FormActions submitLabel={t('core.password.submit')} cancelTo="/" />}
      </Form>
    </Page>
  )
}
