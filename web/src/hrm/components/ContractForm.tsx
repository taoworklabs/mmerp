import { ActionIcon, Button, Group, Text, Tooltip } from '@mantine/core'
import { IconCash, IconFileText, IconPlus, IconTrash } from '@tabler/icons-react'
import { useFieldArray, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type Contract, type ContractFields } from '@/shared/api/hrm'
import { useDocumentMutation } from '@/shared/document'
import { CheckboxField, DateField, Form, FormActions, FormSection, InputRow, MoneyField, SelectField, TextField } from '@/shared/ui/form'
import { icon } from '@/shared/ui/theme'
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

const fieldOfCode: Record<string, keyof Values> = {
  contract_end_before_start: 'end_date',
  contract_end_required: 'end_date',
  contract_end_not_allowed: 'end_date',
  appendix_outside_contract: 'start_date',
  contract_type_inactive: 'contract_type_id',
  invalid_contract_terms: 'salary',
}

type Props = {
  contract?: Contract
  // A new contract: the employee, and for an appendix its original (whose terms it starts from).
  employeeId?: number
  parent?: Contract
  onCreated?: (id: number) => Promise<void>
}

// ContractForm creates a contract or an appendix, or edits a draft when its allowed_actions
// has edit; otherwise it is read-only. Terms are null when the user may not see salaries.
export function ContractForm({ contract, employeeId, parent, onCreated }: Props) {
  const { t } = useTranslation()
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
  const lines = useFieldArray({ control: form.control, name: 'lines' })
  const kinds = form.watch('lines')
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
      return
    }
    const { id } = await unwrap(api.POST('/hrm/contracts', { body: { employee_id: employeeId ?? 0, parent_id: parent?.id, ...body } }))
    await onCreated?.(id)
  }

  return (
    <Form form={form} onSubmit={save} fieldOf={(err) => fieldOfCode[err.code]}>
      <FormSection
        title={t(appendix ? 'hrm.contract.section_appendix' : 'hrm.contract.section')}
        description={t(appendix ? 'hrm.contract.section_appendix_hint' : 'hrm.contract.section_hint')}
        icon={IconFileText}
      >
        <SelectField name="contract_type_id" label={t('hrm.contract.kind')} required readOnly={ro || appendix} data={options} />
        <DateField name="start_date" label={t('hrm.contract.start_date')} required readOnly={ro} />
        {!appendix && (fixedTerm || (ro && contract?.end_date)) && (
          <DateField name="end_date" label={t('hrm.contract.end_date')} description={t('hrm.contract.end_date_required_hint')} required readOnly={ro} />
        )}
      </FormSection>
      <FormSection title={t('hrm.contract.section_terms')} description={t('hrm.contract.section_terms_hint')} icon={IconCash}>
        {terms === null && contract ? (
          <Text size="sm" c="dimmed">
            {t('hrm.contract.terms_hidden')}
          </Text>
        ) : (
          <>
            <MoneyField name="salary" label={t('hrm.contract.salary')} required readOnly={ro} />
            <span />
            {lines.fields.map((line, i) => (
              <LineFields key={line.id} index={i} support={kinds[i]?.kind === 'support'} readOnly={ro} onRemove={() => lines.remove(i)} />
            ))}
            {!ro && (
              <Group>
                <Button size="xs" variant="light" leftSection={<IconPlus {...icon.text} />} onClick={() => lines.append({ kind: 'allowance', name: '', amount: null, taxable: true })}>
                  {t('hrm.contract.add_line')}
                </Button>
              </Group>
            )}
          </>
        )}
      </FormSection>
      {editable && <FormActions submitLabel={contract ? t('hrm.common.save') : t('hrm.common.create_draft')} />}
      {mutation.dialog}
    </Form>
  )
}

// LineFields is one allowance or support of the terms; only supports may be tax-free.
function LineFields({ index, support, readOnly, onRemove }: { index: number; support: boolean; readOnly: boolean; onRemove: () => void }) {
  const { t } = useTranslation()
  const n = index + 1
  return (
    <>
      <SelectField
        name={`lines.${index}.kind`}
        label={t('hrm.contract.line_kind', { n })}
        required
        readOnly={readOnly}
        data={[
          { value: 'allowance', label: t('hrm.contract.line.allowance') },
          { value: 'support', label: t('hrm.contract.line.support') },
        ]}
        description={t(support ? 'hrm.contract.line.support_hint' : 'hrm.contract.line.allowance_hint')}
      />
      <TextField name={`lines.${index}.name`} label={t('hrm.contract.line_name', { n })} required maxLength={200} readOnly={readOnly} />
      <MoneyField name={`lines.${index}.amount`} label={t('hrm.contract.line_amount', { n })} required readOnly={readOnly} />
      <InputRow>
        {support ? <CheckboxField name={`lines.${index}.taxable`} label={t('hrm.contract.line.taxable')} readOnly={readOnly} /> : <span />}
        {!readOnly && (
          <Tooltip label={t('hrm.contract.remove_line', { n })}>
            <ActionIcon variant="subtle" color="danger" size="lg" aria-label={t('hrm.contract.remove_line', { n })} onClick={onRemove}>
              <IconTrash {...icon.button} />
            </ActionIcon>
          </Tooltip>
        )}
      </InputRow>
    </>
  )
}
