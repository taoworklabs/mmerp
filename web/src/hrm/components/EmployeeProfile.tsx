import { Divider, Stack } from '@mantine/core'
import { IconAddressBook, IconLock, IconUser, type Icon as TablerIcon } from '@tabler/icons-react'
import { useRef, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import type { Employee } from '@/shared/api/hrm'
import { formatDate } from '@/shared/i18n'
import { documentKeys } from '@/shared/document'
import { FieldList } from '@/shared/ui/FieldList'
import { EditableSection, Form, FormActions, SensitiveValue } from '@/shared/ui/form'
import { FormModal } from '@/shared/ui/page'
import { employeeDocType } from '../keys'
import { MainFields, PersonalFields, revealSensitive, SensitiveFields, sensitiveFields, useEmployeeForm } from './EmployeeForm'

type Part = 'main' | 'personal' | 'sensitive'

const orDash = (s: string | null | undefined) => s || '—'

// EmployeeProfile is an employee's values in three parts, each read as label–value pairs and
// edited on its own in a modal: the column is too narrow for a form.
export function EmployeeProfile({ employee: e, onSaved }: { employee: Employee; onSaved: () => Promise<void> }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [open, setOpen] = useState<Part | null>(null)
  // Bumped by each save, so a sensitive value shown before it is masked again rather than left stale.
  const [saves, setSaves] = useState(0)
  // The modal unmounts on closing, before it could hand the focus back; this does it.
  const opener = useRef<HTMLElement | null>(null)
  const canEdit = e.allowed_actions.includes('edit')
  const part = (p: Part, title: string, icon: TablerIcon) => ({
    title,
    icon,
    editable: canEdit,
    onEdit: () => {
      opener.current = document.activeElement as HTMLElement | null
      setOpen(p)
    },
  })
  const close = () => {
    setOpen(null)
    requestAnimationFrame(() => opener.current?.focus())
  }

  return (
    <Stack gap="md">
      <EditableSection {...part('main', t('hrm.employee.section.main'), IconUser)}>
        <FieldList
          rows={[
            [t('hrm.employee.full_name'), e.full_name],
            [t('hrm.employee.code'), e.code],
            [t('hrm.employee.org_unit'), e.org_unit_name],
            [t('hrm.employee.hire_date'), formatDate(e.hire_date)],
            [t('hrm.employee.manager'), e.manager_code && e.manager_name ? `${e.manager_code} · ${e.manager_name}` : '—'],
            [t('hrm.employee.user_login'), orDash(e.user_login)],
            ...(e.termination_date ? [[t('hrm.employee.termination_date'), formatDate(e.termination_date)] as [string, string]] : []),
          ]}
        />
      </EditableSection>
      <Divider />
      <EditableSection {...part('personal', t('hrm.employee.section.personal'), IconAddressBook)}>
        <FieldList
          rows={[
            [t('hrm.employee.date_of_birth'), orDash(formatDate(e.date_of_birth))],
            [t('hrm.employee.gender'), e.gender ? t(`hrm.employee.gender.${e.gender}`) : '—'],
            [t('hrm.employee.phone'), orDash(e.phone)],
            [t('hrm.employee.email'), orDash(e.email)],
            [t('hrm.employee.address'), orDash(e.address)],
          ]}
        />
      </EditableSection>
      <Divider />
      {/* Masked in both views: showing a value is its own audited read, editing or not. */}
      <EditableSection {...part('sensitive', t('hrm.employee.section.sensitive'), IconLock)}>
        <FieldList
          key={saves}
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
                void qc.invalidateQueries({ queryKey: documentKeys.history(employeeDocType, e.id) })
                return value
              }}
            />,
          ])}
        />
      </EditableSection>
      {open && (
        <PartModal
          employee={e}
          part={open}
          onClose={close}
          onSaved={async () => {
            await onSaved()
            setSaves((n) => n + 1)
          }}
        />
      )}
    </Stack>
  )
}

// PartModal edits one part with a form holding the whole employee: the API saves it as a whole.
function PartModal({ employee: e, part, onClose, onSaved }: { employee: Employee; part: Part; onClose: () => void; onSaved: () => Promise<void> }) {
  const { t } = useTranslation()
  const { form, save, fieldOf } = useEmployeeForm(e, async () => {
    await onSaved()
    onClose()
  })
  return (
    <FormModal opened title={t(`hrm.employee.edit.${part}`)} onClose={onClose}>
      {/* Errors mapped to a field belong to the main part; elsewhere they show as the form's banner. */}
      <Form form={form} onSubmit={save} fieldOf={part === 'main' ? fieldOf : undefined}>
        {part === 'main' && <MainFields employee={e} />}
        {part === 'personal' && <PersonalFields />}
        {part === 'sensitive' && <SensitiveFields employee={e} canView={e.allowed_actions.includes('view_sensitive')} canEdit />}
        <FormActions submitLabel={t('hrm.common.save')} onCancel={onClose} cancelLabel={t('hrm.common.cancel')} />
      </Form>
    </FormModal>
  )
}
