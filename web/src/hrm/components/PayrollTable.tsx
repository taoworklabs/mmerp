import { Menu, Text } from '@mantine/core'
import { IconPrinter } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'
import type { Payroll, PayrollLine, PayrollTotal } from '@/shared/api/hrm'
import { formatDecimal, formatNumber } from '@/shared/i18n'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { icon } from '@/shared/ui/theme'

type Amounts = Omit<PayrollTotal, 'org_unit_id' | 'org_unit_name' | 'employees'>
type Key = keyof Amounts

// The money columns by band, so each band reads as one step of the computation: earnings,
// the employee's insurance, income tax, net pay, then what the employer pays on top.
const bands: [string | null, Key[]][] = [
  ['income', ['earned', 'overtime', 'unused_leave', 'adjustment', 'gross']],
  ['insurance', ['insurance_base', 'social_insurance', 'health_insurance', 'unemployment_insurance']],
  ['tax', ['exempt', 'personal_deduction', 'dependent_deduction', 'taxable', 'income_tax']],
  [null, ['net']],
  ['employer', ['employer_social_insurance', 'employer_health_insurance', 'employer_unemployment_insurance', 'employer_union_fee', 'cost']],
]
const amounts: Key[] = bands.flatMap(([, keys]) => keys)
const bandOf = new Map(bands.flatMap(([band, keys]) => keys.map((k) => [k, band] as const)))
// The results of the computation.
const results: Key[] = ['net', 'cost']
// Department totals show only what adds up meaningfully across people.
const totalsOnly: Key[] = ['gross', 'social_insurance', 'health_insurance', 'unemployment_insurance', 'income_tax', 'net', 'employer_social_insurance', 'employer_health_insurance', 'employer_unemployment_insurance', 'employer_union_fee', 'cost']

// What a card shows below 1024px, where the table becomes a list.
const onCards: Key[] = ['gross', 'income_tax', 'net', 'cost']

function sum(rows: Amounts[]): Amounts {
  const out = Object.fromEntries(amounts.map((k) => [k, 0])) as Amounts
  for (const r of rows) for (const k of amounts) out[k] += r[k]
  return out
}

// A zero shows as a quiet dash, so the eye stops on the figures that are there.
function amount(n: number) {
  return n === 0 ? (
    <Text span c="dimmed">
      –
    </Text>
  ) : (
    formatNumber(n)
  )
}

function moneyColumns<T extends Amounts>(t: (k: string) => string, keys: Key[]): Column<T>[] {
  return keys.map((k) => {
    const band = bandOf.get(k)
    return {
      key: k,
      header: t(`hrm.payroll.col.${k}`),
      role: onCards.includes(k) ? 'meta' : 'hidden',
      numeric: true,
      band: band ? t(`hrm.payroll.band.${band}`) : undefined,
      strong: results.includes(k),
      render: (r: T) => amount(r[k]),
    }
  })
}

// PayrollTable shows each employee's pay grouped by department, with each department's total
// and the grand total; without the amounts of persons, only the department totals. With
// onPrint, each employee's row offers to print their payslip.
export function PayrollTable({ payroll, onPrint }: { payroll: Payroll; onPrint?: (line: PayrollLine) => void }) {
  const { t } = useTranslation()
  const names = new Map(payroll.totals.map((x) => [x.org_unit_id, x.org_unit_name]))
  const grand = sum(payroll.totals)

  if (!payroll.lines) {
    type Row = PayrollTotal & { key: string }
    const columns: Column<Row>[] = [
      { key: 'org_unit', header: t('hrm.payroll.org_unit'), role: 'title', render: (r) => r.org_unit_name },
      { key: 'employees', header: t('hrm.payroll.employees'), role: 'meta', numeric: true, render: (r) => formatNumber(r.employees) },
      ...moneyColumns<Row>(t, totalsOnly),
    ]
    const rows = payroll.totals.map((x) => ({ ...x, key: String(x.org_unit_id) }))
    const employees = payroll.totals.reduce((n, x) => n + x.employees, 0)
    return (
      <DataTable
        label={t('hrm.payroll.totals')}
        columns={columns}
        rows={rows}
        rowKey={(r) => r.key}
        footer={{ ...grand, key: 'total', org_unit_id: 0, org_unit_name: t('hrm.payroll.grand_total'), employees }}
        maxHeight="70dvh"
      />
    )
  }

  const lines = [...payroll.lines].sort((a, b) => (names.get(a.org_unit_id) ?? '').localeCompare(names.get(b.org_unit_id) ?? ''))
  const total = (name: string, rows: Amounts[]): PayrollLine => ({
    ...sum(rows),
    employee_id: 0,
    employee_code: '',
    employee_name: name,
    org_unit_id: 0,
    standard_days: 0,
    paid_days: '',
    overtime_hours: '',
    warnings: [],
  })
  const columns: Column<PayrollLine>[] = [
    {
      key: 'employee',
      header: t('hrm.payroll.employee'),
      role: 'title',
      render: (l) =>
        l.employee_code ? (
          <>
            <Text span size="xs" c="dimmed">
              {l.employee_code}
            </Text>{' '}
            {l.employee_name}
          </>
        ) : (
          l.employee_name
        ),
    },
    { key: 'days', header: t('hrm.payroll.paid_days'), role: 'meta', numeric: true, band: t('hrm.payroll.band.attendance'), render: (l) => (l.paid_days ? `${formatDecimal(l.paid_days)}/${l.standard_days}` : '') },
    {
      key: 'hours',
      header: t('hrm.payroll.overtime_hours'),
      role: 'hidden',
      numeric: true,
      band: t('hrm.payroll.band.attendance'),
      render: (l) => (l.overtime_hours && l.overtime_hours !== '0' ? formatDecimal(l.overtime_hours) : ''),
    },
    ...moneyColumns<PayrollLine>(t, amounts),
  ]
  return (
    <DataTable
      label={t('hrm.payroll.lines')}
      columns={columns}
      rows={lines}
      rowKey={(l) => l.employee_id}
      menu={
        onPrint &&
        ((l) =>
          l.employee_id ? (
            <Menu.Item leftSection={<IconPrinter {...icon.button} />} onClick={() => onPrint(l)}>
              {t('hrm.payroll.print_payslip')}
            </Menu.Item>
          ) : null)
      }
      group={{ of: (l) => names.get(l.org_unit_id) ?? '', total: (rows) => total(t('hrm.payroll.group_total'), rows) }}
      footer={total(t('hrm.payroll.grand_total'), lines)}
      maxHeight="70dvh"
    />
  )
}
