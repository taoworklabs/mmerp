import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type Rule } from '@/shared/api/accounting'
import { useCan, useProductOn } from '@/shared/auth/me'
import { type Column } from '@/shared/ui/DataTable'
import { Form, FormActions, SelectField } from '@/shared/ui/form'
import { notifySuccess } from '@/shared/ui/notify'
import { FormModal, ListPage } from '@/shared/ui/page'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { kindLabel } from '../balance'
import { accountLabel, accountOptions, asList, useAccounts } from '../hooks/useAccounting'
import { accountingKeys } from '../keys'

// RulesPage: one row per kind of business line, booked to an account on a side; administrators set each in a modal.
export function RulesPage() {
  const { t } = useTranslation()
  const on = useProductOn('accounting')
  const writable = useCan('accounting.manage') && on
  const accounts = useAccounts()
  const [editing, setEditing] = useState<Rule | null>(null)
  const rules = useQuery({
    queryKey: accountingKeys.rules(),
    queryFn: async () => (await unwrap(api.GET('/accounting/rules'))).items,
  })
  const accountOf = (id: number | null | undefined) => accounts.data?.find((a) => a.id === id)

  const columns: Column<Rule>[] = [
    { key: 'kind', header: t('accounting.rule.kind'), role: 'title', render: (r) => kindLabel(t, r.kind) },
    {
      key: 'account',
      header: t('accounting.rule.account'),
      role: 'meta',
      render: (r) => {
        const a = accountOf(r.account_id)
        return a ? accountLabel(a) : ''
      },
    },
    { key: 'side', header: t('accounting.rule.side'), role: 'meta', render: (r) => (r.side ? t(`accounting.side.${r.side}`) : '') },
    {
      key: 'state',
      header: t('accounting.rule.state'),
      role: 'status',
      kind: 'status',
      render: (r) =>
        r.account_id ? (
          <StatusBadge tone="success">{t('accounting.rule.set')}</StatusBadge>
        ) : (
          <StatusBadge tone="neutral">{t('accounting.rule.unset')}</StatusBadge>
        ),
    },
  ]

  return (
    <ListPage
      title={t('accounting.rules.title')}
      description={t('accounting.rules.description')}
      list={{
        query: asList(rules, rules.data),
        table: {
          label: t('accounting.rules.title'),
          columns,
          rowKey: (r) => r.kind,
          onRowClick: writable ? (r) => setEditing(r) : undefined,
        },
        empty: { title: t('accounting.rules.empty') },
        noResults: t('accounting.common.no_results'),
      }}
    >
      {editing && (
        <RuleModal
          rule={editing}
          // A retired or parent account is no longer offered, but the saved one still shows.
          options={accountOptions(accounts.data, (a) => (a.active && a.postable) || a.id === editing.account_id)}
          onClose={() => setEditing(null)}
        />
      )}
    </ListPage>
  )
}

type Values = { account_id: string | null; side: string | null }

function RuleModal({ rule, options, onClose }: { rule: Rule; options: { value: string; label: string }[]; onClose: () => void }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const form = useForm<Values>({ mode: 'onBlur', defaultValues: { account_id: rule.account_id ? String(rule.account_id) : null, side: rule.side ?? null } })
  async function save(v: Values) {
    await unwrap(
      api.PUT('/accounting/rules/{kind}', { params: { path: { kind: rule.kind } }, body: { account_id: Number(v.account_id), side: v.side as 'debit' | 'credit' } }),
    )
    // Documents waiting for this rule can now be posted.
    await Promise.all([qc.invalidateQueries({ queryKey: accountingKeys.rules() }), qc.invalidateQueries({ queryKey: accountingKeys.books() })])
    notifySuccess(t('accounting.rules.saved'))
    onClose()
  }
  return (
    <FormModal open title={kindLabel(t, rule.kind)} onOpenChange={onClose}>
      <Form form={form} onSubmit={save} errorFields={{ account_not_postable: 'account_id', rule_side_invalid: 'side' }}>
        <SelectField name="account_id" label={t('accounting.rule.account')} required data={options} />
        <SelectField
          name="side"
          label={t('accounting.rule.side')}
          required
          data={[
            { value: 'debit', label: t('accounting.side.debit') },
            { value: 'credit', label: t('accounting.side.credit') },
          ]}
        />
        <FormActions submitLabel={t('accounting.common.save')} onCancel={onClose} cancelLabel={t('accounting.common.cancel')} />
      </Form>
    </FormModal>
  )
}
