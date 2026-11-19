import { useTranslation } from 'react-i18next'
import { CreatePage } from '@/shared/document'
import { TransferForm } from '../components/TransferForm'
import { useTransferActions } from '../hooks/useStock'
import { transferDocType } from '../keys'

export function NewTransferPage() {
  const { t } = useTranslation()
  return (
    <CreatePage
      docType={transferDocType}
      actions={useTransferActions()}
      title={t('stock.transfers.create')}
      breadcrumbs={[{ label: t('stock.transfers.title'), to: '/stock/transfers' }]}
      description={t('stock.transfers.create_description')}
      created={t('stock.transfer.created')}
    >
      {(onCreated) => <TransferForm onCreated={onCreated} />}
    </CreatePage>
  )
}
