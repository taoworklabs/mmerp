import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router'
import { api, unwrap } from '@/shared/api/hrm'
import { CreatePage } from '@/shared/document'
import { errorText } from '@/shared/i18n'
import { ErrorState, NotFoundPage, PageSkeleton } from '@/shared/ui/states'
import { TerminationForm } from '../components/TerminationForm'
import { hrmKeys, terminationDocType } from '../keys'

// NewTerminationPage starts the termination of ?employee=, as the employee's actions allow.
export function NewTerminationPage() {
  const { t } = useTranslation()
  const [params] = useSearchParams()
  const employeeId = Number(params.get('employee'))
  const employee = useQuery({
    queryKey: hrmKeys.employees.detail(employeeId),
    queryFn: () => unwrap(api.GET('/hrm/employees/{id}', { params: { path: { id: employeeId } } })),
    enabled: employeeId > 0,
  })

  const back = { to: '/hrm/employees', label: t('hrm.employees.back') }
  if (!employeeId) return <NotFoundPage back={back} />
  if (employee.isPending) return <PageSkeleton />
  if (employee.isError) return <ErrorState message={errorText(t, employee.error)} onRetry={() => void employee.refetch()} />
  const e = employee.data
  return (
    <CreatePage
      docType={terminationDocType}
      actions={e.allowed_actions}
      action="create_termination"
      title={t('hrm.termination.create')}
      breadcrumbs={[
        { label: t('hrm.employees.title'), to: '/hrm/employees' },
        { label: e.full_name, to: `/hrm/employees/${e.id}` },
      ]}
      description={t('hrm.termination.create_description', { name: e.full_name })}
      created={t('hrm.termination.created')}
    >
      {(onCreated) => <TerminationForm employeeId={e.id} onCreated={onCreated} />}
    </CreatePage>
  )
}
