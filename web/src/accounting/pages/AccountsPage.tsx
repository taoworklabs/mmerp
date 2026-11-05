import { Button, Group, Text } from '@mmerp/ui'
import { IconArchive, IconArchiveOff, IconPencil, IconPlus } from '@tabler/icons-react'
import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type Account, type AccountFields } from '@/shared/api/accounting'
import { useCan, useProductOn } from '@/shared/auth/me'
import { errorText } from '@/shared/i18n'
import { CheckboxField, Form, FormActions, SelectField, TextField } from '@/shared/ui/form'
import { MenuAction } from '@/shared/ui/MenuAction'
import { notifySuccess, notifyUndo } from '@/shared/ui/notify'
import { FormModal, ListPage } from '@/shared/ui/page'
import { ContentSkeleton, EmptyState, ErrorState } from '@/shared/ui/states'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { icon } from '@/shared/ui/theme'
import { TreeList, type TreeNode } from '@/shared/ui/TreeList'
import { accountOptions, useAccounts } from '../hooks/useAccounting'
import { parentCandidates } from '../tree'
import { accountingKeys } from '../keys'

const types = ['asset', 'liability', 'equity', 'revenue', 'expense'] as const satisfies readonly Account['type'][]

// accountTree hangs each account under its parent; the list comes ordered by code.
function accountTree(accounts: Account[]): TreeNode<Account>[] {
  const nodes = new Map(accounts.map((a) => [a.id, { key: a.id, item: a, children: [] as TreeNode<Account>[] }]))
  const roots: TreeNode<Account>[] = []
  for (const n of nodes.values()) {
    const parent = n.item.parent_id ? nodes.get(n.item.parent_id) : undefined
    ;(parent ? parent.children : roots).push(n)
  }
  return roots
}

const fieldsOf = (a: Account): AccountFields => ({ code: a.code, name: a.name, type: a.type, parent_id: a.parent_id ?? null, active: a.active })

// AccountsPage: the chart of accounts as a tree; accounting administrators add, edit and retire accounts.
export function AccountsPage() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  // The chart has no allowed_actions; a disabled product is read-only.
  const on = useProductOn('accounting')
  const writable = useCan('accounting.manage') && on
  const accounts = useAccounts()
  const [editing, setEditing] = useState<Account | 'new' | null>(null)
  const [error, setError] = useState<string | null>(null)

  // Retiring is reversible and touches no document, so it applies at once and offers Undo.
  async function toggle(a: Account, undo = false) {
    setError(null)
    try {
      await unwrap(api.PUT('/accounting/accounts/{id}', { params: { path: { id: a.id } }, body: { ...fieldsOf(a), active: !a.active } }))
    } catch (err) {
      setError(errorText(t, err))
      return
    } finally {
      await qc.invalidateQueries({ queryKey: accountingKeys.accounts() })
    }
    if (!undo)
      notifyUndo(t(a.active ? 'accounting.accounts.retired_done' : 'accounting.accounts.reactivated_done', { name: a.name }), t('shared.undo'), () =>
        void toggle({ ...a, active: !a.active }, true),
      )
  }

  function body() {
    if (accounts.isPending) return <ContentSkeleton />
    if (accounts.isError) return <ErrorState message={errorText(t, accounts.error)} onRetry={() => void accounts.refetch()} />
    if (accounts.data.length === 0) return <EmptyState title={t('accounting.accounts.empty')} description={t('accounting.accounts.empty_description')} />
    return (
      <TreeList
        label={t('accounting.accounts.title')}
        nodes={accountTree(accounts.data)}
        render={(a) => (
          <Group gap="xs" wrap={false}>
            <Text size="sm" weight={500}>
              {a.code}
            </Text>
            <Text size="sm" truncate>
              {a.name}
            </Text>
            {!a.active && <StatusBadge tone="neutral">{t('accounting.account.retired')}</StatusBadge>}
          </Group>
        )}
        meta={(a) => t(`accounting.account.type_${a.type}`)}
        menu={
          writable
            ? (a) => (
                <>
                  <MenuAction icon={IconPencil} onClick={() => setEditing(a)}>
                    {t('accounting.accounts.edit')}
                  </MenuAction>
                  <MenuAction icon={a.active ? IconArchive : IconArchiveOff} onClick={() => void toggle(a)}>
                    {t(a.active ? 'accounting.accounts.retire' : 'accounting.accounts.reactivate')}
                  </MenuAction>
                </>
              )
            : undefined
        }
        toggleLabel={(a, folded) => t(folded ? 'accounting.accounts.expand' : 'accounting.accounts.collapse', { name: a.name })}
      />
    )
  }

  return (
    <ListPage
      title={t('accounting.accounts.title')}
      description={t('accounting.accounts.description')}
      action={
        writable && (
          <Button leftSection={<IconPlus {...icon.button} />} onClick={() => setEditing('new')}>
            {t('accounting.accounts.create')}
          </Button>
        )
      }
    >
      {error && <ErrorState message={error} />}
      {body()}
      {editing && <AccountModal account={editing === 'new' ? null : editing} accounts={accounts.data ?? []} onClose={() => setEditing(null)} />}
    </ListPage>
  )
}

type Values = { code: string; name: string; type: string; parent_id: string | null; active: boolean }

function AccountModal({ account: a, accounts, onClose }: { account: Account | null; accounts: Account[]; onClose: () => void }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const form = useForm<Values>({
    mode: 'onBlur',
    defaultValues: a ? { ...a, parent_id: a.parent_id ? String(a.parent_id) : null } : { code: '', name: '', type: '', parent_id: null, active: true },
  })
  const type = form.watch('type')
  async function save(v: Values) {
    const body: AccountFields = {
      code: v.code.trim(),
      name: v.name.trim(),
      type: v.type as Account['type'],
      parent_id: v.parent_id ? Number(v.parent_id) : null,
      active: v.active,
    }
    if (a) await unwrap(api.PUT('/accounting/accounts/{id}', { params: { path: { id: a.id } }, body }))
    else await unwrap(api.POST('/accounting/accounts', { body }))
    await qc.invalidateQueries({ queryKey: accountingKeys.accounts() })
    notifySuccess(t('accounting.accounts.saved'))
    onClose()
  }
  return (
    <FormModal opened title={t(a ? 'accounting.accounts.edit' : 'accounting.accounts.create')} onClose={onClose}>
      <Form form={form} onSubmit={save} errorFields={{ account_code_taken: 'code', account_parent_invalid: 'parent_id', account_name_required: 'name' }}>
        <TextField name="code" label={t('accounting.account.code')} required maxLength={20} rules={{ pattern: { value: /^[0-9A-Za-z.]+$/, message: t('accounting.account.code_hint') } }} />
        <TextField name="name" label={t('accounting.account.name')} required maxLength={200} />
        <SelectField name="type" label={t('accounting.account.type')} required data={types.map((k) => ({ value: k, label: t(`accounting.account.type_${k}`) }))} />
        <SelectField
          name="parent_id"
          label={t('accounting.account.parent')}
          description={t('accounting.account.parent_hint')}
          clearable
          data={accountOptions(parentCandidates(accounts, type, a?.id))}
        />
        <CheckboxField name="active" label={t('accounting.account.active')} description={t('accounting.account.active_hint')} />
        <FormActions submitLabel={t('accounting.common.save')} onCancel={onClose} cancelLabel={t('accounting.common.cancel')} />
      </Form>
    </FormModal>
  )
}
