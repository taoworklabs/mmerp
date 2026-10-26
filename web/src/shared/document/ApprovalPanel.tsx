import { Badge, Button, Group, Stack, Text, type BadgeProps } from '@mmerp/ui'
import { Timeline } from '@mantine/core'
import { IconArrowBackUp, IconCheck, IconUserShare, IconX, type Icon as TablerIcon } from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import type { TFunction } from 'i18next'
import { useState, type ReactNode } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, ApiError, unwrap, type ApprovalInstance, type ApprovalPlan, type ApprovalStep } from '@/shared/api/core'
import { useInvalidateRecord } from '@/shared/area'
import { useMe } from '@/shared/auth/me'
import { errorText, formatDateTime } from '@/shared/i18n'
import { useConfirm } from '@/shared/ui/confirm'
import { Form, FormActions, SelectField, TextField } from '@/shared/ui/form'
import { Notice } from '@/shared/ui/Notice'
import { notifySuccess } from '@/shared/ui/notify'
import { FormModal } from '@/shared/ui/page'
import { ContentSkeleton, ErrorState } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import classes from './ApprovalPanel.module.css'
import { documentKeys } from './keys'

// actionLabel names an approval action: post and cancel are core's, the rest the type's product's.
export function actionLabel(t: TFunction, docType: string, action: string) {
  if (action === 'post' || action === 'cancel') return t(`shared.rule.action.${action}`)
  const product = docType.split('.')[0]
  return t(`${product}.action.${action}`, { defaultValue: action })
}

type PanelProps = {
  docType: string
  id: number
  // Runs after a decision (e.g. to leave the inbox item).
  onDone?: () => void
  // Keeps approve and reject at the bottom of the screen where the panel is the page's main task.
  sticky?: boolean
  // Who the document is for, when it may be someone other than its submitter: "sent for …".
  subject?: string
  // Approve and reject are in the page header (ApprovalDecision), so the panel only shows the steps.
  decideElsewhere?: boolean
}

function useApproval(docType: string, id: number) {
  return useQuery({
    queryKey: documentKeys.approval(docType, id),
    queryFn: () => unwrap(api.GET('/documents/{type}/{id}/approval', { params: { path: { type: docType, id } } })),
  })
}

// ApprovalPanel shows the latest submission's steps, every one of them, and, for the actor at an
// open step, approve, reject and reassign.
export function ApprovalPanel({ docType, id, onDone, sticky, subject, decideElsewhere }: PanelProps) {
  const { t } = useTranslation()
  const q = useApproval(docType, id)
  if (q.isPending) return <ContentSkeleton />
  if (q.isError) return <ErrorState message={errorText(t, q.error)} onRetry={() => void q.refetch()} />
  const inst = q.data.instance
  if (!inst)
    return (
      <Text size="sm" tone="dimmed">
        {t('shared.approval.none')}
      </Text>
    )
  return <Instance docType={docType} id={id} inst={inst} onDone={onDone} sticky={sticky} subject={subject} decideElsewhere={decideElsewhere} />
}

// ApprovalDecision is approve and reject for a page header, when the actor may vote on exactly
// one open step; with several, each step in the panel has its own. Errors go to the page's banner.
export function ApprovalDecision({ docType, id, onError }: { docType: string; id: number; onError: (message: string | null) => void }) {
  const inst = useApproval(docType, id).data?.instance
  const only = inst && soleVoting(inst)
  if (!inst || !only) return null
  return <Decision docType={docType} id={id} inst={inst} step={only} onError={onError} />
}

function Decision({ docType, id, inst, step, onError }: { docType: string; id: number; inst: ApprovalInstance; step: ApprovalStep; onError: (message: string | null) => void }) {
  const d = useDecide(docType, id, inst, undefined, onError)
  return (
    <>
      <DecisionButtons d={d} step={step} />
      {d.dialogs}
    </>
  )
}

function votingSteps(inst: ApprovalInstance) {
  return inst.steps.filter((s) => s.status === 'open' && s.allowed_actions.includes('approve'))
}

function soleVoting(inst: ApprovalInstance) {
  const v = votingSteps(inst)
  return v.length === 1 ? v[0] : undefined
}

type Decide = ReturnType<typeof useDecide>

// useDecide runs approve, reject and reassign on the step it is asked for, with the modals they
// ask through.
function useDecide(docType: string, id: number, inst: ApprovalInstance, onDone?: () => void, onError?: (message: string | null) => void) {
  const { t } = useTranslation()
  const invalidate = useInvalidateRecord()
  const [error, setOwnError] = useState<string | null>(null)
  const setError = onError ?? setOwnError
  const [busy, setBusy] = useState(false)
  const [modal, setModal] = useState<{ kind: 'reject'; step: number } | { kind: 'reassign'; step: ApprovalStep } | null>(null)
  const path = { path: { id: inst.id } }

  async function decide(action: 'approve' | 'reject' | 'reassign' | 'withdraw', call: () => Promise<unknown>) {
    setError(null)
    setBusy(true)
    try {
      await call()
      await invalidate(docType, id)
      notifySuccess(t(`shared.approval.done.${action}`))
      if (action !== 'reassign') onDone?.()
    } catch (err) {
      setError(errorText(t, err))
      // The instance moved on without us: show its current state instead of buttons that keep failing.
      if (err instanceof ApiError && (err.code === 'approval_stale' || err.code === 'approval_closed')) await invalidate(docType, id)
      throw err
    } finally {
      setBusy(false)
    }
  }

  const approve = (step: number) => void decide('approve', () => unwrap(api.POST('/approvals/{id}/approve', { params: path, body: { step } }))).catch(() => {})
  const dialogs = (
    <>
      {modal?.kind === 'reject' && (
        <TextModal
          title={t('shared.approval.reject_title')}
          field={t('shared.approval.reason')}
          submitLabel={t('shared.approval.reject')}
          multiline
          onClose={() => setModal(null)}
          onSubmit={(reason) => decide('reject', () => unwrap(api.POST('/approvals/{id}/reject', { params: path, body: { step: modal.step, reason } })))}
        />
      )}
      {modal?.kind === 'reassign' && (
        <ReassignModal
          step={modal.step}
          onClose={() => setModal(null)}
          onSubmit={(from, login) =>
            decide('reassign', () => unwrap(api.POST('/approvals/{id}/reassign', { params: path, body: { step: modal.step.position, from, login } })))
          }
        />
      )}
    </>
  )
  const withdraw = () => void decide('withdraw', () => unwrap(api.POST('/approvals/{id}/withdraw', { params: path }))).catch(() => {})
  return { error, busy, approve, withdraw, setModal, dialogs }
}

// namedAria is set only when several steps show their own buttons, so each pair names its step
// for screen readers; the visible label stays the plain "Approve"/"Reject".
function DecisionButtons({ d, step, namedAria }: { d: Decide; step: ApprovalStep; namedAria?: boolean }) {
  const { t } = useTranslation()
  const can = (a: string) => step.allowed_actions.includes(a)
  return (
    <>
      {can('reject') && (
        <Button
          variant="outline"
          color="danger"
          leftSection={<IconX {...icon.button} />}
          disabled={d.busy}
          aria-label={namedAria ? t('shared.approval.reject_step', { n: step.position }) : undefined}
          onClick={() => d.setModal({ kind: 'reject', step: step.position })}
        >
          {t('shared.approval.reject')}
        </Button>
      )}
      {can('approve') && (
        <Button
          leftSection={<IconCheck {...icon.button} />}
          disabled={d.busy}
          aria-label={namedAria ? t('shared.approval.approve_step', { n: step.position }) : undefined}
          onClick={() => d.approve(step.position)}
        >
          {t('shared.approval.approve')}
        </Button>
      )}
    </>
  )
}

function Instance({ docType, id, inst, onDone, sticky, subject, decideElsewhere }: Omit<PanelProps, 'docType' | 'id'> & { docType: string; id: number; inst: ApprovalInstance }) {
  const { t } = useTranslation()
  const me = useMe()
  const d = useDecide(docType, id, inst, onDone)
  const [ask, confirm] = useConfirm()
  const at = formatDateTime(inst.submitted_at, me.timezone)
  const only = soleVoting(inst)
  const byPos = new Map(inst.steps.map((s) => [s.position, s]))
  // A step the plan does not name still shows, after the plan, so none is ever hidden.
  const planned = new Set(leaves(inst.plan))
  const top = [...(inst.plan.seq ?? [inst.plan]).filter((n) => leaves(n).length > 0), ...inst.steps.filter((s) => !planned.has(s.position)).map((s) => ({ step: s.position }))]
  const renderNode = (node: ApprovalPlan): ReactNode => {
    if (node.par)
      return (
        <Stack gap="xs">
          <Text size="xs" tone="dimmed" weight={600}>
            {t('shared.approval.parallel')}
          </Text>
          <Stack gap="sm" className={classes.parallel}>
            {node.par.map((c, i) => (
              <div key={i}>{renderNode(c)}</div>
            ))}
          </Stack>
        </Stack>
      )
    if (node.seq)
      return (
        <Stack gap="sm">
          {node.seq.map((c, i) => (
            <div key={i}>{renderNode(c)}</div>
          ))}
        </Stack>
      )
    const s = node.step === undefined ? undefined : byPos.get(node.step)
    if (!s) return null
    // With several votable steps, each one carries its own decision instead of the shared bar.
    const own = !only && s.status === 'open' && s.allowed_actions.includes('approve')
    return (
      <StepBody
        step={s}
        onReassign={s.allowed_actions.includes('reassign') ? () => d.setModal({ kind: 'reassign', step: s }) : undefined}
        busy={d.busy}
        decision={own ? <DecisionButtons d={d} step={s} namedAria /> : undefined}
      />
    )
  }
  // Withdrawing drops the approvals given so far, so it asks first, though the document is untouched.
  const withdraw = async () => {
    const label = t('shared.approval.withdraw')
    if (await ask({ title: label, message: t('shared.approval.withdraw_confirm'), confirmLabel: label })) d.withdraw()
  }
  return (
    <Stack gap="md">
      {inst.action !== 'post' && (
        <Text size="sm" weight={600}>
          {requestTitle(t, docType, inst.action)}
        </Text>
      )}
      <Text size="sm" tone="dimmed">
        {subject && subject !== inst.submitted_by_name
          ? t('shared.approval.submitted_for', { name: inst.submitted_by_name, subject, at })
          : t('shared.approval.submitted', { name: inst.submitted_by_name, at })}
      </Text>
      {top.length === 0 && (
        <Text size="sm" tone="dimmed">
          {t('shared.approval.no_steps')}
        </Text>
      )}
      <Timeline active={top.length - 1} bulletSize={20} lineWidth={2} role="list" aria-label={t('shared.approval.steps')}>
        {top.map((node, i) => {
          const status = nodeStatus(node, byPos)
          const look = stepLook[status]
          return (
            <Timeline.Item
              key={i}
              role="listitem"
              aria-current={status === 'open' ? 'step' : undefined}
              color={look.color}
              bullet={look.icon && <look.icon size={12} stroke={2.5} aria-hidden />}
              lineVariant={status === 'approved' || status === 'rejected' ? 'solid' : 'dashed'}
            >
              {renderNode(node)}
            </Timeline.Item>
          )
        })}
      </Timeline>
      {inst.status !== 'open' && <Text size="sm">{t(`shared.approval.status.${inst.status}`)}</Text>}
      {d.error && (
        <Notice tone="danger">
          {d.error}
        </Notice>
      )}
      {!decideElsewhere && only && (
        // In the narrow side column the pair splits the width; as a sticky bar it sits on the right.
        <Group gap="xs" grow={!sticky} justify="end" className={sticky ? classes.bar : undefined}>
          <DecisionButtons d={d} step={only} />
        </Group>
      )}
      {inst.allowed_actions.includes('withdraw') && (
        <Group gap="xs" justify="end">
          <Button variant="default" leftSection={<IconArrowBackUp {...icon.button} />} disabled={d.busy} onClick={() => void withdraw()}>
            {t('shared.approval.withdraw')}
          </Button>
        </Group>
      )}
      {d.dialogs}
      {confirm}
    </Stack>
  )
}

// requestTitle names an approval request by the action it waits to run.
function requestTitle(t: TFunction, docType: string, action: string) {
  if (action === 'cancel') return t('shared.approval.request.cancel')
  return t('shared.approval.request.module', { action: actionLabel(t, docType, action) })
}

// stepLook: approved is a green check, rejected a red cross, an open step amber, the rest grey.
const stepLook: Record<ApprovalStep['status'], { color: BadgeProps['color']; icon?: TablerIcon }> = {
  approved: { color: 'success', icon: IconCheck },
  rejected: { color: 'danger', icon: IconX },
  open: { color: 'warning' },
  waiting: { color: 'gray' },
  cancelled: { color: 'gray' },
}

// leaves lists the step positions a plan node names, in order.
function leaves(node: ApprovalPlan): number[] {
  if (node.step !== undefined && !node.seq && !node.par) return [node.step]
  return [...(node.seq ?? []), ...(node.par ?? [])].flatMap(leaves)
}

// nodeStatus is a step's status, or for a group the status its members add up to.
function nodeStatus(node: ApprovalPlan, byPos: Map<number, ApprovalStep>): ApprovalStep['status'] {
  if (node.step !== undefined && !node.seq && !node.par) return byPos.get(node.step)?.status ?? 'waiting'
  const all = [...(node.seq ?? []), ...(node.par ?? [])].map((c) => nodeStatus(c, byPos))
  if (all.includes('rejected')) return 'rejected'
  if (all.includes('open')) return 'open'
  if (all.length > 0 && all.every((x) => x === 'approved')) return 'approved'
  if (all.length > 0 && all.every((x) => x === 'cancelled')) return 'cancelled'
  return 'waiting'
}

// onReassign is set when the actor may hand one of the step's voters' seats to someone else;
// decision holds the step's own approve and reject.
function StepBody({ step, onReassign, busy, decision }: { step: ApprovalStep; onReassign?: () => void; busy: boolean; decision?: ReactNode }) {
  const { t } = useTranslation()
  const me = useMe()
  const approved = step.voters.filter((v) => v.decision === 'approved').length
  return (
    <Stack gap="xxxs">
      <Group gap="xs" wrap={false}>
        <Text size="sm" weight={600}>
          {t('shared.approval.step', { n: step.position })}
        </Text>
        <Badge color={stepLook[step.status].color}>{t(`shared.approval.step_status.${step.status}`)}</Badge>
        {step.overdue && <Badge color="danger">{t('shared.approval.overdue')}</Badge>}
      </Group>
      {step.due_at && step.status === 'open' && (
        <Text size="xs" tone="dimmed">
          {t('shared.approval.due', { at: formatDateTime(step.due_at, me.timezone) })}
        </Text>
      )}
      {step.approvals > 1 && (
        <Text size="xs" tone="dimmed">
          {t('shared.approval.count', { n: approved, m: step.approvals })}
        </Text>
      )}
      {step.voters.map((v) => (
        <Stack key={v.id} gap={0}>
          <Text size="sm">
            {v.cast_by_name ? t('shared.approval.cast_by', { cast_by: v.cast_by_name, name: v.name }) : v.name}
            {v.decision && (
              <Text render={<span />} size="sm" tone={v.decision === 'approved' ? 'success' : 'danger'}>
                {` · ${t(`shared.approval.decision.${v.decision}`)}`}
              </Text>
            )}
          </Text>
          {v.decided_at && (
            <Text size="xs" tone="dimmed">
              {formatDateTime(v.decided_at, me.timezone)}
            </Text>
          )}
          {v.reason && <Text size="sm">{t('shared.approval.reason_value', { reason: v.reason })}</Text>}
        </Stack>
      ))}
      {(step.fallback || onReassign) && (
        <Group gap="xs">
          {step.fallback && (
            <Text size="xs" tone="dimmed">
              {t('shared.approval.fallback')}
            </Text>
          )}
          {onReassign && (
            <Button variant="subtle" leftSection={<IconUserShare {...icon.button} />} disabled={busy} onClick={onReassign}>
              {t('shared.approval.reassign')}
            </Button>
          )}
        </Group>
      )}
      {decision && <Group gap="xs">{decision}</Group>}
    </Stack>
  )
}

// ReassignModal asks whose seat on the step to hand over (only voters yet to vote), then to whom.
function ReassignModal({ step, onClose, onSubmit }: { step: ApprovalStep; onClose: () => void; onSubmit: (from: number, login: string) => Promise<unknown> }) {
  const { t } = useTranslation()
  const pending = step.voters.filter((v) => !v.decision)
  const form = useForm<{ from: string | null; login: string }>({
    mode: 'onBlur',
    defaultValues: { from: pending.length === 1 ? String(pending[0]?.id) : null, login: '' },
  })
  return (
    <FormModal opened title={t('shared.approval.reassign_title')} onClose={onClose}>
      <Form
        form={form}
        onSubmit={async (v) => {
          await onSubmit(Number(v.from), v.login.trim())
          onClose()
        }}
      >
        <SelectField name="from" label={t('shared.approval.reassign_from')} required data={pending.map((p) => ({ value: String(p.id), label: p.name }))} />
        <TextField name="login" label={t('shared.approval.login')} required maxLength={200} />
        <FormActions submitLabel={t('shared.approval.reassign')} onCancel={onClose} cancelLabel={t('shared.confirm.back')} />
      </Form>
    </FormModal>
  )
}

// TextModal asks for one required text: the reason of a rejection.
function TextModal(props: { title: string; field: string; submitLabel: string; multiline?: boolean; onClose: () => void; onSubmit: (v: string) => Promise<unknown> }) {
  const { t } = useTranslation()
  const form = useForm<{ value: string }>({ mode: 'onBlur', defaultValues: { value: '' } })
  return (
    <FormModal opened title={props.title} onClose={props.onClose}>
      <Form
        form={form}
        onSubmit={async (v) => {
          await props.onSubmit(v.value.trim())
          props.onClose()
        }}
      >
        <TextField name="value" label={props.field} required multiline={props.multiline} maxLength={1000} />
        <FormActions submitLabel={props.submitLabel} onCancel={props.onClose} cancelLabel={t('shared.confirm.back')} />
      </Form>
    </FormModal>
  )
}
