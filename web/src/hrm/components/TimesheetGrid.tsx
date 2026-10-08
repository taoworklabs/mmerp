import { Stack, Text } from '@mantine/core'
import { useController, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type Timesheet } from '@/shared/api/hrm'
import { useDocumentMutation } from '@/shared/document'
import { formatDate, formatDecimal, formatWeekday } from '@/shared/i18n'
import { CellGrid } from '@/shared/ui/CellGrid'
import { Form, FormActions } from '@/shared/ui/form'
import { timesheetDocType } from '../keys'

type Cells = Record<string, string> // "<employee id>:<YYYY-MM-DD>" → the cell as typed

// halves reads a cell as half days: "" and 0 are none, 0,5 one, 1 two; anything else is invalid.
function halves(v: string): number | null {
  const n = v.trim().replace(',', '.')
  if (n === '' || n === '0') return 0
  if (n === '0.5' || n === '.5') return 1
  if (n === '1') return 2
  return null
}

function days(start: string, end: string): string[] {
  const out: string[] = []
  for (let d = new Date(`${start}T00:00:00Z`); d.toISOString().slice(0, 10) <= end; d.setUTCDate(d.getUTCDate() + 1)) out.push(d.toISOString().slice(0, 10))
  return out
}

// TimesheetGrid shows the days worked per employee and day; a draft the user may edit is
// typed in place (0,5 or 1 per cell) and saved as a whole.
export function TimesheetGrid({ timesheet: ts }: { timesheet: Timesheet }) {
  const { t } = useTranslation()
  const editable = ts.allowed_actions.includes('edit')
  const mutation = useDocumentMutation(timesheetDocType, ts.id, ts.version)
  const cells: Cells = Object.fromEntries(ts.lines.map((l) => [`${l.employee_id}:${l.date}`, formatDecimal(l.days)]))
  const form = useForm<{ cells: Cells }>({ defaultValues: { cells } })

  async function save(v: { cells: Cells }) {
    const lines = Object.entries(v.cells)
      .map(([key, value]) => {
        const [employee, date] = key.split(':') as [string, string]
        return { employee_id: Number(employee), date, days: (['', '0.5', '1'] as const)[halves(value) ?? 0] }
      })
      .filter((l): l is { employee_id: number; date: string; days: '0.5' | '1' } => l.days !== '')
    await mutation.run((version) =>
      unwrap(api.PUT('/hrm/timesheets/{id}', { params: { path: { id: ts.id } }, body: { version, month: ts.period_start.slice(0, 7), lines } })),
    )
  }

  if (ts.employees.length === 0) return <Text c="dimmed">{t('hrm.timesheet.no_employees')}</Text>
  return (
    <Form form={form} onSubmit={save}>
      <Stack gap="xs">
        <Text size="sm" c="dimmed">
          {t(editable ? 'hrm.timesheet.grid_hint' : 'hrm.timesheet.grid_hint_read')}
        </Text>
        <Grid timesheet={ts} editable={editable} />
      </Stack>
      {editable && <FormActions submitLabel={t('hrm.common.save')} />}
      {mutation.dialog}
    </Form>
  )
}

function Grid({ timesheet: ts, editable }: { timesheet: Timesheet; editable: boolean }) {
  const { t } = useTranslation()
  const { field } = useController<{ cells: Cells }, 'cells'>({
    name: 'cells',
    rules: { validate: (v) => Object.values(v).every((c) => halves(c) !== null) || t('hrm.timesheet.invalid_cells') },
  })
  const value: Cells = field.value
  const dates = days(ts.period_start, ts.period_end)
  const employees = new Map(ts.employees.map((e) => [String(e.id), e]))
  const offDays = new Set(ts.off_days)
  return (
    <CellGrid
      label={t('hrm.timesheet.grid')}
      corner={t('hrm.timesheet.employee')}
      rows={ts.employees.map((e) => ({ key: String(e.id), header: `${e.code} · ${e.full_name}`, label: e.full_name }))}
      columns={dates.map((d) => ({
        key: d,
        label: formatDate(d),
        header: (
          <Stack gap={0}>
            <span>{Number(d.slice(8))}</span>
            <span>{formatWeekday(d)}</span>
          </Stack>
        ),
      }))}
      value={(r, c) => value[`${r}:${c}`] ?? ''}
      onChange={editable ? (r, c, v) => field.onChange({ ...value, [`${r}:${c}`]: v }) : undefined}
      invalid={(v) => halves(v) === null}
      off={(r, c) => {
        const e = employees.get(r)
        return offDays.has(c) || (!!e && (c < e.hire_date || (e.termination_date !== null && c > e.termination_date)))
      }}
      total={{
        header: t('hrm.timesheet.total'),
        value: (r) => formatDecimal(String(dates.reduce((sum, d) => sum + (halves(value[`${r}:${d}`] ?? '') ?? 0), 0) / 2)),
      }}
    />
  )
}
