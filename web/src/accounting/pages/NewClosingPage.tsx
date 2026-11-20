import { useTranslation } from 'react-i18next'
import { CreatePage } from '@/shared/document'
import { ClosingForm } from '../components/ClosingForm'
import { useClosingActions } from '../hooks/useAccounting'
import { closingDocType } from '../keys'

export function NewClosingPage() {
  const { t } = useTranslation()
  return (
    <CreatePage
      docType={closingDocType}
      actions={useClosingActions()}
      title={t('accounting.closings.create')}
      breadcrumbs={[{ label: t('accounting.closings.title'), to: '/accounting/closings' }]}
      description={t('accounting.closings.create_description')}
      created={t('accounting.closing.created')}
    >
      {(onCreated) => <ClosingForm onCreated={onCreated} />}
    </CreatePage>
  )
}
