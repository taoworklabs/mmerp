import { Anchor } from '@mantine/core'
import { IconAddressBook, IconBriefcase, IconLock } from '@tabler/icons-react'
import { useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { type Employee } from '@/shared/api/hrm'
import { useInvalidateRecord } from '@/shared/area'
import { useMe } from '@/shared/auth/me'
import { formatDate, today } from '@/shared/i18n'
import { FieldList, MissingValue, type FieldRow } from '@/shared/ui/FieldList'
import { DateField, EditablePanel, Form, FormActions, FormFields, OrgUnitField, SensitiveValue } from '@/shared/ui/form'
import { FormModal } from '@/shared/ui/page'
import { currentContract, useEmployeeContracts } from '../hooks/useEmployeeRecords'
import { employeeDocType } from '../keys'
import { tenure } from '../tenure'
import { EmployeeField } from './EmployeeField'
import { useConfirm } from '@/shared/ui/confirm'
import { MainFields, PersonalFields, revealSensitive, SensitiveFields, sensitiveFields, useEmployeeForm, type EmployeeValues } from './EmployeeForm'

type Part = 'main' | 'personal' | 'sensitive'
// A move is saved like any edit, through a form that holds only what it changes.
export type Move = 'transfer' | 'terminate'

// EmployeeProfile is an employee's values in three cards, each read as label–value pairs and
// edited in place. What the page header already shows (name, code, manager, login) is not repeated;
// the work card's form still edits it, as it is one form. An empty value offers to fill it in.
export function EmployeeProfile({ employee: e, onSaved }: { employee: Employee; onSaved: () => Promise<void> }) {
  const { t } = useTranslation()
  const me = useMe()
  const invalidate = useInvalidateRecord()
  // Bumped by each save, so a sensitive value shown before it is masked again rather than left stale.
  const [saves, setSaves] = useState(0)
  const canEdit = e.allowed_actions.includes('edit')
  const contracts = useEmployeeContracts(e)
  const now = today(me.timezone)
  const current = currentContract(contracts.data, now)
  const saved = async () => {
    await onSaved()
    setSaves((n) => n + 1)
  }
  const edit = (part: Part) => (close: () => void) => <PartForm employee={e} part={part} onClose={close} onSaved={saved} />
  // An empty value: "+ Bổ sung" opening the card's form for who may edit.
  const or = (open: () => void, label: string, value: ReactNode) => value || <MissingValue label={label} onAdd={canEdit ? open : undefined} />

  return (
    <>
      <EditablePanel title={t('hrm.employee.section.work')} icon={IconBriefcase} editable={canEdit} edit={edit('main')}>
        <FieldList
          stacked
          rows={[
            [t('hrm.employee.org_unit'), e.org_unit_name],
            [t('hrm.employee.hire_date'), formatDate(e.hire_date)],
            ...(e.allowed_actions.includes('view_contracts')
              ? [
                  [
                    t('hrm.employee.contract_type'),
                    contracts.isPending ? '' : (current?.contract_type_name ?? (
                      <MissingValue
                        label={t('hrm.employee.contract_type')}
                        to={e.allowed_actions.includes('create_contract') ? `/hrm/contracts/new?employee=${e.id}` : undefined}
                      />
                    )),
                  ] as FieldRow,
                ]
              : []),
            ...(e.termination_date ? [[t('hrm.employee.termination_date'), formatDate(e.termination_date)] as FieldRow] : []),
          ]}
        />
      </EditablePanel>
      <EditablePanel title={t('hrm.employee.section.personal')} icon={IconAddressBook} editable={canEdit} edit={edit('personal')}>
        {(open) => (
          <FieldList
            stacked
            rows={[
              [
                t('hrm.employee.date_of_birth'),
                or(open, t('hrm.employee.date_of_birth'), e.date_of_birth && `${formatDate(e.date_of_birth)} (${t('hrm.employee.age', { count: tenure(e.date_of_birth, now).years })})`),
              ],
              [t('hrm.employee.gender'), or(open, t('hrm.employee.gender'), e.gender && t(`hrm.employee.gender.${e.gender}`))],
              [
                t('hrm.employee.phone'),
                or(open, t('hrm.employee.phone'), e.phone && <Anchor href={`tel:${e.phone.replace(/\s/g, '')}`} size="sm" inherit>{e.phone}</Anchor>),
              ],
              [t('hrm.employee.email'), or(open, t('hrm.employee.email'), e.email && <Anchor href={`mailto:${e.email}`} size="sm" inherit>{e.email}</Anchor>)],
              [t('hrm.employee.address'), or(open, t('hrm.employee.address'), e.address), 'wide'],
            ]}
          />
        )}
      </EditablePanel>
      {/* Masked in both views: showing a value is its own audited read, editing or not. */}
      <EditablePanel title={t('hrm.employee.section.sensitive')} icon={IconLock} editable={canEdit} edit={edit('sensitive')}>
        <FieldList
          key={saves}
          stacked
          rows={sensitiveFields.map((f) => [
            t(`hrm.employee.${f}`),
            <SensitiveValue
              key={f}
              label={t(`hrm.employee.${f}`)}
              present={e.sensitive[f]}
              canView={e.allowed_actions.includes('view_sensitive')}
              reveal={async () => {
                const value = await revealSensitive(e.id, f)()
                // Each reveal is a read in the activity feed beside it.
                void invalidate(employeeDocType, e.id)
                return value
              }}
            />,
          ])}
        />
      </EditablePanel>
    </>
  )
}

// PartForm edits one part in its card with a form holding the whole employee: the API saves it as a whole.
function PartForm({ employee: e, part, onClose, onSaved }: { employee: Employee; part: Part; onClose: () => void; onSaved: () => Promise<void> }) {
  const { t } = useTranslation()
  const { form, save, errorFields } = useEmployeeForm(e, async () => {
    await onSaved()
    onClose()
  })
  return (
    // Errors mapped to a field belong to the main part; elsewhere they show as the form's banner.
    <Form form={form} onSubmit={save} errorFields={part === 'main' ? errorFields : undefined}>
      <FormFields>
        {part === 'main' && <MainFields employee={e} />}
        {part === 'personal' && <PersonalFields />}
        {part === 'sensitive' && <SensitiveFields employee={e} canView={e.allowed_actions.includes('view_sensitive')} canEdit />}
      </FormFields>
      <FormActions submitLabel={t('hrm.common.save')} onCancel={onClose} cancelLabel={t('hrm.common.cancel')} />
    </Form>
  )
}

// MoveModal moves an employee to another department (and manager), or records their leaving.
export function MoveModal({ employee: e, move, onClose, onSaved }: { employee: Employee; move: Move; onClose: () => void; onSaved: () => Promise<void> }) {
  const { t } = useTranslation()
  const [ask, dialog] = useConfirm()
  const { form, save, errorFields } = useEmployeeForm(e, async () => {
    await onSaved()
    onClose()
  })
  // Leaving cannot be taken back from here: the date is asked for, then confirmed.
  const submit = async (v: EmployeeValues) => {
    if (move === 'terminate') {
      const message = t('hrm.employee.terminate_confirm', { name: e.full_name, date: formatDate(v.termination_date) })
      if (!(await ask({ title: t('hrm.employee.terminate'), message, confirmLabel: t('hrm.employee.terminate'), danger: true }))) return
    }
    await save(v)
  }
  return (
    <FormModal opened title={t(`hrm.employee.${move}_title`, { name: e.full_name })} onClose={onClose}>
      <Form form={form} onSubmit={submit} errorFields={errorFields}>
        {move === 'transfer' ? (
          <>
            <OrgUnitField name="org_unit_id" label={t('hrm.employee.org_unit')} required product="hrm" permission="hrm.employee.edit" />
            <EmployeeField
              name="manager_id"
              label={t('hrm.employee.manager')}
              managerOf={e.id}
              current={e.manager_id && e.manager_code && e.manager_name ? { id: e.manager_id, code: e.manager_code, name: e.manager_name } : undefined}
            />
          </>
        ) : (
          <DateField name="termination_date" label={t('hrm.employee.termination_date')} description={t('hrm.employee.terminate_hint')} required />
        )}
        <FormActions submitLabel={t(`hrm.employee.${move}`)} onCancel={onClose} cancelLabel={t('hrm.common.cancel')} />
      </Form>
      {dialog}
    </FormModal>
  )
}
