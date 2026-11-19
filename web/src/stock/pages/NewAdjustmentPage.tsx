import { useTranslation } from 'react-i18next'
import { CreatePage } from '@/shared/document'
import { AdjustmentForm } from '../components/AdjustmentForm'
import { useAdjustmentActions } from '../hooks/useStock'
import { adjustmentDocType } from '../keys'

export function NewAdjustmentPage() {
  const { t } = useTranslation()
  return (
    <CreatePage
      docType={adjustmentDocType}
      actions={useAdjustmentActions()}
      title={t('stock.adjustments.create')}
      breadcrumbs={[{ label: t('stock.adjustments.title'), to: '/stock/adjustments' }]}
      description={t('stock.adjustments.create_description')}
      created={t('stock.adjustment.created')}
    >
      {(onCreated) => <AdjustmentForm onCreated={onCreated} />}
    </CreatePage>
  )
}
