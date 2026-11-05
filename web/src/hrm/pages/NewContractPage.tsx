import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router'
import { api, unwrap } from '@/shared/api/hrm'
import { CreatePage } from '@/shared/document'
import { ErrorState, NotFoundPage, PageSkeleton } from '@/shared/ui/states'
import { ContractForm } from '../components/ContractForm'
import { useContract } from '../hooks/useContract'
import { contractDocType, hrmKeys } from '../keys'

// NewContractPage adds a contract for ?employee=, or an appendix to ?parent=, starting from its terms.
export function NewContractPage() {
  const { t } = useTranslation()
  const [params] = useSearchParams()
  const employeeId = Number(params.get('employee'))
  const parentId = Number(params.get('parent'))
  const employee = useQuery({
    queryKey: hrmKeys.employees.detail(employeeId),
    queryFn: () => unwrap(api.GET('/hrm/employees/{id}', { params: { path: { id: employeeId } } })),
    enabled: employeeId > 0,
  })
  const parent = useContract(parentId, parentId > 0)

  const back = { to: '/hrm/employees', label: t('hrm.employees.back') }
  if (!employeeId) return <NotFoundPage back={back} />
  if (employee.isPending || (parentId > 0 && parent.isPending)) return <PageSkeleton />
  if (employee.isError || parent.isError) return <ErrorState queries={[employee, parent]} />
  const e = employee.data
  const p = parentId > 0 ? parent.data : undefined
  if (p && p.employee_id !== e.id) return <NotFoundPage back={back} />
  return (
    <CreatePage
      docType={contractDocType}
      // A contract is created from the employee, as the employee's actions allow.
      actions={e.allowed_actions}
      action="create_contract"
      title={p ? t('hrm.contract.new_appendix', { number: p.number }) : t('hrm.contract.create')}
      breadcrumbs={[
        { label: t('hrm.employees.title'), to: '/hrm/employees' },
        { label: e.full_name, to: `/hrm/employees/${e.id}?tab=contracts` },
      ]}
      description={t(p ? 'hrm.contract.new_appendix_description' : 'hrm.contract.create_description', { name: e.full_name })}
      created={t('hrm.contract.created')}
    >
      {(onCreated) => <ContractForm employeeId={e.id} parent={p} onCreated={onCreated} />}
    </CreatePage>
  )
}
