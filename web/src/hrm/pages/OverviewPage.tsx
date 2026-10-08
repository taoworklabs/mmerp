import { useQuery, type QueryKey } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { api, unwrap } from '@/shared/api/hrm'
import { can, useMe } from '@/shared/auth/me'
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
    total: async () => (await unwrap(api.GET('/hrm/leaves', { params: { query: { status: 'pending_approval', page_size: 20 } } }))).total,
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
