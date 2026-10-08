import { useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router'
import { errorText } from '@/shared/i18n'
import { notifySuccess } from '@/shared/ui/notify'
import { Page } from '@/shared/ui/page'
import { ErrorState, ForbiddenPage, PageSkeleton } from '@/shared/ui/states'
import { LeaveForm } from '../components/LeaveForm'
import { useLeaveActions } from '../hooks/useLeave'
import { hrmKeys } from '../keys'

export function NewLeavePage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const actions = useLeaveActions()
  if (actions.isPending) return <PageSkeleton />
  if (actions.isError) return <ErrorState message={errorText(t, actions.error)} onRetry={() => void actions.refetch()} />
  if (!actions.data.allowed_actions.includes('create')) return <ForbiddenPage />
  // The employee picker shows only when the API allows filing for others; it still checks each choice.
  const forOthers = actions.data.allowed_actions.includes('create_for_others')
  const self = actions.data.self_employee_id ? { id: actions.data.self_employee_id, name: actions.data.self_employee_name ?? '' } : null
  return (
    <Page title={t('hrm.leaves.create')} breadcrumbs={[{ label: t('hrm.leaves.title'), to: '/hrm/leaves' }]} description={t('hrm.leaves.create_description')}>
      <LeaveForm
        self={self}
        chooseEmployee={forOthers}
        onCreated={async (id) => {
          await qc.invalidateQueries({ queryKey: hrmKeys.leaves.lists() })
          notifySuccess(t('hrm.leave.created'))
          // The form is clean after a successful save, so leaving is not blocked.
          setTimeout(() => navigate(`/hrm/leaves/${id}`, { replace: true }))
        }}
      />
    </Page>
  )
}
