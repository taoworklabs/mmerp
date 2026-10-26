import { Button, FieldList, Group, Stack, Text } from '@mmerp/ui'
import { IconCircleDot, IconUsersGroup } from '@tabler/icons-react'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { formatDecimal, formatNumber } from '@/shared/i18n'
import {
  CheckboxField,
  EditablePanel,
  CodeField,
  DateField,
  DecimalField,
  FileField,
  FilterChip,
  Form,
  FormActions,
  FormFields,
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
import { DocumentStatusFilter } from '@/shared/document'
import { MissingValue } from '@/shared/ui/MissingValue'
import { InlineEditField } from '@/shared/ui/InlineEditField'
import { ModalFooter } from '@/shared/ui/ModalFooter'
import { FormModal } from '@/shared/ui/page'
import { EmptyState } from '@/shared/ui/states'
import { Case, rows } from './Demo'
import { DocPage } from './DocPage'

// The main props of each component; data, not interface text.
const code = {
  form: `const form = useForm({ defaultValues })
<Form form={form} onSubmit={save}>
  <FormSection title>  // or <FormFields> inside a card
    <TextField name label required? description? />  // DateField, MoneyField, SelectField…
  </FormSection>
  <FormActions submitLabel />
</Form>`,
  filters: `import { DocumentStatusFilter } from '@/shared/document'
// values live on the URL: useListParams from '@/shared/url/listParams'`,
  inline: `<InlineEditField label value onSave validate? readOnly? />`,
  lines: `<LinesTable name label columns newLine addLabel lineLabel removeLabel minLines? footer? />
<LinesTable readOnly label lines columns lineLabel footer? />
import { LinesTotals } from '@/shared/ui/form'`,
}

// Searched in the browser: the catalogue has no server.
const people = [
  { id: 1, code: 'NV001', name: 'Nguyễn Văn An' },
  { id: 2, code: 'NV002', name: 'Trần Thị Bình' },
  { id: 3, code: 'NV003', name: 'Lê Minh Châu' },
]
const searchPeople = async (q: string) => people.filter((p) => `${p.code} ${p.name}`.toLowerCase().includes(q.toLowerCase()))

export function FormDoc() {
  const { t } = useTranslation()
  // A card's fields edited in place: no section title, two to a row.
  const pair = useForm({ defaultValues: { name: 'Nguyễn Văn An', code: 'NV001', date: null, salary: undefined } })
  const form = useForm({ defaultValues: { code: '', password: '', unit: null, entity: null, person: '3', date: null, month: '2026-03', file: null, choice: null, many: ['a'], masked: undefined, open: undefined, denied: undefined, days: '2.5', flag: true } })
  return (
    <DocPage
      path="mmerp/form"
      example={
        <Case label={t('shared.devui.case_form_fields')}>
          <Form form={pair} onSubmit={async () => {}}>
            <FormFields>
              <TextField name="name" label={t('shared.devui.name')} required />
              <TextField name="code" label={t('shared.devui.code')} />
              <DateField name="date" label={t('shared.devui.date')} />
              <MoneyField name="salary" label={t('shared.devui.money')} />
            </FormFields>
          </Form>
        </Case>
      }
      variants={
        <>
          <Case label={t('shared.devui.case_form_section')}>
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
                <MultiSelectField
                  name="many"
                  label={t('shared.devui.multi_choice')}
                  data={[
                    { value: 'a', label: 'A' },
                    { value: 'b', label: 'B' },
                    { value: 'c', label: 'C' },
                  ]}
                />
                <DecimalField name="days" label={t('shared.devui.decimal')} scale={1} step={0.5} />
                <MoneyField name="salary" label={t('shared.devui.money')} />
                <InputRow>
                  <CheckboxField name="beside" label={t('shared.devui.checkbox')} />
                </InputRow>
                <CheckboxField name="flag" label={t('shared.devui.checkbox')} description={t('shared.devui.hint')} />
                <FormRow>
                  <Text size="sm" tone="dimmed">
                    {t('shared.devui.hint')}
                  </Text>
                </FormRow>
                <SensitiveField name="masked" label={t('shared.devui.sensitive')} present canView canEdit reveal={async () => '079123456789'} />
                <SensitiveField name="open" label={t('shared.devui.sensitive')} present={false} canView canEdit reveal={async () => null} />
                <SensitiveField name="denied" label={t('shared.devui.sensitive')} present canView={false} canEdit={false} reveal={async () => null} />
              </FormSection>
              <FormActions submitLabel={t('shared.devui.primary')} />
            </Form>
          </Case>
        </>
      }
      code={code.form}
    />
  )
}

export function FiltersDoc() {
  const { t } = useTranslation()
  const [year, setYear] = useState(2026)
  const [choice, setChoice] = useState<string | null>('positive')
  const [month, setMonth] = useState<string | null>('2026-08')
  const [q, setQ] = useState('')
  const [unit, setUnit] = useState<string | null>(null)
  const [status, setStatus] = useState<string | null>(null)
  const [entity, setEntity] = useState('')
  const entities = useLegalEntities({}, entity)
  const choices = [
    { value: 'positive', label: t('shared.devui.status_positive') },
    { value: 'neutral', label: t('shared.devui.status_neutral') },
    { value: 'long', label: t('shared.devui.long_value') },
  ]
  return (
    <DocPage
      path="mmerp/filters"
      example={
        <Case label={t('shared.devui.case_search')}>
          <Group gap="xs">
            <SearchInput label={t('shared.devui.name')} value={q} onSearch={setQ} />
          </Group>
        </Case>
      }
      variants={
        <>
          <Case label={t('shared.devui.case_filter_chip')}>
            <Group gap="xs">
              <FilterChip label={t('shared.devui.choice')} icon={IconCircleDot} data={choices} value={null} onChange={() => {}} searchable />
              <FilterChip label={t('shared.devui.choice')} icon={IconCircleDot} data={choices} value={choice} onChange={setChoice} />
              <FilterChip label={t('shared.devui.choice')} icon={IconCircleDot} data={choices} value="long" onChange={() => {}} />
            </Group>
          </Case>
          <Case label={t('shared.devui.case_filter_status')}>
            <Group gap="xs">
              <DocumentStatusFilter docType="shared" value={status} onChange={setStatus} />
            </Group>
          </Case>
          <Case label={t('shared.devui.case_filter_org')}>
            <Group gap="xs">
              <OrgUnitFilter label={t('shared.devui.org_unit')} value={unit} onChange={setUnit} />
              <LegalEntityFilter label={t('shared.devui.legal_entity')} options={entities.options} value={entities.chosen} onChange={setEntity} />
            </Group>
          </Case>
          <Case label={t('shared.devui.case_filter_period')}>
            <Group gap="xs">
              <YearFilter label={t('shared.devui.year')} value={year} years={[2025, 2026, 2027]} onChange={setYear} />
              <MonthFilter label={t('shared.devui.month')} value={month} onChange={setMonth} />
            </Group>
          </Case>
        </>
      }
      code={code.filters}
    />
  )
}

export function InlineEditFieldDoc() {
  const { t } = useTranslation()
  const [name, setName] = useState('Công ty CP Demo')
  return (
    <DocPage
      path="mmerp/inline-edit-field"
      example={
        <Case label={t('shared.devui.case_inline_edit')}>
          <InlineEditField
            label={t('shared.devui.name')}
            value={name}
            validate={(v) => (v ? null : t('shared.form.required'))}
            onSave={async (v) => {
              await new Promise((r) => setTimeout(r, 400))
              setName(v)
            }}
          />
        </Case>
      }
      variants={
        <>
          <Case label={t('shared.devui.case_inline_read_only')}>
            <InlineEditField label={t('shared.devui.code')} value="KH001" readOnly onSave={async () => {}} />
          </Case>
        </>
      }
      code={code.inline}
    />
  )
}

export function LinesTableDoc() {
  const { t } = useTranslation()
  const form = useForm({ defaultValues: { levels: 2, lines: [{ description: 'Dịch vụ tư vấn', quantity: '2', unit_price: 1500000 }] } })
  return (
    <DocPage
      path="mmerp/lines-table"
      example={
        <Case label={t('shared.devui.case_lines_edit')}>
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
        </Case>
      }
      variants={
        <>
          <Case label={t('shared.devui.case_lines_read_only')}>
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
          </Case>
        </>
      }
      code={code.lines}
    />
  )
}

const editableCode = `<EditablePanel title icon? editable edit={(close) => <Form …><FormActions onCancel={close} /></Form>}>
  {(open) => <FieldList stacked rows={…} />}   // open() starts editing, e.g. from MissingValue
</EditablePanel>`

export function EditablePanelDoc() {
  const { t } = useTranslation()
  const form = useForm({ defaultValues: { department: 'Kế toán', email: '' } })
  return (
    <DocPage
      path="mmerp/editable-panel"
      example={
        <Case>
          <EditablePanel
            title={t('shared.devui.section_staff')}
            icon={IconUsersGroup}
            editable
            edit={(close) => (
              <Form form={form} onSubmit={async () => close()}>
                <FormFields>
                  <TextField name="department" label="Phòng ban" />
                  <TextField name="email" label="Email" />
                </FormFields>
                <FormActions submitLabel={t('shared.devui.save')} onCancel={close} cancelLabel={t('shared.form.cancel')} />
              </Form>
            )}
          >
            {(open) => (
              <FieldList
                stacked
                rows={[
                  ['Phòng ban', 'Kế toán'],
                  ['Email', <MissingValue key="e" label="Email" onAdd={open} />],
                ]}
              />
            )}
          </EditablePanel>
        </Case>
      }
      code={editableCode}
    />
  )
}

const modalCode = `<FormModal title opened onClose wide?>{form}</FormModal>
import { ModalFooter } from '@/shared/ui/ModalFooter'   // a footer for a modal without a Form`

export function FormModalDoc() {
  const { t } = useTranslation()
  const [modal, setModal] = useState<'form' | 'footer' | null>(null)
  return (
    <DocPage
      path="mmerp/form-modal"
      example={
        <Case>
          <Group>
            <Button variant="default" onClick={() => setModal('form')}>
              {t('shared.devui.open_modal')}
            </Button>
          </Group>
          <FormModal opened={modal === 'form'} onClose={() => setModal(null)} title={t('shared.devui.open_modal')}>
            <EmptyState title={t('shared.devui.empty')} />
          </FormModal>
        </Case>
      }
      variants={
        <Case label={t('shared.devui.case_modal_footer')}>
          <Group>
            <Button variant="default" onClick={() => setModal('footer')}>
              {t('shared.devui.open_modal')}
            </Button>
          </Group>
          <FormModal opened={modal === 'footer'} onClose={() => setModal(null)} title={t('shared.devui.export')}>
            <Stack gap="md">
              <Text>{`${rows[0]!.code} · ${rows[0]!.name}`}</Text>
              <ModalFooter>
                <Button onClick={() => setModal(null)}>{t('shared.devui.export')}</Button>
              </ModalFooter>
            </Stack>
          </FormModal>
        </Case>
      }
      code={modalCode}
    />
  )
}
