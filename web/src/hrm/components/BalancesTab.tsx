import { Button, Stack } from '@mantine/core'
import { IconPlusMinus } from '@tabler/icons-react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, ApiError, unwrap, type Employee, type LeaveBalance } from '@/shared/api/hrm'
import { useMe } from '@/shared/auth/me'
import { currentYear, errorText, formatDecimal } from '@/shared/i18n'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { DecimalField, Form, FormActions, TextField } from '@/shared/ui/form'
import { notifySuccess } from '@/shared/ui/notify'
import { FormModal, TabActions } from '@/shared/ui/page'
import { ContentSkeleton, EmptyState, ErrorState } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import { hrmKeys } from '../keys'

// BalancesTab: an employee's remaining leave by year; leave admins grant and adjust it, with a reason.
export function BalancesTab({ employee }: { employee: Employee }) {
  const { t } = useTranslation()
  const [adjusting, setAdjusting] = useState(false)
  const list = useQuery({
    queryKey: hrmKeys.balances.employee(employee.id),
    queryFn: async () => (await unwrap(api.GET('/hrm/employees/{id}/leave-balances', { params: { path: { id: employee.id } } }))) ?? [],
  })
  const canAdjust = employee.allowed_actions.includes('adjust_leave_balance')
  const columns: Column<LeaveBalance>[] = [
    { key: 'year', header: t('hrm.balance.year'), role: 'title', render: (b) => String(b.year) },
    { key: 'days', header: t('hrm.balance.days'), role: 'meta', numeric: true, render: (b) => formatDecimal(b.days) },
  ]
  let body
  if (list.isPending) body = <ContentSkeleton />
  // Without the right to see it the API answers 404.
  else if (list.error instanceof ApiError && list.error.status === 404) body = <EmptyState title={t('hrm.balance.no_permission')} />
  else if (list.isError) body = <ErrorState message={errorText(t, list.error)} onRetry={() => void list.refetch()} />
  else if (list.data.length === 0) body = <EmptyState title={t('hrm.balance.empty')} />
  else body = <DataTable label={t('hrm.employee.tab.balances')} columns={columns} rows={list.data} rowKey={(b) => b.year} />
  return (
    <Stack gap="md">
      {canAdjust && (
        <TabActions>
          <Button size="xs" leftSection={<IconPlusMinus {...icon.text} />} onClick={() => setAdjusting(true)}>
            {t('hrm.balance.adjust')}
          </Button>
        </TabActions>
      )}
      {body}
      {adjusting && <AdjustModal employeeId={employee.id} onClose={() => setAdjusting(false)} />}
    </Stack>
  )
}

type Values = { year: string; delta: string; reason: string }

function AdjustModal({ employeeId, onClose }: { employeeId: number; onClose: () => void }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const me = useMe()
  const form = useForm<Values>({ mode: 'onBlur', defaultValues: { year: String(currentYear(me.timezone)), delta: '', reason: '' } })
  async function save(v: Values) {
    await unwrap(
      api.POST('/hrm/employees/{id}/leave-balances/{year}/adjustments', {
        params: { path: { id: employeeId, year: Number(v.year) } },
        body: { delta: v.delta, reason: v.reason.trim() },
      }),
    )
    await qc.invalidateQueries({ queryKey: hrmKeys.balances.all() })
    notifySuccess(t('hrm.balance.adjusted'))
    onClose()
  }
  return (
    <FormModal opened title={t('hrm.balance.adjust')} onClose={onClose}>
      <Form form={form} onSubmit={save} fieldOf={(err) => (err.code === 'leave_balance_negative' ? 'delta' : undefined)}>
        <DecimalField name="year" label={t('hrm.balance.year')} required scale={0} min={2000} />
        <DecimalField name="delta" label={t('hrm.balance.delta')} description={t('hrm.balance.delta_hint')} required scale={1} step={0.5} allowNegative />
        <TextField name="reason" label={t('hrm.balance.reason')} required multiline maxLength={1000} />
        <FormActions submitLabel={t('hrm.balance.adjust_submit')} onCancel={onClose} cancelLabel={t('hrm.common.cancel')} />
      </Form>
    </FormModal>
  )
}
