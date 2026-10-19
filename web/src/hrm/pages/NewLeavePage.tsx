import { useTranslation } from 'react-i18next'
import { CreatePage } from '@/shared/document'
import { LeaveForm } from '../components/LeaveForm'
import { useLeaveActions, useLeaveSelf } from '../hooks/useLeave'
import { leaveDocType } from '../keys'

export function NewLeavePage() {
  const { t } = useTranslation()
  const actions = useLeaveActions()
  // The same query as the actions, so it loads and fails with them.
  const self = useLeaveSelf()
  return (
    <CreatePage
      docType={leaveDocType}
      actions={actions}
      title={t('hrm.leaves.create')}
      breadcrumbs={[{ label: t('hrm.leaves.title'), to: '/hrm/leaves' }]}
      description={t('hrm.leaves.create_description')}
      created={t('hrm.leave.created')}
    >
      {(onCreated, allowed) => (
        // The employee picker shows only when the API allows filing for others; it still checks each choice.
        <LeaveForm self={self.data ?? null} chooseEmployee={allowed.includes('create_for_others')} onCreated={onCreated} />
      )}
    </CreatePage>
  )
}
