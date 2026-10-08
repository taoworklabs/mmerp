import { useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router'
import { errorText } from '@/shared/i18n'
import { notifySuccess } from '@/shared/ui/notify'
import { Page } from '@/shared/ui/page'
import { ErrorState, ForbiddenPage, PageSkeleton } from '@/shared/ui/states'
import { OvertimeForm } from '../components/OvertimeForm'
import { useOvertimeActions } from '../hooks/useOvertime'
import { hrmKeys } from '../keys'

export function NewOvertimePage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const actions = useOvertimeActions()
  if (actions.isPending) return <PageSkeleton />
  if (actions.isError) return <ErrorState message={errorText(t, actions.error)} onRetry={() => void actions.refetch()} />
  if (!actions.data.allowed_actions.includes('create')) return <ForbiddenPage />
  // The employee picker shows only when the API allows filing for others; it still checks each choice.
  const forOthers = actions.data.allowed_actions.includes('create_for_others')
  const self = actions.data.self_employee_id ? { id: actions.data.self_employee_id, name: actions.data.self_employee_name ?? '' } : null
  return (
    <Page title={t('hrm.overtimes.create')} breadcrumbs={[{ label: t('hrm.overtimes.title'), to: '/hrm/overtimes' }]} description={t('hrm.overtimes.create_description')}>
      <OvertimeForm
        self={self}
        chooseEmployee={forOthers}
        onCreated={async (id) => {
          await qc.invalidateQueries({ queryKey: hrmKeys.overtimes.lists() })
          notifySuccess(t('hrm.overtime.created'))
          // The form is clean after a successful save, so leaving is not blocked.
          setTimeout(() => navigate(`/hrm/overtimes/${id}`, { replace: true }))
        }}
      />
    </Page>
  )
}
