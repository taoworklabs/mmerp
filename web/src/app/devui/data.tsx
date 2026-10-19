import { Button, Group, Menu, Text } from '@mantine/core'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { formatNumber } from '@/shared/i18n'
import { BulkBar } from '@/shared/ui/BulkBar'
import { CellGrid } from '@/shared/ui/CellGrid'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { FieldList } from '@/shared/ui/FieldList'
import { useListView } from '@/shared/ui/ListView'
import { SidePanel, usePeek } from '@/shared/ui/SidePanel'
import { Case, Demo, rows, type Row } from './Demo'

function useColumns(sortable = false): Column<Row>[] {
  const { t } = useTranslation()
  return [
    { key: 'code', header: t('shared.devui.code'), role: 'title', kind: 'code', sortable, render: (r) => r.code },
    { key: 'name', header: t('shared.devui.name'), role: 'meta', sortable, render: (r) => r.name },
    { key: 'amount', header: t('shared.devui.amount'), role: 'meta', kind: 'number', render: (r) => formatNumber(r.amount) },
  ]
}

export function Table() {
  const { t } = useTranslation()
  const [sort, setSort] = useState('code')
  return (
    <Demo title={t('shared.devui.table')}>
      <Case>
        <DataTable
          label={t('shared.devui.table')}
          rows={rows}
          rowKey={(r) => r.id}
          href={() => '/dev/ui/table'}
          menu={() => <Menu.Item>{t('shared.devui.secondary')}</Menu.Item>}
          sort={sort}
          onSort={setSort}
          columns={useColumns(true)}
        />
      </Case>
    </Demo>
  )
}

export function GroupedTable() {
  const { t } = useTranslation()
  const total = (name: string, of: Row[]) => ({ id: 0, code: '', name: t('shared.devui.total', { name }), amount: of.reduce((s, r) => s + r.amount, 0) })
  return (
    <Demo title={t('shared.devui.grouped')}>
      <Case>
        <DataTable
          label={t('shared.devui.grouped')}
          rows={rows}
          rowKey={(r) => r.id}
          group={{ of: (r) => (r.id === 1 ? 'A' : 'B'), total: (g, name) => total(name, g) }}
          footer={total('', rows)}
          columns={useColumns()}
        />
      </Case>
    </Demo>
  )
}

export function Grid() {
  const { t } = useTranslation()
  const [cells, setCells] = useState<Record<string, string>>({ 'NV001:1': '1', 'NV001:2': '0,5', 'NV002:2': '2' })
  return (
    <Demo title={t('shared.devui.grid')}>
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
    </Demo>
  )
}

export function Bulk() {
  const { t } = useTranslation()
  const [selected, setSelected] = useState<Set<string | number>>(new Set())
  const peek = usePeek()
  const view = useListView('dev.ui', useColumns())
  const open = rows.find((r) => String(r.id) === peek.id)
  return (
    <Demo title={t('shared.devui.bulk')} description={t('shared.devui.bulk_hint')}>
      <Case>
        {selected.size > 0 ? (
          <BulkBar count={selected.size} onClear={() => setSelected(new Set())}>
            <Button variant="default">{t('shared.devui.export')}</Button>
          </BulkBar>
        ) : (
          <Group justify="space-between">
            <Text size="sm" c="dimmed">
              {t('shared.devui.bulk_none')}
            </Text>
            {view.tools}
          </Group>
        )}
        <DataTable
          label={t('shared.devui.bulk')}
          columns={view.columns}
          rows={rows}
          rowKey={(r) => r.id}
          href={(r) => `/dev/ui/bulk?peek=${r.id}`}
          onRowClick={(r) => peek.open(r.id)}
          selection={{ selected, onChange: setSelected }}
          onResize={view.onResize}
        />
        <SidePanel title={open?.name ?? ''} href="/dev/ui/bulk" opened={!!open} onClose={peek.close}>
          {open && (
            <FieldList
              rows={[
                [t('shared.devui.code'), open.code],
                [t('shared.devui.amount'), formatNumber(open.amount)],
              ]}
            />
          )}
        </SidePanel>
      </Case>
    </Demo>
  )
}
