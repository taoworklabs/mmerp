import { IconFileText, IconListDetails } from '@tabler/icons-react'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type Closing, type ClosingLine } from '@/shared/api/accounting'
import { useDocumentMutation } from '@/shared/document'
import { formatNumber } from '@/shared/i18n'
import { DateField, Form, FormActions, FormRow, FormSection, LegalEntityField, LinesTable, SelectField, TextField, useLegalEntities, useOrgUnits, type ErrorFields, type LineColumn } from '@/shared/ui/form'
import { Notice } from '@/shared/ui/Notice'
import { accountLabel } from '../hooks/useAccounting'
import { closingDocType } from '../keys'

type Values = { date: string | null; legal_entity_id: string | null; description: string }

const errorFields: ErrorFields<Values> = {
  journal_entry_description_required: 'description',
  accounting_not_started: 'date',
}

type Props = {
  doc?: Closing
  // A new closing: called with its id once created.
  onCreated?: (id: number) => Promise<void>
}

// ClosingForm creates a closing entry or edits a draft's date and legal entity; its lines are
// computed by the server: a preview at the date while not posted, the entry once posted.
export function ClosingForm({ doc, onCreated }: Props) {
  const { t } = useTranslation()
  const mutation = useDocumentMutation(closingDocType, doc?.id ?? 0, doc?.version ?? 0)
  const [requestId] = useState(() => crypto.randomUUID())
  const ro = doc ? !doc.allowed_actions.includes('edit') : false
  const entities = useLegalEntities({}).options
  const units = useOrgUnits({}).data ?? []
  const form = useForm<Values>({
    mode: 'onBlur',
    defaultValues: { date: doc?.date ?? null, legal_entity_id: doc ? String(doc.legal_entity_id) : null, description: doc?.description ?? '' },
  })

  async function save(v: Values) {
    const body = { date: v.date ?? '', legal_entity_id: Number(v.legal_entity_id), description: v.description.trim() }
    if (doc) {
      await mutation.run((version) => unwrap(api.PUT('/accounting/closings/{id}', { params: { path: { id: doc.id } }, body: { version, ...body } })))
      return
    }
    const { id } = await unwrap(api.POST('/accounting/closings', { body: { request_id: requestId, ...body } }))
    await onCreated?.(id)
  }

  const columns: LineColumn<ClosingLine>[] = [
    { key: 'account', header: t('accounting.line.account_col'), value: (l) => accountLabel({ code: l.account_code, name: l.account_name }) },
    { key: 'org_unit', header: t('accounting.line.org_unit_col'), width: 200, value: (l) => units.find((u) => u.id === l.org_unit_id)?.name ?? '' },
    { key: 'debit', header: t('accounting.side.debit_money'), numeric: true, width: 160, value: (l) => formatNumber(l.debit) },
    { key: 'credit', header: t('accounting.side.credit_money'), numeric: true, width: 160, value: (l) => formatNumber(l.credit) },
  ]
  const posted = doc?.status === 'posted' || doc?.status === 'cancelled'

  return (
    <Form form={form} onSubmit={save} errorFields={errorFields}>
      <FormSection title={t('accounting.closing.section.general')} description={t('accounting.closing.section.general_hint')} icon={IconFileText}>
        <DateField name="date" label={t('accounting.closing.date')} description={ro ? undefined : t('accounting.closing.date_hint')} required readOnly={ro} />
        {ro ? (
          <SelectField name="legal_entity_id" label={t('accounting.common.legal_entity')} data={entities} readOnly />
        ) : (
          <LegalEntityField name="legal_entity_id" label={t('accounting.common.legal_entity')} product="accounting" permission="accounting.entry.edit" required />
        )}
        <TextField name="description" label={t('accounting.entry.description')} required maxLength={500} wide readOnly={ro} />
      </FormSection>
      {doc && (
        <FormSection
          title={t('accounting.closing.section.lines')}
          description={t(posted ? 'accounting.closing.section.lines_hint_posted' : 'accounting.closing.section.lines_hint')}
          icon={IconListDetails}
          wide
        >
          <FormRow>
            {doc.lines.length > 0 ? (
              <LinesTable readOnly label={t('accounting.closing.section.lines')} columns={columns} lines={doc.lines} lineLabel={(n) => t('accounting.line.title', { n })} />
            ) : (
              <Notice tone="info">{t('accounting.closing.no_lines')}</Notice>
            )}
          </FormRow>
        </FormSection>
      )}
      {!ro && <FormActions submitLabel={t(doc ? 'accounting.common.save' : 'accounting.common.create_draft')} cancelTo={doc ? undefined : '/accounting/closings'} />}
      {mutation.dialog}
    </Form>
  )
}
