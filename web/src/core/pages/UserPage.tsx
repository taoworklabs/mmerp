import { Button, Stack, Text } from '@mantine/core'
import { IconMail, IconPlus, IconTrash, IconUserCircle } from '@tabler/icons-react'
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
import { Form, FormActions, OrgUnitField, SelectField } from '@/shared/ui/form'
import { FieldList, MissingValue } from '@/shared/ui/FieldList'
import { KeyFacts } from '@/shared/ui/KeyFacts'
import { InlineEditField } from '@/shared/ui/InlineEditField'
import { MenuAction } from '@/shared/ui/MenuAction'
import { notifySuccess } from '@/shared/ui/notify'
import { FormModal, Panel, RecordColumns, RecordPage, SeeAll, TabContent } from '@/shared/ui/page'
import { QueryView } from '@/shared/ui/QueryView'
import { ContentSkeleton, EmptyState, ErrorState, NotFoundPage, PageSkeleton } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import { useAction } from '@/shared/ui/useAction'
import { coreKeys } from '../keys'

export function UserPage() {
  const { t } = useTranslation()
  const id = Number(useParams().id)
  const [granting, setGranting] = useState(false)
  const user = useQuery({ queryKey: coreKeys.users.detail(id), queryFn: () => unwrap(api.GET('/users/{id}', { params: { path: { id } } })) })
  const grants = useGrants(id)

  if (user.isPending) return <PageSkeleton />
  if (user.error instanceof ApiError && user.error.status === 404) return <NotFoundPage back={{ to: '/admin/users', label: t('core.user.back') }} />
  if (user.isError) return <ErrorState message={errorText(t, user.error)} onRetry={() => void user.refetch()} />
  const u = user.data
  const count = grants.data?.length
  return (
    <RecordPage
      title={u.name}
      description={u.login}
      leading={<PersonAvatar name={u.name} size="lg" />}
      breadcrumbs={[{ label: t('core.user.title'), to: '/admin/users' }]}
      summary={
        <KeyFacts
          facts={[
            [t('core.user.login'), u.login],
            [t('core.user.email'), u.email || <MissingValue label={t('core.user.email')} />],
            [t('core.user.roles'), count === undefined ? '…' : count],
          ]}
        />
      }
      actions={
        <Button variant="default" leftSection={<IconPlus {...icon.button} />} onClick={() => setGranting(true)}>
          {t('core.user.grant')}
        </Button>
      }
      tabs={[
        {
          value: 'overview',
          label: t('shared.record.overview'),
          content: <RecordColumns main={<UserProfile userId={id} login={u.login} name={u.name} email={u.email} />} side={<RolesDigest userId={id} />} />,
        },
        { value: 'roles', label: count ? t('core.user.roles_count', { count }) : t('core.user.roles'), content: <RolesTab userId={id} /> },
      ]}
    >
      {granting && <GrantModal userId={id} onClose={() => setGranting(false)} />}
    </RecordPage>
  )
}

function useGrants(userId: number) {
  return useQuery({
    queryKey: coreKeys.users.roles(userId),
    queryFn: async () => (await unwrap(api.GET('/users/{id}/roles', { params: { path: { id: userId } } }))) ?? [],
  })
}

// RolesDigest is the overview's right column: the user's first roles, leading to the tab of them all.
function RolesDigest({ userId }: { userId: number }) {
  const { t } = useTranslation()
  const grants = useGrants(userId)
  return (
    <Panel plain title={t('core.user.roles')} actions={<SeeAll tab="roles" />}>
      {grants.isPending ? (
        <ContentSkeleton />
      ) : grants.isError ? (
        <ErrorState message={errorText(t, grants.error)} onRetry={() => void grants.refetch()} />
      ) : grants.data.length === 0 ? (
        <Text size="sm" c="dimmed">
          {t('core.user.no_roles')}
        </Text>
      ) : (
        <Stack gap="sm">
          {grants.data.slice(0, 5).map((g) => (
            <Stack key={g.id} gap={0}>
              <Text fw={600}>{roleLabel(t, g.product, g.role)}</Text>
              <Text size="sm" c="dimmed">
                {g.org_unit_name ?? t('core.user.tenant_wide')}
              </Text>
            </Stack>
          ))}
        </Stack>
      )}
    </Panel>
  )
}

// roleLabel names a role by its product's translation, e.g. hrm.role.hr.
const roleLabel = (t: TFunction, product: string, role: string) => t(`${product}.role.${role}`, { defaultValue: `${product}.${role}` })

function RolesTab({ userId }: { userId: number }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const grants = useGrants(userId)
  const action = useAction()

  const revoke = (g: Grant) =>
    action.run(
      async () => {
        await unwrap(api.DELETE('/users/{id}/roles/{grant}', { params: { path: { id: userId, grant: g.id } } }))
        await qc.invalidateQueries({ queryKey: coreKeys.users.roles(userId) })
      },
      {
        confirm: {
          title: t('core.user.revoke_title'),
          message: t('core.user.revoke_confirm', { role: roleLabel(t, g.product, g.role) }),
          confirmLabel: t('core.user.revoke'),
          danger: true,
        },
        success: t('core.user.revoked'),
      },
    )

  const columns: Column<Grant>[] = [
    { key: 'role', header: t('core.user.role'), role: 'title', render: (g) => roleLabel(t, g.product, g.role) },
    { key: 'unit', header: t('core.user.org_unit'), role: 'meta', render: (g) => g.org_unit_name ?? t('core.user.tenant_wide') },
  ]

  const body = (
    <QueryView query={grants} empty={<EmptyState title={t('core.user.no_roles')} />}>
      {(data) => (
        <DataTable
          label={t('core.user.roles')}
          columns={columns}
          rows={data}
          rowKey={(g) => g.id}
          menu={(g) => (
            <MenuAction icon={IconTrash} danger onClick={() => void revoke(g)}>
              {t('core.user.revoke')}
            </MenuAction>
          )}
        />
      )}
    </QueryView>
  )

  return (
    <TabContent error={action.error}>
      {body}
      {action.dialog}
    </TabContent>
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
      <Form form={form} onSubmit={save} errorFields={{ role_already_granted: 'role' }}>
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

// UserProfile: the account, read only, and where the user's email notifications go, if anywhere;
// the address is one value changed on its own, so it is edited in place.
function UserProfile({ userId, login, name, email }: { userId: number; login: string; name: string; email: string | null }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  return (
    <>
      <Panel title={t('core.user.account')} icon={IconUserCircle}>
        <FieldList
          stacked
          rows={[
            [t('core.user.login'), login],
            [t('core.user.name'), name],
          ]}
        />
      </Panel>
      <Panel title={t('core.user.contact')} icon={IconMail}>
        <Stack gap="xxs">
          <InlineEditField
            label={t('core.user.email')}
            value={email ?? ''}
            maxLength={254}
            validate={(v) => (v && !/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(v) ? t('core.user.email_invalid') : null)}
            onSave={async (v) => {
              await unwrap(api.PUT('/users/{id}/email', { params: { path: { id: userId } }, body: { email: v || null } }))
              await qc.invalidateQueries({ queryKey: coreKeys.users.detail(userId) })
            }}
          />
          <Text size="xs" c="dimmed">
            {t('core.user.email_hint')}
          </Text>
        </Stack>
      </Panel>
    </>
  )
}
