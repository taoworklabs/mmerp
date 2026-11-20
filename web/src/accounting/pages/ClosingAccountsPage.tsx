import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, ApiError, unwrap } from '@/shared/api/accounting'
import { useCan, useProductOn } from '@/shared/auth/me'
import { type Column } from '@/shared/ui/DataTable'
import { Form, FormActions, SelectField } from '@/shared/ui/form'
import { notifySuccess } from '@/shared/ui/notify'
import { FormModal, ListPage } from '@/shared/ui/page'
import { accountLabel, accountOptions, asList, useAccounts } from '../hooks/useAccounting'
import { accountingKeys } from '../keys'

type Row = { key: 'result' | 'retained'; account: string }

// ClosingAccountsPage: the two accounts closing entries post to, the result account (911) and
// retained earnings (4212); both are edited in one modal.
export function ClosingAccountsPage() {
  const { t } = useTranslation()
  const writable = useCan('accounting.manage') && useProductOn('accounting')
  const [editing, setEditing] = useState(false)
  const current = useQuery({
    queryKey: accountingKeys.closingAccounts(),
    // Not set yet answers 404: the rows show as unset.
    queryFn: () =>
      unwrap(api.GET('/accounting/closing-accounts')).catch((e: unknown) => {
        if (e instanceof ApiError && e.status === 404) return null
        throw e
      }),
  })
  const c = current.data
  const rows: Row[] | undefined =
    current.data === undefined
      ? undefined
      : [
          { key: 'result', account: c ? accountLabel({ code: c.result_code, name: c.result_name }) : '' },
          { key: 'retained', account: c ? accountLabel({ code: c.retained_code, name: c.retained_name }) : '' },
        ]
  const columns: Column<Row>[] = [
    { key: 'role', header: t('accounting.closing_accounts.role'), role: 'title', render: (r) => t(`accounting.closing_accounts.${r.key}`) },
    { key: 'account', header: t('accounting.rule.account'), role: 'meta', render: (r) => r.account },
  ]
  return (
    <ListPage
      title={t('accounting.closing_accounts.title')}
      description={t('accounting.closing_accounts.description')}
      list={{
        query: asList(current, rows),
        table: { label: t('accounting.closing_accounts.title'), columns, rowKey: (r) => r.key, onRowClick: writable ? () => setEditing(true) : undefined },
        empty: { title: t('accounting.closing_accounts.title') },
        noResults: t('accounting.common.no_results'),
      }}
    >
      {editing && <ClosingAccountsModal result={c?.result_account_id} retained={c?.retained_account_id} onClose={() => setEditing(false)} />}
    </ListPage>
  )
}

type Values = { result: string | null; retained: string | null }

function ClosingAccountsModal({ result, retained, onClose }: { result?: number; retained?: number; onClose: () => void }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const accounts = useAccounts().data
  const options = accountOptions(accounts, (a) => (a.active && a.postable) || a.id === result || a.id === retained)
  const form = useForm<Values>({ mode: 'onBlur', defaultValues: { result: result ? String(result) : null, retained: retained ? String(retained) : null } })
  async function save(v: Values) {
    await unwrap(api.PUT('/accounting/closing-accounts', { body: { result_account_id: Number(v.result), retained_account_id: Number(v.retained) } }))
    await qc.invalidateQueries({ queryKey: accountingKeys.closingAccounts() })
    notifySuccess(t('accounting.closing_accounts.saved'))
    onClose()
  }
  return (
    <FormModal open title={t('accounting.closing_accounts.title')} onOpenChange={onClose}>
      <Form form={form} onSubmit={save} errorFields={{ account_not_postable: 'result' }}>
        <SelectField name="result" label={t('accounting.closing_accounts.result')} required data={options} />
        <SelectField name="retained" label={t('accounting.closing_accounts.retained')} required data={options} />
        <FormActions submitLabel={t('accounting.common.save')} onCancel={onClose} cancelLabel={t('accounting.common.cancel')} />
      </Form>
    </FormModal>
  )
}
