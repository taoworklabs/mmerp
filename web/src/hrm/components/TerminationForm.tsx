import { IconCash, IconClipboardList, IconUserOff } from '@tabler/icons-react'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type Termination, type TerminationFields, type TerminationReason } from '@/shared/api/hrm'
import { useDocumentMutation } from '@/shared/document'
import { notifySuccess } from '@/shared/ui/notify'
import { CheckboxField, DateField, Form, FormActions, FormFields, FormSection, IntegerField, SelectField, TextField, type ErrorFields } from '@/shared/ui/form'
import { terminationDocType } from '../keys'

// The reasons of the Labour Code, in its order; `satisfies` fails tsc if the API changes them.
export const terminationReasons = [
  'contract_expired',
  'work_completed',
  'mutual_agreement',
  'imprisonment',
  'foreigner_expelled',
  'death',
  'employer_ceased',
  'dismissal',
  'employee_resignation',
  'employer_unilateral',
  'absent_without_reason',
  'restructuring',
  'work_permit_expired',
  'probation_failed',
] as const satisfies readonly TerminationReason[]

type Values = {
  termination_date: string | null
  reason: string | null
  decision_date: string | null
  pension_eligible: boolean
  insured_months: number | null
  paid_months: number | null
  handover: string
  note: string
}

const errorFields: ErrorFields<Values> = {
  termination_before_hire: 'termination_date',
  invalid_termination_reason: 'reason',
  invalid_termination_months: 'insured_months',
}

export type TerminationPart = 'main' | 'allowance' | 'handover'

type Props = {
  termination?: Termination
  employeeId?: number
  onCreated?: (id: number) => Promise<void>
  // Editing one card of a draft: that part's fields only; the whole termination saves.
  part?: TerminationPart
  onClose?: () => void
}

// TerminationForm creates a termination, or edits one card of a draft in place.
export function TerminationForm({ termination: x, employeeId, onCreated, part, onClose }: Props) {
  const { t } = useTranslation()
  // One id per opened form: sending it twice returns the document made the first time.
  const [requestId] = useState(() => crypto.randomUUID())
  const mutation = useDocumentMutation(terminationDocType, x?.id ?? 0, x?.version ?? 0)
  const form = useForm<Values>({
    mode: 'onBlur',
    defaultValues: {
      termination_date: x?.termination_date ?? null,
      reason: x?.reason ?? null,
      decision_date: x?.decision_date ?? null,
      pension_eligible: x?.pension_eligible ?? false,
      insured_months: x?.insured_months ?? null,
      paid_months: x?.paid_months ?? 0,
      handover: x?.handover ?? '',
      note: x?.note ?? '',
    },
  })

  async function save(v: Values) {
    const body: TerminationFields = {
      termination_date: v.termination_date ?? '',
      reason: v.reason as TerminationReason,
      decision_date: v.decision_date,
      pension_eligible: v.pension_eligible,
      insured_months: v.insured_months ?? 0,
      paid_months: v.paid_months ?? 0,
      handover: v.handover.trim() || null,
      note: v.note.trim() || null,
    }
    if (x) {
      await mutation.run((version) => unwrap(api.PUT('/hrm/terminations/{id}', { params: { path: { id: x.id } }, body: { version, ...body } })))
      notifySuccess(t('hrm.termination.saved'))
      onClose?.()
      return
    }
    const { id } = await unwrap(api.POST('/hrm/terminations', { body: { request_id: requestId, employee_id: employeeId ?? 0, ...body } }))
    await onCreated?.(id)
  }

  const reasons = terminationReasons.map((r) => ({ value: r, label: t(`hrm.termination.reason.${r}`) }))
  const fields = {
    main: (
      <>
        <DateField name="termination_date" label={t('hrm.termination.date')} description={t('hrm.termination.date_hint')} required />
        <SelectField name="reason" label={t('hrm.termination.reason')} required data={reasons} />
        <DateField name="decision_date" label={t('hrm.termination.decision_date')} />
      </>
    ),
    allowance: (
      <>
        <IntegerField name="insured_months" label={t('hrm.termination.insured_months')} description={t('hrm.termination.insured_months_hint')} required min={0} max={1200} />
        <IntegerField name="paid_months" label={t('hrm.termination.paid_months')} description={t('hrm.termination.paid_months_hint')} min={0} max={1200} />
        <CheckboxField name="pension_eligible" label={t('hrm.termination.pension_eligible')} description={t('hrm.termination.pension_eligible_hint')} />
      </>
    ),
    handover: (
      <>
        <TextField name="handover" label={t('hrm.termination.handover')} description={t('hrm.termination.handover_hint')} multiline wide maxLength={4000} />
        <TextField name="note" label={t('hrm.termination.note')} multiline wide maxLength={1000} />
      </>
    ),
  }

  if (part && onClose)
    return (
      <Form form={form} onSubmit={save} errorFields={errorFields}>
        <FormFields>{fields[part]}</FormFields>
        <FormActions submitLabel={t('hrm.common.save')} onCancel={onClose} cancelLabel={t('hrm.common.cancel')} />
        {mutation.dialog}
      </Form>
    )
  return (
    <Form form={form} onSubmit={save} errorFields={errorFields}>
      <FormSection title={t('hrm.termination.section')} description={t('hrm.termination.section_hint')} icon={IconUserOff}>
        {fields.main}
      </FormSection>
      <FormSection title={t('hrm.termination.section_allowance')} description={t('hrm.termination.section_allowance_hint')} icon={IconCash}>
        {fields.allowance}
      </FormSection>
      <FormSection title={t('hrm.termination.section_handover')} description={t('hrm.termination.section_handover_hint')} icon={IconClipboardList}>
        {fields.handover}
      </FormSection>
      <FormActions submitLabel={t('hrm.common.create_draft')} cancelTo={`/hrm/employees/${employeeId}`} />
    </Form>
  )
}
