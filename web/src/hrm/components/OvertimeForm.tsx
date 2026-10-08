import { IconClockHour4 } from '@tabler/icons-react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type Overtime } from '@/shared/api/hrm'
import { useDocumentMutation } from '@/shared/document'
import { DateField, DecimalField, Form, FormActions, FormSection, SelectField, TextField } from '@/shared/ui/form'
import { overtimeDocType } from '../keys'
import { EmployeeField } from './EmployeeField'

type Values = {
  employee_id: string | null
  date: string | null
  day_kind: string | null
  day_hours: string
  night_hours: string
  reason: string
}

type DayKind = Overtime['day_kind']
// Keyed by the API's day kinds, so tsc fails when that list changes.
const dayKindSet: Record<DayKind, true> = { weekday: true, weekly_off: true, holiday: true }
const dayKinds = Object.keys(dayKindSet) as DayKind[]

const fieldOfCode: Record<string, keyof Values> = {
  invalid_overtime_hours: 'day_hours',
  not_an_employee: 'employee_id',
}

type Props = {
  overtime?: Overtime
  // A new request: who it is for. Without chooseEmployee it is the user's own.
  self?: { id: number; name: string } | null
  chooseEmployee?: boolean
  onCreated?: (id: number) => Promise<void>
}

// OvertimeForm creates an overtime request, or edits a draft when its allowed_actions has edit;
// otherwise it is read-only.
export function OvertimeForm({ overtime, self, chooseEmployee, onCreated }: Props) {
  const { t } = useTranslation()
  const mutation = useDocumentMutation(overtimeDocType, overtime?.id ?? 0, overtime?.version ?? 0)
  const form = useForm<Values>({
    mode: 'onBlur',
    defaultValues: {
      employee_id: overtime ? String(overtime.employee_id) : self ? String(self.id) : null,
      date: overtime?.date ?? null,
      day_kind: overtime?.day_kind ?? 'weekday',
      day_hours: overtime?.day_hours ?? '0',
      night_hours: overtime?.night_hours ?? '0',
      reason: overtime?.reason ?? '',
    },
  })
  const editable = overtime ? overtime.allowed_actions.includes('edit') : true
  const ro = !editable

  async function save(v: Values) {
    const fields = {
      date: v.date ?? '',
      day_kind: v.day_kind as DayKind,
      day_hours: v.day_hours || '0',
      night_hours: v.night_hours || '0',
      reason: v.reason.trim() === '' ? null : v.reason.trim(),
    }
    if (overtime) {
      await mutation.run((version) => unwrap(api.PUT('/hrm/overtimes/{id}', { params: { path: { id: overtime.id } }, body: { version, ...fields } })))
      return
    }
    const employee = chooseEmployee && v.employee_id && Number(v.employee_id) !== self?.id ? Number(v.employee_id) : undefined
    const { id } = await unwrap(api.POST('/hrm/overtimes', { body: { employee_id: employee, ...fields } }))
    await onCreated?.(id)
  }

  return (
    <Form form={form} onSubmit={save} fieldOf={(err) => fieldOfCode[err.code]}>
      <FormSection title={t('hrm.overtime.section')} description={t('hrm.overtime.section_hint')} icon={IconClockHour4}>
        {!overtime && chooseEmployee && (
          <EmployeeField name="employee_id" label={t('hrm.overtime.employee')} required current={self ? { id: self.id, code: '', name: self.name } : undefined} />
        )}
        <DateField name="date" label={t('hrm.overtime.date')} required readOnly={ro} />
        <SelectField
          name="day_kind"
          label={t('hrm.overtime.day_kind')}
          description={t('hrm.overtime.day_kind_hint')}
          required
          readOnly={ro}
          data={dayKinds.map((k) => ({ value: k, label: t(`hrm.overtime.day_kind.${k}`) }))}
        />
        <DecimalField name="day_hours" label={t('hrm.overtime.day_hours')} description={t('hrm.overtime.hours_hint')} required readOnly={ro} scale={1} step={0.5} min={0} />
        <DecimalField name="night_hours" label={t('hrm.overtime.night_hours')} description={t('hrm.overtime.night_hint')} required readOnly={ro} scale={1} step={0.5} min={0} />
        <TextField name="reason" label={t('hrm.overtime.reason')} multiline maxLength={1000} readOnly={ro} />
      </FormSection>
      {editable && <FormActions submitLabel={overtime ? t('hrm.common.save') : t('hrm.common.create_draft')} />}
      {mutation.dialog}
    </Form>
  )
}
