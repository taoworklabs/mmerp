import { Tabs } from '@mantine/core'
import { IconHierarchy, IconTable } from '@tabler/icons-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router'
import type { OrgUnit } from '@/shared/api/core'
import { errorText } from '@/shared/i18n'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { orgUnitOptions, useOrgUnits } from '@/shared/ui/form'
import { OrgUnitName } from '@/shared/ui/OrgUnitName'
import { ContentSkeleton, EmptyState, ErrorState } from '@/shared/ui/states'
import { TreeList, type TreeNode } from '@/shared/ui/TreeList'
import { icon } from '@/shared/ui/theme'

export type OrgView = 'tree' | 'table'

// useOrgView keeps the chosen view in the URL (?view=table); the tree is the default.
export function useOrgView() {
  const { t } = useTranslation()
  const [search, setSearch] = useSearchParams()
  const view: OrgView = search.get('view') === 'table' ? 'table' : 'tree'
  const tabs = (
    <Tabs
      value={view}
      onChange={(v) =>
        setSearch((s) => {
          if (v === 'table') s.set('view', v)
          else s.delete('view')
          return s
        })
      }
    >
      <Tabs.List aria-label={t('shared.org.view')}>
        <Tabs.Tab value="tree" leftSection={<IconHierarchy {...icon.text} />}>
          {t('shared.org.view.tree')}
        </Tabs.Tab>
        <Tabs.Tab value="table" leftSection={<IconTable {...icon.text} />}>
          {t('shared.org.view.table')}
        </Tabs.Tab>
      </Tabs.List>
    </Tabs>
  )
  return { view, tabs }
}

// OrgTree shows the whole org tree, which every signed-in user may read; menu adds row actions.
// As a tree, units hang from their parents by guide lines; as a table, they are flat by name with
// their parent in a column.
export function OrgTree({ label, view = 'tree', emptyDescription, menu }: { label: string; view?: OrgView; emptyDescription?: string; menu?: (unit: OrgUnit) => ReactNode }) {
  const { t } = useTranslation()
  const units = useOrgUnits({})

  if (units.isPending) return <ContentSkeleton />
  if (units.isError) return <ErrorState message={errorText(t, units.error)} onRetry={() => void units.refetch()} />
  if (units.data.length === 0) return <EmptyState title={t('shared.org.empty')} description={emptyDescription} />
  const byId = new Map(units.data.map((u) => [u.id, u]))
  const kindOf = (u: OrgUnit) => t(`shared.org.kind.${u.kind}`)

  if (view === 'table') {
    const rows = [...units.data].sort((a, b) => a.name.localeCompare(b.name))
    const columns: Column<OrgUnit>[] = [
      { key: 'name', header: t('shared.org.name'), role: 'title', render: (u) => <OrgUnitName unit={u} /> },
      { key: 'kind', header: t('shared.org.kind'), role: 'status', render: kindOf },
      { key: 'parent', header: t('shared.org.parent'), role: 'meta', render: (u) => (u.parent_id === null ? '' : (byId.get(u.parent_id)?.name ?? '')) },
      { key: 'tax_code', header: t('shared.org.tax_code'), role: 'meta', kind: 'code', render: (u) => u.tax_code ?? '' },
    ]
    return <DataTable label={label} columns={columns} rows={rows} rowKey={(u) => u.id} menu={menu} />
  }

  // Nest the shared picker's tree order: each unit goes under the last one a level up.
  const roots: TreeNode<OrgUnit>[] = []
  const path: TreeNode<OrgUnit>[] = []
  for (const o of orgUnitOptions(units.data)) {
    const node: TreeNode<OrgUnit> = { key: o.value, item: byId.get(Number(o.value)) as OrgUnit, children: [] }
    path.length = o.depth
    const siblings = path[o.depth - 1]?.children ?? roots
    siblings.push(node)
    path.push(node)
  }
  return (
    <TreeList
      label={label}
      nodes={roots}
      render={(u) => <OrgUnitName unit={u} />}
      meta={(u) => [kindOf(u), u.tax_code].filter(Boolean).join(' · ')}
      menu={menu}
      toggleLabel={(u, folded) => t(folded ? 'shared.org.expand' : 'shared.org.collapse', { name: u.name })}
    />
  )
}
