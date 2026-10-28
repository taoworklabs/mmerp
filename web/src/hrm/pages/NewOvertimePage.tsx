import { useTranslation } from 'react-i18next'
import { CreatePage } from '@/shared/document'
import { OvertimeForm } from '../components/OvertimeForm'
import { useOvertimeActions, useOvertimeSelf } from '../hooks/useOvertime'
import { overtimeDocType } from '../keys'
import { useRequestPaths } from '../requestPaths'

export function NewOvertimePage() {
  const { t } = useTranslation()
  const actions = useOvertimeActions()
  // The same query as the actions, so it loads and fails with them.
  const self = useOvertimeSelf()
  const paths = useRequestPaths('overtime')
  return (
    <CreatePage
      docType={overtimeDocType}
      actions={actions}
      title={t('hrm.overtimes.create')}
      breadcrumbs={[{ label: t('hrm.overtimes.title'), to: paths.list }]}
      openAt={paths.open}
      description={t('hrm.overtimes.create_description')}
      created={t('hrm.overtime.created')}
    >
      {(onCreated, allowed) => (
        // The employee picker shows only when the API allows filing for others; it still checks each choice.
        <OvertimeForm self={self.data ?? null} chooseEmployee={allowed.includes('create_for_others')} onCreated={onCreated} />
      )}
    </CreatePage>
  )
}
