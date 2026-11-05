import { useTranslation } from 'react-i18next'
import { CreatePage } from '@/shared/document'
import { JournalEntryForm } from '../components/JournalEntryForm'
import { useJournalEntryActions } from '../hooks/useAccounting'
import { journalEntryDocType } from '../keys'

export function NewJournalEntryPage() {
  const { t } = useTranslation()
  return (
    <CreatePage
      docType={journalEntryDocType}
      actions={useJournalEntryActions()}
      title={t('accounting.entries.create')}
      breadcrumbs={[{ label: t('accounting.entries.title'), to: '/accounting/journal-entries' }]}
      description={t('accounting.entries.create_description')}
      created={t('accounting.entry.created')}
    >
      {(onCreated) => <JournalEntryForm onCreated={onCreated} />}
    </CreatePage>
  )
}
