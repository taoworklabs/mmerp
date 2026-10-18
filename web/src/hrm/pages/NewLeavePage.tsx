import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router'
import { useInvalidateRecord } from '@/shared/area'
import { errorText } from '@/shared/i18n'
import { notifySuccess } from '@/shared/ui/notify'
import { Page } from '@/shared/ui/page'
import { ErrorState, ForbiddenPage, PageSkeleton } from '@/shared/ui/states'
import { LeaveForm } from '../components/LeaveForm'
import { useLeaveActions, useLeaveSelf } from '../hooks/useLeave'
import { leaveDocType } from '../keys'

export function NewLeavePage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const invalidate = useInvalidateRecord()
  const actions = useLeaveActions()
  // The same query as the actions, so it loads and fails with them.
  const self = useLeaveSelf()
  if (actions.isPending) return <PageSkeleton />
  if (actions.isError) return <ErrorState message={errorText(t, actions.error)} onRetry={() => void actions.refetch()} />
  if (!actions.data.includes('create')) return <ForbiddenPage />
  // The employee picker shows only when the API allows filing for others; it still checks each choice.
  const forOthers = actions.data.includes('create_for_others')
  return (
    <Page centered title={t('hrm.leaves.create')} breadcrumbs={[{ label: t('hrm.leaves.title'), to: '/hrm/leaves' }]} description={t('hrm.leaves.create_description')}>
      <LeaveForm
        self={self.data ?? null}
        chooseEmployee={forOthers}
        onCreated={async (id) => {
          await invalidate(leaveDocType, id)
          notifySuccess(t('hrm.leave.created'))
          // The form is clean after a successful save, so leaving is not blocked.
          setTimeout(() => navigate(`/hrm/leaves/${id}`, { replace: true }))
        }}
      />
    </Page>
  )
}
