import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router'
import { DocumentScreen } from '@/shared/document'
import { formatDate } from '@/shared/i18n'
import { useLegalEntities } from '@/shared/ui/form'
import { loaded } from '@/shared/ui/loaded'
import { JournalEntryForm } from '../components/JournalEntryForm'
import { useJournalEntry } from '../hooks/useAccounting'
import { journalEntryDocType } from '../keys'

const list = '/accounting/journal-entries'

// JournalEntryPage shows a manual journal entry; a draft the user may edit is edited in place.
export function JournalEntryPage() {
  const { t } = useTranslation()
  const id = Number(useParams().id)
  const entities = useLegalEntities({}).options
  const state = loaded(useJournalEntry(id), t, { to: list, label: t('accounting.entries.back') })
  if (!state.ok) return state.fallback
  const d = state.data
  return (
    <DocumentScreen
      docType={journalEntryDocType}
      doc={d}
      list={{ label: t('accounting.entries.title'), to: list }}
      facts={[entities.find((o) => o.value === String(d.legal_entity_id))?.label, formatDate(d.date), d.description]}
      attachments
      // The editable lines table needs the whole width; approval and history become tabs.
      fullWidth
    >
      {/* Keyed by version: a reload after a conflict starts from the saved data. */}
      <JournalEntryForm key={d.version} doc={d} />
    </DocumentScreen>
  )
}
