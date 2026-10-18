import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router'
import { DocumentScreen } from '@/shared/document'
import { formatDate } from '@/shared/i18n'
import { OvertimeDetails } from '../components/OvertimeDetails'
import { EmployeeLink } from '../components/EmployeeLink'
import { loaded } from '@/shared/ui/loaded'
import { useOvertime } from '../hooks/useOvertime'
import { overtimeDocType } from '../keys'

export function OvertimePage() {
  const { t } = useTranslation()
  const id = Number(useParams().id)
  const overtime = useOvertime(id)

  const state = loaded(overtime, t, { to: '/hrm/overtimes', label: t('hrm.overtimes.back') })
  if (!state.ok) return state.fallback
  const o = state.data
  return (
    <DocumentScreen
      docType={overtimeDocType}
      doc={o}
      list={{ label: t('hrm.overtimes.title'), to: '/hrm/overtimes' }}
      facts={[
        <EmployeeLink key="employee" id={o.employee_id}>
          {o.employee_name}
        </EmployeeLink>,
        formatDate(o.date),
      ]}
      subject={o.employee_name}
      attachments
    >
      <OvertimeDetails overtime={o} />
    </DocumentScreen>
  )
}
