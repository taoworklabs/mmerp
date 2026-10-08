import { useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router'
import { notifySuccess } from '@/shared/ui/notify'
import { errorText } from '@/shared/i18n'
import { Page } from '@/shared/ui/page'
import { ErrorState, ForbiddenPage, PageSkeleton } from '@/shared/ui/states'
import { EmployeeForm } from '../components/EmployeeForm'
import { useEmployeeActions } from '../hooks/useEmployeeActions'
import { hrmKeys } from '../keys'

export function NewEmployeePage() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const actions = useEmployeeActions()
  if (actions.isPending) return <PageSkeleton />
  if (actions.isError) return <ErrorState message={errorText(t, actions.error)} onRetry={() => void actions.refetch()} />
  if (!actions.data.includes('create')) return <ForbiddenPage />
  return (
    <Page title={t('hrm.employees.create')} breadcrumbs={[{ label: t('hrm.employees.title'), to: '/hrm/employees' }]}>
      <EmployeeForm
        actions={actions.data}
        onSaved={async (id) => {
          await qc.invalidateQueries({ queryKey: hrmKeys.employees.lists() })
          notifySuccess(t('hrm.employee.created'))
          // The form is clean after a successful save, so leaving is not blocked.
          setTimeout(() => navigate(`/hrm/employees/${id}`, { replace: true }))
        }}
      />
    </Page>
  )
}
