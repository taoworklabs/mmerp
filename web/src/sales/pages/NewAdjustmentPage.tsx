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
      title={t('sales.adjustments.create')}
      breadcrumbs={[{ label: t('sales.adjustments.title'), to: '/sales/stock/adjustments' }]}
      description={t('sales.adjustments.create_description')}
      created={t('sales.adjustment.created')}
    >
      {(onCreated) => <AdjustmentForm onCreated={onCreated} />}
    </CreatePage>
  )
}
