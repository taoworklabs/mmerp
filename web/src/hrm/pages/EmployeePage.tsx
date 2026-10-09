import { Group } from '@mantine/core'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router'
import { api, unwrap } from '@/shared/api/hrm'
import { useAttachmentSection, useDiscussionSection } from '@/shared/document'
import { PersonAvatar } from '@/shared/ui/avatar'
import { notifySuccess } from '@/shared/ui/notify'
import { RecordPage } from '@/shared/ui/page'
import { BalancesTab } from '../components/BalancesTab'
import { ContractsTab } from '../components/ContractsTab'
import { DependentsTab } from '../components/DependentsTab'
import { EmployeeForm } from '../components/EmployeeForm'
import { EmployeeStatus } from '../components/EmployeeStatus'
import { loaded } from '@/shared/ui/loaded'
import { employeeDocType, hrmKeys } from '../keys'

export function EmployeePage() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const id = Number(useParams().id)
  const employee = useQuery({
    queryKey: hrmKeys.employees.detail(id),
    queryFn: () => unwrap(api.GET('/hrm/employees/{id}', { params: { path: { id } } })),
  })
  const sections = [useAttachmentSection(employeeDocType, id), useDiscussionSection(employeeDocType, id)].map((s) => ({
    value: s.key,
    label: s.title,
    content: s.content,
    panel: true,
  }))

  const state = loaded(employee, t, { to: '/hrm/employees', label: t('hrm.employees.back') })
  if (!state.ok) return state.fallback
  const e = state.data
  return (
    <RecordPage
      title={e.full_name}
      breadcrumbs={[{ label: t('hrm.employees.title'), to: '/hrm/employees' }]}
      leading={<PersonAvatar name={e.full_name} size="lg" />}
      description={
        <Group gap="xs" component="span">
          <span>{e.code}</span>
          <span aria-hidden>·</span>
          <span>{e.org_unit_name}</span>
          <EmployeeStatus status={e.status} />
        </Group>
      }
      tabs={[
        {
          value: 'info',
          label: t('hrm.employee.tab.info'),
          content: (
            <EmployeeForm
              key={e.id}
              employee={e}
              actions={e.allowed_actions}
              onSaved={async () => {
                await qc.invalidateQueries({ queryKey: hrmKeys.employees.all() })
                notifySuccess(t('hrm.employee.saved'))
              }}
            />
          ),
        },
        { value: 'dependents', label: t('hrm.employee.tab.dependents'), content: <DependentsTab employee={e} /> },
        { value: 'balances', label: t('hrm.employee.tab.balances'), content: <BalancesTab employee={e} /> },
        ...(e.allowed_actions.includes('view_contracts')
          ? [{ value: 'contracts', label: t('hrm.employee.tab.contracts'), content: <ContractsTab employee={e} /> }]
          : []),
        ...sections,
      ]}
    />
  )
}
