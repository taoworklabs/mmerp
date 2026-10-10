import { useTranslation } from 'react-i18next'
import type { DocListItem } from '@/shared/api/sales'
import { StatusBadge } from '@/shared/ui/StatusBadge'

// DocProgress is a quotation's derived progress: ordered, or expired while still posted.
export function DocProgress({ doc }: { doc: Pick<DocListItem, 'status' | 'ordered' | 'expired'> }) {
  const { t } = useTranslation()
  if (doc.ordered) return <StatusBadge tone="info">{t('sales.quote.progress.ordered')}</StatusBadge>
  if (doc.expired && doc.status === 'posted') return <StatusBadge tone="neutral">{t('sales.quote.progress.expired')}</StatusBadge>
  return null
}
