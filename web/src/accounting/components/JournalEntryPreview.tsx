import { useTranslation } from 'react-i18next'
import { DocumentPreview } from '@/shared/document'
import { formatDate } from '@/shared/i18n'
import { useJournalEntry } from '../hooks/useAccounting'
import { journalEntryDocType } from '../keys'

// JournalEntryPreview is a manual journal entry in the approval inbox, when a rule holds one for
// approval. It is loaded outside the accounting screens, so its strings live in the meta translations.
export default function JournalEntryPreview({ id }: { id: number }) {
  const { t } = useTranslation()
  return (
    <DocumentPreview
      docType={journalEntryDocType}
      query={useJournalEntry(id)}
      rows={(d) => [
        [t('accounting.preview.date'), formatDate(d.date)],
        [t('accounting.preview.description'), d.description],
        [t('accounting.preview.lines'), String(d.lines.length)],
      ]}
    />
  )
}
