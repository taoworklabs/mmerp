import { useTranslation } from 'react-i18next'
import { StatusBadge } from '@/shared/ui/StatusBadge'

// DocProgress is a quotation's derived progress: ordered, or expired while still posted.
export function DocProgress({ doc }: { doc: { status: string; ordered: boolean; expired: boolean } }) {
  const { t } = useTranslation()
  if (doc.ordered) return <StatusBadge tone="info">{t('sales.quote.progress.ordered')}</StatusBadge>
  if (doc.expired && doc.status === 'posted') return <StatusBadge tone="neutral">{t('sales.quote.progress.expired')}</StatusBadge>
  return null
}
