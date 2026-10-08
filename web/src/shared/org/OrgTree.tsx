import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import type { OrgUnit } from '@/shared/api/core'
import { errorText } from '@/shared/i18n'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { orgUnitOptions, useOrgUnits } from '@/shared/ui/form'
import { OrgUnitName } from '@/shared/ui/OrgUnitName'
import { ContentSkeleton, EmptyState, ErrorState } from '@/shared/ui/states'

type Row = { unit: OrgUnit; depth: number }

// OrgTree shows the whole org tree, which every signed-in user may read; menu adds row actions.
export function OrgTree({ label, emptyDescription, menu }: { label: string; emptyDescription?: string; menu?: (unit: OrgUnit) => ReactNode }) {
  const { t } = useTranslation()
  const units = useOrgUnits({})

  if (units.isPending) return <ContentSkeleton />
  if (units.isError) return <ErrorState message={errorText(t, units.error)} onRetry={() => void units.refetch()} />
  const byId = new Map(units.data.map((u) => [u.id, u]))
  // Tree order with indented names, from the shared picker's ordering.
  const rows: Row[] = orgUnitOptions(units.data).map((o) => ({ unit: byId.get(Number(o.value)) as OrgUnit, depth: o.depth }))
  if (rows.length === 0) return <EmptyState title={t('shared.org.empty')} description={emptyDescription} />
  const columns: Column<Row>[] = [
    { key: 'name', header: t('shared.org.name'), role: 'title', render: (r) => <OrgUnitName unit={r.unit} depth={r.depth} /> },
    { key: 'kind', header: t('shared.org.kind'), role: 'status', render: (r) => t(`shared.org.kind.${r.unit.kind}`) },
    { key: 'tax_code', header: t('shared.org.tax_code'), role: 'meta', render: (r) => r.unit.tax_code ?? '' },
  ]
  return <DataTable label={label} columns={columns} rows={rows} rowKey={(r) => r.unit.id} menu={menu && ((r) => menu(r.unit))} />
}
