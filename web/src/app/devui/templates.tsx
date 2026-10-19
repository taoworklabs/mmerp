import { Button, Group, Stack, Text } from '@mantine/core'
import { IconUsersGroup } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router'
import { WithSampleSections } from '@/shared/document/samples'
import { formatMonth, formatNumber } from '@/shared/i18n'
import { PersonAvatar } from '@/shared/ui/avatar'
import { DataTable } from '@/shared/ui/DataTable'
import { FieldList, MissingValue } from '@/shared/ui/FieldList'
import { KeyFacts } from '@/shared/ui/KeyFacts'
import { EditablePanel, MonthFilter, OrgUnitFilter, SearchInput, SensitiveValue } from '@/shared/ui/form'
import { DocumentPage, InboxList, InboxPage, InboxRow, ListPage, Page, Panel, RecordColumns, RecordPage, TabActions } from '@/shared/ui/page'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { useListParams } from '@/shared/url/listParams'
import { orgUnits } from './api'
import { type Row, rows } from './Demo'

// The body every template shows, so the templates differ only in their frame.
function Sample({ rows: data = rows }: { rows?: Row[] }) {
  const { t } = useTranslation()
  return (
    <DataTable
      label={t('shared.devui.table')}
      rows={data}
      rowKey={(r) => r.id}
      columns={[
        { key: 'code', header: t('shared.devui.code'), role: 'title', kind: 'code', render: (r) => r.code },
        { key: 'name', header: t('shared.devui.name'), role: 'meta', render: (r) => r.name },
        { key: 'amount', header: t('shared.devui.amount'), role: 'meta', kind: 'number', render: (r) => formatNumber(r.amount) },
      ]}
    />
  )
}

// Each template is titled with its own name, as in the sidebar.
function useHeader(title: string) {
  const { t } = useTranslation()
  return { title, breadcrumbs: [{ label: t('shared.devui.title'), to: '/dev/ui' }], action: <Button>{t('shared.devui.primary')}</Button> }
}

export function PageTemplate() {
  const { t } = useTranslation()
  const h = useHeader('Page')
  return (
    <Page title={h.title} description={t('shared.devui.description')} breadcrumbs={h.breadcrumbs} actions={h.action}>
      <Sample />
    </Page>
  )
}

const unitName = new Map(orgUnits.map((u) => [u.id, u.name]))
const unitParent = new Map(orgUnits.map((u) => [u.id, u.parent_id]))
// inUnit says whether a unit is the filtered one or below it, as the real lists filter by org unit.
const inUnit = (id: number | null | undefined, filter: number): boolean => id != null && (id === filter || inUnit(unitParent.get(id), filter))

const names = ['Nguyễn Văn An', 'Trần Thị Bình', 'Lê Văn Cường', 'Phạm Thị Dung', 'Hoàng Văn Em', 'Vũ Thị Giang', 'Đặng Văn Hùng']
// Enough rows across three months and the leaf units for the filters and paging to have something to do.
const listRows = Array.from({ length: 60 }, (_, i) => ({
  id: i + 1,
  code: `NV${String(i + 1).padStart(3, '0')}`,
  name: names[i % names.length]!,
  amount: 8_000_000 + ((i * 731_000) % 9_000_000),
  orgUnit: [2, 3, 4][i % 3]!,
  month: `2026-0${7 + Math.floor(i / 20)}`,
}))

const listDefaults = { filters: { q: '', org_unit: '', month: '' }, sort: '', sorts: [] as string[], pageSize: 20 }

// ListTemplate filters and pages its samples in the browser, with the state on the URL as on a real list.
export function ListTemplate() {
  const { t } = useTranslation()
  const h = useHeader('ListPage')
  const [params, set, filterBar] = useListParams(listDefaults)
  const { q, org_unit, month } = params.filters as typeof listDefaults.filters
  const needle = q.toLowerCase()
  const found = listRows.filter(
    (r) => (!needle || `${r.code} ${r.name}`.toLowerCase().includes(needle)) && (!org_unit || inUnit(r.orgUnit, Number(org_unit))) && (!month || r.month === month),
  )
  const start = (params.page - 1) * params.pageSize
  return (
    <ListPage
      title={h.title}
      action={h.action}
      filters={
        <>
          <SearchInput label={t('shared.devui.name')} value={q} onSearch={(v) => set({ filters: { q: v } })} />
          <OrgUnitFilter label={t('shared.devui.org_unit')} {...filterBar.bind('org_unit')} />
          <MonthFilter label={t('shared.devui.month')} {...filterBar.bind('month')} />
        </>
      }
      onClearFilters={filterBar.active ? filterBar.clear : undefined}
      list={{
        query: { isPending: false, isError: false, error: null, refetch: () => undefined, data: { items: found.slice(start, start + params.pageSize), total: found.length } },
        table: {
          label: t('shared.devui.table'),
          rowKey: (r) => r.id,
          columns: [
            { key: 'code', header: t('shared.devui.code'), role: 'title', kind: 'code', render: (r) => r.code },
            { key: 'name', header: t('shared.devui.name'), role: 'meta', render: (r) => r.name },
            { key: 'unit', header: t('shared.devui.org_unit'), role: 'meta', render: (r) => unitName.get(r.orgUnit) },
            { key: 'month', header: t('shared.devui.month'), role: 'meta', render: (r) => formatMonth(r.month) },
            { key: 'amount', header: t('shared.devui.amount'), role: 'meta', kind: 'number', render: (r) => formatNumber(r.amount) },
          ],
        },
        empty: { title: t('shared.devui.no_results') },
        noResults: t('shared.devui.no_results'),
        paging: { page: params.page, pageSize: params.pageSize, onPage: (page) => set({ page }), onPageSize: (pageSize) => set({ pageSize }) },
      }}
    />
  )
}

export function RecordTemplate() {
  const { t } = useTranslation()
  const h = useHeader('RecordPage')
  return (
    <WithSampleSections>
      {(_, activity) => (
        <RecordPage
          title={h.title}
          breadcrumbs={h.breadcrumbs}
          leading={<PersonAvatar name="Nguyễn Văn An" size="lg" />}
          status={<StatusBadge tone="positive">{t('shared.devui.status_positive')}</StatusBadge>}
          description="NV0019 · Kế toán"
          summary={
            <KeyFacts
              facts={[
                ['Quản lý trực tiếp', <MissingValue key="m" label="Quản lý trực tiếp" onAdd={() => {}} />],
                ['Thâm niên', '4 năm 2 tháng'],
                ['Loại hợp đồng', 'Không xác định thời hạn'],
                ['Phép còn lại', <MissingValue key="l" label="Phép còn lại" />],
              ]}
            />
          }
          actions={<Button variant="default">{t('shared.devui.secondary')}</Button>}
          tabs={[
            {
              value: 'overview',
              label: t('shared.record.overview'),
              content: (
                <RecordColumns
                  main={
                    <EditablePanel title={t('shared.devui.section')} icon={IconUsersGroup} editable edit={(close) => <Button variant="default" onClick={close}>{t('shared.devui.secondary')}</Button>}>
                      {(open) => (
                        <FieldList
                          stacked
                          rows={[
                            ['Phòng ban', 'Kế toán'],
                            ['Ngày vào làm', '05/08/2022'],
                            ['Số CCCD', <SensitiveValue key="m" label="Số CCCD" present canView tail="0123" reveal={async () => '001088000123'} />],
                            ['Email', <MissingValue key="e" label="Email" onAdd={open} />],
                            ['Địa chỉ', '4 phố số 1, Đống Đa, Hà Nội', 'wide'],
                          ]}
                        />
                      )}
                    </EditablePanel>
                  }
                  side={<Panel plain title={t('shared.document.tab.activity')}>{activity}</Panel>}
                />
              ),
            },
            {
              value: 'list',
              label: `${t('shared.devui.tab')} (3)`,
              content: (
                <Stack gap="md">
                  <TabActions>
                    <Button>{t('shared.devui.primary')}</Button>
                  </TabActions>
                  <Sample />
                </Stack>
              ),
            },
            { value: 'activity', label: t('shared.document.tab.activity'), content: activity },
          ]}
        />
      )}
    </WithSampleSections>
  )
}

export function DocumentTemplate() {
  const h = useHeader('DocumentPage')
  return (
    <WithSampleSections>
      {(sections, activity, approval) => (
        <DocumentPage
          title="NP-2026-00001"
          breadcrumbs={h.breadcrumbs}
          actions={h.action}
          approval={approval}
          activity={activity}
          sections={sections}
        >
          <Sample />
        </DocumentPage>
      )}
    </WithSampleSections>
  )
}

export function InboxTemplate() {
  const { t } = useTranslation()
  const [search, setSearch] = useSearchParams()
  const item = search.get('item')
  return (
    <InboxPage
      title="InboxPage"
      list={
        <InboxList label={t('shared.devui.list')}>
          {rows.map((r, i) => (
            <InboxRow
              key={r.id}
              active={item === String(r.id)}
              unread={i === 0}
              onClick={() => setSearch({ item: String(r.id) })}
              label={
                <Group justify="space-between" wrap="nowrap">
                  <Text size="sm" fw="inherit">
                    {r.code}
                  </Text>
                  <Text size="xs" c="dimmed">
                    {t('shared.devui.ago')}
                  </Text>
                </Group>
              }
              description={r.name}
            />
          ))}
        </InboxList>
      }
      detail={item ? <Sample /> : null}
      onBack={() => setSearch({})}
    />
  )
}
