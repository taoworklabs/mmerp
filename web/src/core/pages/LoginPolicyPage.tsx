import { useQuery, useQueryClient } from '@tanstack/react-query'
import { IconLock, IconPassword } from '@tabler/icons-react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type LoginPolicy } from '@/shared/api/core'
import { CheckboxField, Form, FormActions, FormSection, IntegerField } from '@/shared/ui/form'
import { notifySuccess } from '@/shared/ui/notify'
import { Page } from '@/shared/ui/page'
import { QueryView } from '@/shared/ui/QueryView'

const key = ['core', 'login-policy'] as const

// LoginPolicyPage: the password rules and the lockout of failed sign-ins, one policy for the company.
export function LoginPolicyPage() {
  const { t } = useTranslation()
  const policy = useQuery({
    queryKey: key,
    queryFn: () => unwrap(api.GET('/login-policy')),
  })
  return (
    <Page centered title={t('core.policy.title')} description={t('core.policy.description')}>
      <QueryView query={policy}>{(data) => <PolicyForm key={JSON.stringify(data)} current={data} />}</QueryView>
    </Page>
  )
}

function PolicyForm({ current }: { current: LoginPolicy }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const form = useForm<LoginPolicy>({ mode: 'onBlur', defaultValues: current })

  async function save(v: LoginPolicy) {
    await unwrap(
      api.PUT('/login-policy', {
        body: {
          min_length: v.min_length,
          upper: v.upper,
          lower: v.lower,
          digit: v.digit,
          symbol: v.symbol,
          max_failures: v.max_failures,
          lock_minutes: v.lock_minutes,
        },
      }),
    )
    await qc.invalidateQueries({ queryKey: key })
    notifySuccess(t('core.policy.saved'))
  }

  return (
    <Form form={form} onSubmit={save}>
      <FormSection title={t('core.policy.section.password')} description={t('core.policy.section.password_hint')} icon={IconPassword}>
        <IntegerField name="min_length" label={t('core.policy.min_length')} required min={8} max={64} />
        <CheckboxField name="upper" label={t('core.policy.upper')} />
        <CheckboxField name="lower" label={t('core.policy.lower')} />
        <CheckboxField name="digit" label={t('core.policy.digit')} />
        <CheckboxField name="symbol" label={t('core.policy.symbol')} />
      </FormSection>
      <FormSection title={t('core.policy.section.lockout')} description={t('core.policy.section.lockout_hint')} icon={IconLock}>
        <IntegerField name="max_failures" label={t('core.policy.max_failures')} required min={3} max={20} />
        <IntegerField name="lock_minutes" label={t('core.policy.lock_minutes')} required min={1} max={1440} />
      </FormSection>
      <FormActions submitLabel={t('core.common.save')} />
    </Form>
  )
}
