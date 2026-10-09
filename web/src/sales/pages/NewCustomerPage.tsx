import { useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router'
import { errorText } from '@/shared/i18n'
import { notifySuccess } from '@/shared/ui/notify'
import { Page } from '@/shared/ui/page'
import { ErrorState, ForbiddenPage, PageSkeleton } from '@/shared/ui/states'
import { CustomerForm } from '../components/CustomerForm'
import { useCustomerActions } from '../hooks/useSales'
import { salesKeys } from '../keys'

export function NewCustomerPage() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const actions = useCustomerActions()
  if (actions.isPending) return <PageSkeleton />
  if (actions.isError) return <ErrorState message={errorText(t, actions.error)} onRetry={() => void actions.refetch()} />
  if (!actions.data.includes('create')) return <ForbiddenPage />
  return (
    <Page title={t('sales.customers.create')} breadcrumbs={[{ label: t('sales.customers.title'), to: '/sales/customers' }]}>
      <CustomerForm
        canEdit
        onSaved={async (id) => {
          await qc.invalidateQueries({ queryKey: salesKeys.customers.all() })
          notifySuccess(t('sales.customer.created'))
          // The form is clean after a successful save, so leaving is not blocked.
          setTimeout(() => navigate(`/sales/customers/${id}`, { replace: true }))
        }}
      />
    </Page>
  )
}
