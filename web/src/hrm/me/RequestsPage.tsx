import { Button, FieldList, Group } from '@mmerp/ui'
import { IconPlus } from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { api, unwrap, type LeaveListItem, type OvertimeListItem } from '@/shared/api/hrm'
import { useMe } from '@/shared/auth/me'
import { DocumentStatus } from '@/shared/document'
import { currentYear, formatDate, formatDecimal } from '@/shared/i18n'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { Page } from '@/shared/ui/page'
import { QueryView } from '@/shared/ui/QueryView'
import { Section } from '@/shared/ui/Section'
import { EmptyState } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import { useLeaveActions, useLeaveSelf } from '../hooks/useLeave'
import { useOvertimeActions } from '../hooks/useOvertime'
import { hrmKeys, leaveDocType, overtimeDocType } from '../keys'

// RequestsPage: the user's leave balance for this year, and their own leave and overtime requests.
export function RequestsPage() {
  const { t } = useTranslation()
  const me = useMe()
  const self = useLeaveSelf().data
  const employee = self?.id ?? 0
  const year = currentYear(me.timezone)
  const canLeave = useLeaveActions().data?.includes('create') ?? false
  const canOvertime = useOvertimeActions().data?.includes('create') ?? false

  const balances = useQuery({
    queryKey: hrmKeys.balances.employee(employee),
    queryFn: async () => (await unwrap(api.GET('/hrm/employees/{id}/leave-balances', { params: { path: { id: employee } } }))) ?? [],
    enabled: !!employee,
  })
  const leaves = useQuery({
    queryKey: hrmKeys.me.leaves(employee),
    queryFn: async () => (await unwrap(api.GET('/hrm/leaves', { params: { query: { employee_id: employee, sort: '-start_date', page: 1, page_size: 50 } } }))).items,
    enabled: !!employee,
  })
  const overtimes = useQuery({
    queryKey: hrmKeys.me.overtimes(employee),
    queryFn: async () => (await unwrap(api.GET('/hrm/overtimes', { params: { query: { employee_id: employee, sort: '-date', page: 1, page_size: 50 } } }))).items,
    enabled: !!employee,
  })

  const leaveColumns: Column<LeaveListItem>[] = [
    { key: 'number', header: t('hrm.leave.number'), role: 'title', kind: 'code', render: (l) => l.number },
    { key: 'type', header: t('hrm.leave.type'), role: 'hidden', render: (l) => l.leave_type_name },
    { key: 'period', header: t('hrm.leave.period'), role: 'meta', kind: 'date', render: (l) => `${formatDate(l.start_date)} – ${formatDate(l.end_date)}` },
    { key: 'days', header: t('hrm.leave.days'), role: 'hidden', kind: 'number', render: (l) => formatDecimal(l.days) },
    { key: 'status', header: t('hrm.leave.status'), role: 'status', kind: 'status', render: (l) => <DocumentStatus docType={leaveDocType} status={l.status} /> },
  ]
  const overtimeColumns: Column<OvertimeListItem>[] = [
    { key: 'number', header: t('hrm.overtime.number'), role: 'title', kind: 'code', render: (o) => o.number },
    { key: 'date', header: t('hrm.overtime.date'), role: 'meta', kind: 'date', render: (o) => formatDate(o.date) },
    { key: 'day_hours', header: t('hrm.overtime.day_hours'), role: 'hidden', kind: 'number', render: (o) => formatDecimal(o.day_hours) },
    { key: 'night_hours', header: t('hrm.overtime.night_hours'), role: 'hidden', kind: 'number', render: (o) => formatDecimal(o.night_hours) },
    { key: 'status', header: t('hrm.overtime.status'), role: 'status', kind: 'status', render: (o) => <DocumentStatus docType={overtimeDocType} status={o.status} /> },
  ]
  const add = (to: string, label: string) => (
    <Button render={<Link to={to} />} leftSection={<IconPlus {...icon.button} />}>
      {label}
    </Button>
  )

  return (
    <Page title={t('hrm.me.requests.title')} description={t('hrm.me.requests.description')} actions={<Group gap="xs">{canLeave && add('/me/requests/leave/new', t('hrm.leaves.create'))}{canOvertime && add('/me/requests/overtime/new', t('hrm.overtimes.create'))}</Group>}>
      <Section title={t('hrm.me.requests.balance', { year })}>
        <QueryView query={balances} select={(b) => b.find((x) => x.year === year)} empty={<EmptyState title={t('hrm.balance.empty')} />} notFound={<EmptyState title={t('hrm.balance.empty')} />}>
          {(b) => <FieldList rows={[[t('hrm.balance.days'), formatDecimal(b?.days)]]} />}
        </QueryView>
      </Section>
      <Section title={t('hrm.leaves.title')}>
        <QueryView query={leaves} empty={<EmptyState title={t('hrm.leaves.empty')} />}>
          {(rows) => <DataTable label={t('hrm.leaves.title')} columns={leaveColumns} rows={rows} rowKey={(l) => l.id} href={(l) => `/me/requests/leave/${l.id}`} />}
        </QueryView>
      </Section>
      <Section title={t('hrm.overtimes.title')}>
        <QueryView query={overtimes} empty={<EmptyState title={t('hrm.overtimes.empty')} />}>
          {(rows) => <DataTable label={t('hrm.overtimes.title')} columns={overtimeColumns} rows={rows} rowKey={(o) => o.id} href={(o) => `/me/requests/overtime/${o.id}`} />}
        </QueryView>
      </Section>
    </Page>
  )
}
