import { Button, Menu, Stack } from '@mantine/core'
import { IconFilePlus, IconPlus } from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Link, useNavigate } from 'react-router'
import { api, unwrap, type ContractListItem, type Employee } from '@/shared/api/hrm'
import { DocumentStatus } from '@/shared/document'
import { errorText, formatDate } from '@/shared/i18n'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { TabActions } from '@/shared/ui/page'
import { ContentSkeleton, EmptyState, ErrorState } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import { contractDocType, hrmKeys } from '../keys'

// ContractsTab: an employee's contracts, each original followed by its appendices.
export function ContractsTab({ employee }: { employee: Employee }) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const list = useQuery({
    queryKey: hrmKeys.contracts.employee(employee.id),
    queryFn: async () => (await unwrap(api.GET('/hrm/employees/{id}/contracts', { params: { path: { id: employee.id } } }))) ?? [],
  })
  const canCreate = employee.allowed_actions.includes('create_contract')
  const numbers = new Map((list.data ?? []).map((c) => [c.id, c.number]))
  const newPath = `/hrm/contracts/new?employee=${employee.id}`
  const columns: Column<ContractListItem>[] = [
    { key: 'number', header: t('hrm.contract.number'), role: 'title', render: (c) => c.number },
    {
      key: 'kind',
      header: t('hrm.contract.kind'),
      role: 'meta',
      render: (c) => (c.parent_id ? t('hrm.contract.appendix_of', { number: numbers.get(c.parent_id) ?? '' }) : c.contract_type_name),
    },
    { key: 'start_date', header: t('hrm.contract.start_date'), role: 'meta', render: (c) => formatDate(c.start_date) },
    { key: 'end_date', header: t('hrm.contract.end_date'), role: 'hidden', render: (c) => (c.parent_id ? '' : c.end_date ? formatDate(c.end_date) : t('hrm.contract.no_end')) },
    { key: 'status', header: t('hrm.contract.status'), role: 'status', render: (c) => <DocumentStatus docType={contractDocType} status={c.status} /> },
  ]
  let body
  if (list.isPending) body = <ContentSkeleton />
  else if (list.isError) body = <ErrorState message={errorText(t, list.error)} onRetry={() => void list.refetch()} />
  else if (list.data.length === 0) body = <EmptyState title={t('hrm.contract.empty')} description={canCreate ? t('hrm.contract.empty_description') : undefined} />
  else
    body = (
      <DataTable
        label={t('hrm.employee.tab.contracts')}
        columns={columns}
        rows={list.data}
        rowKey={(c) => c.id}
        href={(c) => `/hrm/contracts/${c.id}`}
        menu={(c) =>
          c.allowed_actions.includes('add_appendix') && (
            <Menu.Item leftSection={<IconFilePlus {...icon.text} />} onClick={() => navigate(`${newPath}&parent=${c.id}`)}>
              {t('hrm.contract.add_appendix')}
            </Menu.Item>
          )
        }
      />
    )
  return (
    <Stack gap="md">
      {canCreate && (
        <TabActions>
          <Button size="xs" component={Link} to={newPath} leftSection={<IconPlus {...icon.text} />}>
            {t('hrm.contract.create')}
          </Button>
        </TabActions>
      )}
      {body}
    </Stack>
  )
}
