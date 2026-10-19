import { Button, Group, Text } from '@mantine/core'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { formatNumber } from '@/shared/i18n'
import { BulkBar } from '@/shared/ui/BulkBar'
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
          href={(r) => `/dev/ui/patterns/bulk?peek=${r.id}`}
          onRowClick={(r) => peek.open(r.id)}
          selection={{ selected, onChange: setSelected }}
          onResize={view.onResize}
        />
        <SidePanel title={open?.name ?? ''} href="/dev/ui/patterns/bulk" opened={!!open} onClose={peek.close}>
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
