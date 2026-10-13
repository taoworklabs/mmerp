import { Button } from '@mantine/core'
import { IconAddressBook, IconLock, IconPlus, IconUser } from '@tabler/icons-react'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type Employee, type EmployeeInput } from '@/shared/api/hrm'
import { DateField, Form, FormActions, FormRow, FormSection, OrgUnitField, SelectField, SensitiveField, TextField } from '@/shared/ui/form'
import { icon } from '@/shared/ui/theme'
import { EmployeeField } from './EmployeeField'

const sensitiveFields = ['national_id', 'social_insurance_no', 'tax_code', 'bank_account'] as const
type SensitiveName = (typeof sensitiveFields)[number]

export type EmployeeValues = {
  code: string
  full_name: string
  date_of_birth: string | null
  gender: string | null
  phone: string
  email: string
  address: string
  org_unit_id: string | null
  manager_id: string | null
  user_login: string
  hire_date: string | null
  termination_date: string | null
} & Partial<Record<SensitiveName, string>>

function defaults(e?: Employee): EmployeeValues {
  return {
    code: e?.code ?? '',
    full_name: e?.full_name ?? '',
    date_of_birth: e?.date_of_birth ?? null,
    gender: e?.gender ?? null,
    phone: e?.phone ?? '',
    email: e?.email ?? '',
    address: e?.address ?? '',
    org_unit_id: e ? String(e.org_unit_id) : null,
    manager_id: e?.manager_id ? String(e.manager_id) : null,
    user_login: e?.user_login ?? '',
    hire_date: e?.hire_date ?? null,
    termination_date: e?.termination_date ?? null,
  }
}

const orNull = (s: string) => (s.trim() === '' ? null : s.trim())

// toInput sends every plain field, and only the sensitive fields the user edited.
function toInput(v: EmployeeValues, edited: SensitiveName[]): EmployeeInput {
  const sensitive = Object.fromEntries(edited.map((f) => [f, v[f] ?? '']))
  return {
    code: v.code.trim(),
    full_name: v.full_name.trim(),
    date_of_birth: v.date_of_birth,
    gender: v.gender as EmployeeInput['gender'],
    phone: orNull(v.phone),
    email: orNull(v.email),
    address: orNull(v.address),
    org_unit_id: Number(v.org_unit_id),
    manager_id: v.manager_id ? Number(v.manager_id) : null,
    user_login: orNull(v.user_login),
    hire_date: v.hire_date ?? '',
    termination_date: v.termination_date,
    sensitive: edited.length ? sensitive : undefined,
  }
}

const fieldOfCode: Record<string, keyof EmployeeValues> = {
  employee_code_taken: 'code',
  user_not_found: 'user_login',
  user_already_linked: 'user_login',
  manager_not_found: 'manager_id',
  manager_cycle: 'manager_id',
  manager_self: 'manager_id',
  termination_before_hire: 'termination_date',
}

// EmployeeForm creates (no employee) or edits an employee: read-only without `edit`, or for a new one without `create`.
// actions: the employee's allowed_actions, or for a new one, the list-level actions from the API.
export function EmployeeForm({ employee, actions, onSaved }: { employee?: Employee; actions: string[]; onSaved: (id: number) => Promise<void> | void }) {
  const { t } = useTranslation()
  const form = useForm<EmployeeValues>({ mode: 'onBlur', defaultValues: defaults(employee) })
  const canEdit = actions.includes(employee ? 'edit' : 'create')
  const canViewSensitive = actions.includes('view_sensitive')
  // Creating, the sensitive details stay folded until asked for: they usually come later, with the papers.
  const [sensitiveOpen, setSensitiveOpen] = useState(false)

  async function save(v: EmployeeValues) {
    const dirty = form.formState.dirtyFields
    const edited = sensitiveFields.filter((f) => dirty[f])
    const body = toInput(v, edited)
    if (employee) {
      await unwrap(api.PUT('/hrm/employees/{id}', { params: { path: { id: employee.id } }, body }))
      await onSaved(employee.id)
    } else {
      const { id } = await unwrap(api.POST('/hrm/employees', { body }))
      await onSaved(id)
    }
  }

  const ro = !canEdit
  const reveal = (field: SensitiveName) => async () => {
    if (!employee) return null
    const { value } = await unwrap(api.GET('/hrm/employees/{id}/sensitive/{field}', { params: { path: { id: employee.id, field } } }))
    return value
  }

  return (
    <Form form={form} onSubmit={save} fieldOf={(err) => fieldOfCode[err.code]}>
      {/* What every employee needs comes first, the four required fields together. */}
      <FormSection title={t('hrm.employee.section.main')} description={t('hrm.employee.section.main_hint')} icon={IconUser}>
        <TextField name="full_name" label={t('hrm.employee.full_name')} required maxLength={200} readOnly={ro} wide />
        <TextField name="code" label={t('hrm.employee.code')} required maxLength={50} readOnly={ro} />
        <OrgUnitField
          name="org_unit_id"
          label={t('hrm.employee.org_unit')}
          required
          product="hrm"
          permission={ro ? 'hrm.employee.view' : 'hrm.employee.edit'}
          readOnly={ro}
        />
        <DateField name="hire_date" label={t('hrm.employee.hire_date')} required readOnly={ro} />
        <EmployeeField
          name="manager_id"
          label={t('hrm.employee.manager')}
          readOnly={ro}
          managerOf={employee?.id}
          current={
            employee?.manager_id && employee.manager_code && employee.manager_name
              ? { id: employee.manager_id, code: employee.manager_code, name: employee.manager_name }
              : undefined
          }
        />
        <TextField name="user_login" label={t('hrm.employee.user_login')} description={t('hrm.employee.user_login_hint')} maxLength={200} readOnly={ro} />
        {/* Leaving is recorded on an existing employee, never when creating one. */}
        {employee && <DateField name="termination_date" label={t('hrm.employee.termination_date')} clearable readOnly={ro} />}
      </FormSection>
      <FormSection title={t('hrm.employee.section.personal')} description={t('hrm.employee.section.personal_hint')} icon={IconAddressBook}>
        <DateField name="date_of_birth" label={t('hrm.employee.date_of_birth')} clearable readOnly={ro} short />
        <SelectField
          name="gender"
          label={t('hrm.employee.gender')}
          clearable
          readOnly={ro}
          short
          data={['male', 'female', 'other'].map((g) => ({ value: g, label: t(`hrm.employee.gender.${g}`) }))}
        />
        <TextField name="phone" label={t('hrm.employee.phone')} type="tel" maxLength={50} readOnly={ro} short placeholder={t('hrm.employee.phone_placeholder')} />
        <TextField name="email" label={t('hrm.employee.email')} type="email" maxLength={200} readOnly={ro} />
        <TextField name="address" label={t('hrm.employee.address')} maxLength={500} readOnly={ro} wide />
      </FormSection>
      <FormSection title={t('hrm.employee.section.sensitive')} description={t('hrm.employee.section.sensitive_hint')} icon={IconLock}>
        {employee || sensitiveOpen ? (
          sensitiveFields.map((f) => (
            <SensitiveField
              key={f}
              name={f}
              label={t(`hrm.employee.${f}`)}
              present={employee?.sensitive[f] ?? false}
              canView={canViewSensitive}
              canEdit={canEdit}
              reveal={reveal(f)}
            />
          ))
        ) : (
          // Where the fields will appear, so the section keeps its place in the form.
          <FormRow>
            <Button
              variant="light"
              leftSection={<IconPlus {...icon.button} />}
              onClick={() => {
                setSensitiveOpen(true)
                // The button goes away; the keyboard carries on in the first field it opened.
                requestAnimationFrame(() => form.setFocus(sensitiveFields[0]))
              }}
            >
              {t('hrm.employee.add_sensitive')}
            </Button>
          </FormRow>
        )}
      </FormSection>
      {canEdit && (
        <FormActions submitLabel={t(employee ? 'hrm.employee.save' : 'hrm.employees.create')} cancelTo={employee ? undefined : '/hrm/employees'} />
      )}
    </Form>
  )
}
