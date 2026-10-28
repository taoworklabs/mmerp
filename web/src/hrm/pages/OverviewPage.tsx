import { Stack, Text } from '@mmerp/ui'
import { useQuery, type QueryKey } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { api as coreApi, unwrap as coreUnwrap } from '@/shared/api/core'
import { api, unwrap } from '@/shared/api/hrm'
import { can, useMe } from '@/shared/auth/me'
import { documentKeys } from '@/shared/document'
import { errorText } from '@/shared/i18n'
import { Page } from '@/shared/ui/page'
import { EmptyState } from '@/shared/ui/states'
import { StatTile, TileGrid } from '@/shared/ui/Tile'
import { hrmKeys } from '../keys'

type Block = { label: string; permission: string; to: string; key: QueryKey; total: () => Promise<number> }

// Each block counts the target list with the block's filter, so the number matches the list
// and follows the list's scope checks.
const blocks: Block[] = [
  {
    label: 'hrm.overview.active_employees',
    permission: 'hrm.employee.view',
    to: '/hrm/employees?status=active',
    key: [...hrmKeys.employees.lists(), 'overview'],
    total: async () => (await unwrap(api.GET('/hrm/employees', { params: { query: { status: 'active', page_size: 20 } } }))).total,
  },
  {
    label: 'hrm.overview.expiring_contracts',
    permission: 'hrm.contract.view',
    to: '/hrm/contracts?expiring=1',
    key: [...hrmKeys.contracts.all(), 'overview', 'expiring'],
    total: async () => (await unwrap(api.GET('/hrm/contracts', { params: { query: { expiring: true, page_size: 20 } } }))).total,
  },
  {
    label: 'hrm.overview.pending_leaves',
    permission: 'hrm.leave.view',
    to: '/hrm/leaves?status=pending_approval',
    key: [...hrmKeys.leaves.lists(), 'overview'],
    total: async () => (await unwrap(api.GET('/hrm/leaves', { params: { query: { status: ['pending_approval'], page_size: 20 } } }))).total,
  },
  {
    label: 'hrm.overview.pending_overtimes',
    permission: 'hrm.overtime.view',
    to: '/hrm/overtimes?status=pending_approval',
    key: [...hrmKeys.overtimes.lists(), 'overview'],
    total: async () => (await unwrap(api.GET('/hrm/overtimes', { params: { query: { status: 'pending_approval', page_size: 20 } } }))).total,
  },
  {
    label: 'hrm.overview.pending_contracts',
    permission: 'hrm.contract.view',
    to: '/hrm/contracts?status=pending_approval',
    key: [...hrmKeys.contracts.all(), 'overview', 'pending'],
    total: async () => (await unwrap(api.GET('/hrm/contracts', { params: { query: { status: 'pending_approval', page_size: 20 } } }))).total,
  },
]

// OverviewPage: the numbers HRM users act on, each opening its filtered list; blocks the
// user has no permission for are neither fetched nor shown.
export function OverviewPage() {
  const { t } = useTranslation()
  const me = useMe()
  const shown = blocks.filter((b) => can(me, b.permission))
  return (
    <Page title={t('hrm.overview.title')} description={t('hrm.overview.description')}>
      <NeedsAttention />
      {shown.length === 0 ? (
        <EmptyState title={t('hrm.overview.empty')} description={t('hrm.overview.empty_description')} />
      ) : (
        <TileGrid>
          {shown.map((b) => (
            <BlockTile key={b.label} block={b} />
          ))}
        </TileGrid>
      )}
    </Page>
  )
}

function BlockTile({ block }: { block: Block }) {
  const { t } = useTranslation()
  const total = useQuery({ queryKey: block.key, queryFn: block.total })
  return (
    <StatTile
      label={t(block.label)}
      value={total.data}
      to={block.to}
      error={total.isError ? errorText(t, total.error) : undefined}
      onRetry={() => void total.refetch()}
    />
  )
}

const approvalTypes = ['hrm.leave_request', 'hrm.overtime_request']

// NeedsAttention: what waits for this user, each entry hidden when zero or not theirs to
// see; the whole block disappears when nothing waits. A failed count hides its entry.
function NeedsAttention() {
  const { t } = useTranslation()
  const inbox = useQuery({ queryKey: documentKeys.inbox(), queryFn: () => coreUnwrap(coreApi.GET('/approvals/inbox')) })
  const close = useQuery({ queryKey: [...hrmKeys.timesheets.lists(), 'month-close'], queryFn: () => unwrap(api.GET('/hrm/overview/month-close')) })
  const approvals = inbox.data?.items.filter((i) => approvalTypes.includes(i.doc_type)).length ?? 0
  const month = close.data?.period_start.slice(0, 7)
  const entries = [
    { label: 'hrm.overview.attention_approvals', value: approvals, to: '/hrm/inbox' },
    { label: 'hrm.overview.attention_timesheets', value: close.data?.open_timesheet_units ?? 0, to: `/hrm/timesheets?month=${month}` },
    { label: 'hrm.overview.attention_payrolls', value: close.data?.open_payroll_entities ?? 0, to: '/hrm/payrolls' },
  ].filter((e) => e.value > 0)
  if (entries.length === 0) return null
  return (
    <Stack gap="sm">
      <Text weight={600}>{t('hrm.overview.attention')}</Text>
      <TileGrid>
        {entries.map((e) => (
          <StatTile key={e.label} label={t(e.label)} value={e.value} to={e.to} />
        ))}
      </TileGrid>
    </Stack>
  )
}
