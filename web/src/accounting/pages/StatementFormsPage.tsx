import { Button } from '@mmerp/ui'
import { IconFileSpreadsheet, IconPlus, IconTrash } from '@tabler/icons-react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router'
import { api, unwrap, type StatementRow } from '@/shared/api/accounting'
import { useCan, useProductOn } from '@/shared/auth/me'
import { type Column } from '@/shared/ui/DataTable'
import { CheckboxField, FilterChip, Form, FormActions, TextField } from '@/shared/ui/form'
import { useConfirm } from '@/shared/ui/confirm'
import { notifySuccess } from '@/shared/ui/notify'
import { FormModal, ListPage } from '@/shared/ui/page'
import { icon } from '@/shared/ui/theme'
import { asList } from '../hooks/useAccounting'
import { accountingKeys, type StatementFormName } from '../keys'

const forms: StatementFormName[] = ['balance_sheet', 'income_statement']

// StatementFormsPage: the rows of a financial statement form, each a code, a label and the formula
// computing it from the ledger. A row is edited in a modal; the whole form is saved at once.
export function StatementFormsPage() {
  const { t } = useTranslation()
  const writable = useCan('accounting.manage') && useProductOn('accounting')
  const [search, setSearch] = useSearchParams()
  const form = forms.find((f) => f === search.get('form')) ?? 'balance_sheet'
  const [editing, setEditing] = useState<number | 'new' | null>(null)
  const rows = useQuery({
    queryKey: accountingKeys.form(form),
    queryFn: async () => (await unwrap(api.GET('/accounting/statement-forms/{form}', { params: { path: { form } } }))).rows,
  })
  const columns: Column<StatementRow>[] = [
    { key: 'code', header: t('accounting.statement.code'), role: 'title', kind: 'code', render: (r) => r.code },
    { key: 'label', header: t('accounting.statement.label'), role: 'meta', render: (r) => r.label },
    { key: 'formula', header: t('accounting.statement_form.formula'), role: 'meta', kind: 'code', render: (r) => r.formula },
  ]
  return (
    <ListPage
      title={t('accounting.statement_form.title')}
      description={t('accounting.statement_form.description')}
      action={
        writable && (
          <Button leftSection={<IconPlus {...icon.button} />} onClick={() => setEditing('new')}>
            {t('accounting.statement_form.add')}
          </Button>
        )
      }
      filters={
        <FilterChip
          label={t('accounting.statement_form.form')}
          icon={IconFileSpreadsheet}
          clearable={false}
          data={forms.map((f) => ({ value: f, label: t(`accounting.statement.${f}.title`) }))}
          value={form}
          onChange={(v) => setSearch(new URLSearchParams({ form: v ?? 'balance_sheet' }))}
        />
      }
      list={{
        query: asList(rows, rows.data),
        table: { label: t('accounting.statement_form.title'), columns, rowKey: (r) => r.code, onRowClick: writable ? (r) => setEditing(rows.data?.indexOf(r) ?? null) : undefined },
        empty: { title: t('accounting.statement_form.empty'), description: t('accounting.statement_form.empty_description') },
        noResults: t('accounting.common.no_results'),
      }}
    >
      {editing !== null && rows.data && <RowModal form={form} rows={rows.data} index={editing} onClose={() => setEditing(null)} />}
    </ListPage>
  )
}

type Values = { code: string; label: string; formula: string; bold: boolean }

// RowModal edits one row, adds one at the end or removes one, and saves the whole form.
function RowModal({ form: name, rows, index, onClose }: { form: StatementFormName; rows: StatementRow[]; index: number | 'new'; onClose: () => void }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const row = index === 'new' ? undefined : rows[index]
  const form = useForm<Values>({ mode: 'onBlur', defaultValues: { code: row?.code ?? '', label: row?.label ?? '', formula: row?.formula ?? '', bold: row?.bold ?? false } })
  const [confirm, dialog] = useConfirm()
  async function remove() {
    const ok = await confirm({
      title: t('accounting.statement_form.remove'),
      message: t('accounting.statement_form.remove_confirm', { code: row?.code }),
      confirmLabel: t('accounting.statement_form.remove'),
      danger: true,
    })
    if (ok) await put(rows.filter((_, i) => i !== index), 'accounting.statement_form.removed')
  }
  async function put(next: StatementRow[], done: string) {
    await unwrap(api.PUT('/accounting/statement-forms/{form}', { params: { path: { form: name } }, body: { rows: next } }))
    await qc.invalidateQueries({ queryKey: ['accounting'] })
    notifySuccess(t(done))
    onClose()
  }
  const save = (v: Values) => put(index === 'new' ? [...rows, v] : rows.map((r, i) => (i === index ? v : r)), 'accounting.statement_form.saved')
  return (
    <FormModal open title={row ? row.code : t('accounting.statement_form.add')} onOpenChange={onClose} wide>
      <Form
        form={form}
        onSubmit={save}
        errorFields={{ statement_formula_invalid: 'formula', statement_formula_cycle: 'formula', statement_row_unknown: 'formula', statement_row_duplicate: 'code' }}
      >
        <TextField name="code" label={t('accounting.statement.code')} required maxLength={10} />
        <TextField name="label" label={t('accounting.statement.label')} required maxLength={300} />
        <TextField name="formula" label={t('accounting.statement_form.formula')} description={t('accounting.statement_form.formula_hint')} maxLength={1000} />
        <CheckboxField name="bold" label={t('accounting.statement_form.bold')} />
        {row && (
          <Button variant="subtle" tone="danger" leftSection={<IconTrash {...icon.button} />} onClick={() => void remove()}>
            {t('accounting.statement_form.remove')}
          </Button>
        )}
        <FormActions submitLabel={t('accounting.common.save')} onCancel={onClose} cancelLabel={t('accounting.common.cancel')} />
      </Form>
      {dialog}
    </FormModal>
  )
}
