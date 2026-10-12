import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type Payroll } from '@/shared/api/hrm'
import { useDocumentMutation } from '@/shared/document'
import { formatMonth, formatNumber } from '@/shared/i18n'
import { DataTable } from '@/shared/ui/DataTable'
import { Form, FormActions, LinesTable, MoneyField, MonthField, TextField } from '@/shared/ui/form'
import { Section } from '@/shared/ui/Section'
import { payrollDocType } from '../keys'
import { EmployeeField } from './EmployeeField'

type Item = { employee_id: string | null; amount: number | null; source_period: string | null; reason: string }

// AdjustmentsSection holds the back pay and recoveries of a payroll: edited in place, one row
// each, while the payroll may be adjusted; saving queues a new computation, whose job onQueued
// follows. Otherwise it lists them read-only.
export function AdjustmentsSection({ payroll, editable, onQueued }: { payroll: Payroll; editable: boolean; onQueued: (job: number) => void }) {
  const { t } = useTranslation()
  const current = payroll.adjustments ?? []
  if (!editable)
    return current.length === 0 ? null : (
      <Section title={t('hrm.payroll.adjustments')}>
        <DataTable
          label={t('hrm.payroll.adjustments')}
          rows={current}
          rowKey={(a) => `${a.employee_id}:${a.source_period}:${a.reason}:${a.amount}`}
          columns={[
            { key: 'employee', header: t('hrm.payroll.employee'), role: 'title', render: (a) => `${a.employee_code} · ${a.employee_name}` },
            { key: 'period', header: t('hrm.payroll.source_period'), role: 'meta', render: (a) => formatMonth(a.source_period) },
            { key: 'reason', header: t('hrm.payroll.reason'), role: 'meta', render: (a) => a.reason },
            { key: 'amount', header: t('hrm.payroll.col.adjustment'), role: 'meta', numeric: true, render: (a) => formatNumber(a.amount) },
          ]}
        />
      </Section>
    )
  return <AdjustmentsForm key={payroll.version} payroll={payroll} onQueued={onQueued} />
}

function AdjustmentsForm({ payroll, onQueued }: { payroll: Payroll; onQueued: (job: number) => void }) {
  const { t } = useTranslation()
  const mutation = useDocumentMutation(payrollDocType, payroll.id, payroll.version)
  const current = payroll.adjustments ?? []
  const form = useForm<{ items: Item[] }>({
    mode: 'onBlur',
    defaultValues: {
      items: current.map((a) => ({ employee_id: String(a.employee_id), amount: a.amount, source_period: a.source_period, reason: a.reason })),
    },
  })

  async function save(v: { items: Item[] }) {
    const body = v.items.map((i) => ({ employee_id: Number(i.employee_id), amount: i.amount ?? 0, source_period: i.source_period ?? '', reason: i.reason.trim() }))
    const out = await mutation.run((version) => unwrap(api.PUT('/hrm/payrolls/{id}/adjustments', { params: { path: { id: payroll.id } }, body: { version, items: body } })))
    onQueued(out.job_id)
  }

  return (
    <Section title={t('hrm.payroll.adjustments')} description={t('hrm.payroll.adjustments_hint')}>
      <Form form={form} onSubmit={save}>
        <LinesTable
          name="items"
          label={t('hrm.payroll.adjustments')}
          columns={[
            {
              key: 'employee',
              header: t('hrm.payroll.employee'),
              required: true,
              width: 240,
              cell: (i, bare) => {
                const was = current[i]
                return (
                  <EmployeeField
                    name={`items.${i}.employee_id`}
                    label={t('hrm.payroll.adjustment_employee', { n: i + 1 })}
                    current={was && { id: was.employee_id, code: was.employee_code, name: was.employee_name }}
                    withTerminated
                    required
                    bare={bare}
                  />
                )
              },
            },
            {
              key: 'amount',
              header: t('hrm.payroll.adjustment_amount_col'),
              numeric: true,
              required: true,
              width: 144,
              cell: (i, bare) => <MoneyField name={`items.${i}.amount`} label={t('hrm.payroll.adjustment_amount', { n: i + 1 })} bare={bare} min={-1e12} required />,
            },
            {
              key: 'period',
              header: t('hrm.payroll.source_period'),
              required: true,
              width: 128,
              cell: (i, bare) => <MonthField name={`items.${i}.source_period`} label={t('hrm.payroll.adjustment_period', { n: i + 1 })} bare={bare} required />,
            },
            {
              key: 'reason',
              header: t('hrm.payroll.reason'),
              required: true,
              cell: (i, bare) => <TextField name={`items.${i}.reason`} label={t('hrm.payroll.adjustment_reason', { n: i + 1 })} bare={bare} required maxLength={1000} />,
            },
          ]}
          newLine={() => ({ employee_id: null, amount: null, source_period: null, reason: '' })}
          addLabel={t('hrm.payroll.add_adjustment')}
          lineLabel={(n) => t('hrm.payroll.adjustment_title', { n })}
          removeLabel={(n) => t('hrm.payroll.remove_adjustment', { n })}
        />
        <FormActions submitLabel={t('hrm.payroll.save_compute')} />
      </Form>
      {mutation.dialog}
    </Section>
  )
}
