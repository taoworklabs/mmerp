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
import { Form, FormActions, TextField } from '@/shared/ui/form'
import { notifySuccess } from '@/shared/ui/notify'
import { FormModal } from '@/shared/ui/page'
import { ContentSkeleton, ErrorState } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import classes from './ApprovalPanel.module.css'
import { documentKeys } from './keys'

// ApprovalPanel shows the latest submission's steps and, for the actor at the current
// step, approve, reject and reassign. onDone runs after a decision (e.g. to leave the inbox item);
// sticky keeps approve and reject at the bottom of the screen where the panel is the page's main task.
export function ApprovalPanel({ docType, id, onDone, sticky }: { docType: string; id: number; onDone?: () => void; sticky?: boolean }) {
  const { t } = useTranslation()
  const q = useQuery({
    queryKey: documentKeys.approval(docType, id),
    queryFn: () => unwrap(api.GET('/documents/{type}/{id}/approval', { params: { path: { type: docType, id } } })),
  })
  if (q.isPending) return <ContentSkeleton />
  if (q.isError) return <ErrorState message={errorText(t, q.error)} onRetry={() => void q.refetch()} />
  const inst = q.data.instance
  if (!inst)
    return (
      <Text size="sm" c="dimmed">
        {t('shared.approval.none')}
      </Text>
    )
  return <Instance docType={docType} id={id} inst={inst} onDone={onDone} sticky={sticky} />
}

function Instance({ docType, id, inst, onDone, sticky }: { docType: string; id: number; inst: ApprovalInstance; onDone?: () => void; sticky?: boolean }) {
  const { t } = useTranslation()
  const me = useMe()
  const invalidate = useInvalidateRecord()
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [modal, setModal] = useState<'reject' | 'reassign' | null>(null)
  const step = inst.current_step

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

  const path = { path: { id: inst.id } }
  const has = (a: string) => inst.allowed_actions.includes(a)
  return (
    <Stack gap="md">
      <Text size="sm" c="dimmed">
        {t('shared.approval.submitted', { name: inst.submitted_by_name, at: formatDateTime(inst.submitted_at, me.timezone) })}
      </Text>
      <Timeline active={inst.steps.length - 1} bulletSize={20} lineWidth={2} role="list" aria-label={t('shared.approval.steps')}>
        {inst.steps.map((s) => {
          const current = inst.status === 'open' && s.position === step
          const look = stepLook(s, current)
          return (
            <Timeline.Item
              key={s.position}
              role="listitem"
              aria-current={current ? 'step' : undefined}
              color={look.color}
              bullet={look.icon && <look.icon size={12} stroke={2.5} aria-hidden />}
              lineVariant={s.decision ? 'solid' : 'dashed'}
            >
              <StepBody step={s} current={current} onReassign={current && has('reassign') ? () => setModal('reassign') : undefined} busy={busy} />
            </Timeline.Item>
          )
        })}
      </Timeline>
      {inst.status !== 'open' && <Text size="sm">{t(`shared.approval.status.${inst.status}`)}</Text>}
      {error && (
        <Alert color="danger" icon={<IconAlertCircle {...icon.button} />} role="alert">
          {error}
        </Alert>
      )}
      {(has('reject') || has('approve')) && (
        // In the narrow side column the pair splits the width; as a sticky bar it sits on the right.
        <Group gap="xs" grow={!sticky} justify="flex-end" className={sticky ? classes.bar : undefined}>
          {has('reject') && (
            <Button size="xs" variant="outline" color="danger" leftSection={<IconX {...icon.text} />} disabled={busy} onClick={() => setModal('reject')}>
              {t('shared.approval.reject')}
            </Button>
          )}
          {has('approve') && (
            <Button
              size="xs"
              leftSection={<IconCheck {...icon.text} />}
              disabled={busy}
              onClick={() => void decide('approve', () => unwrap(api.POST('/approvals/{id}/approve', { params: path, body: { step } }))).catch(() => {})}
            >
              {t('shared.approval.approve')}
            </Button>
          )}
        </Group>
      )}
      {modal === 'reject' && (
        <TextModal
          title={t('shared.approval.reject_title')}
          field={t('shared.approval.reason')}
          submitLabel={t('shared.approval.reject')}
          multiline
          onClose={() => setModal(null)}
          onSubmit={(reason) => decide('reject', () => unwrap(api.POST('/approvals/{id}/reject', { params: path, body: { step, reason } })))}
        />
      )}
      {modal === 'reassign' && (
        <TextModal
          title={t('shared.approval.reassign_title')}
          field={t('shared.approval.login')}
          submitLabel={t('shared.approval.reassign')}
          onClose={() => setModal(null)}
          onSubmit={(login) => decide('reassign', () => unwrap(api.POST('/approvals/{id}/reassign', { params: path, body: { step, login } })))}
        />
      )}
    </Stack>
  )
}

// stepLook: approved is a green check, rejected a red cross, the waiting step amber, later steps grey.
function stepLook(step: ApprovalStep, current: boolean): { color: string; icon?: TablerIcon } {
  if (step.decision === 'approved') return { color: 'success', icon: IconCheck }
  if (step.decision === 'rejected') return { color: 'danger', icon: IconX }
  return { color: current ? 'warning' : 'gray' }
}

// onReassign is set on the current step when the actor may hand it to someone else.
function StepBody({ step, current, onReassign, busy }: { step: ApprovalStep; current: boolean; onReassign?: () => void; busy: boolean }) {
  const { t } = useTranslation()
  const me = useMe()
  return (
    <Stack gap={2}>
      <Group gap="xs" wrap="nowrap">
        <Text size="sm" fw={600}>
          {t('shared.approval.step', { n: step.position })}
        </Text>
        {step.decision ? (
          <Badge color={step.decision === 'approved' ? 'success' : 'danger'}>{t(`shared.approval.decision.${step.decision}`)}</Badge>
        ) : (
          current && <Badge color="warning">{t('shared.approval.waiting')}</Badge>
        )}
      </Group>
      <Group gap="xs">
        <Text size="sm">
          {step.approver_names.join(', ')}
          {step.fallback && ` · ${t('shared.approval.fallback')}`}
        </Text>
        {onReassign && (
          <Button size="compact-xs" variant="subtle" leftSection={<IconUserShare {...icon.text} />} disabled={busy} onClick={onReassign}>
            {t('shared.approval.reassign')}
          </Button>
        )}
      </Group>
      {step.decided_by_name && (
        <Text size="xs" c="dimmed">
          {t('shared.approval.decided', { name: step.decided_by_name, at: formatDateTime(step.decided_at, me.timezone) })}
        </Text>
      )}
      {step.reason && <Text size="sm">{t('shared.approval.reason_value', { reason: step.reason })}</Text>}
    </Stack>
  )
}

// TextModal asks for one required text: the reason of a rejection, the login of a new approver.
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
