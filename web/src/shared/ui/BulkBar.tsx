import { BulkBar as KitBulkBar } from '@mmerp/ui'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { formatNumber } from '@/shared/i18n'

// BulkBar takes the filter bar's place while rows are selected: how many, the actions that apply
// to all of them (each starts a job), and a way to clear the selection.
export function BulkBar({ count, onClear, children }: { count: number; onClear: () => void; children: ReactNode }) {
  const { t } = useTranslation()
  return (
    <KitBulkBar countText={t('shared.bulk.selected', { count, n: formatNumber(count) })} label={t('shared.bulk.label')} clearLabel={t('shared.bulk.clear')} onClear={onClear}>
      {children}
    </KitBulkBar>
  )
}
