import { Button } from '@mantine/core'
import { IconFilePlus, IconPlus } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'
import { Link, useNavigate } from 'react-router'
import { type ContractListItem, type Employee } from '@/shared/api/hrm'
import { DocumentStatus } from '@/shared/document'
import { formatDate } from '@/shared/i18n'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { MenuAction } from '@/shared/ui/MenuAction'
import { TabContent } from '@/shared/ui/page'
import { QueryView } from '@/shared/ui/QueryView'
import { EmptyState } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import { useEmployeeContracts } from '../hooks/useEmployeeRecords'
import { contractDocType } from '../keys'

// ContractsTab: an employee's contracts, each original followed by its appendices.
export function ContractsTab({ employee }: { employee: Employee }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const list = useEmployeeContracts(employee)
  const canCreate = employee.allowed_actions.includes('create_contract')
  const numbers = new Map((list.data ?? []).map((c) => [c.id, c.number]))
  const newPath = `/hrm/contracts/new?employee=${employee.id}`
  const columns: Column<ContractListItem>[] = [
    { key: 'number', header: t('hrm.contract.number'), role: 'title', kind: 'code', render: (c) => c.number },
    {
      key: 'kind',
      header: t('hrm.contract.kind'),
      role: 'meta',
      render: (c) => (c.parent_id ? t('hrm.contract.appendix_of', { number: numbers.get(c.parent_id) ?? '' }) : c.contract_type_name),
    },
    { key: 'start_date', header: t('hrm.contract.start_date'), role: 'meta', kind: 'date', render: (c) => formatDate(c.start_date) },
    { key: 'end_date', header: t('hrm.contract.end_date'), role: 'hidden', kind: 'date', render: (c) => (c.parent_id ? '' : c.end_date ? formatDate(c.end_date) : t('hrm.contract.no_end')) },
    { key: 'status', header: t('hrm.contract.status'), role: 'status', kind: 'status', render: (c) => <DocumentStatus docType={contractDocType} status={c.status} /> },
  ]
  const body = (
    <QueryView query={list} empty={<EmptyState title={t('hrm.contract.empty')} description={canCreate ? t('hrm.contract.empty_description') : undefined} />}>
      {(data) => (
        <DataTable
          label={t('hrm.employee.tab.contracts')}
          columns={columns}
          rows={data}
          rowKey={(c) => c.id}
          href={(c) => `/hrm/contracts/${c.id}`}
          menu={(c) =>
            c.allowed_actions.includes('add_appendix') && (
              <MenuAction icon={IconFilePlus} onClick={() => navigate(`${newPath}&parent=${c.id}`)}>
                {t('hrm.contract.add_appendix')}
              </MenuAction>
            )
          }
        />
      )}
    </QueryView>
  )
  return (
    <TabContent
      actions={
        canCreate && (
          <Button component={Link} to={newPath} leftSection={<IconPlus {...icon.button} />}>
            {t('hrm.contract.create')}
          </Button>
        )
      }
    >
      {body}
    </TabContent>
  )
}
