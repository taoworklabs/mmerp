import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router'
import { DocumentScreen } from '@/shared/document'
import { formatDate } from '@/shared/i18n'
import { loaded } from '@/shared/ui/loaded'
import { EmployeeLink } from '../components/EmployeeLink'
import { TerminationDetails } from '../components/TerminationDetails'
import { useTermination } from '../hooks/useTermination'
import { terminationDocType } from '../keys'

export function TerminationPage() {
  const { t } = useTranslation()
  const id = Number(useParams().id)
  const termination = useTermination(id)

  const state = loaded(termination, t, { to: '/hrm/terminations', label: t('hrm.terminations.back') })
  if (!state.ok) return state.fallback
  const x = state.data
  return (
    <DocumentScreen
      docType={terminationDocType}
      doc={x}
      list={{ label: t('hrm.terminations.title'), to: '/hrm/terminations' }}
      facts={[
        <EmployeeLink key="employee" id={x.employee_id}>
          {x.employee_name}
        </EmployeeLink>,
        formatDate(x.termination_date),
      ]}
      subject={x.employee_name}
      attachments
      printLabel={t('hrm.termination.print')}
    >
      <TerminationDetails termination={x} />
    </DocumentScreen>
  )
}
