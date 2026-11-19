import { useTranslation } from 'react-i18next'
import { CreatePage } from '@/shared/document'
import { StocktakeForm } from '../components/StocktakeForm'
import { useStocktakeActions } from '../hooks/useStock'
import { stocktakeDocType } from '../keys'

export function NewStocktakePage() {
  const { t } = useTranslation()
  return (
    <CreatePage
      docType={stocktakeDocType}
      actions={useStocktakeActions()}
      title={t('stock.stocktakes.create')}
      breadcrumbs={[{ label: t('stock.stocktakes.title'), to: '/stock/stocktakes' }]}
      description={t('stock.stocktakes.create_description')}
      created={t('stock.stocktake.created')}
    >
      {(onCreated) => <StocktakeForm onCreated={onCreated} />}
    </CreatePage>
  )
}
