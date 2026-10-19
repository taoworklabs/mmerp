import { Group, Text } from '@mantine/core'
import { IconCircleDot } from '@tabler/icons-react'
import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { formatDecimal, formatNumber } from '@/shared/i18n'
import {
  CheckboxField,
  CodeField,
  DateField,
  DecimalField,
  FileField,
  FilterChip,
  Form,
  FormActions,
  FormRow,
  FormSection,
  InputRow,
  IntegerField,
  LegalEntityField,
  LegalEntityFilter,
  LinesTable,
  LinesTotals,
  MoneyField,
  MonthField,
  MonthFilter,
  MultiSelectField,
  OrgUnitField,
  OrgUnitFilter,
  PasswordField,
  RemoteSelectField,
  SearchInput,
  SelectField,
  SensitiveField,
  TextField,
  useLegalEntities,
  YearFilter,
} from '@/shared/ui/form'
import { InlineEditField } from '@/shared/ui/InlineEditField'
import { Demo } from './Demo'

// Searched in the browser: the catalogue has no server.
const people = [
  { id: 1, code: 'NV001', name: 'Nguyễn Văn An' },
  { id: 2, code: 'NV002', name: 'Trần Thị Bình' },
  { id: 3, code: 'NV003', name: 'Lê Minh Châu' },
]
const searchPeople = async (q: string) => people.filter((p) => `${p.code} ${p.name}`.toLowerCase().includes(q.toLowerCase()))

const tagOptions = [
  { value: 'a', label: 'A' },
  { value: 'b', label: 'B' },
  { value: 'c', label: 'C' },
]

export function Fields() {
  const { t } = useTranslation()
  const form = useForm({
    defaultValues: {
      code: '',
      password: '',
      unit: null,
      entity: null,
      person: '3',
      date: null,
      month: '2026-03',
      file: null,
      choice: null,
      tags: [],
      tags_chosen: ['a', 'b'],
      tags_required: [] as string[],
      tags_readonly: ['a'],
      masked: undefined,
      open: undefined,
      denied: undefined,
      days: '2.5',
      flag: true,
    },
  })
  // Shows the required error without the user touching the field first.
  useEffect(() => void form.trigger('tags_required'), [form])
  return (
    <Demo title={t('shared.devui.form')}>
      <Form form={form} onSubmit={async () => {}}>
        <FormSection title={t('shared.devui.form')} description={t('shared.devui.hint')}>
          <TextField name="code" label={t('shared.devui.code')} required description={t('shared.devui.hint')} />
          <CodeField name="issued_code" label={t('shared.devui.code')} existing={false} />
          <PasswordField name="password" label={t('shared.devui.password')} />
          <OrgUnitField name="unit" label={t('shared.devui.org_unit')} clearable />
          <LegalEntityField name="entity" label={t('shared.devui.legal_entity')} required />
          <RemoteSelectField name="person" label={t('shared.devui.remote')} queryKey={(q) => ['devui', 'people', q]} search={searchPeople} current={people[2]} />
          <FileField name="file" label={t('shared.devui.file')} accept=".xlsx" description={t('shared.devui.hint')} />
          <DateField name="date" label={t('shared.devui.date')} clearable short />
          <MonthField name="month" label={t('shared.devui.month')} short />
          <SelectField
            name="choice"
            label={t('shared.devui.choice')}
            short
            data={[
              { value: 'a', label: 'A' },
              { value: 'b', label: 'B' },
            ]}
          />
          <MultiSelectField name="tags" label={t('shared.devui.multi_choice')} short data={tagOptions} />
          <MultiSelectField name="tags_chosen" label={t('shared.devui.multi_choice')} short data={tagOptions} />
          <MultiSelectField name="tags_required" label={t('shared.devui.multi_choice')} short required data={tagOptions} />
          <MultiSelectField name="tags_readonly" label={t('shared.devui.multi_choice')} short readOnly data={tagOptions} />
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
    </Demo>
  )
}

export function Filters() {
  const { t } = useTranslation()
  const [year, setYear] = useState(2026)
  const [choice, setChoice] = useState<string | null>('positive')
  const [month, setMonth] = useState<string | null>('2026-08')
  const [q, setQ] = useState('')
  const [unit, setUnit] = useState<string | null>(null)
  const [entity, setEntity] = useState('')
  const entities = useLegalEntities({}, entity)
  const choices = [
    { value: 'positive', label: t('shared.devui.status_positive') },
    { value: 'neutral', label: t('shared.devui.status_neutral') },
    { value: 'long', label: t('shared.devui.long_value') },
  ]
  return (
    <Demo title={t('shared.devui.filters')}>
      {/* Unset with a search (type to see no results), set (tinted, with its clear button), a value
          too long for the chip, always set (a year), a month. */}
      <Group gap="xs">
        <SearchInput label={t('shared.devui.name')} value={q} onSearch={setQ} />
        <FilterChip label={t('shared.devui.choice')} icon={IconCircleDot} data={choices} value={null} onChange={() => {}} searchable />
        <FilterChip label={t('shared.devui.choice')} icon={IconCircleDot} data={choices} value={choice} onChange={setChoice} />
        <FilterChip label={t('shared.devui.choice')} icon={IconCircleDot} data={choices} value="long" onChange={() => {}} />
        <OrgUnitFilter label={t('shared.devui.org_unit')} value={unit} onChange={setUnit} />
        <LegalEntityFilter label={t('shared.devui.legal_entity')} options={entities.options} value={entities.chosen} onChange={setEntity} />
        <YearFilter label={t('shared.devui.year')} value={year} years={[2025, 2026, 2027]} onChange={setYear} />
        <MonthFilter label={t('shared.devui.month')} value={month} onChange={setMonth} />
      </Group>
    </Demo>
  )
}

export function Inline() {
  const { t } = useTranslation()
  const [name, setName] = useState('Công ty CP Demo')
  return (
    <Demo title={t('shared.devui.inline')} description={t('shared.devui.inline_hint')}>
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
    </Demo>
  )
}

export function Lines() {
  const { t } = useTranslation()
  const form = useForm({ defaultValues: { levels: 2, lines: [{ description: 'Dịch vụ tư vấn', quantity: '2', unit_price: 1500000 }] } })
  return (
    <Demo title={t('shared.devui.lines')} description={t('shared.devui.lines_hint')}>
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
          <FormRow>
            <LinesTable
              readOnly
              label={t('shared.devui.lines')}
              lines={[
                { description: t('shared.devui.name'), quantity: '2', unit_price: 1000000 },
                { description: t('shared.devui.name'), quantity: '1.5', unit_price: 666667 },
              ]}
              columns={[
                { key: 'description', header: t('shared.devui.name'), value: (l) => l.description },
                { key: 'quantity', header: t('shared.devui.quantity'), numeric: true, value: (l) => formatDecimal(l.quantity, 3) },
                { key: 'unit_price', header: t('shared.devui.amount'), numeric: true, value: (l) => formatNumber(l.unit_price) },
              ]}
              lineLabel={(n) => `# ${n}`}
              footer={<LinesTotals rows={[[t('shared.devui.subtotal'), formatNumber(3000000)], [t('shared.devui.grand_total'), formatNumber(3240000)]]} />}
            />
          </FormRow>
        </FormSection>
        <FormActions submitLabel={t('shared.devui.primary')} />
      </Form>
    </Demo>
  )
}
