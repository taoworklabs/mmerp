import { IconCalendarEvent } from '@tabler/icons-react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type Leave } from '@/shared/api/hrm'
import { useDocumentMutation } from '@/shared/document'
import { notifySuccess } from '@/shared/ui/notify'
import { DateField, DecimalField, Form, FormActions, FormFields, FormSection, SelectField, TextField } from '@/shared/ui/form'
import { useLeaveTypes, useLeaveWorkingDays } from '../hooks/useLeave'
import { EmployeeField } from './EmployeeField'
import { leaveDocType } from '../keys'

type Values = {
  employee_id: string | null
  leave_type_id: string | null
  start_date: string | null
  end_date: string | null
  days: string
  reason: string
}

const fieldOfCode: Record<string, keyof Values> = {
  invalid_leave_days: 'days',
  leave_days_exceed_working_days: 'days',
  leave_end_before_start: 'end_date',
  leave_no_working_days: 'start_date',
  leave_overlaps: 'start_date',
  leave_spans_years: 'end_date',
  leave_type_inactive: 'leave_type_id',
  not_an_employee: 'employee_id',
}

type Props = {
  leave?: Leave
  // A new request: who it is for. Without chooseEmployee it is the user's own.
  self?: { id: number; name: string } | null
  chooseEmployee?: boolean
  onCreated?: (id: number) => Promise<void>
  // Editing a draft in its card on the request's page: Cancel, and after saving, close the form.
  onClose?: () => void
}

// LeaveForm creates a leave request, or edits a draft when its allowed_actions has edit (in its
// card with onClose); otherwise it is read-only.
export function LeaveForm({ leave, self, chooseEmployee, onCreated, onClose }: Props) {
  const { t } = useTranslation()
  const types = useLeaveTypes()
  const mutation = useDocumentMutation(leaveDocType, leave?.id ?? 0, leave?.version ?? 0)
  const form = useForm<Values>({
    mode: 'onBlur',
    defaultValues: {
      employee_id: leave ? String(leave.employee_id) : self ? String(self.id) : null,
      leave_type_id: leave ? String(leave.leave_type_id) : null,
      start_date: leave?.start_date ?? null,
      end_date: leave?.end_date ?? null,
      days: leave?.days ?? '',
      reason: leave?.reason ?? '',
    },
  })
  const editable = leave ? leave.allowed_actions.includes('edit') : true
  const ro = !editable
  // The day count may not exceed the working days of the range, so show that ceiling as
  // soon as both dates are known rather than letting the user find it by being refused.
  const [start, end, chosen] = form.watch(['start_date', 'end_date', 'employee_id'])
  const forEmployee = leave?.employee_id ?? (chosen && Number(chosen) !== self?.id ? Number(chosen) : undefined)
  const working = useLeaveWorkingDays(ro ? null : start, ro ? null : end, forEmployee)
  const daysHint = working.data
    ? `${t('hrm.leave.days_hint')} ${t('hrm.leave.working_days', { days: working.data.working_days })}`
    : t('hrm.leave.days_hint')
  // Inactive kinds stay visible on requests that already use them.
  const options = (types.data ?? []).filter((x) => x.active || String(x.id) === form.getValues('leave_type_id')).map((x) => ({ value: String(x.id), label: x.name }))

  async function save(v: Values) {
    const fields = {
      leave_type_id: Number(v.leave_type_id),
      start_date: v.start_date ?? '',
      end_date: v.end_date ?? '',
      days: v.days,
      reason: v.reason.trim() === '' ? null : v.reason.trim(),
    }
    if (leave) {
      await mutation.run((version) => unwrap(api.PUT('/hrm/leaves/{id}', { params: { path: { id: leave.id } }, body: { version, ...fields } })))
      notifySuccess(t('hrm.leave.saved'))
      onClose?.()
      return
    }
    const employee = chooseEmployee && v.employee_id && Number(v.employee_id) !== self?.id ? Number(v.employee_id) : undefined
    const { id } = await unwrap(api.POST('/hrm/leaves', { body: { employee_id: employee, ...fields } }))
    await onCreated?.(id)
  }

  const inputs = (
    <>
      {!leave && chooseEmployee && (
        <EmployeeField name="employee_id" label={t('hrm.leave.employee')} required current={self ? { id: self.id, code: '', name: self.name } : undefined} />
      )}
      <SelectField name="leave_type_id" label={t('hrm.leave.type')} required readOnly={ro} data={options} />
      <DecimalField name="days" label={t('hrm.leave.days')} description={daysHint} required readOnly={ro} scale={1} step={0.5} min={0.5} />
      <DateField name="start_date" label={t('hrm.leave.start_date')} required readOnly={ro} />
      <DateField name="end_date" label={t('hrm.leave.end_date')} required readOnly={ro} />
      <TextField name="reason" label={t('hrm.leave.reason')} description={t('hrm.leave.reason_hint')} multiline maxLength={1000} readOnly={ro} />
    </>
  )

  if (onClose)
    return (
      <Form form={form} onSubmit={save} fieldOf={(err) => fieldOfCode[err.code]}>
        <FormFields>{inputs}</FormFields>
        <FormActions submitLabel={t('hrm.common.save')} onCancel={onClose} cancelLabel={t('hrm.common.cancel')} />
        {mutation.dialog}
      </Form>
    )
  return (
    <Form form={form} onSubmit={save} fieldOf={(err) => fieldOfCode[err.code]}>
      <FormSection title={t('hrm.leave.section')} description={t('hrm.leave.section_hint')} icon={IconCalendarEvent} compact>
        {inputs}
      </FormSection>
      {editable && <FormActions submitLabel={leave ? t('hrm.common.save') : t('hrm.common.create_draft')} cancelTo={leave ? undefined : '/hrm/leaves'} />}
      {mutation.dialog}
    </Form>
  )
}
