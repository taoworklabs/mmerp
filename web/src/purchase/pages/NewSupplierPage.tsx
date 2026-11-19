import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router'
import { useInvalidateRecord } from '@/shared/area'
import { notifySuccess } from '@/shared/ui/notify'
import { Page } from '@/shared/ui/page'
import { ErrorState, ForbiddenPage, PageSkeleton } from '@/shared/ui/states'
import { SupplierForm } from '../components/SupplierForm'
import { useSupplierActions } from '../hooks/usePurchase'
import { supplierDocType } from '../keys'

export function NewSupplierPage() {
  const { t } = useTranslation()
  const invalidate = useInvalidateRecord()
  const navigate = useNavigate()
  const actions = useSupplierActions()
  if (actions.isPending) return <PageSkeleton />
  if (actions.isError) return <ErrorState queries={[actions]} />
  if (!actions.data.includes('create')) return <ForbiddenPage />
  return (
    <Page centered title={t('purchase.suppliers.create')} breadcrumbs={[{ label: t('purchase.suppliers.title'), to: '/purchase/suppliers' }]}>
      <SupplierForm
        onSaved={async (id) => {
          await invalidate(supplierDocType, id)
          notifySuccess(t('purchase.supplier.created'))
          // The form is clean after a successful save, so leaving is not blocked.
          setTimeout(() => navigate(`/purchase/suppliers/${id}`, { replace: true }))
        }}
      />
    </Page>
  )
}
