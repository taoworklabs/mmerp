import { Group } from '@mantine/core'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router'
import { api, unwrap } from '@/shared/api/hrm'
import { ActivityFeed, documentKeys, useAttachmentSection } from '@/shared/document'
import { PersonAvatar } from '@/shared/ui/avatar'
import { notifySuccess } from '@/shared/ui/notify'
import { RecordPage } from '@/shared/ui/page'
import { BalancesTab } from '../components/BalancesTab'
import { ContractsTab } from '../components/ContractsTab'
import { DependentsTab } from '../components/DependentsTab'
import { EmployeeProfile } from '../components/EmployeeProfile'
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
  const attachments = useAttachmentSection(employeeDocType, id)

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
      aside={{
        label: t('hrm.employee.tab.info'),
        content: (
          <EmployeeProfile
            key={e.id}
            employee={e}
            onSaved={async () => {
              await Promise.all([
                qc.invalidateQueries({ queryKey: hrmKeys.employees.all() }),
                qc.invalidateQueries({ queryKey: documentKeys.history(employeeDocType, e.id) }),
              ])
              notifySuccess(t('hrm.employee.saved'))
            }}
          />
        ),
      }}
      tabs={[
        { value: 'dependents', label: t('hrm.employee.tab.dependents'), content: <DependentsTab employee={e} /> },
        { value: 'balances', label: t('hrm.employee.tab.balances'), content: <BalancesTab employee={e} /> },
        ...(e.allowed_actions.includes('view_contracts')
          ? [{ value: 'contracts', label: t('hrm.employee.tab.contracts'), content: <ContractsTab employee={e} /> }]
          : []),
        { value: attachments.key, label: attachments.title, content: attachments.content, panel: true },
      ]}
      activity={<ActivityFeed docType={employeeDocType} id={e.id} />}
    />
  )
}
