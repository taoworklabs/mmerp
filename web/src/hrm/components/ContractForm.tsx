import { Text } from '@mmerp/ui'
import { IconCash, IconFileText, IconListDetails } from '@tabler/icons-react'
import { useState } from 'react'
import type { TFunction } from 'i18next'
import { useForm, useWatch } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type Contract, type ContractFields } from '@/shared/api/hrm'
import { useDocumentMutation } from '@/shared/document'
import { notifySuccess } from '@/shared/ui/notify'
import { formatNumber } from '@/shared/i18n'
import { CheckboxField, DateField, Form, FormActions, FormFields, FormRow, FormSection, LinesTable, MoneyField, SelectField, TextField, type LineColumn, type ErrorFields } from '@/shared/ui/form'
import { useContractTypes } from '../hooks/useContract'
import { contractDocType } from '../keys'

type Line = { kind: 'allowance' | 'support'; name: string; amount: number | null; taxable: boolean }

type Values = {
  contract_type_id: string | null
  start_date: string | null
  end_date: string | null
  salary: number | null
  lines: Line[]
}

const errorFields: ErrorFields<Values> = {
  contract_end_before_start: 'end_date',
  contract_end_required: 'end_date',
  contract_end_not_allowed: 'end_date',
  appendix_outside_contract: 'start_date',
  contract_type_inactive: 'contract_type_id',
  invalid_contract_terms: 'salary',
  contract_line_name_required: 'lines.*.name',
  invalid_contract_line_amount: 'lines.*.amount',
}

type Props = {
  contract?: Contract
  // A new contract: the employee, and for an appendix its original (whose terms it starts from).
  employeeId?: number
  parent?: Contract
  onCreated?: (id: number) => Promise<void>
  // Editing one card of a draft on the contract's page: its part's fields only (the form still holds
  // the whole contract, which saves as a whole), Cancel, and after saving, close the form.
  part?: ContractPart
  onClose?: () => void
}

export type ContractPart = 'main' | 'terms' | 'lines'

// ContractForm creates a contract or an appendix, or edits a draft when its allowed_actions
// has edit; otherwise it is read-only. Terms are null when the user may not see salaries.
export function ContractForm({ contract, employeeId, parent, onCreated, part, onClose }: Props) {
  const { t } = useTranslation()
  // One id per opened form: sending it twice returns the document made the first time.
  const [requestId] = useState(() => crypto.randomUUID())
  const types = useContractTypes()
  const mutation = useDocumentMutation(contractDocType, contract?.id ?? 0, contract?.version ?? 0)
  const appendix = contract ? contract.parent_id !== null && contract.parent_id !== undefined : !!parent
  const terms = contract ? contract.terms : (parent?.terms ?? null)
  const form = useForm<Values>({
    mode: 'onBlur',
    defaultValues: {
      contract_type_id: contract ? String(contract.contract_type_id) : parent ? String(parent.contract_type_id) : null,
      start_date: contract?.start_date ?? null,
      end_date: contract?.end_date ?? null,
      salary: terms?.salary ?? null,
      lines: terms?.lines.map((l) => ({ ...l })) ?? [],
    },
  })
  const editable = contract ? contract.allowed_actions.includes('edit') : true
  const ro = !editable
  // Inactive kinds stay visible on contracts that already use them.
  const current = form.watch('contract_type_id')
  // The kind decides whether an original has an end date; the API checks the same.
  const fixedTerm = types.data?.find((x) => String(x.id) === current)?.fixed_term ?? false
  const options = (types.data ?? []).filter((x) => x.active || String(x.id) === current).map((x) => ({ value: String(x.id), label: x.name }))

  async function save(v: Values) {
    const body: ContractFields = {
      contract_type_id: Number(v.contract_type_id),
      start_date: v.start_date ?? '',
      end_date: appendix || !fixedTerm ? null : v.end_date,
      terms: {
        salary: v.salary ?? 0,
        lines: v.lines.map((l) => ({ kind: l.kind, name: l.name.trim(), amount: l.amount ?? 0, taxable: l.kind === 'allowance' || l.taxable })),
      },
    }
    if (contract) {
      await mutation.run((version) => unwrap(api.PUT('/hrm/contracts/{id}', { params: { path: { id: contract.id } }, body: { version, ...body } })))
      notifySuccess(t('hrm.contract.saved'))
      onClose?.()
      return
    }
    const { id } = await unwrap(api.POST('/hrm/contracts', { body: { request_id: requestId, employee_id: employeeId ?? 0, parent_id: parent?.id, ...body } }))
    await onCreated?.(id)
  }

  const mainInputs = (
    <>
      <SelectField name="contract_type_id" label={t('hrm.contract.kind')} required readOnly={ro || appendix} data={options} />
      <DateField name="start_date" label={t('hrm.contract.start_date')} required readOnly={ro} />
      {!appendix && (fixedTerm || (ro && contract?.end_date)) && (
        <DateField name="end_date" label={t('hrm.contract.end_date')} description={t('hrm.contract.end_date_required_hint')} required readOnly={ro} />
      )}
    </>
  )
  const linesTable = (
    <LinesTable
      name="lines"
      label={t('hrm.contract.section_lines')}
      columns={lineColumns(t)}
      newLine={() => ({ kind: 'allowance', name: '', amount: null, taxable: true })}
      addLabel={t('hrm.contract.add_line')}
      lineLabel={(n) => t('hrm.contract.line_title', { n })}
      removeLabel={(n) => t('hrm.contract.remove_line', { n })}
    />
  )

  if (part && onClose)
    return (
      <Form form={form} onSubmit={save} errorFields={errorFields}>
        {part === 'lines' ? (
          linesTable
        ) : (
          <FormFields>{part === 'main' ? mainInputs : <MoneyField name="salary" label={t('hrm.contract.salary')} required />}</FormFields>
        )}
        <FormActions submitLabel={t('hrm.common.save')} onCancel={onClose} cancelLabel={t('hrm.common.cancel')} />
        {mutation.dialog}
      </Form>
    )
  return (
    <Form form={form} onSubmit={save} errorFields={errorFields}>
      <FormSection
        title={t(appendix ? 'hrm.contract.section_appendix' : 'hrm.contract.section')}
        description={t(appendix ? 'hrm.contract.section_appendix_hint' : 'hrm.contract.section_hint')}
        icon={IconFileText}
      >
        {mainInputs}
      </FormSection>
      <FormSection title={t('hrm.contract.section_terms')} description={t('hrm.contract.section_terms_hint')} icon={IconCash}>
        {terms === null && contract ? (
          <Text size="sm" tone="dimmed">
            {t('hrm.contract.terms_hidden')}
          </Text>
        ) : (
          <MoneyField name="salary" label={t('hrm.contract.salary')} required readOnly={ro} />
        )}
      </FormSection>
      {!(terms === null && contract) && (
        <FormSection title={t('hrm.contract.section_lines')} description={t('hrm.contract.section_lines_hint')} icon={IconListDetails} wide>
          <FormRow>
            {ro ? <SavedLines lines={terms?.lines ?? []} /> : linesTable}
          </FormRow>
        </FormSection>
      )}
      {editable && <FormActions submitLabel={contract ? t('hrm.common.save') : t('hrm.common.create_draft')} cancelTo={contract ? undefined : `/hrm/employees/${employeeId}?tab=contracts`} />}
      {mutation.dialog}
    </Form>
  )
}

// lineColumns are the allowances and supports of the terms, one row each; only a support may be tax-free.
function lineColumns(t: TFunction): LineColumn<Line>[] {
  return [
    {
      key: 'kind',
      header: t('hrm.contract.line_kind_col'),
      required: true,
      width: 168,
      value: (l) => t(`hrm.contract.line.${l.kind}`),
      cell: (i, bare) => (
        <SelectField
          name={`lines.${i}.kind`}
          label={t('hrm.contract.line_kind', { n: i + 1 })}
          bare={bare}
          required
          data={[
            { value: 'allowance', label: t('hrm.contract.line.allowance') },
            { value: 'support', label: t('hrm.contract.line.support') },
          ]}
        />
      ),
    },
    {
      key: 'name',
      header: t('hrm.contract.line_name_col'),
      required: true,
      width: 144,
      value: (l) => l.name,
      cell: (i, bare) => <TextField name={`lines.${i}.name`} label={t('hrm.contract.line_name', { n: i + 1 })} bare={bare} required maxLength={200} />,
    },
    {
      key: 'amount',
      header: t('hrm.contract.line_amount_col'),
      numeric: true,
      required: true,
      width: 128,
      value: (l) => (l.amount == null ? '' : formatNumber(l.amount)),
      cell: (i, bare) => <MoneyField name={`lines.${i}.amount`} label={t('hrm.contract.line_amount', { n: i + 1 })} bare={bare} required />,
    },
    {
      key: 'taxable',
      header: t('hrm.contract.line_taxable_col'),
      width: 72,
      value: (l) => (l.kind === 'support' ? t(l.taxable ? 'hrm.common.yes' : 'hrm.common.no') : ''),
      cell: (i, bare) => <TaxableCell index={i} bare={bare} />,
    },
  ]
}

// TaxableCell asks only of a support; an allowance is always taxable.
function TaxableCell({ index, bare }: { index: number; bare: boolean }) {
  const { t } = useTranslation()
  const kind = useWatch({ name: `lines.${index}.kind` }) as Line['kind']
  if (kind !== 'support') return null
  return <CheckboxField name={`lines.${index}.taxable`} label={t('hrm.contract.line.taxable')} bare={bare} />
}

// SavedLines shows the allowances and supports of a contract that is not edited.
export function SavedLines({ lines }: { lines: Line[] }) {
  const { t } = useTranslation()
  if (lines.length === 0)
    return (
      <Text size="sm" tone="dimmed">
        {t('hrm.contract.no_lines')}
      </Text>
    )
  return <LinesTable readOnly label={t('hrm.contract.section_lines')} columns={lineColumns(t)} lines={lines} lineLabel={(n) => t('hrm.contract.line_title', { n })} />
}
