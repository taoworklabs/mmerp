import { Button, Menu } from '@mantine/core'
import { IconPencil, IconPlus } from '@tabler/icons-react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type LegalParam } from '@/shared/api/hrm'
import { errorText, formatDate, formatDecimal, formatNumber } from '@/shared/i18n'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { DateField, Form, FormActions, SelectField, TextField } from '@/shared/ui/form'
import { notifySuccess } from '@/shared/ui/notify'
import { FormModal, ListPage } from '@/shared/ui/page'
import { ContentSkeleton, EmptyState, ErrorState } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import { hrmKeys } from '../keys'

// Every key a payroll reads, in the order they are listed.
const keys = [
  'base_salary',
  'min_wage_region_1',
  'min_wage_region_2',
  'min_wage_region_3',
  'min_wage_region_4',
  'si_employee',
  'hi_employee',
  'ui_employee',
  'si_employer',
  'hi_employer',
  'ui_employer',
  'union_employer',
  'insurance_cap_multiplier',
  'personal_deduction',
  'dependent_deduction',
  'pit_brackets',
  'ot_exempt_hours_month',
  'ot_exempt_hours_year',
  'ot_hours_per_day',
  'insurance_skip_days',
]

// Rates are stored as fractions (0.08) and shown as percentages (8%).
const rates = new Set(['si_employee', 'hi_employee', 'ui_employee', 'si_employer', 'hi_employer', 'ui_employer', 'union_employer'])

// percent moves the decimal point two places in the string itself, so no float rounding shows.
function percent(v: string): string {
  const [whole = '0', frac = ''] = v.split('.')
  const digits = (frac + '00').slice(0, 2)
  const rest = frac.slice(2)
  return formatDecimal(`${Number(whole + digits)}${rest ? `.${rest}` : ''}`, 4)
}

// paramValue shows a stored value as people read it; the tax brackets bracket by bracket.
function paramValue(t: (k: string, o?: Record<string, unknown>) => string, p: LegalParam): string {
  if (rates.has(p.key)) return `${percent(p.value)}%`
  if (p.key !== 'pit_brackets') return formatDecimal(p.value, 6)
  try {
    const brackets = JSON.parse(p.value) as [number | null, number][]
    return brackets
      .map(([upper, rate], i) => {
        const r = `${percent(String(rate))}%`
        if (upper !== null) return t('hrm.legal_param.bracket_upto', { upper: formatNumber(upper), rate: r })
        return t('hrm.legal_param.bracket_over', { lower: formatNumber(brackets[i - 1]?.[0] ?? 0), rate: r })
      })
      .join(' · ')
  } catch {
    return p.value
  }
}

// LegalParamsPage: every version of the legal parameters payrolls use; a payroll takes each
// key's version in force on its last day.
export function LegalParamsPage() {
  const { t } = useTranslation()
  const [editing, setEditing] = useState<LegalParam | 'new' | null>(null)
  const list = useQuery({ queryKey: hrmKeys.legalParams(), queryFn: () => unwrap(api.GET('/hrm/legal-params')) })
  const manage = list.data?.allowed_actions.includes('manage') ?? false
  const rows = [...(list.data?.items ?? [])].sort((a, b) => keys.indexOf(a.key) - keys.indexOf(b.key) || a.effective_from.localeCompare(b.effective_from))
  const columns: Column<LegalParam>[] = [
    { key: 'key', header: t('hrm.legal_param.key'), role: 'title', render: (p) => t(`hrm.legal_param.${p.key}`) },
    { key: 'from', header: t('hrm.legal_param.effective_from'), role: 'meta', render: (p) => formatDate(p.effective_from) },
    { key: 'value', header: t('hrm.legal_param.value'), role: 'meta', numeric: true, render: (p) => paramValue(t, p) },
  ]
  let body
  if (list.isPending) body = <ContentSkeleton />
  else if (list.isError) body = <ErrorState message={errorText(t, list.error)} onRetry={() => void list.refetch()} />
  else if (rows.length === 0) body = <EmptyState title={t('hrm.legal_param.empty')} />
  else
    body = (
      <DataTable
        label={t('hrm.legal_param.title')}
        columns={columns}
        rows={rows}
        rowKey={(p) => `${p.key}:${p.effective_from}`}
        menu={
          manage
            ? (p) => (
                <Menu.Item leftSection={<IconPencil {...icon.text} />} onClick={() => setEditing(p)}>
                  {t('hrm.legal_param.edit')}
                </Menu.Item>
              )
            : undefined
        }
      />
    )
  return (
    <ListPage
      title={t('hrm.legal_param.title')}
      description={t('hrm.legal_param.description')}
      action={
        manage && (
          <Button leftSection={<IconPlus {...icon.button} />} onClick={() => setEditing('new')}>
            {t('hrm.legal_param.create')}
          </Button>
        )
      }
    >
      {body}
      {editing && <ParamModal param={editing === 'new' ? null : editing} onClose={() => setEditing(null)} />}
    </ListPage>
  )
}

type ParamForm = { key: string | null; effective_from: string | null; value: string }

function ParamModal({ param, onClose }: { param: LegalParam | null; onClose: () => void }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const form = useForm<ParamForm>({ mode: 'onBlur', defaultValues: param ?? { key: null, effective_from: null, value: '' } })
  async function save(v: ParamForm) {
    await unwrap(
      api.PUT('/hrm/legal-params/{key}/{effective_from}', { params: { path: { key: v.key ?? '', effective_from: v.effective_from ?? '' } }, body: { value: v.value.trim() } }),
    )
    await qc.invalidateQueries({ queryKey: hrmKeys.legalParams() })
    notifySuccess(t('hrm.legal_param.saved'))
    onClose()
  }
  return (
    <FormModal opened title={t(param ? 'hrm.legal_param.edit' : 'hrm.legal_param.create')} onClose={onClose}>
      <Form form={form} onSubmit={save} fieldOf={(err) => (err.code === 'invalid_legal_param' ? 'value' : undefined)}>
        <SelectField name="key" label={t('hrm.legal_param.key')} data={keys.map((k) => ({ value: k, label: t(`hrm.legal_param.${k}`) }))} readOnly={!!param} required />
        <DateField name="effective_from" label={t('hrm.legal_param.effective_from')} readOnly={!!param} required />
        <TextField name="value" label={t('hrm.legal_param.value')} description={t('hrm.legal_param.value_hint')} required maxLength={500} />
        <FormActions submitLabel={t('hrm.common.save')} onCancel={onClose} cancelLabel={t('hrm.common.cancel')} />
      </Form>
    </FormModal>
  )
}
