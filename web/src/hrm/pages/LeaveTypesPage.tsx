import { Button } from '@mantine/core'
import { IconPlus } from '@tabler/icons-react'
import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { useProductOn } from '@/shared/auth/me'
import { api, unwrap, type LeaveType, type LeaveTypeInput } from '@/shared/api/hrm'
import { errorText, formatDecimal } from '@/shared/i18n'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { CheckboxField, DecimalField, Form, FormActions, TextField } from '@/shared/ui/form'
import { notifySuccess } from '@/shared/ui/notify'
import { FormModal, ListPage } from '@/shared/ui/page'
import { ContentSkeleton, EmptyState, ErrorState } from '@/shared/ui/states'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { icon } from '@/shared/ui/theme'
import { useLeaveTypes } from '../hooks/useLeave'
import { hrmKeys } from '../keys'

// LeaveTypesPage: the kinds of leave the customer uses; only deducting kinds touch the balance.
export function LeaveTypesPage() {
  const { t } = useTranslation()
  const types = useLeaveTypes()
  const [editing, setEditing] = useState<LeaveType | 'new' | null>(null)
  // Catalog writes have no allowed_actions; a disabled product is read-only.
  const writable = useProductOn('hrm')
  const columns: Column<LeaveType>[] = [
    { key: 'name', header: t('hrm.leave_type.name'), role: 'title', render: (x) => x.name },
    { key: 'deducts', header: t('hrm.leave_type.deducts_balance'), role: 'meta', render: (x) => t(x.deducts_balance ? 'hrm.leave_type.deducts' : 'hrm.leave_type.not_deducts') },
    { key: 'paid', header: t('hrm.leave_type.paid'), role: 'meta', render: (x) => t(x.paid ? 'hrm.leave_type.paid_yes' : 'hrm.leave_type.paid_no') },
    {
      key: 'max_days',
      header: t('hrm.leave_type.max_days'),
      role: 'meta',
      kind: 'number',
      render: (x) => (x.max_days ? t('hrm.leave_type.max_days_value', { days: formatDecimal(x.max_days) }) : '–'),
    },
    {
      key: 'active',
      header: t('hrm.leave_type.active'),
      role: 'status',
      kind: 'status',
      render: (x) => <StatusBadge tone={x.active ? 'positive' : 'neutral'}>{t(x.active ? 'hrm.leave_type.in_use' : 'hrm.leave_type.retired')}</StatusBadge>,
    },
  ]
  let body
  if (types.isPending) body = <ContentSkeleton />
  else if (types.isError) body = <ErrorState message={errorText(t, types.error)} onRetry={() => void types.refetch()} />
  else if (types.data.length === 0) body = <EmptyState title={t('hrm.leave_type.empty')} description={t('hrm.leave_type.empty_description')} />
  else
    body = (
      <DataTable
        label={t('hrm.leave_type.title')}
        columns={columns}
        rows={types.data}
        rowKey={(x) => x.id}
        // One action per row: the row itself opens it.
        onRowClick={writable ? (x) => setEditing(x) : undefined}
      />
    )
  return (
    <ListPage
      title={t('hrm.leave_type.title')}
      description={t('hrm.leave_type.description')}
      action={
        writable && (
          <Button leftSection={<IconPlus {...icon.button} />} onClick={() => setEditing('new')}>
            {t('hrm.leave_type.create')}
          </Button>
        )
      }
    >
      {body}
      {editing && <LeaveTypeModal type={editing === 'new' ? null : editing} onClose={() => setEditing(null)} />}
    </ListPage>
  )
}

function LeaveTypeModal({ type, onClose }: { type: LeaveType | null; onClose: () => void }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const form = useForm<LeaveTypeInput>({
    mode: 'onBlur',
    defaultValues: type ?? { name: '', deducts_balance: false, paid: false, max_days: '', requires_attachment: false, active: true },
  })
  async function save(v: LeaveTypeInput) {
    const body = {
      name: v.name.trim(),
      deducts_balance: v.deducts_balance,
      paid: v.paid,
      // Empty means no cap; the field is left out rather than sent empty.
      max_days: v.max_days || undefined,
      requires_attachment: v.requires_attachment,
      active: v.active,
    }
    if (type) await unwrap(api.PUT('/hrm/leave-types/{id}', { params: { path: { id: type.id } }, body }))
    else await unwrap(api.POST('/hrm/leave-types', { body }))
    await qc.invalidateQueries({ queryKey: hrmKeys.leaveTypes() })
    notifySuccess(t('hrm.leave_type.saved'))
    onClose()
  }
  return (
    <FormModal opened title={t(type ? 'hrm.leave_type.edit' : 'hrm.leave_type.create')} onClose={onClose}>
      <Form form={form} onSubmit={save} errorFields={{ leave_type_taken: 'name', invalid_leave_days: 'max_days' }}>
        <TextField name="name" label={t('hrm.leave_type.name')} required maxLength={100} />
        <CheckboxField name="deducts_balance" label={t('hrm.leave_type.deducts_balance')} description={t('hrm.leave_type.deducts_hint')} />
        <CheckboxField name="paid" label={t('hrm.leave_type.paid')} description={t('hrm.leave_type.paid_hint')} />
        <DecimalField name="max_days" label={t('hrm.leave_type.max_days')} description={t('hrm.leave_type.max_days_hint')} scale={1} step={0.5} min={0.5} />
        <CheckboxField name="requires_attachment" label={t('hrm.leave_type.requires_attachment')} description={t('hrm.leave_type.requires_attachment_hint')} />
        <CheckboxField name="active" label={t('hrm.leave_type.active')} description={t('hrm.leave_type.active_hint')} />
        <FormActions submitLabel={t('hrm.common.save')} onCancel={onClose} cancelLabel={t('hrm.common.cancel')} />
      </Form>
    </FormModal>
  )
}
