import { Button } from '@mantine/core'
import { IconAddressBook, IconLock, IconPlus, IconUser } from '@tabler/icons-react'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type Employee, type EmployeeInput } from '@/shared/api/hrm'
import { DateField, Form, FormActions, FormRow, FormSection, OrgUnitField, SelectField, SensitiveField, TextField } from '@/shared/ui/form'
import { icon } from '@/shared/ui/theme'
import { EmployeeField } from './EmployeeField'

export const sensitiveFields = ['national_id', 'social_insurance_no', 'tax_code', 'bank_account'] as const
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

function employeeValues(e?: Employee): EmployeeValues {
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

// useEmployeeForm holds the whole employee whichever part of it is on screen: the API saves it as a whole.
export function useEmployeeForm(employee: Employee | undefined, onSaved: (id: number) => Promise<void> | void) {
  const form = useForm<EmployeeValues>({ mode: 'onBlur', defaultValues: employeeValues(employee) })
  async function save(v: EmployeeValues) {
    const dirty = form.formState.dirtyFields
    const edited = sensitiveFields.filter((f) => dirty[f])
    const body = toInput(v, edited)
    if (employee) {
      await unwrap(api.PUT('/hrm/employees/{id}', { params: { path: { id: employee.id } }, body }))
      await onSaved(employee.id)
    } else {
      const { id } = await unwrap(api.POST('/hrm/employees', { body: { ...body, code: v.code.trim() } }))
      await onSaved(id)
    }
  }
  return { form, save, fieldOf: (err: { code: string }) => fieldOfCode[err.code] }
}

// MainFields: what every employee needs, the four required fields together.
export function MainFields({ employee }: { employee?: Employee }) {
  const { t } = useTranslation()
  return (
    <>
      <TextField name="full_name" label={t('hrm.employee.full_name')} required maxLength={200} wide />
      {/* The code is typed or issued at creation and never changes after. */}
      {employee ? (
        <TextField name="code" label={t('hrm.employee.code')} readOnly />
      ) : (
        <TextField name="code" label={t('hrm.employee.code')} description={t('hrm.employee.code_hint')} maxLength={50} />
      )}
      <OrgUnitField
        name="org_unit_id"
        label={t('hrm.employee.org_unit')}
        required
        product="hrm"
        permission="hrm.employee.edit"
       
      />
      <DateField name="hire_date" label={t('hrm.employee.hire_date')} required />
      <EmployeeField
        name="manager_id"
        label={t('hrm.employee.manager')}
       
        managerOf={employee?.id}
        current={
          employee?.manager_id && employee.manager_code && employee.manager_name
            ? { id: employee.manager_id, code: employee.manager_code, name: employee.manager_name }
            : undefined
        }
      />
      <TextField name="user_login" label={t('hrm.employee.user_login')} description={t('hrm.employee.user_login_hint')} maxLength={200} />
      {/* Leaving is recorded on an existing employee, never when creating one. */}
      {employee && <DateField name="termination_date" label={t('hrm.employee.termination_date')} clearable />}
    </>
  )
}

export function PersonalFields() {
  const { t } = useTranslation()
  return (
    <>
      <DateField name="date_of_birth" label={t('hrm.employee.date_of_birth')} clearable short />
      <SelectField
        name="gender"
        label={t('hrm.employee.gender')}
        clearable
       
        short
        data={['male', 'female', 'other'].map((g) => ({ value: g, label: t(`hrm.employee.gender.${g}`) }))}
      />
      <TextField name="phone" label={t('hrm.employee.phone')} type="tel" maxLength={50} short placeholder={t('hrm.employee.phone_placeholder')} />
      <TextField name="email" label={t('hrm.employee.email')} type="email" maxLength={200} />
      <TextField name="address" label={t('hrm.employee.address')} maxLength={500} wide />
    </>
  )
}

// revealSensitive fetches one sensitive value; the backend audits each call.
export const revealSensitive = (id: number, field: SensitiveName) => async () => {
  const { value } = await unwrap(api.GET('/hrm/employees/{id}/sensitive/{field}', { params: { path: { id, field } } }))
  return value
}

// SensitiveFields are masked until shown; each reveal is an audited read.
export function SensitiveFields({ employee, canView, canEdit }: { employee?: Employee; canView: boolean; canEdit: boolean }) {
  const { t } = useTranslation()
  const reveal = (field: SensitiveName) => (employee ? revealSensitive(employee.id, field) : async () => null)
  return sensitiveFields.map((f) => (
    <SensitiveField
      key={f}
      name={f}
      label={t(`hrm.employee.${f}`)}
      present={employee?.sensitive[f] ?? false}
      canView={canView}
      canEdit={canEdit}
      reveal={reveal(f)}
    />
  ))
}

// EmployeeForm creates an employee; an existing one is edited a part at a time on its page.
// actions: the list-level actions from the API (it is shown only with create).
export function EmployeeForm({ actions, onSaved }: { actions: string[]; onSaved: (id: number) => Promise<void> | void }) {
  const { t } = useTranslation()
  const { form, save, fieldOf } = useEmployeeForm(undefined, onSaved)
  // The sensitive details stay folded until asked for: they usually come later, with the papers.
  const [sensitiveOpen, setSensitiveOpen] = useState(false)

  return (
    <Form form={form} onSubmit={save} fieldOf={fieldOf}>
      <FormSection title={t('hrm.employee.section.main')} description={t('hrm.employee.section.main_hint')} icon={IconUser}>
        <MainFields />
      </FormSection>
      <FormSection title={t('hrm.employee.section.personal')} description={t('hrm.employee.section.personal_hint')} icon={IconAddressBook}>
        <PersonalFields />
      </FormSection>
      <FormSection title={t('hrm.employee.section.sensitive')} description={t('hrm.employee.section.sensitive_hint')} icon={IconLock}>
        {sensitiveOpen ? (
          <SensitiveFields canView={actions.includes('view_sensitive')} canEdit />
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
      <FormActions submitLabel={t('hrm.employees.create')} cancelTo="/hrm/employees" />
    </Form>
  )
}
