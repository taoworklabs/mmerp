import { Button, Group, Stack, Text } from '@mantine/core'
import { IconUsersGroup } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router'
import { WithSampleSections } from '@/shared/document/samples'
import { formatNumber } from '@/shared/i18n'
import { PersonAvatar } from '@/shared/ui/avatar'
import { DataTable } from '@/shared/ui/DataTable'
import { FieldList } from '@/shared/ui/FieldList'
import { EditablePanel, MonthFilter, OrgUnitFilter, SearchInput, SensitiveValue } from '@/shared/ui/form'
import { DetailPage, DetailTabs, DocumentPage, InboxList, InboxPage, InboxRow, ListPage, Page, Panel, TabActions } from '@/shared/ui/page'
import { EmptyState } from '@/shared/ui/states'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { rows } from './Demo'

// The body every template shows, so the templates differ only in their frame.
function Sample() {
  const { t } = useTranslation()
  return (
    <DataTable
      label={t('shared.devui.table')}
      rows={rows}
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

export function ListTemplate() {
  const { t } = useTranslation()
  const h = useHeader('ListPage')
  return (
    <ListPage
      title={h.title}
      action={h.action}
      filters={
        <>
          <SearchInput label={t('shared.devui.name')} value="" onSearch={() => {}} />
          <OrgUnitFilter label={t('shared.devui.org_unit')} value={null} onChange={() => {}} />
          <MonthFilter label={t('shared.devui.month')} value="2026-08" onChange={() => {}} />
        </>
      }
      onClearFilters={() => {}}
      paging={{ page: 1, pageSize: 50, total: 120, onPage: () => {}, onPageSize: () => {} }}
    >
      <Sample />
    </ListPage>
  )
}

export function DetailTemplate() {
  const { t } = useTranslation()
  const h = useHeader('DetailPage')
  return (
    <WithSampleSections>
      {(_, activity) => (
        <DetailPage
          title={h.title}
          breadcrumbs={h.breadcrumbs}
          leading={<PersonAvatar name="Nguyễn Văn An" size="lg" />}
          side={
            <>
              <DetailTabs
                tabs={[
                  { value: 'one', label: `${t('shared.devui.tab')} 1`, content: <Sample /> },
                  {
                    value: 'two',
                    label: `${t('shared.devui.tab')} 2`,
                    content: (
                      <Stack gap="md">
                        <TabActions>
                          <Button>{t('shared.devui.primary')}</Button>
                        </TabActions>
                        <EmptyState title={t('shared.devui.empty')} />
                      </Stack>
                    ),
                  },
                ]}
              />
              <Panel title={t('shared.document.tab.activity')}>{activity}</Panel>
            </>
          }
        >
          {/* One group edited in place, one that only reads, with a masked value. */}
          <EditablePanel
            title={t('shared.devui.section')}
            icon={IconUsersGroup}
            editable
            edit={(close) => (
              <Group justify="flex-end">
                <Button variant="default" onClick={close}>
                  {t('shared.devui.secondary')}
                </Button>
              </Group>
            )}
          >
            <FieldList
              rows={[
                ['Mã nhân viên', 'NV0019'],
                ['Phòng ban', 'Kế toán'],
                ['Trạng thái', <StatusBadge key="s" tone="positive">{t('shared.devui.status_positive')}</StatusBadge>],
              ]}
            />
          </EditablePanel>
          <Panel title={t('shared.devui.section_hint')}>
            <FieldList
              rows={[
                ['Ngày sinh', '02/11/1988'],
                ['Số CCCD', <SensitiveValue key="m" label="Số CCCD" present canView reveal={async () => '001088000123'} />],
              ]}
            />
          </Panel>
        </DetailPage>
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
