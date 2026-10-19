import { Skeleton } from '@mantine/core'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { ApiError, type Employee } from '@/shared/api/hrm'
import { useMe } from '@/shared/auth/me'
import { currentYear, formatDecimal, today } from '@/shared/i18n'
import { MissingValue } from '@/shared/ui/FieldList'
import { KeyFacts } from '@/shared/ui/KeyFacts'
import { currentContract, useEmployeeContracts, useLeaveBalances } from '../hooks/useEmployeeRecords'
import { tenure } from '../tenure'
import { EmployeeLink } from './EmployeeLink'

const loading = <Skeleton h="1lh" w="6em" />

// useTenureText is how long the employee has served, to their last day once they have left.
export function useTenureText(e: Employee): string {
  const { t } = useTranslation()
  const now = today(useMe().timezone)
  const served = tenure(e.hire_date, e.termination_date && e.termination_date < now ? e.termination_date : now)
  if (served.years + served.months === 0) return t('hrm.employee.tenure_new')
  return [served.years && t('hrm.employee.tenure_years', { count: served.years }), served.months && t('hrm.employee.tenure_months', { count: served.months })]
    .filter(Boolean)
    .join(' ')
}

// EmployeeSummary is the employee's key facts under the page title. An empty fact links to where it
// is filled in for who may (the manager through Transfer), and says "not filled in" to the rest;
// a fact the user may not read is left out.
export function EmployeeSummary({ employee: e, onTransfer, balancesTab }: { employee: Employee; onTransfer: () => void; balancesTab: string }) {
  const { t } = useTranslation()
  const me = useMe()
  const contracts = useEmployeeContracts(e)
  const balances = useLeaveBalances(e)
  const can = (a: string) => e.allowed_actions.includes(a)
  const year = currentYear(me.timezone)
  const served = useTenureText(e)
  const facts: [string, ReactNode][] = [
    [
      t('hrm.employee.manager'),
      e.manager_id && e.manager_code && e.manager_name ? (
        <EmployeeLink id={e.manager_id}>{e.manager_name}</EmployeeLink>
      ) : (
        <MissingValue label={t('hrm.employee.manager')} onAdd={can('edit') ? onTransfer : undefined} />
      ),
    ],
    [t('hrm.employee.tenure'), served],
  ]
  if (can('view_contracts')) {
    const current = currentContract(contracts.data, today(me.timezone))
    facts.push([
      t('hrm.employee.contract_type'),
      contracts.isPending
        ? loading
        : (current?.contract_type_name ?? (
            <MissingValue label={t('hrm.employee.contract_type')} to={can('create_contract') ? `/hrm/contracts/new?employee=${e.id}` : undefined} />
          )),
    ])
  }
  // Without the right to see the balances the API answers 404: the fact is left out.
  if (!(balances.error instanceof ApiError && balances.error.status === 404)) {
    const left = balances.data?.find((b) => b.year === year)
    const label = t('hrm.employee.leave_left', { year })
    facts.push([
      label,
      balances.isPending
        ? loading
        : left
          ? t('hrm.leave.balance_left', { left: formatDecimal(left.days) })
          : <MissingValue label={label} to={can('adjust_leave_balance') ? `?tab=${balancesTab}` : undefined} />,
    ])
  }
  return <KeyFacts facts={facts} />
}
