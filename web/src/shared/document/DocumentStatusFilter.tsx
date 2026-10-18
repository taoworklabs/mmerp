import { IconCircleDot } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'
import { FilterChip } from '@/shared/ui/form'
import { useStatusLabel, type DocStatus } from './DocumentStatus'

const statuses = ['draft', 'pending_approval', 'posted', 'cancelled'] as const satisfies readonly DocStatus[]

// DocumentStatusFilter is a document list's status filter, labelled as the document type names its statuses.
export function DocumentStatusFilter({ docType, value, onChange }: { docType: string; value: string | null; onChange: (value: string | null) => void }) {
  const { t } = useTranslation()
  const label = useStatusLabel()
  return (
    <FilterChip
      label={t('shared.document.status_filter')}
      icon={IconCircleDot}
      data={statuses.map((s) => ({ value: s, label: label(docType, s) }))}
      value={value}
      onChange={onChange}
    />
  )
}
