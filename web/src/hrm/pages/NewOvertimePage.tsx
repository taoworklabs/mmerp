import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router'
import { useInvalidateRecord } from '@/shared/area'
import { errorText } from '@/shared/i18n'
import { notifySuccess } from '@/shared/ui/notify'
import { Page } from '@/shared/ui/page'
import { ErrorState, ForbiddenPage, PageSkeleton } from '@/shared/ui/states'
import { OvertimeForm } from '../components/OvertimeForm'
import { useOvertimeActions, useOvertimeSelf } from '../hooks/useOvertime'
import { overtimeDocType } from '../keys'

export function NewOvertimePage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const invalidate = useInvalidateRecord()
  const actions = useOvertimeActions()
  // The same query as the actions, so it loads and fails with them.
  const self = useOvertimeSelf()
  if (actions.isPending) return <PageSkeleton />
  if (actions.isError) return <ErrorState message={errorText(t, actions.error)} onRetry={() => void actions.refetch()} />
  if (!actions.data.includes('create')) return <ForbiddenPage />
  // The employee picker shows only when the API allows filing for others; it still checks each choice.
  const forOthers = actions.data.includes('create_for_others')
  return (
    <Page centered title={t('hrm.overtimes.create')} breadcrumbs={[{ label: t('hrm.overtimes.title'), to: '/hrm/overtimes' }]} description={t('hrm.overtimes.create_description')}>
      <OvertimeForm
        self={self.data ?? null}
        chooseEmployee={forOthers}
        onCreated={async (id) => {
          await invalidate(overtimeDocType, id)
          notifySuccess(t('hrm.overtime.created'))
          // The form is clean after a successful save, so leaving is not blocked.
          setTimeout(() => navigate(`/hrm/overtimes/${id}`, { replace: true }))
        }}
      />
    </Page>
  )
}
