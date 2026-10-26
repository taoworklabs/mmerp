import { Anchor, Button, FieldList, Group, KeyFacts, Text } from '@mmerp/ui'
import { Menu } from '@mantine/core'
import { IconDownload, IconFileDescription, IconPencil, IconPlus, IconUsersGroup } from '@tabler/icons-react'
import { type ReactNode, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { formatNumber } from '@/shared/i18n'
import { PersonAvatar, PersonName } from '@/shared/ui/avatar'
import { CellGrid } from '@/shared/ui/CellGrid'
import { type Column, DataTable } from '@/shared/ui/DataTable'
import { useListView } from '@/shared/ui/ListView'
import { MenuAction } from '@/shared/ui/MenuAction'
import { MoreActions } from '@/shared/ui/MoreActions'
import { OrgUnitName } from '@/shared/ui/OrgUnitName'
import { Panel } from '@/shared/ui/page'
import { Section } from '@/shared/ui/Section'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { icon } from '@/shared/ui/theme'
import { LinkTile, StatTile, TileGrid } from '@/shared/ui/Tile'
import { TreeList } from '@/shared/ui/TreeList'
import { Case, rows, type Row, facts } from './Demo'
import { DocPage, DoDont } from './DocPage'

function useColumns(sortable = false): Column<Row>[] {
  const { t } = useTranslation()
  return [
    { key: 'code', header: t('shared.devui.code'), role: 'title', kind: 'code', sortable, render: (r) => r.code },
    { key: 'name', header: t('shared.devui.name'), role: 'meta', sortable, render: (r) => r.name },
    { key: 'amount', header: t('shared.devui.amount'), role: 'meta', kind: 'number', render: (r) => formatNumber(r.amount) },
  ]
}

// The main props of each component; data, not interface text.
const code = {
  dataTable: `<DataTable label rows rowKey columns
  href? onRowClick? menu? sort? onSort? group? footer? selection? onResize? />
// column: { key, header, role: 'title' | 'meta', kind?: 'code' | 'number', sortable?, render }`,
  cellGrid: `<CellGrid label corner rows columns value onChange invalid? off? warning? total? />`,
  fieldList: `<FieldList rows={[[label, value, 'wide'?], …]} stacked? />
import { MissingValue } from '@/shared/ui/MissingValue'   // an empty value users can fill`,
  keyFacts: `import { KeyFacts } from '@mmerp/ui'
<KeyFacts facts={[[label, value], …]} />   // in the page header's summary`,
  treeList: `<TreeList label nodes render meta? toggleLabel />
import { OrgUnitName } from '@/shared/ui/OrgUnitName'   // render an org unit`,
  statusBadge: `<StatusBadge tone="positive" | "neutral" | "info" | "warning" | "negative">{label}</StatusBadge>`,
  personAvatar: `<PersonAvatar name size="sm" | "md" | "lg" />
import { PersonName } from '@/shared/ui/avatar'   // avatar and name together`,
  tiles: `<TileGrid><StatTile label value to error? onRetry? /><LinkTile icon title to /></TileGrid>`,
  section: `<Section title description? actions?>{content}</Section>`,
  panel: `<Panel title icon? actions? plain?>{content}</Panel>`,
}

const values: [string, ReactNode][] = [
  ['Phòng ban', 'Kế toán'],
  ['Ngày vào làm', '05/08/2022'],
  ['Email', 'an.nguyen@demo.vn'],
  ['Địa chỉ', '4 phố số 1, Đống Đa, Hà Nội'],
]

export function DataTableDoc() {
  const { t } = useTranslation()
  const [sort, setSort] = useState('code')
  const columns = useColumns(true)
  const view = useListView('dev.ui', useColumns())
  const total = (name: string, of: Row[]) => ({ id: 0, code: '', name: t('shared.devui.total', { name }), amount: of.reduce((s, r) => s + r.amount, 0) })
  return (
    <DocPage
      path="mmerp/data-table"
      example={
        <Case>
          <DataTable
            label={t('shared.devui.table')}
            rows={rows}
            rowKey={(r) => r.id}
            href={() => '/dev/ui/mmerp/data-table'}
            menu={() => <Menu.Item>{t('shared.devui.secondary')}</Menu.Item>}
            sort={sort}
            onSort={setSort}
            columns={columns}
          />
        </Case>
      }
      variants={
        <>
          <Case label={t('shared.devui.grouped')}>
            <DataTable
              label={t('shared.devui.grouped')}
              rows={rows}
              rowKey={(r) => r.id}
              group={{ of: (r) => (r.id === 1 ? 'A' : 'B'), total: (g, name) => total(name, g) }}
              footer={total('', rows)}
              columns={useColumns()}
            />
          </Case>
          <Case label={t('shared.devui.case_column_tools')}>
            <Group justify="end">{view.tools}</Group>
            <DataTable label={t('shared.devui.table')} columns={view.columns} rows={rows} rowKey={(r) => r.id} onResize={view.onResize} />
          </Case>
        </>
      }
      doDont={
        <DoDont
          good={<Text size="sm">{t('shared.devui.do_table_numbers')}</Text>}
          bad={<Text size="sm">{t('shared.devui.dont_table_own')}</Text>}
        />
      }
      code={code.dataTable}
    />
  )
}

export function CellGridDoc() {
  const { t } = useTranslation()
  const [cells, setCells] = useState<Record<string, string>>({ 'NV001:1': '1', 'NV001:2': '0,5', 'NV002:2': '2' })
  return (
    <DocPage
      path="mmerp/cell-grid"
      example={
        <Case>
          <CellGrid
            label={t('shared.devui.grid')}
            corner={t('shared.devui.name')}
            rows={rows.map((r) => ({ key: r.code, header: r.name, label: r.name }))}
            columns={Array.from({ length: 10 }, (_, i) => ({ key: String(i + 1), header: i + 1, label: String(i + 1) }))}
            value={(r, c) => cells[`${r}:${c}`] ?? ''}
            onChange={(r, c, v) => setCells({ ...cells, [`${r}:${c}`]: v })}
            invalid={(v) => !['', '1', '0,5', '0.5'].includes(v)}
            off={(r, c) => r === 'NV002' && c === '1'}
            warning={(r, c) => (r === 'NV001' && c === '3' ? t('shared.devui.hint') : undefined)}
            total={{ header: t('shared.devui.total'), value: (r) => Object.entries(cells).filter(([k]) => k.startsWith(`${r}:`)).length }}
          />
        </Case>
      }
      code={code.cellGrid}
    />
  )
}

// Sample values the links point at; data, not interface text.
const person = 'NV0019 · Lâm Đức Thịnh'

export function FieldListDoc() {
  const { t } = useTranslation()
  return (
    <DocPage
      path="mantine/field-list"
      example={
        <Case>
          <FieldList
            rows={[
              [
                'Nhân viên',
                <Anchor key="p" render={<Link to="/dev/ui/mantine/field-list" />}>
                  {person}
                </Anchor>,
              ],
              ['Thời gian nghỉ', '12/10/2026 – 16/10/2026'],
              ['Trạng thái', <StatusBadge key="s" tone="positive">{t('shared.devui.status_positive')}</StatusBadge>],
              ['Nơi làm việc', t('shared.devui.long_value')],
              ['Ghi chú', '—'],
            ]}
          />
        </Case>
      }
      variants={
        <Case label={t('shared.devui.case_stacked')}>
          <FieldList stacked rows={values} />
        </Case>
      }
      code={code.fieldList}
    />
  )
}

export function KeyFactsDoc() {
  return (
    <DocPage
      path="mantine/key-facts"
      example={
        <Case>
          <KeyFacts facts={facts} />
        </Case>
      }
      code={code.keyFacts}
    />
  )
}

export function TreeListDoc() {
  const { t } = useTranslation()
  return (
    <DocPage
      path="mmerp/tree-list"
      example={
        <Case>
          <TreeList
            label={t('shared.devui.tree')}
            nodes={[
              {
                key: 1,
                item: { kind: 'company' as const, name: 'Công ty CP Demo' },
                children: [
                  { key: 2, item: { kind: 'department' as const, name: 'Kế toán' }, children: [] },
                  { key: 3, item: { kind: 'branch' as const, name: 'Chi nhánh Hà Nội' }, children: [{ key: 4, item: { kind: 'department' as const, name: 'Kinh doanh' }, children: [] }] },
                ],
              },
            ]}
            render={(unit) => <OrgUnitName unit={unit} />}
            meta={() => 'Meta'}
            toggleLabel={(unit) => unit.name}
          />
        </Case>
      }
      code={code.treeList}
    />
  )
}

export function StatusBadgeDoc() {
  const { t } = useTranslation()
  return (
    <DocPage
      path="mmerp/status-badge"
      example={
        <Case>
          <Group>
            <StatusBadge tone="positive">{t('shared.devui.status_positive')}</StatusBadge>
          </Group>
        </Case>
      }
      variants={
        <Case label={t('shared.devui.case_record_status')}>
          <Group>
            <StatusBadge tone="positive">{t('shared.devui.status_positive')}</StatusBadge>
            <StatusBadge tone="neutral">{t('shared.devui.status_neutral')}</StatusBadge>
            <StatusBadge tone="info">{t('shared.devui.status_info')}</StatusBadge>
            <StatusBadge tone="warning">{t('shared.devui.status_warning')}</StatusBadge>
            <StatusBadge tone="negative">{t('shared.devui.status_negative')}</StatusBadge>
          </Group>
        </Case>
      }
      doDont={
        <DoDont
          good={
            <Group>
              <StatusBadge tone="neutral">{t('shared.devui.status_neutral')}</StatusBadge>
            </Group>
          }
          bad={
            <>
              <Group>
                <StatusBadge tone="positive">{t('shared.devui.status_neutral')}</StatusBadge>
              </Group>
              <Text size="sm" tone="dimmed">
                {t('shared.devui.dont_badge_tone')}
              </Text>
            </>
          }
        />
      }
      code={code.statusBadge}
    />
  )
}

export function PersonAvatarDoc() {
  const { t } = useTranslation()
  return (
    <DocPage
      path="mmerp/person-avatar"
      example={
        <Case>
          <Group>
            <PersonName name="Nguyễn Văn An" />
            <PersonName name="Trần Thị Bình" />
            <PersonName name="Lê Văn Cường" />
          </Group>
        </Case>
      }
      variants={
        <Case label={t('shared.devui.case_avatar_sizes')}>
          <Group>
            <PersonAvatar name="Trần Thị Bình" size="sm" />
            <PersonAvatar name="Trần Thị Bình" size="md" />
            <PersonAvatar name="Trần Thị Bình" size="lg" />
          </Group>
        </Case>
      }
      code={code.personAvatar}
    />
  )
}

export function TilesDoc() {
  const { t } = useTranslation()
  return (
    <DocPage
      path="mmerp/tiles"
      example={
        <Case>
          <TileGrid>
            <StatTile label={t('shared.devui.name')} value={128} to="/dev/ui/mmerp/tiles" />
            <LinkTile icon={IconUsersGroup} title={t('shared.devui.name')} to="/dev/ui/mmerp/tiles" />
          </TileGrid>
        </Case>
      }
      variants={
        <Case label={t('shared.devui.case_stat_tile')}>
          <TileGrid>
            <StatTile label={t('shared.devui.name')} value={128} to="/dev/ui/mmerp/tiles" />
            <StatTile label={t('shared.devui.name')} value={undefined} to="/dev/ui/mmerp/tiles" />
            <StatTile label={t('shared.devui.name')} value={undefined} to="/dev/ui/mmerp/tiles" error={t('shared.devui.error')} />
          </TileGrid>
        </Case>
      }
      code={code.tiles}
    />
  )
}

function StaffTable() {
  const { t } = useTranslation()
  return <DataTable label={t('shared.devui.section_staff')} rows={rows} rowKey={(r) => r.id} columns={useColumns()} />
}

export function SectionDoc() {
  const { t } = useTranslation()
  return (
    <DocPage
      path="mmerp/section"
      example={
        <Case>
          <Section
            title={t('shared.devui.section_staff')}
            description={t('shared.devui.section_hint')}
            actions={
              <>
                <Button variant="default" leftSection={<IconPlus {...icon.button} />}>
                  {t('shared.devui.add')}
                </Button>
                <MoreActions place="page">
                  <MenuAction icon={IconDownload} onClick={() => {}}>
                    {t('shared.devui.export')}
                  </MenuAction>
                </MoreActions>
              </>
            }
          >
            <StaffTable />
          </Section>
        </Case>
      }
      variants={
        <>
          <Case label={t('shared.devui.case_section_title')}>
            <Section title={t('shared.devui.section_staff')}>
              <StaffTable />
            </Section>
          </Case>
          <Case label={t('shared.devui.case_section_stack')}>
            <Section title={t('shared.devui.section_company')}>
              <FieldList rows={[[t('shared.devui.name'), 'Công ty CP Demo']]} />
            </Section>
            <Section title={t('shared.devui.section_staff')}>
              <StaffTable />
            </Section>
          </Case>
        </>
      }
      code={code.section}
    />
  )
}

export function PanelDoc() {
  const { t } = useTranslation()
  return (
    <DocPage
      path="mmerp/panel"
      example={
        <Case>
          <Panel
            title={t('shared.devui.section_company')}
            icon={IconFileDescription}
            actions={
              <Button variant="subtle" leftSection={<IconPencil {...icon.button} />}>
                {t('shared.devui.edit')}
              </Button>
            }
          >
            <FieldList stacked rows={values} />
          </Panel>
        </Case>
      }
      variants={
        <Case label={t('shared.devui.case_panel_plain')}>
          <Panel plain title={t('shared.devui.section_staff')}>
            <FieldList rows={[['NV001', 'Nguyễn Văn An'], ['NV002', 'Trần Thị Bình']]} />
          </Panel>
        </Case>
      }
      code={code.panel}
    />
  )
}
