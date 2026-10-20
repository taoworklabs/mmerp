import { Button } from '@mantine/core'
import { IconTrash } from '@tabler/icons-react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import type { TFunction } from 'i18next'
import { useEffect, useRef, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { Route, Routes, useNavigate, useParams } from 'react-router'
import { api, unwrap, type Role, type TypeRule } from '@/shared/api/core'
import { ApiError } from '@/shared/api/error'
import { errorText } from '@/shared/i18n'
import { DataTable, type Column } from '@/shared/ui/DataTable'
import { useConfirm } from '@/shared/ui/confirm'
import { DecimalField, fieldId, Form, FormActions, FormRow, FormSection, SelectField, type Option } from '@/shared/ui/form'
import { Notice } from '@/shared/ui/Notice'
import { notifySuccess } from '@/shared/ui/notify'
import { ListPage, Page } from '@/shared/ui/page'
import { ContentSkeleton, EmptyState, ErrorState, NotFoundPage, PageSkeleton } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import { actionLabel } from './ApprovalPanel'
import { documentKeys } from './keys'
import { FlowBlock } from './flowEditor/FlowBlock'
import { flowSummary, formPathOf, toRuleInput, toRuleValues, type RuleValues } from './flowForm'

type Props = { product: string; basePath: string } // basePath: where the area mounts this, e.g. /hrm/approval-rules

function useRules(product: string) {
  return useQuery({
    queryKey: documentKeys.approvalRules(product),
    queryFn: async () => (await unwrap(api.GET('/approval-rules', { params: { query: { product } } }))) ?? [],
  })
}

// ruleLabel names one rule: the document type, then the action it gates.
function ruleLabel(t: TFunction, r: TypeRule) {
  return `${t(`${r.doc_type}.name`, { defaultValue: r.doc_type })} · ${actionLabel(t, r.doc_type, r.action)}`
}

// A post rule keeps the type's own path; other actions get theirs under it.
const rulePath = (basePath: string, r: TypeRule) => (r.action === 'post' ? `${basePath}/${r.doc_type}` : `${basePath}/${r.doc_type}/${r.action}`)

// ApprovalRules lists and edits the approval rules of one product's document types, one per action.
// The area mounts it at a splat route ("approval-rules/*").
export function ApprovalRules(props: Props) {
  return (
    <Routes>
      <Route index element={<RuleList {...props} />} />
      <Route path=":docType" element={<RuleEditor {...props} />} />
      <Route path=":docType/:action" element={<RuleEditor {...props} />} />
    </Routes>
  )
}

function RuleList({ product, basePath }: Props) {
  const { t } = useTranslation()
  const rules = useRules(product)
  const columns: Column<TypeRule>[] = [
    { key: 'doc_type', header: t('shared.rule.doc_type'), role: 'title', render: (r) => ruleLabel(t, r) },
    {
      key: 'rule',
      header: t('shared.rule.steps'),
      role: 'meta',
      render: (r) => {
        if (!r.rule) return t('shared.rule.none')
        const s = flowSummary(r.rule.flow)
        return t('shared.rule.summary', { count: s.steps }) + (s.branched ? t('shared.rule.summary_branched') : '')
      },
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
        rowKey={(r) => `${r.doc_type}:${r.action}`}
        href={(r) => rulePath(basePath, r)}
      />
    )
  return (
    <ListPage title={t('shared.rule.title')} description={t('shared.rule.description')}>
      {body}
    </ListPage>
  )
}

const roleValue = (r: Role) => `${r.product}.${r.role}`

// RuleEditor edits the rule of one document type's action: its approval flow, plus the fallback role.
function RuleEditor({ product, basePath }: Props) {
  const { t } = useTranslation()
  const params = useParams()
  const docType = params.docType ?? ''
  const action = params.action ?? 'post'
  const rules = useRules(product)
  const roles = useQuery({ queryKey: documentKeys.roles(), queryFn: async () => (await unwrap(api.GET('/roles'))) ?? [] })
  const users = useQuery({ queryKey: documentKeys.approvalRuleUsers(), queryFn: async () => (await unwrap(api.GET('/approval-rules/users'))) ?? [] })
  if (rules.isPending || roles.isPending || users.isPending) return <PageSkeleton />
  if (rules.isError) return <ErrorState message={errorText(t, rules.error)} onRetry={() => void rules.refetch()} />
  if (roles.isError) return <ErrorState message={errorText(t, roles.error)} onRetry={() => void roles.refetch()} />
  if (users.isError) return <ErrorState message={errorText(t, users.error)} onRetry={() => void users.refetch()} />
  const rule = rules.data.find((r) => r.doc_type === docType && r.action === action)
  if (!rule) return <NotFoundPage back={{ to: basePath, label: t('shared.rule.back') }} />
  const userOptions = users.data.map((u) => ({ value: String(u.id), label: `${u.name} (${u.login})` }))
  // Tenant-wide roles administer the system; they do not approve documents.
  return <RuleForm key={`${docType}:${action}`} rule={rule} basePath={basePath} roles={roles.data.filter((r) => !r.tenant_wide)} userOptions={userOptions} />
}

function RuleForm({ rule, basePath, roles, userOptions }: { rule: TypeRule; basePath: string; roles: Role[]; userOptions: Option[] }) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const [ask, confirm] = useConfirm()
  // A failed delete, shown at the top of the page until the next attempt.
  const [error, setError] = useState<string | null>(null)
  const form = useForm<RuleValues>({ mode: 'onBlur', defaultValues: toRuleValues(rule) })
  const name = ruleLabel(t, rule)
  const params = { path: { doc_type: rule.doc_type }, query: { action: rule.action } }
  const allowed = rule.allowed_actions
  // Read-only while the product is disabled.
  const readOnly = !allowed.includes('save')
  const roleOptions: Option[] = roles.map((r) => ({ value: roleValue(r), label: t(`${r.product}.role.${r.role}`, { defaultValue: roleValue(r) }) }))
  const ctx = { rule, roleOptions, userOptions, readOnly }

  // The server's complaint about one element goes away once the user edits, not only on the next submit.
  const issue = useRef<string | null>(null)
  useEffect(() => {
    const sub = form.watch(() => {
      if (!issue.current) return
      form.clearErrors(issue.current as never)
      issue.current = null
    })
    return () => sub.unsubscribe()
  }, [form])

  async function save(v: RuleValues): Promise<void | false> {
    try {
      await unwrap(api.PUT('/approval-rules/{doc_type}', { params, body: toRuleInput(v) }))
    } catch (err) {
      if (!(err instanceof ApiError) || err.code !== 'invalid_rule') throw err
      const path = String(err.params.path ?? '')
      const message = t(`shared.rule.reason.${String(err.params.reason)}`, { defaultValue: errorText(t, err) })
      // The only rule-level complaint is about the fallback role.
      const at = path ? `${formPathOf(path)}.issue` : 'fallback'
      issue.current = path ? at : null
      form.setError(at as never, { message })
      requestAnimationFrame(() => document.getElementById(fieldId(path ? formPathOf(path) : at))?.scrollIntoView({ block: 'center' }))
      return false
    }
    await qc.invalidateQueries({ queryKey: documentKeys.approvalRules() })
    notifySuccess(t('shared.rule.saved'))
  }

  async function remove() {
    if (!(await ask({ title: t('shared.rule.delete'), message: t('shared.rule.delete_confirm', { name }), confirmLabel: t('shared.rule.delete'), danger: true }))) return
    setError(null)
    try {
      await unwrap(api.DELETE('/approval-rules/{doc_type}', { params }))
      await qc.invalidateQueries({ queryKey: documentKeys.approvalRules() })
      notifySuccess(t('shared.rule.deleted'))
      navigate(basePath)
    } catch (err) {
      setError(errorText(t, err))
    }
  }

  return (
    <Page
      centered
      title={name}
      breadcrumbs={[{ label: t('shared.rule.title'), to: basePath }]}
      actions={
        rule.rule &&
        allowed.includes('delete') && (
          <Button variant="subtle" color="danger" leftSection={<IconTrash {...icon.button} />} onClick={() => void remove()}>
            {t('shared.rule.delete')}
          </Button>
        )
      }
    >
      {error && (
        <Notice tone="danger">
          {error}
        </Notice>
      )}
      <Form form={form} onSubmit={save}>
        <FormSection title={t('shared.rule.flow_section')} description={t('shared.rule.flow_hint')} wide>
          <FormRow>
            <FlowBlock path="flow" ctx={ctx} depth={0} />
          </FormRow>
        </FormSection>
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
