import { Button, Menu, Stack } from '@mantine/core'
import { IconPencil, IconPlus, IconTrash } from '@tabler/icons-react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type Dependent, type DependentInput, type Employee } from '@/shared/api/hrm'
import { errorText, formatDate, formatMonth } from '@/shared/i18n'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { DateField, Form, FormActions, SelectField, TextField } from '@/shared/ui/form'
import { useConfirm } from '@/shared/ui/confirm'
import { notifySuccess } from '@/shared/ui/notify'
import { FormModal, TabActions } from '@/shared/ui/page'
import { ContentSkeleton, EmptyState, ErrorState } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import { hrmKeys } from '../keys'

const relationships = ['child', 'spouse', 'parent', 'other'] as const

// Dependents are sensitive as a whole: shown only with view_sensitive, and each load is audited.
export function DependentsTab({ employee }: { employee: Employee }) {
  const { t } = useTranslation()
  if (!employee.allowed_actions.includes('view_sensitive')) return <EmptyState title={t('shared.sensitive.no_permission')} />
  return <Dependents employee={employee} />
}

function Dependents({ employee }: { employee: Employee }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const canEdit = employee.allowed_actions.includes('edit')
  const [editing, setEditing] = useState<Dependent | 'new' | null>(null)
  const [error, setError] = useState<string | null>(null)
  const key = hrmKeys.employees.dependents(employee.id)
  const list = useQuery({
    queryKey: key,
    queryFn: async () => (await unwrap(api.GET('/hrm/employees/{id}/dependents', { params: { path: { id: employee.id } } }))) ?? [],
    // Every load writes an audit row; reload only when asked.
    refetchOnWindowFocus: false,
  })

  const [ask, dialog] = useConfirm()

  async function remove(d: Dependent) {
    if (!(await ask({ title: t('hrm.dependent.delete_title'), message: t('hrm.dependent.delete_confirm', { name: d.full_name }), confirmLabel: t('hrm.dependent.delete'), danger: true })))
      return
    setError(null)
    try {
      await unwrap(api.DELETE('/hrm/employees/{id}/dependents/{dependent}', { params: { path: { id: employee.id, dependent: d.id } } }))
    } catch (err) {
      setError(errorText(t, err))
      return
    }
    await qc.invalidateQueries({ queryKey: key })
    notifySuccess(t('hrm.dependent.deleted'))
  }

  const months = (d: Dependent) => [d.deduction_from, d.deduction_to].map((m) => formatMonth(m) || '…').join(' – ')
  const columns: Column<Dependent>[] = [
    { key: 'full_name', header: t('hrm.dependent.full_name'), role: 'title', render: (d) => d.full_name },
    { key: 'relationship', header: t('hrm.dependent.relationship'), role: 'status', render: (d) => t(`hrm.dependent.relationship.${d.relationship}`) },
    { key: 'date_of_birth', header: t('hrm.dependent.date_of_birth'), role: 'meta', render: (d) => formatDate(d.date_of_birth) },
    { key: 'id_number', header: t('hrm.dependent.id_number'), role: 'hidden', render: (d) => d.id_number ?? '' },
    { key: 'deduction', header: t('hrm.dependent.deduction'), role: 'meta', render: months },
  ]

  let body
  if (list.isPending) body = <ContentSkeleton />
  else if (list.isError) body = <ErrorState message={errorText(t, list.error)} onRetry={() => void list.refetch()} />
  else if (list.data.length === 0) body = <EmptyState title={t('hrm.dependent.empty')} />
  else
    body = (
      <DataTable
        label={t('hrm.employee.tab.dependents')}
        columns={columns}
        rows={list.data}
        rowKey={(d) => d.id}
        menu={
          canEdit
            ? (d) => (
                <>
                  <Menu.Item leftSection={<IconPencil {...icon.text} />} onClick={() => setEditing(d)}>
                    {t('hrm.dependent.edit')}
                  </Menu.Item>
                  <Menu.Item color="danger" leftSection={<IconTrash {...icon.text} />} onClick={() => void remove(d)}>
                    {t('hrm.dependent.delete')}
                  </Menu.Item>
                </>
              )
            : undefined
        }
      />
    )

  return (
    <Stack gap="md">
      {canEdit && (
        <TabActions>
          <Button size="xs" leftSection={<IconPlus {...icon.text} />} onClick={() => setEditing('new')}>
            {t('hrm.dependent.create')}
          </Button>
        </TabActions>
      )}
      {error && <ErrorState message={error} />}
      {body}
      {dialog}
      {editing && <DependentModal employeeId={employee.id} dependent={editing === 'new' ? null : editing} onClose={() => setEditing(null)} />}
    </Stack>
  )
}

type Values = {
  full_name: string
  relationship: string | null
  date_of_birth: string | null
  id_number: string
  deduction_from: string
  deduction_to: string
}

function DependentModal({ employeeId, dependent, onClose }: { employeeId: number; dependent: Dependent | null; onClose: () => void }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const form = useForm<Values>({
    mode: 'onBlur',
    defaultValues: {
      full_name: dependent?.full_name ?? '',
      relationship: dependent?.relationship ?? null,
      date_of_birth: dependent?.date_of_birth ?? null,
      id_number: dependent?.id_number ?? '',
      deduction_from: dependent?.deduction_from ?? '',
      deduction_to: dependent?.deduction_to ?? '',
    },
  })

  async function save(v: Values) {
    const orNull = (s: string) => (s.trim() === '' ? null : s.trim())
    const body: DependentInput = {
      full_name: v.full_name.trim(),
      relationship: v.relationship as DependentInput['relationship'],
      date_of_birth: v.date_of_birth,
      id_number: orNull(v.id_number),
      deduction_from: orNull(v.deduction_from),
      deduction_to: orNull(v.deduction_to),
    }
    const path = { id: employeeId }
    if (dependent) await unwrap(api.PUT('/hrm/employees/{id}/dependents/{dependent}', { params: { path: { ...path, dependent: dependent.id } }, body }))
    else await unwrap(api.POST('/hrm/employees/{id}/dependents', { params: { path }, body }))
    await qc.invalidateQueries({ queryKey: hrmKeys.employees.dependents(employeeId) })
    notifySuccess(t('hrm.dependent.saved'))
    onClose()
  }

  const month = { pattern: { value: /^\d{4}-\d{2}$/, message: t('hrm.dependent.month_format') } }
  return (
    <FormModal opened title={t(dependent ? 'hrm.dependent.edit' : 'hrm.dependent.create')} onClose={onClose}>
      <Form form={form} onSubmit={save}>
        <TextField name="full_name" label={t('hrm.dependent.full_name')} required maxLength={200} />
        <SelectField
          name="relationship"
          label={t('hrm.dependent.relationship')}
          required
          data={relationships.map((r) => ({ value: r, label: t(`hrm.dependent.relationship.${r}`) }))}
        />
        <DateField name="date_of_birth" label={t('hrm.dependent.date_of_birth')} clearable />
        <TextField name="id_number" label={t('hrm.dependent.id_number')} maxLength={50} />
        <TextField name="deduction_from" label={t('hrm.dependent.deduction_from')} description={t('hrm.dependent.month_format')} rules={month} />
        <TextField name="deduction_to" label={t('hrm.dependent.deduction_to')} description={t('hrm.dependent.month_format')} rules={month} />
        <FormActions submitLabel={t('hrm.common.save')} onCancel={onClose} cancelLabel={t('hrm.common.cancel')} />
      </Form>
    </FormModal>
  )
}
