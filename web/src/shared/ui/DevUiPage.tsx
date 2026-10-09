import { Button, Group, Menu, SegmentedControl, Stack, Text, Title } from '@mantine/core'
import { IconUsersGroup } from '@tabler/icons-react'
import { useState, type ReactNode } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router'
import { formatNumber } from '@/shared/i18n'
import { PersonAvatar, PersonName } from './avatar'
import { DataTable } from './DataTable'
import { FieldList } from './FieldList'
import { OrgUnitName } from './OrgUnitName'
import { StatusBadge } from './StatusBadge'
import { LinkTile, StatTile, TileGrid } from './Tile'
import { CellGrid } from './CellGrid'
import { CheckboxField, DateField, DecimalField, FileField, Form, MoneyField, MonthField, MonthFilter, FormActions, FormRow, FormSection, InputRow, OrgUnitField, OrgUnitSelect, PasswordField, SearchInput, SelectField, SensitiveField, TextField } from './form'
import { useConfirm } from './confirm'
import { DocumentPage, type DocumentSection, FormModal, InboxList, InboxPage, InboxRow, ListPage, Page, RecordPage, TabActions } from './page'
import { ContentSkeleton, EmptyState, ErrorState, NotFoundPage, ReadOnlyBanner } from './states'

const rows = [
  { id: 1, code: 'NV001', name: 'Nguyễn Văn An', amount: 12500000 },
  { id: 2, code: 'NV002', name: 'Trần Thị Bình', amount: 9800000 },
]

// DevUiPage shows every shared/ui template and primitive in each state; dev builds only.
// The template is chosen with ?template=page|list|record|document|inbox, so each one can be opened directly.
type Props = {
  // Renders its child with sample side sections (attachments, discussion) of a document.
  withSections: (p: { children: (sections: DocumentSection[]) => ReactNode }) => ReactNode
}

export default function DevUiPage({ withSections: WithSections }: Props) {
  const { t } = useTranslation()
  const [search, setSearch] = useSearchParams()
  const template = search.get('template') ?? 'page'
  const [sort, setSort] = useState('code')
  const [modal, setModal] = useState(false)
  const [ask, dialog] = useConfirm()
  const form = useForm({ defaultValues: { code: '', password: '', unit: null, date: null, month: '2026-03', file: null, choice: null, masked: undefined, open: undefined, denied: undefined, days: '2.5', flag: true } })
  const [cells, setCells] = useState<Record<string, string>>({ 'NV001:1': '1', 'NV001:2': '0,5', 'NV002:2': '2' })
  const title = t('shared.devui.title')
  const action = <Button>{t('shared.devui.primary')}</Button>
  const content = (
    <Stack gap="lg">
      <Stack gap="xs">
        <Title order={2}>{t('shared.devui.template')}</Title>
        <SegmentedControl
          w="fit-content"
          value={template}
          onChange={(v) => setSearch({ template: v })}
          data={['page', 'list', 'record', 'document', 'inbox'].map((v) => ({
            value: v,
            label: { page: 'Page', list: 'ListPage', record: 'RecordPage', document: 'DocumentPage', inbox: 'InboxPage' }[v] ?? v,
          }))}
        />
        <Group>
          <Button variant="default" onClick={() => setModal(true)}>
            {t('shared.devui.open_modal')}
          </Button>
          <Button
            variant="default"
            onClick={() =>
              void ask({ title: t('shared.devui.confirm'), message: t('shared.devui.confirm_message'), confirmLabel: t('shared.devui.danger'), danger: true })
            }
          >
            {t('shared.devui.confirm')}
          </Button>
        </Group>
        <FormModal opened={modal} onClose={() => setModal(false)} title={t('shared.devui.open_modal')}>
          <EmptyState title={t('shared.devui.empty')} />
        </FormModal>
        {dialog}
      </Stack>
      <Stack gap="xs">
        <Title order={2}>{t('shared.devui.buttons')}</Title>
        <Group>
          <Button>{t('shared.devui.primary')}</Button>
          <Button variant="default">{t('shared.devui.secondary')}</Button>
          <Button color="danger">{t('shared.devui.danger')}</Button>
          <Button loading>{t('shared.devui.primary')}</Button>
          <Button disabled>{t('shared.devui.primary')}</Button>
        </Group>
      </Stack>
      <Stack gap="xs">
        <Title order={2}>{t('shared.devui.people')}</Title>
        <Group>
          <PersonName name="Nguyễn Văn An" />
          <OrgUnitName unit={{ kind: 'company', name: 'Công ty CP Demo' }} />
          <OrgUnitName unit={{ kind: 'department', name: 'Kế toán' }} depth={1} />
          <PersonAvatar name="Trần Thị Bình" size="lg" />
          <StatusBadge tone="positive">{t('shared.devui.status_positive')}</StatusBadge>
          <StatusBadge tone="neutral">{t('shared.devui.status_neutral')}</StatusBadge>
          <StatusBadge tone="info">{t('shared.devui.status_info')}</StatusBadge>
          <StatusBadge tone="warning">{t('shared.devui.status_warning')}</StatusBadge>
          <StatusBadge tone="negative">{t('shared.devui.status_negative')}</StatusBadge>
        </Group>
        <FieldList
          rows={[
            ['Nhân viên', 'NV0019 · Lâm Đức Thịnh'],
            ['Thời gian nghỉ', '12/10/2026 – 16/10/2026'],
            ['Lý do', 'Du lịch nước ngoài'],
          ]}
        />
      </Stack>
      <Stack gap="xs">
        <Title order={2}>{t('shared.devui.tiles')}</Title>
        <TileGrid>
          <StatTile label={t('shared.devui.name')} value={128} to="/dev/ui" />
          <StatTile label={t('shared.devui.name')} value={undefined} to="/dev/ui" />
          <StatTile label={t('shared.devui.name')} value={undefined} to="/dev/ui" error={t('shared.devui.error')} />
          <LinkTile icon={IconUsersGroup} title={t('shared.devui.name')} to="/dev/ui" />
        </TileGrid>
      </Stack>
      <Stack gap="xs">
        <Title order={2}>{t('shared.devui.table')}</Title>
        <DataTable
          label={t('shared.devui.table')}
          rows={rows}
          rowKey={(r) => r.id}
          href={() => '/dev/ui'}
          menu={() => <Menu.Item>{t('shared.devui.secondary')}</Menu.Item>}
          sort={sort}
          onSort={setSort}
          columns={[
            { key: 'code', header: t('shared.devui.code'), role: 'title', sortable: true, render: (r) => r.code },
            { key: 'name', header: t('shared.devui.name'), role: 'meta', sortable: true, render: (r) => r.name },
            { key: 'amount', header: t('shared.devui.amount'), role: 'meta', numeric: true, render: (r) => formatNumber(r.amount) },
          ]}
        />
      </Stack>
      <Stack gap="xs">
        <Title order={2}>{t('shared.devui.grouped')}</Title>
        <DataTable
          label={t('shared.devui.grouped')}
          rows={rows}
          rowKey={(r) => r.id}
          group={{ of: (r) => (r.id === 1 ? 'A' : 'B'), total: (g, name) => ({ id: 0, code: '', name: t('shared.devui.total', { name }), amount: g.reduce((s, r) => s + r.amount, 0) }) }}
          footer={{ id: 0, code: '', name: t('shared.devui.total', { name: '' }), amount: rows.reduce((s, r) => s + r.amount, 0) }}
          columns={[
            { key: 'code', header: t('shared.devui.code'), role: 'title', render: (r) => r.code },
            { key: 'name', header: t('shared.devui.name'), role: 'meta', render: (r) => r.name },
            { key: 'amount', header: t('shared.devui.amount'), role: 'meta', numeric: true, render: (r) => formatNumber(r.amount) },
          ]}
        />
      </Stack>
      <Stack gap="xs">
        <Title order={2}>{t('shared.devui.grid')}</Title>
        <CellGrid
          label={t('shared.devui.grid')}
          corner={t('shared.devui.name')}
          rows={rows.map((r) => ({ key: r.code, header: r.name, label: r.name }))}
          columns={Array.from({ length: 10 }, (_, i) => ({ key: String(i + 1), header: i + 1, label: String(i + 1) }))}
          value={(r, c) => cells[`${r}:${c}`] ?? ''}
          onChange={(r, c, v) => setCells({ ...cells, [`${r}:${c}`]: v })}
          invalid={(v) => !['', '1', '0,5', '0.5'].includes(v)}
          off={(r, c) => r === 'NV002' && c === '1'}
          total={{ header: t('shared.devui.total'), value: (r) => Object.entries(cells).filter(([k]) => k.startsWith(`${r}:`)).length }}
        />
      </Stack>
      <Stack gap="xs">
        <Title order={2}>{t('shared.devui.form')}</Title>
        <Form form={form} onSubmit={async () => {}}>
          <FormSection title={t('shared.devui.form')} description={t('shared.devui.hint')}>
            <TextField name="code" label={t('shared.devui.code')} required description={t('shared.devui.hint')} />
            <PasswordField name="password" label={t('shared.devui.password')} />
            <OrgUnitField name="unit" label={t('shared.devui.org_unit')} clearable />
            <DateField name="date" label={t('shared.devui.date')} clearable />
            <MonthField name="month" label={t('shared.devui.month')} />
            <FileField name="file" label={t('shared.devui.file')} accept=".xlsx" description={t('shared.devui.hint')} />
            <SelectField
              name="choice"
              label={t('shared.devui.choice')}
              data={[
                { value: 'a', label: 'A' },
                { value: 'b', label: 'B' },
              ]}
            />
            <DecimalField name="days" label={t('shared.devui.decimal')} scale={1} step={0.5} />
            <MoneyField name="salary" label={t('shared.devui.money')} />
            <InputRow>
              <CheckboxField name="beside" label={t('shared.devui.checkbox')} />
            </InputRow>
            <CheckboxField name="flag" label={t('shared.devui.checkbox')} description={t('shared.devui.hint')} />
            <FormRow>
              <Text size="sm" c="dimmed">
                {t('shared.devui.hint')}
              </Text>
            </FormRow>
            <SensitiveField name="masked" label={t('shared.devui.sensitive')} present canView canEdit reveal={async () => '079123456789'} />
            <SensitiveField name="open" label={t('shared.devui.sensitive')} present={false} canView canEdit reveal={async () => null} />
            <SensitiveField name="denied" label={t('shared.devui.sensitive')} present canView={false} canEdit={false} reveal={async () => null} />
          </FormSection>
          <FormActions submitLabel={t('shared.devui.primary')} />
        </Form>
      </Stack>
      <Stack gap="xs">
        <Title order={2}>{t('shared.devui.states')}</Title>
        <ContentSkeleton />
        <ReadOnlyBanner />
        <EmptyState title={t('shared.devui.empty')} description={t('shared.devui.empty_description')} />
        <EmptyState title={t('shared.devui.no_results')} action={<Button variant="default">{t('shared.devui.clear_filters')}</Button>} />
        <ErrorState message={t('shared.error.network_error')} onRetry={() => {}} />
        <NotFoundPage back={{ to: '/dev/ui', label: t('shared.devui.back') }} />
      </Stack>
    </Stack>
  )
  if (template === 'list')
    return (
      <ListPage
        title={title}
        action={action}
        filters={
          <>
            <SearchInput label={t('shared.devui.name')} value="" onSearch={() => {}} />
            <OrgUnitSelect label={t('shared.devui.org_unit')} value={null} onChange={() => {}} clearable />
            <MonthFilter label={t('shared.devui.month')} value={null} onChange={() => {}} />
          </>
        }
        paging={{ page: 1, pageSize: 50, total: 120, onPage: () => {}, onPageSize: () => {} }}
      >
        {content}
      </ListPage>
    )
  if (template === 'document')
    return (
      <WithSections>
        {(sections) => (
          <DocumentPage
            title="NP-2026-00001"
            actions={action}
            error={t('shared.error.network_error')}
            approval={<EmptyState title={t('shared.devui.empty')} />}
            history={<EmptyState title={t('shared.devui.empty')} />}
            sections={sections}
          >
            {content}
          </DocumentPage>
        )}
      </WithSections>
    )
  if (template === 'inbox')
    return (
      <InboxPage
        title={title}
        list={
          <InboxList label={t('shared.devui.list')}>
            {rows.map((r, i) => (
              <InboxRow
                key={r.id}
                active={search.get('item') === String(r.id)}
                unread={i === 0}
                onClick={() => setSearch({ template: 'inbox', item: String(r.id) })}
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
        detail={search.get('item') ? content : null}
        onBack={() => setSearch({ template: 'inbox' })}
      />
    )
  if (template === 'record')
    return (
      <RecordPage
        title={title}
        tabs={[
          { value: 'one', label: `${t('shared.devui.tab')} 1`, content },
          {
            value: 'two',
            label: `${t('shared.devui.tab')} 2`,
            content: (
              <Stack gap="md">
                <TabActions>
                  <Button size="xs">{t('shared.devui.primary')}</Button>
                </TabActions>
                <EmptyState title={t('shared.devui.empty')} />
              </Stack>
            ),
          },
        ]}
      />
    )
  return (
    <Page title={title} description={t('shared.devui.description')} breadcrumbs={[{ label: t('shared.app.title'), to: '/' }]} actions={action}>
      {content}
    </Page>
  )
}
