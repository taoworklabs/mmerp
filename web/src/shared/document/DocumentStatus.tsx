import { useTranslation } from 'react-i18next'
import { StatusBadge, type StatusTone } from '@/shared/ui/StatusBadge'

export type DocStatus = 'draft' | 'pending_approval' | 'posted' | 'cancelled'

// Colours are fixed for every document type; only the label may be overridden.
const tone: Record<DocStatus, StatusTone> = { draft: 'neutral', pending_approval: 'warning', posted: 'positive', cancelled: 'negative' }

// statusLabel is <doc_type>.status.<status> when the area defines it, else the default label.
export function useStatusLabel() {
  const { t } = useTranslation()
  return (docType: string, status: string) => t(`${docType}.status.${status}`, { defaultValue: t(`shared.document.status.${status}`) })
}

// DocumentStatus is the only way to show a document's status.
export function DocumentStatus({ docType, status }: { docType: string; status: DocStatus }) {
  const label = useStatusLabel()
  return <StatusBadge tone={tone[status]}>{label(docType, status)}</StatusBadge>
}
