import { Button, Group, Menu, Stack, Text } from '@mantine/core'
import { IconCircleDot, IconDownload, IconPlus } from '@tabler/icons-react'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { formatNumber } from '@/shared/i18n'
import { BulkBar } from './BulkBar'
import { DataTable, type Column } from './DataTable'
import { FieldList } from './FieldList'
import { DecimalField, FilterChip, Form, FormActions, FormRow, FormSection, IntegerField, LinesTable, LinesTotals, MoneyField, MonthFilter, TextField, YearFilter } from './form'
import { InlineEditField } from './InlineEditField'
import { useListView } from './ListView'
import { MoreActions } from './MoreActions'
import { notifyUndo } from './notify'
import { Section } from './Section'
import { SidePanel, usePeek } from './SidePanel'
import { icon } from './theme'

type Row = { id: number; code: string; name: string; amount: number }
const rows: Row[] = [
  { id: 1, code: 'NV001', name: 'Nguyễn Văn An', amount: 12500000 },
  { id: 2, code: 'NV002', name: 'Trần Thị Bình', amount: 9800000 },
  { id: 3, code: 'NV003', name: 'Lê Văn Cường', amount: 11200000 },
]

// DevUiPatterns shows the interaction patterns of shared/ui: sections, bulk selection, the side
// panel, inline edit, editable lines, undo. Part of /dev/ui.
export function DevUiPatterns() {
  const { t } = useTranslation()
  const [selected, setSelected] = useState<Set<string | number>>(new Set())
  const [name, setName] = useState('Công ty CP Demo')
  const [year, setYear] = useState(2026)
  const [choice, setChoice] = useState<string | null>('positive')
  const [month, setMonth] = useState<string | null>('2026-08')
  const choices = [
    { value: 'positive', label: t('shared.devui.status_positive') },
    { value: 'neutral', label: t('shared.devui.status_neutral') },
    { value: 'long', label: t('shared.devui.long_value') },
  ]
  const peek = usePeek()
  const form = useForm({ defaultValues: { levels: 2, lines: [{ description: 'Dịch vụ tư vấn', quantity: '2', unit_price: 1500000 }] } })
  const columns: Column<Row>[] = [
    { key: 'code', header: t('shared.devui.code'), role: 'title', kind: 'code', render: (r) => r.code },
    { key: 'name', header: t('shared.devui.name'), role: 'meta', render: (r) => r.name },
    { key: 'amount', header: t('shared.devui.amount'), role: 'meta', kind: 'number', render: (r) => formatNumber(r.amount) },
  ]
  const view = useListView('dev.ui', columns)
  const open = rows.find((r) => String(r.id) === peek.id)
  return (
    <Stack gap="lg">
      <Section
        title={t('shared.devui.section')}
        description={t('shared.devui.section_hint')}
        actions={
          <>
            <Button size="xs" variant="default" leftSection={<IconPlus {...icon.text} />}>
              {t('shared.devui.add')}
            </Button>
            <MoreActions place="page">
              <Menu.Item leftSection={<IconDownload {...icon.text} />}>{t('shared.devui.export')}</Menu.Item>
            </MoreActions>
          </>
        }
      >
        <FieldList rows={[[t('shared.devui.name'), 'Công ty CP Demo']]} />
      </Section>

      <Section title={t('shared.devui.bulk')} description={t('shared.devui.bulk_hint')}>
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
          href={(r) => `/dev/ui?template=list&peek=${r.id}`}
          onRowClick={(r) => peek.open(r.id)}
          selection={{ selected, onChange: setSelected }}
          onResize={view.onResize}
        />
        <SidePanel title={open?.name ?? ''} href={`/dev/ui?template=list`} opened={!!open} onClose={peek.close}>
          {open && (
            <FieldList
              rows={[
                [t('shared.devui.code'), open.code],
                [t('shared.devui.amount'), formatNumber(open.amount)],
              ]}
            />
          )}
        </SidePanel>
      </Section>

      <Section title={t('shared.devui.inline')} description={t('shared.devui.inline_hint')}>
        <Group align="flex-start" grow maw={640}>
          <InlineEditField
            label={t('shared.devui.name')}
            value={name}
            validate={(v) => (v ? null : t('shared.form.required'))}
            onSave={async (v) => {
              await new Promise((r) => setTimeout(r, 400))
              setName(v)
            }}
          />
          <InlineEditField label={t('shared.devui.code')} value="KH001" readOnly onSave={async () => {}} />
        </Group>
        <Group>
          <Button variant="default" onClick={() => notifyUndo(t('shared.devui.undo_done'), t('shared.undo'), () => undefined)}>
            {t('shared.devui.undo')}
          </Button>
        </Group>
        {/* Filter chips: unset with a search (type to see no results), set (tinted, with its clear
            button), a value too long for the chip, always set (a year), a month. */}
        <Group gap="xs">
          <FilterChip label={t('shared.devui.choice')} icon={IconCircleDot} data={choices} value={null} onChange={() => {}} searchable />
          <FilterChip label={t('shared.devui.choice')} icon={IconCircleDot} data={choices} value={choice} onChange={setChoice} />
          <FilterChip label={t('shared.devui.choice')} icon={IconCircleDot} data={choices} value="long" onChange={() => {}} />
          <YearFilter label={t('shared.devui.year')} value={year} years={[2025, 2026, 2027]} onChange={setYear} />
          <MonthFilter label={t('shared.devui.month')} value={month} onChange={setMonth} />
        </Group>
      </Section>

      <Section title={t('shared.devui.lines')} description={t('shared.devui.lines_hint')}>
        <Form form={form} onSubmit={async () => {}}>
          <FormSection title={t('shared.devui.form')}>
            <IntegerField name="levels" label={t('shared.devui.levels')} min={1} max={5} />
            <FormRow>
              <LinesTable
                name="lines"
                label={t('shared.devui.lines')}
                columns={[
                  { key: 'description', header: t('shared.devui.name'), required: true, cell: (i, bare) => <TextField name={`lines.${i}.description`} label={`${t('shared.devui.name')} (${i + 1})`} bare={bare} required /> },
                  { key: 'quantity', header: t('shared.devui.quantity'), numeric: true, width: 110, cell: (i, bare) => <DecimalField name={`lines.${i}.quantity`} label={`${t('shared.devui.quantity')} (${i + 1})`} bare={bare} scale={3} min={0} /> },
                  { key: 'unit_price', header: t('shared.devui.amount'), numeric: true, width: 140, cell: (i, bare) => <MoneyField name={`lines.${i}.unit_price`} label={`${t('shared.devui.amount')} (${i + 1})`} bare={bare} /> },
                ]}
                newLine={() => ({ description: '', quantity: '1', unit_price: 0 })}
                addLabel={t('shared.devui.add')}
                lineLabel={(n) => `# ${n}`}
                removeLabel={(n) => `${t('shared.devui.remove')} ${n}`}
                minLines={1}
                footer={<LinesTotals note={t('shared.devui.totals_note')} rows={[[t('shared.devui.subtotal'), formatNumber(3000000)], [t('shared.devui.grand_total'), formatNumber(3240000)]]} />}
              />
            </FormRow>
          </FormSection>
          <FormActions submitLabel={t('shared.devui.primary')} />
        </Form>
      </Section>
    </Stack>
  )
}
