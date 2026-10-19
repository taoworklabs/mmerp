import { Button } from '@mantine/core'
import { IconArrowsExchange, IconUserOff } from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router'
import { api, unwrap } from '@/shared/api/hrm'
import { useInvalidateRecord } from '@/shared/area'
import { ActivityFeed, useAttachmentSection } from '@/shared/document'
import { PersonAvatar } from '@/shared/ui/avatar'
import { MenuAction } from '@/shared/ui/MenuAction'
import { notifySuccess } from '@/shared/ui/notify'
import { RecordColumns, RecordPage } from '@/shared/ui/page'
import { icon } from '@/shared/ui/theme'
import { BalancesTab } from '../components/BalancesTab'
import { ContractsTab } from '../components/ContractsTab'
import { DependentsTab } from '../components/DependentsTab'
import { EmployeeDigest } from '../components/EmployeeDigest'
import { EmployeeProfile, MoveModal, type Move } from '../components/EmployeeProfile'
import { EmployeeStatus } from '../components/EmployeeStatus'
import { EmployeeSummary } from '../components/EmployeeSummary'
import { useEmployeeContracts } from '../hooks/useEmployeeRecords'
import { loaded } from '@/shared/ui/loaded'
import { employeeDocType, hrmKeys } from '../keys'

export function EmployeePage() {
  const { t } = useTranslation()
  const invalidate = useInvalidateRecord()
  const id = Number(useParams().id)
  const [move, setMove] = useState<Move | null>(null)
  const employee = useQuery({
    queryKey: hrmKeys.employees.detail(id),
    queryFn: () => unwrap(api.GET('/hrm/employees/{id}', { params: { path: { id } } })),
  })
  const attachments = useAttachmentSection(employeeDocType, id)
  const contracts = useEmployeeContracts(employee.data)

  const state = loaded(employee, t, { to: '/hrm/employees', label: t('hrm.employees.back') })
  if (!state.ok) return state.fallback
  const e = state.data
  const canEdit = e.allowed_actions.includes('edit')
  const saved = async () => {
    await invalidate(employeeDocType, e.id)
    notifySuccess(t('hrm.employee.saved'))
  }
  const contractCount = contracts.data?.length
  const tabs = [
    {
      value: 'overview',
      label: t('shared.record.overview'),
      content: <RecordColumns main={<EmployeeProfile key={e.id} employee={e} onSaved={saved} />} side={<EmployeeDigest employee={e} />} />,
    },
    ...(e.allowed_actions.includes('view_contracts')
      ? [{ value: 'contracts', label: contractCount ? t('hrm.employee.tab.contracts_count', { count: contractCount }) : t('hrm.employee.tab.contracts'), content: <ContractsTab employee={e} /> }]
      : []),
    { value: 'balances', label: t('hrm.employee.tab.leave'), content: <BalancesTab employee={e} /> },
    { value: 'dependents', label: t('hrm.employee.tab.dependents'), content: <DependentsTab employee={e} /> },
    { value: attachments.key, label: attachments.title, content: attachments.content },
    { value: 'activity', label: t('shared.document.tab.activity'), content: <ActivityFeed docType={employeeDocType} id={e.id} /> },
  ]
  return (
    <RecordPage
      title={e.full_name}
      breadcrumbs={[{ label: t('hrm.employees.title'), to: '/hrm/employees' }]}
      leading={<PersonAvatar name={e.full_name} size="lg" />}
      status={<EmployeeStatus status={e.status} />}
      description={[e.code, e.org_unit_name, e.user_login].filter(Boolean).join(' · ')}
      summary={<EmployeeSummary employee={e} onTransfer={() => setMove('transfer')} />}
      actions={
        canEdit && (
          <Button variant="default" leftSection={<IconArrowsExchange {...icon.button} />} onClick={() => setMove('transfer')}>
            {t('hrm.employee.transfer')}
          </Button>
        )
      }
      more={
        canEdit &&
        !e.termination_date && (
          <MenuAction icon={IconUserOff} danger onClick={() => setMove('terminate')}>
            {t('hrm.employee.terminate')}
          </MenuAction>
        )
      }
      tabs={tabs}
      // The activity was a card beside the profile; its old links land on the history.
      aliases={{ history: 'activity', discussion: 'activity' }}
    >
      {move && <MoveModal employee={e} move={move} onClose={() => setMove(null)} onSaved={saved} />}
    </RecordPage>
  )
}
