import { FieldList, Stack } from '@mmerp/ui'
import { IconPrinter } from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router'
import { api, unwrap } from '@/shared/api/hrm'
import { formatDecimal, formatMoney, formatMonth, formatMonthShort } from '@/shared/i18n'
import { ExportButton } from '@/shared/jobs'
import { loaded } from '@/shared/ui/loaded'
import { Page } from '@/shared/ui/page'
import { Section } from '@/shared/ui/Section'
import { hrmKeys } from '../keys'
import { useLeaveSelf } from '../hooks/useLeave'

// PayslipPage shows one payslip with the sections of the printed one, and its PDF.
export function PayslipPage() {
  const { t } = useTranslation()
  const id = Number(useParams().payrollId)
  const self = useLeaveSelf().data
  const q = useQuery({ queryKey: hrmKeys.me.payslip(id), queryFn: () => unwrap(api.GET('/hrm/me/payslips/{payroll_id}', { params: { path: { payroll_id: id } } })) })
  const list = { to: '/me/payslips', label: t('hrm.me.payslips.title') }
  const state = loaded(q, t, { to: list.to, label: t('hrm.me.payslips.back') })
  if (!state.ok) return state.fallback
  const s = state.data
  const a = s.amounts
  const p = (k: string) => t(`hrm.print.payslip.${k}`)
  const money = (...keys: [string, number][]) => keys.map(([k, v]) => [p(k), formatMoney(v)] as [string, string])
  return (
    <Page
      centered
      withBack
      title={p('title')}
      breadcrumbs={[list]}
      description={`${t('hrm.print.payslip.period', { month: formatMonthShort(s.period_start.slice(0, 7)) })} · ${t('hrm.print.payslip.payroll', { number: s.number })}`}
      actions={
        self && <ExportButton kind="printing.hrm.payslip" params={{ id, parts: [self.id] }} label={t('hrm.me.payslips.print')} icon={IconPrinter} />
      }
    >
      <Stack gap="lg">
        <FieldList
          rows={[
            [p('paid_days'), t('hrm.print.payslip.days_of', { paid: formatDecimal(s.paid_days), standard: s.standard_days })],
            [p('overtime_hours'), formatDecimal(s.overtime_hours)],
          ]}
        />
        <Section title={p('income')}>
          <FieldList rows={money(['earned', a.earned], ['overtime', a.overtime], ['unused_leave', a.unused_leave], ['adjustment', a.adjustment], ['gross', a.gross])} />
        </Section>
        <Section title={p('deductions')}>
          <FieldList
            rows={money(
              ['social_insurance', a.social_insurance],
              ['health_insurance', a.health_insurance],
              ['unemployment_insurance', a.unemployment_insurance],
              ['income_tax', a.income_tax],
              ['total_deductions', a.social_insurance + a.health_insurance + a.unemployment_insurance + a.income_tax],
            )}
          />
        </Section>
        <Section title={p('tax')}>
          <FieldList
            rows={money(['insurance_base', a.insurance_base], ['exempt', a.exempt], ['personal_deduction', a.personal_deduction], ['dependent_deduction', a.dependent_deduction], ['taxable', a.taxable])}
          />
        </Section>
        {s.adjustments.length > 0 && (
          <Section title={p('adjustments')}>
            <FieldList rows={s.adjustments.map((x) => [formatMonth(x.source_period), `${formatMoney(x.amount)} · ${x.reason}`])} />
          </Section>
        )}
        <Section title={p('net')}>
          <FieldList rows={money(['net', a.net])} />
        </Section>
      </Stack>
    </Page>
  )
}
