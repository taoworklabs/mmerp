import { IconCalendarEvent } from '@tabler/icons-react'
import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type Leave } from '@/shared/api/hrm'
import { useDocumentMutation } from '@/shared/document'
import { formatDecimal } from '@/shared/i18n'
import { notifySuccess } from '@/shared/ui/notify'
import { CheckboxField, DateField, DecimalField, Form, FormActions, FormFields, FormSection, SelectField, TextField, type ErrorFields } from '@/shared/ui/form'
import { useLeaveTypes, useLeaveWorkingDays } from '../hooks/useLeave'
import { EmployeeField } from './EmployeeField'
import { leaveDocType } from '../keys'
import { useRequestPaths } from '../requestPaths'
import { leaveDays } from '../leaveDays'

type Values = {
  employee_id: string | null
  leave_type_id: string | null
  start_date: string | null
  end_date: string | null
  days: string
  start_half: boolean
  end_half: boolean
  reason: string
}

const errorFields: ErrorFields<Values> = {
  invalid_leave_days: 'days',
  leave_days_changed: 'days',
  leave_days_exceed_working_days: 'days',
  leave_days_exceed_type_max: 'days',
  leave_end_before_start: 'end_date',
  leave_half_days: 'end_half',
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
  const paths = useRequestPaths('leave')
  // One id per opened form: sending it twice returns the document made the first time.
  const [requestId] = useState(() => crypto.randomUUID())
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
      start_half: leave?.start_half ?? false,
      end_half: leave?.end_half ?? false,
      reason: leave?.reason ?? '',
    },
  })
  const editable = leave ? leave.allowed_actions.includes('edit') : true
  const ro = !editable
  // The day count comes from the calendar: the working days of the range less the half days,
  // shown as soon as both dates are known.
  const [start, end, chosen, typeID, startHalf, endHalf] = form.watch(['start_date', 'end_date', 'employee_id', 'leave_type_id', 'start_half', 'end_half'])
  const forEmployee = leave?.employee_id ?? (chosen && Number(chosen) !== self?.id ? Number(chosen) : undefined)
  const working = useLeaveWorkingDays(ro ? null : start, ro ? null : end, forEmployee)
  const oneDay = !!start && start === end
  const computed = working.data ? leaveDays(working.data.working_days, startHalf, !oneDay && endHalf) : null
  useEffect(() => {
    if (!ro) form.setValue('days', computed ?? '')
  }, [form, ro, computed])
  // The chosen kind's own rules, shown before they refuse anything.
  const kind = types.data?.find((x) => String(x.id) === typeID)
  const daysHint = [
    t('hrm.leave.days_hint'),
    working.data && t('hrm.leave.working_days', { days: working.data.working_days }),
    !ro && kind?.max_days && t('hrm.leave.type_max_days', { days: formatDecimal(kind.max_days) }),
  ]
    .filter(Boolean)
    .join(' ')
  // Inactive kinds stay visible on requests that already use them.
  const options = (types.data ?? []).filter((x) => x.active || String(x.id) === form.getValues('leave_type_id')).map((x) => ({ value: String(x.id), label: x.name }))

  async function save(v: Values) {
    const fields = {
      leave_type_id: Number(v.leave_type_id),
      start_date: v.start_date ?? '',
      end_date: v.end_date ?? '',
      // The count the user saw: refused if the calendar now gives another.
      days: v.days === '' ? undefined : v.days,
      start_half: v.start_half,
      end_half: !oneDay && v.end_half,
      reason: v.reason.trim() === '' ? null : v.reason.trim(),
    }
    if (leave) {
      await mutation.run((version) => unwrap(api.PUT('/hrm/leaves/{id}', { params: { path: { id: leave.id } }, body: { version, ...fields } })))
      notifySuccess(t('hrm.leave.saved'))
      onClose?.()
      return
    }
    const employee = chooseEmployee && v.employee_id && Number(v.employee_id) !== self?.id ? Number(v.employee_id) : undefined
    const { id } = await unwrap(api.POST('/hrm/leaves', { body: { request_id: requestId, employee_id: employee, ...fields } }))
    await onCreated?.(id)
  }

  const inputs = (
    <>
      {!leave && chooseEmployee && (
        <EmployeeField name="employee_id" label={t('hrm.leave.employee')} required current={self ? { id: self.id, code: '', name: self.name } : undefined} />
      )}
      <SelectField
        name="leave_type_id"
        label={t('hrm.leave.type')}
        description={!ro && kind?.requires_attachment ? t('hrm.leave.type_needs_attachment') : undefined}
        required
        readOnly={ro}
        data={options}
      />
      <DateField name="start_date" label={t('hrm.leave.start_date')} required readOnly={ro} />
      <DateField name="end_date" label={t('hrm.leave.end_date')} description={daysHint} required readOnly={ro} />
      <CheckboxField name="start_half" label={oneDay ? t('hrm.leave.half_day') : t('hrm.leave.start_half')} readOnly={ro} />
      {!oneDay && <CheckboxField name="end_half" label={t('hrm.leave.end_half')} readOnly={ro} />}
      <DecimalField name="days" label={t('hrm.leave.days')} readOnly scale={1} />
      <TextField name="reason" label={t('hrm.leave.reason')} description={t('hrm.leave.reason_hint')} multiline maxLength={1000} readOnly={ro} />
    </>
  )

  if (onClose)
    return (
      <Form form={form} onSubmit={save} errorFields={errorFields}>
        <FormFields>{inputs}</FormFields>
        <FormActions submitLabel={t('hrm.common.save')} onCancel={onClose} cancelLabel={t('hrm.common.cancel')} />
        {mutation.dialog}
      </Form>
    )
  return (
    <Form form={form} onSubmit={save} errorFields={errorFields}>
      <FormSection title={t('hrm.leave.section')} description={t('hrm.leave.section_hint')} icon={IconCalendarEvent} compact>
        {inputs}
      </FormSection>
      {editable && <FormActions submitLabel={leave ? t('hrm.common.save') : t('hrm.common.create_draft')} cancelTo={leave ? undefined : paths.list} />}
      {mutation.dialog}
    </Form>
  )
}
