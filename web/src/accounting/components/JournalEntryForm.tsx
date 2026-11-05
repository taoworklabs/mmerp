import { IconFileText, IconListDetails } from '@tabler/icons-react'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type JournalEntry, type JournalEntryLine } from '@/shared/api/accounting'
import { useDocumentMutation } from '@/shared/document'
import { formatNumber } from '@/shared/i18n'
import {
  DateField,
  Form,
  FormActions,
  FormRow,
  FormSection,
  LegalEntityField,
  LinesTable,
  MoneyField,
  SelectField,
  TextField,
  orgUnitOptions,
  useLegalEntities,
  useOrgUnits,
  type ErrorFields,
  type LineColumn,
} from '@/shared/ui/form'
import { accountLabel, accountOptions, useAccounts } from '../hooks/useAccounting'
import { journalEntryDocType } from '../keys'

type LineValues = { account_id: string | null; org_unit_id: string | null; description: string; debit: number | null; credit: number | null }

type Values = { date: string | null; legal_entity_id: string | null; description: string; lines: LineValues[] }

const emptyLine: LineValues = { account_id: null, org_unit_id: null, description: '', debit: null, credit: null }

function defaults(doc?: JournalEntry): Values {
  return {
    date: doc?.date ?? null,
    legal_entity_id: doc ? String(doc.legal_entity_id) : null,
    description: doc?.description ?? '',
    lines: doc
      ? doc.lines.map((l) => ({
          account_id: String(l.account_id),
          org_unit_id: l.org_unit_id ? String(l.org_unit_id) : null,
          description: l.description ?? '',
          debit: l.debit || null,
          credit: l.credit || null,
        }))
      : [{ ...emptyLine }, { ...emptyLine }],
  }
}

const errorFields: ErrorFields<Values> = {
  journal_entry_description_required: 'description',
  accounting_not_started: 'date',
  line_side_invalid: 'lines.*.debit',
  line_amount_invalid: 'lines.*.debit',
  line_org_unit_invalid: 'lines.*.org_unit_id',
}

type Line = JournalEntry['lines'][number]

type Props = {
  doc?: JournalEntry
  // A new entry: called with its id once created.
  onCreated?: (id: number) => Promise<void>
}

// JournalEntryForm creates a manual journal entry, or edits one whose allowed_actions has edit;
// otherwise it shows it read-only. Each line books one amount on one side of one account.
export function JournalEntryForm({ doc, onCreated }: Props) {
  const { t } = useTranslation()
  const mutation = useDocumentMutation(journalEntryDocType, doc?.id ?? 0, doc?.version ?? 0)
  // One id per opened form: sending it twice returns the entry made the first time.
  const [requestId] = useState(() => crypto.randomUUID())
  const form = useForm<Values>({ mode: 'onBlur', defaultValues: defaults(doc) })
  const ro = doc ? !doc.allowed_actions.includes('edit') : false
  const accounts = useAccounts().data
  const entities = useLegalEntities({}).options
  const units = useOrgUnits({}).data ?? []
  // Only postable accounts take lines; a saved line's account still shows.
  const saved = new Set(doc?.lines.map((l) => l.account_id))
  const accountChoices = accountOptions(accounts, (a) => (a.active && a.postable) || saved.has(a.id))
  const unitChoices = orgUnitOptions(units).map(({ value, label }) => ({ value, label }))

  async function save(v: Values) {
    const body = {
      date: v.date ?? '',
      legal_entity_id: Number(v.legal_entity_id),
      description: v.description.trim(),
      lines: v.lines.map(
        (l): JournalEntryLine => ({
          account_id: Number(l.account_id),
          org_unit_id: l.org_unit_id ? Number(l.org_unit_id) : undefined,
          description: l.description.trim() || undefined,
          debit: l.debit ?? 0,
          credit: l.credit ?? 0,
        }),
      ),
    }
    if (doc) {
      await mutation.run((version) => unwrap(api.PUT('/accounting/journal-entries/{id}', { params: { path: { id: doc.id } }, body: { version, ...body } })))
      return
    }
    const { id } = await unwrap(api.POST('/accounting/journal-entries', { body: { request_id: requestId, ...body } }))
    await onCreated?.(id)
  }

  const lineColumns: LineColumn<Line>[] = [
    {
      key: 'account',
      header: t('accounting.line.account_col'),
      required: true,
      width: 240,
      value: (l) => accountLabel({ code: l.account_code, name: l.account_name }),
      cell: (i, bare) => <SelectField name={`lines.${i}.account_id`} label={t('accounting.line.account', { n: i + 1 })} bare={bare} required data={accountChoices} />,
    },
    {
      key: 'description',
      header: t('accounting.line.description_col'),
      value: (l) => l.description ?? '',
      cell: (i, bare) => <TextField name={`lines.${i}.description`} label={t('accounting.line.description', { n: i + 1 })} bare={bare} maxLength={500} />,
    },
    {
      key: 'org_unit',
      header: t('accounting.line.org_unit_col'),
      width: 200,
      value: (l) => units.find((u) => u.id === l.org_unit_id)?.name ?? '',
      cell: (i, bare) => <SelectField name={`lines.${i}.org_unit_id`} label={t('accounting.line.org_unit', { n: i + 1 })} bare={bare} clearable data={unitChoices} />,
    },
    {
      key: 'debit',
      header: t('accounting.side.debit_money'),
      numeric: true,
      width: 160,
      value: (l) => formatNumber(l.debit),
      cell: (i, bare) => <MoneyField name={`lines.${i}.debit`} label={t('accounting.line.debit', { n: i + 1 })} bare={bare} />,
    },
    {
      key: 'credit',
      header: t('accounting.side.credit_money'),
      numeric: true,
      width: 160,
      value: (l) => formatNumber(l.credit),
      cell: (i, bare) => <MoneyField name={`lines.${i}.credit`} label={t('accounting.line.credit', { n: i + 1 })} bare={bare} />,
    },
  ]

  return (
    <Form form={form} onSubmit={save} errorFields={errorFields}>
      <FormSection title={t('accounting.entry.section.general')} description={t('accounting.entry.section.general_hint')} icon={IconFileText}>
        <DateField name="date" label={t('accounting.entry.date')} required readOnly={ro} />
        {ro ? (
          <SelectField name="legal_entity_id" label={t('accounting.common.legal_entity')} data={entities} readOnly />
        ) : (
          <LegalEntityField name="legal_entity_id" label={t('accounting.common.legal_entity')} product="accounting" permission="accounting.entry.edit" required />
        )}
        <TextField name="description" label={t('accounting.entry.description')} required maxLength={500} wide readOnly={ro} />
      </FormSection>
      <FormSection
        title={t('accounting.entry.section.lines')}
        description={t(ro ? 'accounting.entry.section.lines_hint_ro' : 'accounting.entry.section.lines_hint')}
        icon={IconListDetails}
        wide
      >
        <FormRow>
          {ro && doc ? (
            <LinesTable readOnly label={t('accounting.entry.section.lines')} columns={lineColumns} lines={doc.lines} lineLabel={(n) => t('accounting.line.title', { n })} />
          ) : (
            <LinesTable
              name="lines"
              label={t('accounting.entry.section.lines')}
              columns={lineColumns}
              newLine={() => ({ ...emptyLine })}
              addLabel={t('accounting.line.add')}
              lineLabel={(n) => t('accounting.line.title', { n })}
              removeLabel={(n) => t('accounting.line.remove', { n })}
              minLines={1}
            />
          )}
        </FormRow>
      </FormSection>
      {!ro && <FormActions submitLabel={t(doc ? 'accounting.common.save' : 'accounting.common.create_draft')} cancelTo={doc ? undefined : '/accounting/journal-entries'} />}
      {mutation.dialog}
    </Form>
  )
}
