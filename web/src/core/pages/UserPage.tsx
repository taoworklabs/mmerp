import { Button, Menu, Stack } from '@mantine/core'
import { IconPencil, IconPlus, IconTrash } from '@tabler/icons-react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import type { TFunction } from 'i18next'
import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router'
import { api, ApiError, unwrap, type Grant } from '@/shared/api/core'
import { errorText } from '@/shared/i18n'
import { PersonAvatar } from '@/shared/ui/avatar'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { Form, FormActions, OrgUnitField, SelectField, TextField } from '@/shared/ui/form'
import { FieldList } from '@/shared/ui/FieldList'
import { useConfirm } from '@/shared/ui/confirm'
import { notifySuccess } from '@/shared/ui/notify'
import { FormModal, RecordPage, TabActions } from '@/shared/ui/page'
import { ContentSkeleton, EmptyState, ErrorState, NotFoundPage, PageSkeleton } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import { coreKeys } from '../keys'

export function UserPage() {
  const { t } = useTranslation()
  const id = Number(useParams().id)
  const user = useQuery({ queryKey: coreKeys.users.detail(id), queryFn: () => unwrap(api.GET('/users/{id}', { params: { path: { id } } })) })

  if (user.isPending) return <PageSkeleton />
  if (user.error instanceof ApiError && user.error.status === 404) return <NotFoundPage back={{ to: '/admin/users', label: t('core.user.back') }} />
  if (user.isError) return <ErrorState message={errorText(t, user.error)} onRetry={() => void user.refetch()} />
  return (
    <RecordPage
      title={user.data.name}
      description={user.data.login}
      leading={<PersonAvatar name={user.data.name} size="lg" />}
      breadcrumbs={[{ label: t('core.user.title'), to: '/admin/users' }]}
      tabs={[
        { value: 'roles', label: t('core.user.roles'), content: <RolesTab userId={id} /> },
        { value: 'contact', label: t('core.user.contact'), content: <ContactTab userId={id} email={user.data.email} /> },
      ]}
    />
  )
}

// roleLabel names a role by its product's translation, e.g. hrm.role.hr.
const roleLabel = (t: TFunction, product: string, role: string) => t(`${product}.role.${role}`, { defaultValue: `${product}.${role}` })

function RolesTab({ userId }: { userId: number }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [granting, setGranting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const grants = useQuery({
    queryKey: coreKeys.users.roles(userId),
    queryFn: async () => (await unwrap(api.GET('/users/{id}/roles', { params: { path: { id: userId } } }))) ?? [],
  })

  const [ask, dialog] = useConfirm()

  async function revoke(g: Grant) {
    const role = roleLabel(t, g.product, g.role)
    if (!(await ask({ title: t('core.user.revoke_title'), message: t('core.user.revoke_confirm', { role }), confirmLabel: t('core.user.revoke'), danger: true }))) return
    setError(null)
    try {
      await unwrap(api.DELETE('/users/{id}/roles/{grant}', { params: { path: { id: userId, grant: g.id } } }))
    } catch (err) {
      setError(errorText(t, err))
      return
    }
    await qc.invalidateQueries({ queryKey: coreKeys.users.roles(userId) })
    notifySuccess(t('core.user.revoked'))
  }

  const columns: Column<Grant>[] = [
    { key: 'role', header: t('core.user.role'), role: 'title', render: (g) => roleLabel(t, g.product, g.role) },
    { key: 'unit', header: t('core.user.org_unit'), role: 'meta', render: (g) => g.org_unit_name ?? t('core.user.tenant_wide') },
  ]

  let body
  if (grants.isPending) body = <ContentSkeleton />
  else if (grants.isError) body = <ErrorState message={errorText(t, grants.error)} onRetry={() => void grants.refetch()} />
  else if (grants.data.length === 0) body = <EmptyState title={t('core.user.no_roles')} />
  else
    body = (
      <DataTable
        label={t('core.user.roles')}
        columns={columns}
        rows={grants.data}
        rowKey={(g) => g.id}
        menu={(g) => (
          <Menu.Item color="danger" leftSection={<IconTrash {...icon.text} />} onClick={() => void revoke(g)}>
            {t('core.user.revoke')}
          </Menu.Item>
        )}
      />
    )

  return (
    <Stack gap="md">
      <TabActions>
        <Button size="xs" leftSection={<IconPlus {...icon.text} />} onClick={() => setGranting(true)}>
          {t('core.user.grant')}
        </Button>
      </TabActions>
      {error && <ErrorState message={error} />}
      {body}
      {granting && <GrantModal userId={userId} onClose={() => setGranting(false)} />}
      {dialog}
    </Stack>
  )
}

type Values = { role: string | null; org_unit_id: string | null }

function GrantModal({ userId, onClose }: { userId: number; onClose: () => void }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const roles = useQuery({ queryKey: coreKeys.roles(), queryFn: async () => (await unwrap(api.GET('/roles'))) ?? [] })
  const form = useForm<Values>({ mode: 'onBlur', defaultValues: { role: null, org_unit_id: null } })
  const picked = form.watch('role')
  const tenantWide = roles.data?.some((r) => `${r.product}:${r.role}` === picked && r.tenant_wide) ?? false

  async function save(v: Values) {
    const [product = '', role = ''] = (v.role ?? '').split(':')
    await unwrap(
      api.POST('/users/{id}/roles', {
        params: { path: { id: userId } },
        body: { product, role, org_unit_id: v.org_unit_id && !tenantWide ? Number(v.org_unit_id) : null },
      }),
    )
    await qc.invalidateQueries({ queryKey: coreKeys.users.roles(userId) })
    notifySuccess(t('core.user.granted'))
    onClose()
  }

  return (
    <FormModal opened title={t('core.user.grant')} onClose={onClose}>
      <Form form={form} onSubmit={save} fieldOf={(err) => (err.code === 'role_already_granted' ? 'role' : undefined)}>
        <SelectField
          name="role"
          label={t('core.user.role')}
          required
          data={(roles.data ?? []).map((r) => ({ value: `${r.product}:${r.role}`, label: roleLabel(t, r.product, r.role) }))}
        />
        {!tenantWide && <OrgUnitField name="org_unit_id" label={t('core.user.org_unit')} clearable />}
        <FormActions submitLabel={t('core.user.grant')} onCancel={onClose} cancelLabel={t('core.common.cancel')} />
      </Form>
    </FormModal>
  )
}

// ContactTab: where the user's email notifications go, if anywhere.
function ContactTab({ userId, email }: { userId: number; email: string | null }) {
  const { t } = useTranslation()
  const [editing, setEditing] = useState(false)
  return (
    <Stack gap="md">
      <TabActions>
        <Button size="xs" variant="default" leftSection={<IconPencil {...icon.text} />} onClick={() => setEditing(true)}>
          {t('core.user.edit_email')}
        </Button>
      </TabActions>
      <FieldList rows={[[t('core.user.email'), email ?? t('core.user.no_email')]]} />
      {editing && <EmailModal userId={userId} email={email} onClose={() => setEditing(false)} />}
    </Stack>
  )
}

function EmailModal({ userId, email, onClose }: { userId: number; email: string | null; onClose: () => void }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const form = useForm<{ email: string }>({ mode: 'onBlur', defaultValues: { email: email ?? '' } })
  async function save(v: { email: string }) {
    await unwrap(api.PUT('/users/{id}/email', { params: { path: { id: userId } }, body: { email: v.email.trim() || null } }))
    await qc.invalidateQueries({ queryKey: coreKeys.users.detail(userId) })
    notifySuccess(t('core.user.email_saved'))
    onClose()
  }
  return (
    <FormModal opened title={t('core.user.edit_email')} onClose={onClose}>
      <Form form={form} onSubmit={save}>
        <TextField name="email" label={t('core.user.email')} type="email" maxLength={254} description={t('core.user.email_hint')} />
        <FormActions submitLabel={t('core.common.save')} onCancel={onClose} cancelLabel={t('core.common.cancel')} />
      </Form>
    </FormModal>
  )
}
