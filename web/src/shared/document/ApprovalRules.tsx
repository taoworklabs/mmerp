import { ActionIcon, Button, Group, Paper, Stack, Text, Tooltip } from '@mantine/core'
import { IconPlus, IconTrash } from '@tabler/icons-react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useFieldArray, useForm, useWatch, type Control } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { Route, Routes, useNavigate, useParams } from 'react-router'
import { api, unwrap, type Role, type RuleCondition, type RuleInput, type TypeRule } from '@/shared/api/core'
import { errorText } from '@/shared/i18n'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { useConfirm } from '@/shared/ui/confirm'
import { DecimalField, Form, FormActions, FormSection, SelectField, type Option } from '@/shared/ui/form'
import { notifySuccess } from '@/shared/ui/notify'
import { ListPage, Page } from '@/shared/ui/page'
import { ContentSkeleton, EmptyState, ErrorState, NotFoundPage, PageSkeleton } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import { documentKeys } from './keys'

type Props = { product: string; basePath: string } // basePath: where the area mounts this, e.g. /hrm/approval-rules

function useRules(product: string) {
  return useQuery({
    queryKey: documentKeys.approvalRules(product),
    queryFn: async () => (await unwrap(api.GET('/approval-rules', { params: { query: { product } } }))) ?? [],
  })
}

// ApprovalRules lists and edits the approval rules of one product's document types.
// The area mounts it at a splat route ("approval-rules/*").
export function ApprovalRules(props: Props) {
  return (
    <Routes>
      <Route index element={<RuleList {...props} />} />
      <Route path=":docType" element={<RuleEditor {...props} />} />
    </Routes>
  )
}

function RuleList({ product, basePath }: Props) {
  const { t } = useTranslation()
  const rules = useRules(product)
  const columns: Column<TypeRule>[] = [
    { key: 'doc_type', header: t('shared.rule.doc_type'), role: 'title', render: (r) => t(`${r.doc_type}.name`, { defaultValue: r.doc_type }) },
    {
      key: 'rule',
      header: t('shared.rule.steps'),
      role: 'meta',
      render: (r) => (r.rule ? t('shared.rule.step_count', { count: r.rule.steps.length }) : t('shared.rule.none')),
    },
  ]
  let body
  if (rules.isPending) body = <ContentSkeleton />
  else if (rules.isError) body = <ErrorState message={errorText(t, rules.error)} onRetry={() => void rules.refetch()} />
  else if (rules.data.length === 0) body = <EmptyState title={t('shared.rule.empty')} />
  else
    body = (
      <DataTable
        label={t('shared.rule.title')}
        columns={columns}
        rows={rules.data}
        rowKey={(r) => r.doc_type}
        href={(r) => `${basePath}/${r.doc_type}`}
      />
    )
  return (
    <ListPage title={t('shared.rule.title')} description={t('shared.rule.description')}>
      {body}
    </ListPage>
  )
}

type StepValues = { kind: string | null; role: string | null; user: string | null; field: string | null; op: string | null; value: string }
type Values = { steps: StepValues[]; max_levels: string; fallback: string | null }

const roleValue = (r: Role) => `${r.product}.${r.role}`
const splitRole = (v: string) => {
  const [product = '', ...rest] = v.split('.')
  return { product, role: rest.join('.') }
}

const newStep = (r: TypeRule): StepValues => ({ kind: r.module_approvers ? 'module' : 'role', role: null, user: null, field: null, op: null, value: '' })

function toValues(r: TypeRule): Values {
  if (!r.rule) return { steps: [newStep(r)], max_levels: '3', fallback: null }
  return {
    steps: r.rule.steps.map((s) => ({
      kind: s.approver.kind,
      role: s.approver.kind === 'role' ? `${s.approver.product}.${s.approver.role}` : null,
      user: s.approver.kind === 'user' && s.approver.user_id ? String(s.approver.user_id) : null,
      field: s.condition?.field ?? null,
      op: s.condition?.op ?? null,
      value: s.condition?.value ?? '',
    })),
    max_levels: String(r.rule.max_levels),
    fallback: `${r.rule.fallback_product}.${r.rule.fallback_role}`,
  }
}

function toInput(v: Values): RuleInput {
  const fallback = splitRole(v.fallback ?? '')
  return {
    steps: v.steps.map((s) => ({
      approver:
        s.kind === 'role' ? { kind: 'role', ...splitRole(s.role ?? '') } : s.kind === 'user' ? { kind: 'user', user_id: Number(s.user) } : { kind: 'module' },
      condition: s.field && s.op ? { field: s.field, op: s.op as RuleCondition['op'], value: s.value } : undefined,
    })),
    max_levels: Number(v.max_levels),
    fallback_product: fallback.product,
    fallback_role: fallback.role,
  }
}

// RuleEditor edits the rule of one document type: a chain of steps, each with an
// optional condition on an approval field, plus the fallback role.
function RuleEditor({ product, basePath }: Props) {
  const { t } = useTranslation()
  const docType = useParams().docType ?? ''
  const rules = useRules(product)
  const roles = useQuery({ queryKey: documentKeys.roles(), queryFn: async () => (await unwrap(api.GET('/roles'))) ?? [] })
  const users = useQuery({ queryKey: documentKeys.approvalRuleUsers(), queryFn: async () => (await unwrap(api.GET('/approval-rules/users'))) ?? [] })
  if (rules.isPending || roles.isPending || users.isPending) return <PageSkeleton />
  if (rules.isError) return <ErrorState message={errorText(t, rules.error)} onRetry={() => void rules.refetch()} />
  if (roles.isError) return <ErrorState message={errorText(t, roles.error)} onRetry={() => void roles.refetch()} />
  if (users.isError) return <ErrorState message={errorText(t, users.error)} onRetry={() => void users.refetch()} />
  const rule = rules.data.find((r) => r.doc_type === docType)
  if (!rule) return <NotFoundPage back={{ to: basePath, label: t('shared.rule.back') }} />
  const userOptions = users.data.map((u) => ({ value: String(u.id), label: `${u.name} (${u.login})` }))
  // Tenant-wide roles administer the system; they do not approve documents.
  return <RuleForm key={docType} rule={rule} basePath={basePath} roles={roles.data.filter((r) => !r.tenant_wide)} userOptions={userOptions} />
}

function RuleForm({ rule, basePath, roles, userOptions }: { rule: TypeRule; basePath: string; roles: Role[]; userOptions: Option[] }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const [ask, confirm] = useConfirm()
  const form = useForm<Values>({ mode: 'onBlur', defaultValues: toValues(rule) })
  const steps = useFieldArray({ control: form.control, name: 'steps' })
  const name = t(`${rule.doc_type}.name`, { defaultValue: rule.doc_type })
  const allowed = rule.allowed_actions
  // Read-only while the product is disabled.
  const readOnly = !allowed.includes('save')
  const roleOptions: Option[] = roles.map((r) => ({ value: roleValue(r), label: t(`${r.product}.role.${r.role}`, { defaultValue: roleValue(r) }) }))

  async function save(v: Values) {
    await unwrap(api.PUT('/approval-rules/{doc_type}', { params: { path: { doc_type: rule.doc_type } }, body: toInput(v) }))
    await qc.invalidateQueries({ queryKey: documentKeys.approvalRules() })
    notifySuccess(t('shared.rule.saved'))
  }

  async function remove() {
    if (!(await ask({ title: t('shared.rule.delete'), message: t('shared.rule.delete_confirm', { name }), confirmLabel: t('shared.rule.delete'), danger: true }))) return
    await unwrap(api.DELETE('/approval-rules/{doc_type}', { params: { path: { doc_type: rule.doc_type } } }))
    await qc.invalidateQueries({ queryKey: documentKeys.approvalRules() })
    notifySuccess(t('shared.rule.deleted'))
    navigate(basePath)
  }

  return (
    <Page
      title={name}
      breadcrumbs={[{ label: t('shared.rule.title'), to: basePath }]}
      description={t('shared.rule.form_description')}
      actions={
        rule.rule &&
        allowed.includes('delete') && (
          <Button variant="outline" color="danger" leftSection={<IconTrash {...icon.button} />} onClick={() => void remove()}>
            {t('shared.rule.delete')}
          </Button>
        )
      }
    >
      <Form form={form} onSubmit={save}>
        <Stack gap="md">
          {steps.fields.map((s, i) => (
            <Paper key={s.id} withBorder p="md">
              <Stack gap="sm">
                <Group justify="space-between">
                  <Text fw={600}>{t('shared.rule.step', { n: i + 1 })}</Text>
                  {!readOnly && (
                    <Tooltip label={t('shared.rule.remove_step')}>
                      <ActionIcon variant="subtle" color="gray" aria-label={t('shared.rule.remove_step')} disabled={steps.fields.length === 1} onClick={() => steps.remove(i)}>
                        <IconTrash {...icon.button} />
                      </ActionIcon>
                    </Tooltip>
                  )}
                </Group>
                <StepFields index={i} readOnly={readOnly} rule={rule} roleOptions={roleOptions} userOptions={userOptions} control={form.control} />
              </Stack>
            </Paper>
          ))}
          {!readOnly && (
            <Group>
              <Button size="xs" variant="light" leftSection={<IconPlus {...icon.text} />} onClick={() => steps.append(newStep(rule))}>
                {t('shared.rule.add_step')}
              </Button>
            </Group>
          )}
        </Stack>
        <FormSection title={t('shared.rule.fallback_section')} description={t('shared.rule.fallback_hint')}>
          <SelectField name="fallback" label={t('shared.rule.fallback')} required readOnly={readOnly} data={roleOptions} />
          <DecimalField name="max_levels" label={t('shared.rule.max_levels')} description={t('shared.rule.max_levels_hint')} required readOnly={readOnly} scale={0} min={1} />
        </FormSection>
        {!readOnly && <FormActions submitLabel={t('shared.rule.save')} />}
      </Form>
      {confirm}
    </Page>
  )
}

function StepFields(props: { index: number; readOnly: boolean; rule: TypeRule; roleOptions: Option[]; userOptions: Option[]; control: Control<Values> }) {
  const { index, readOnly, rule, roleOptions, userOptions, control } = props
  const { t } = useTranslation()
  const step = useWatch({ control, name: `steps.${index}` })
  const field = rule.fields.find((f) => f.key === step?.field)
  const p = `steps.${index}`
  const kinds = [...(rule.module_approvers ? ['module'] : []), 'role', 'user']
  return (
    <>
      <SelectField readOnly={readOnly} name={`${p}.kind`} label={t('shared.rule.approver')} required data={kinds.map((k) => ({ value: k, label: t(`shared.rule.kind.${k}`) }))} />
      {step?.kind === 'role' && <SelectField readOnly={readOnly} name={`${p}.role`} label={t('shared.rule.role')} required data={roleOptions} />}
      {step?.kind === 'user' && <SelectField readOnly={readOnly} name={`${p}.user`} label={t('shared.rule.user')} required data={userOptions} />}
      <SelectField
        readOnly={readOnly}
        name={`${p}.field`}
        label={t('shared.rule.condition')}
        description={t('shared.rule.condition_hint')}
        clearable
        data={rule.fields.map((f) => ({ value: f.key, label: t(f.label, { defaultValue: f.key }) }))}
      />
      {field && (
        <>
          <SelectField readOnly={readOnly} name={`${p}.op`} label={t('shared.rule.op')} required data={field.ops.map((o) => ({ value: o, label: t(`shared.rule.op.${o}`) }))} />
          {field.kind === 'choice' ? (
            <SelectField readOnly={readOnly} name={`${p}.value`} label={t('shared.rule.value')} required data={field.options ?? []} />
          ) : (
            <DecimalField readOnly={readOnly} name={`${p}.value`} label={t('shared.rule.value')} required scale={1} />
          )}
        </>
      )}
    </>
  )
}
