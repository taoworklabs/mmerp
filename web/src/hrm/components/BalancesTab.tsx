import { Button } from '@mantine/core'
import { IconPlusMinus } from '@tabler/icons-react'
import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type Employee, type LeaveBalance } from '@/shared/api/hrm'
import { useMe } from '@/shared/auth/me'
import { currentYear, formatDecimal } from '@/shared/i18n'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { DecimalField, Form, FormActions, IntegerField, TextField } from '@/shared/ui/form'
import { notifySuccess } from '@/shared/ui/notify'
import { FormModal, TabContent } from '@/shared/ui/page'
import { QueryView } from '@/shared/ui/QueryView'
import { EmptyState } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import { useLeaveBalances } from '../hooks/useEmployeeRecords'
import { hrmKeys } from '../keys'

// BalancesTab: an employee's remaining leave by year; leave admins grant and adjust it, with a reason.
export function BalancesTab({ employee }: { employee: Employee }) {
  const { t } = useTranslation()
  const [adjusting, setAdjusting] = useState(false)
  const list = useLeaveBalances(employee)
  const canAdjust = employee.allowed_actions.includes('adjust_leave_balance')
  const columns: Column<LeaveBalance>[] = [
    { key: 'year', header: t('hrm.balance.year'), role: 'title', kind: 'date', render: (b) => String(b.year) },
    { key: 'days', header: t('hrm.balance.days'), role: 'meta', kind: 'number', render: (b) => formatDecimal(b.days) },
  ]
  return (
    <TabContent
      actions={
        canAdjust && (
          <Button leftSection={<IconPlusMinus {...icon.button} />} onClick={() => setAdjusting(true)}>
            {t('hrm.balance.adjust')}
          </Button>
        )
      }
    >
      <QueryView
        query={list}
        // Without the right to see it the API answers 404.
        notFound={<EmptyState title={t('hrm.balance.no_permission')} />}
        empty={<EmptyState title={t('hrm.balance.empty')} />}
      >
        {(data) => <DataTable label={t('hrm.employee.tab.balances')} columns={columns} rows={data} rowKey={(b) => b.year} />}
      </QueryView>
      {adjusting && <AdjustModal employeeId={employee.id} onClose={() => setAdjusting(false)} />}
    </TabContent>
  )
}

type Values = { year: number | null; delta: string; reason: string }

function AdjustModal({ employeeId, onClose }: { employeeId: number; onClose: () => void }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const me = useMe()
  const form = useForm<Values>({ mode: 'onBlur', defaultValues: { year: currentYear(me.timezone), delta: '', reason: '' } })
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
      <Form form={form} onSubmit={save} errorFields={{ leave_balance_negative: 'delta' }}>
        <IntegerField name="year" label={t('hrm.balance.year')} required min={2000} max={2100} />
        <DecimalField name="delta" label={t('hrm.balance.delta')} description={t('hrm.balance.delta_hint')} required scale={1} step={0.5} allowNegative />
        <TextField name="reason" label={t('hrm.balance.reason')} required multiline maxLength={1000} />
        <FormActions submitLabel={t('hrm.balance.adjust_submit')} onCancel={onClose} cancelLabel={t('hrm.common.cancel')} />
      </Form>
    </FormModal>
  )
}
