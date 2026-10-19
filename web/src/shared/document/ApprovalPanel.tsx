import { Alert, Badge, Button, Group, Stack, Text, Timeline } from '@mantine/core'
import { IconAlertCircle, IconCheck, IconUserShare, IconX, type Icon as TablerIcon } from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, ApiError, unwrap, type ApprovalInstance, type ApprovalStep } from '@/shared/api/core'
import { useInvalidateRecord } from '@/shared/area'
import { useMe } from '@/shared/auth/me'
import { errorText, formatDateTime } from '@/shared/i18n'
import { Form, FormActions, SelectField, TextField } from '@/shared/ui/form'
import { notifySuccess } from '@/shared/ui/notify'
import { FormModal } from '@/shared/ui/page'
import { ContentSkeleton, ErrorState } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import classes from './ApprovalPanel.module.css'
import { documentKeys } from './keys'

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
      <Text size="sm" c="dimmed">
        {t('shared.approval.none')}
      </Text>
    )
  return <Instance docType={docType} id={id} inst={inst} onDone={onDone} sticky={sticky} subject={subject} decideElsewhere={decideElsewhere} />
}

// ApprovalDecision is approve and reject for the actor at an open step, for a page header;
// nothing for anyone else. Errors go to the page's banner.
export function ApprovalDecision({ docType, id, onError }: { docType: string; id: number; onError: (message: string | null) => void }) {
  const inst = useApproval(docType, id).data?.instance
  if (!inst) return null
  return <Decision docType={docType} id={id} inst={inst} onError={onError} />
}

function Decision({ docType, id, inst, onError }: { docType: string; id: number; inst: ApprovalInstance; onError: (message: string | null) => void }) {
  const d = useDecide(docType, id, inst, undefined, onError)
  return (
    <>
      <DecisionButtons d={d} />
      {d.dialogs}
    </>
  )
}

type Decide = ReturnType<typeof useDecide>

// useDecide runs approve and reject on the open step the actor may vote on, and reassign on
// the step it is asked for, with the modals they ask through.
function useDecide(docType: string, id: number, inst: ApprovalInstance, onDone?: () => void, onError?: (message: string | null) => void) {
  const { t } = useTranslation()
  const invalidate = useInvalidateRecord()
  const [error, setOwnError] = useState<string | null>(null)
  const setError = onError ?? setOwnError
  const [busy, setBusy] = useState(false)
  const [modal, setModal] = useState<{ kind: 'reject' } | { kind: 'reassign'; step: ApprovalStep } | null>(null)
  // Parallel steps may be open at once; the actor votes on the first one they may vote on.
  const voting = inst.steps.find((s) => s.status === 'open' && s.allowed_actions.includes('approve'))
  const step = voting?.position ?? 0
  const path = { path: { id: inst.id } }

  async function decide(action: 'approve' | 'reject' | 'reassign', call: () => Promise<unknown>) {
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

  const approve = () => void decide('approve', () => unwrap(api.POST('/approvals/{id}/approve', { params: path, body: { step } }))).catch(() => {})
  const dialogs = (
    <>
      {modal?.kind === 'reject' && (
        <TextModal
          title={t('shared.approval.reject_title')}
          field={t('shared.approval.reason')}
          submitLabel={t('shared.approval.reject')}
          multiline
          onClose={() => setModal(null)}
          onSubmit={(reason) => decide('reject', () => unwrap(api.POST('/approvals/{id}/reject', { params: path, body: { step, reason } })))}
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
  const can = (a: string) => !!voting?.allowed_actions.includes(a)
  return { error, busy, approve, can, setModal, dialogs }
}

function DecisionButtons({ d }: { d: Decide }) {
  const { t } = useTranslation()
  return (
    <>
      {d.can('reject') && (
        <Button variant="outline" color="danger" leftSection={<IconX {...icon.button} />} disabled={d.busy} onClick={() => d.setModal({ kind: 'reject' })}>
          {t('shared.approval.reject')}
        </Button>
      )}
      {d.can('approve') && (
        <Button leftSection={<IconCheck {...icon.button} />} disabled={d.busy} onClick={d.approve}>
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
  const at = formatDateTime(inst.submitted_at, me.timezone)
  return (
    <Stack gap="md">
      <Text size="sm" c="dimmed">
        {subject && subject !== inst.submitted_by_name
          ? t('shared.approval.submitted_for', { name: inst.submitted_by_name, subject, at })
          : t('shared.approval.submitted', { name: inst.submitted_by_name, at })}
      </Text>
      <Timeline active={inst.steps.length - 1} bulletSize={20} lineWidth={2} role="list" aria-label={t('shared.approval.steps')}>
        {inst.steps.map((s) => {
          const look = stepLook[s.status]
          return (
            <Timeline.Item
              key={s.position}
              role="listitem"
              aria-current={s.status === 'open' ? 'step' : undefined}
              color={look.color}
              bullet={look.icon && <look.icon size={12} stroke={2.5} aria-hidden />}
              lineVariant={s.status === 'approved' || s.status === 'rejected' ? 'solid' : 'dashed'}
            >
              <StepBody step={s} onReassign={s.allowed_actions.includes('reassign') ? () => d.setModal({ kind: 'reassign', step: s }) : undefined} busy={d.busy} />
            </Timeline.Item>
          )
        })}
      </Timeline>
      {inst.status !== 'open' && <Text size="sm">{t(`shared.approval.status.${inst.status}`)}</Text>}
      {d.error && (
        <Alert color="danger" icon={<IconAlertCircle {...icon.button} />} role="alert">
          {d.error}
        </Alert>
      )}
      {!decideElsewhere && (d.can('reject') || d.can('approve')) && (
        // In the narrow side column the pair splits the width; as a sticky bar it sits on the right.
        <Group gap="xs" grow={!sticky} justify="flex-end" className={sticky ? classes.bar : undefined}>
          <DecisionButtons d={d} />
        </Group>
      )}
      {d.dialogs}
    </Stack>
  )
}

// stepLook: approved is a green check, rejected a red cross, an open step amber, the rest grey.
const stepLook: Record<ApprovalStep['status'], { color: string; icon?: TablerIcon }> = {
  approved: { color: 'success', icon: IconCheck },
  rejected: { color: 'danger', icon: IconX },
  open: { color: 'warning' },
  waiting: { color: 'gray' },
  cancelled: { color: 'gray' },
}

// onReassign is set when the actor may hand one of the step's voters' seats to someone else.
function StepBody({ step, onReassign, busy }: { step: ApprovalStep; onReassign?: () => void; busy: boolean }) {
  const { t } = useTranslation()
  const me = useMe()
  const approved = step.voters.filter((v) => v.decision === 'approved').length
  return (
    <Stack gap="xxxs">
      <Group gap="xs" wrap="nowrap">
        <Text size="sm" fw={600}>
          {t('shared.approval.step', { n: step.position })}
        </Text>
        <Badge color={stepLook[step.status].color}>{t(`shared.approval.step_status.${step.status}`)}</Badge>
      </Group>
      {step.approvals > 1 && (
        <Text size="xs" c="dimmed">
          {t('shared.approval.count', { n: approved, m: step.approvals })}
        </Text>
      )}
      {step.voters.map((v) => (
        <Stack key={v.id} gap={0}>
          <Text size="sm">
            {v.name}
            {v.decision && (
              <Text span size="sm" c={v.decision === 'approved' ? 'success' : 'danger'}>
                {` · ${t(`shared.approval.decision.${v.decision}`)}`}
              </Text>
            )}
          </Text>
          {v.decided_at && (
            <Text size="xs" c="dimmed">
              {formatDateTime(v.decided_at, me.timezone)}
            </Text>
          )}
          {v.reason && <Text size="sm">{t('shared.approval.reason_value', { reason: v.reason })}</Text>}
        </Stack>
      ))}
      {(step.fallback || onReassign) && (
        <Group gap="xs">
          {step.fallback && (
            <Text size="xs" c="dimmed">
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
