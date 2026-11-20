import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap } from '@/shared/api/accounting'
import { Form, FormActions, SelectField } from '@/shared/ui/form'
import { Notice } from '@/shared/ui/Notice'
import { notifySuccess } from '@/shared/ui/notify'
import { FormModal } from '@/shared/ui/page'
import { accountingKeys } from '../keys'

// ApplyTemplateModal applies a chart template to an empty chart: accounts, rules, closing
// accounts and statement forms in one go. It cannot be applied twice.
export function ApplyTemplateModal({ onClose }: { onClose: () => void }) {
  const { t, i18n } = useTranslation()
  const qc = useQueryClient()
  const templates = useQuery({ queryKey: accountingKeys.templates(), queryFn: async () => (await unwrap(api.GET('/accounting/templates'))).items })
  const options = (templates.data ?? []).map((x) => ({ value: x.code, label: x.name[i18n.language] ?? x.name.vi ?? x.code }))
  const form = useForm<{ code: string | null }>({ mode: 'onBlur', defaultValues: { code: null } })
  async function apply(v: { code: string | null }) {
    await unwrap(api.POST('/accounting/templates/{code}/apply', { params: { path: { code: v.code ?? '' } } }))
    await qc.invalidateQueries({ queryKey: ['accounting'] })
    notifySuccess(t('accounting.template.applied'))
    onClose()
  }
  return (
    <FormModal open title={t('accounting.template.title')} onOpenChange={onClose}>
      <Form form={form} onSubmit={apply} errorFields={{ chart_not_empty: 'code' }}>
        <SelectField name="code" label={t('accounting.template.choose')} required data={options} />
        <Notice tone="info">{t('accounting.template.hint')}</Notice>
        <FormActions submitLabel={t('accounting.template.apply')} onCancel={onClose} cancelLabel={t('accounting.common.cancel')} />
      </Form>
    </FormModal>
  )
}
