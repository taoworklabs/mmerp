import { Anchor, Group, Stack, Text } from '@mantine/core'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { ApiError, type Employee } from '@/shared/api/hrm'
import { useMe } from '@/shared/auth/me'
import { ActivityFeed, DocumentStatus } from '@/shared/document'
import { currentYear, formatDate, formatDecimal, today } from '@/shared/i18n'
import { Panel } from '@/shared/ui/page'
import { ContentSkeleton } from '@/shared/ui/states'
import { currentContract, useEmployeeContracts, useLeaveBalances } from '../hooks/useEmployeeRecords'
import { contractDocType, employeeDocType } from '../keys'

// SeeAll opens the tab holding the whole of a digest.
function SeeAll({ tab, label }: { tab: string; label?: string }) {
  const { t } = useTranslation()
  return (
    <Anchor component={Link} to={`?tab=${tab}`} replace size="sm">
      {label ?? t('shared.record.see_all')}
    </Anchor>
  )
}

// EmployeeDigest is the overview's right column: the contract in force, this year's leave and the
// latest comments and changes, each leading to its tab. A part the user may not read is left out.
export function EmployeeDigest({ employee: e }: { employee: Employee }) {
  const { t } = useTranslation()
  const me = useMe()
  const contracts = useEmployeeContracts(e)
  const balances = useLeaveBalances(e)
  const year = currentYear(me.timezone)
  const current = currentContract(contracts.data, today(me.timezone))
  const left = balances.data?.find((b) => b.year === year)
  // Without the right to see the balances the API answers 404.
  const canBalances = !(balances.error instanceof ApiError && balances.error.status === 404)
  return (
    <>
      {e.allowed_actions.includes('view_contracts') && (
        <Panel title={t('hrm.employee.current_contract')} actions={<SeeAll tab="contracts" />}>
          {contracts.isPending ? (
            <ContentSkeleton />
          ) : current ? (
            <Stack gap="xxs">
              <Group gap="sm" justify="space-between" wrap="nowrap">
                <Anchor component={Link} to={`/hrm/contracts/${current.id}`} fw={600}>
                  {current.number}
                </Anchor>
                <DocumentStatus docType={contractDocType} status={current.status} />
              </Group>
              <Text size="sm" c="dimmed">
                {t('hrm.employee.contract_since', { type: current.contract_type_name, date: formatDate(current.start_date) })}
              </Text>
            </Stack>
          ) : (
            <Text size="sm" c="dimmed">
              {t('hrm.employee.no_current_contract')}
            </Text>
          )}
        </Panel>
      )}
      {canBalances && (
        <Panel title={t('hrm.employee.leave_year', { year })} actions={<SeeAll tab="balances" />}>
          {balances.isPending ? (
            <ContentSkeleton />
          ) : (
            <Text fw={600}>{left ? t('hrm.leave.balance_left', { left: formatDecimal(left.days) }) : <Text span c="dimmed" size="sm">{t('hrm.balance.empty')}</Text>}</Text>
          )}
        </Panel>
      )}
      <Panel title={t('hrm.employee.recent_activity')} actions={<SeeAll tab="activity" label={t('shared.record.history')} />}>
        <ActivityFeed docType={employeeDocType} id={e.id} recent={5} />
      </Panel>
    </>
  )
}
