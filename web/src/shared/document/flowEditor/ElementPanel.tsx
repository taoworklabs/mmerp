import { Anchor, Button, Group, Stack, Text } from '@mmerp/ui'
import { Collapse, SegmentedControl } from '@mantine/core'
import { IconChevronDown, IconChevronRight, IconPlus } from '@tabler/icons-react'
import { useEffect, useRef, useState, type ReactNode } from 'react'
import { useFormContext, useWatch } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { useConfirm } from '@/shared/ui/confirm'
import { DecimalField, IntegerField, MultiSelectField, SelectField } from '@/shared/ui/form'
import { Panel } from '@/shared/ui/page'
import { icon } from '@/shared/ui/theme'
import { kindAt, newStep, parentOf, type BranchValues, type ElementValues } from '../flowForm'
import { ConditionEditor } from './ConditionEditor'
import { IssueAlert, useFieldProblem, useFlowEditor } from './editor'

// The server caps a step's list of users at 20, and a due time at 720 hours.
const maxUsers = 20
const maxDueHours = 720

// ElementPanel edits what is selected on the diagram, keyed so a structural edit remounts it.
export function ElementPanel() {
  const { selected, rev } = useFlowEditor()
  // Only the selected value, so typing elsewhere does not redraw the panel.
  const value: unknown = useWatch({ name: selected && selected !== 'settings' ? selected : 'max_levels' })
  const kind = selected ? kindAt(selected, value) : null
  const key = `${selected}:${rev}`
  if (!selected || !kind) return <EmptyPanel />
  if (kind === 'settings') return <Focusing key={key}><SettingsPanel /></Focusing>
  if (kind === 'step') return <Focusing key={key}><StepPanel path={selected} /></Focusing>
  if (kind === 'split') return <Focusing key={key}><SplitPanel path={selected} /></Focusing>
  if (kind === 'branch') return <Focusing key={key}><BranchPanel path={selected} /></Focusing>
  return <OtherwisePanel />
}

// Focusing hands the focus to the first input of a panel opened by an insert, instead of leaving it
// on the + that was pressed.
function Focusing({ children }: { children: ReactNode }) {
  const { takeFocus } = useFlowEditor()
  const ref = useRef<HTMLDivElement>(null)
  useEffect(() => {
    if (takeFocus()) ref.current?.querySelector<HTMLElement>('input:not([type=hidden]), button')?.focus()
  }, [takeFocus])
  return <div ref={ref}>{children}</div>
}

function EmptyPanel() {
  const { t } = useTranslation()
  const { ctx } = useFlowEditor()
  return (
    <Panel title={t('shared.rule.panel_title')}>
      <Text size="sm" tone="dimmed">
        {t(ctx.readOnly ? 'shared.rule.panel_hint_readonly' : 'shared.rule.panel_hint')}
      </Text>
    </Panel>
  )
}

// useSiblings tells where an item sits in its list, for the move buttons.
function useSiblings(path: string) {
  const [list, i] = parentOf(path)
  const count = (useWatch({ name: list }) as unknown[] | undefined)?.length ?? 0
  return { first: i === 0, last: i === count - 1, count, index: i }
}

function StepPanel({ path }: { path: string }) {
  const { t } = useTranslation()
  const { ctx, move, remove } = useFlowEditor()
  const { getValues, setValue } = useFormContext()
  const kind: string | null = useWatch({ name: `${path}.kind` })
  const users: string[] = useWatch({ name: `${path}.users` }) ?? []
  const approvals: unknown = useWatch({ name: `${path}.approvals` })
  // Open when the rejections are not the usual 1, or when they are what needs fixing.
  const [advanced, setAdvanced] = useState(() => String(getValues(`${path}.rejections`)) !== '1')
  const { first, last } = useSiblings(path)
  const { readOnly, rule } = ctx
  const problems = {
    role: useFieldProblem(`${path}.role`),
    users: useFieldProblem(`${path}.users`),
    approvals: useFieldProblem(`${path}.approvals`),
    rejections: useFieldProblem(`${path}.rejections`),
    dueHours: useFieldProblem(`${path}.dueHours`),
  }
  const kinds = [...(rule.module_approvers ? ['module'] : []), 'role', 'user']
  return (
    <Panel title={t('shared.rule.step')}>
      <Stack gap="sm">
        <IssueAlert name={`${path}.issue`} />
        <SelectField
          readOnly={readOnly}
          name={`${path}.kind`}
          label={t('shared.rule.approver')}
          // What "direct manager" means is the document type's to say (whose manager, how far up).
          description={kind === 'module' ? t(`${rule.doc_type}.module_approvers`, { defaultValue: '' }) || undefined : undefined}
          required
          data={kinds.map((k) => ({ value: k, label: t(`shared.rule.kind.${k}`) }))}
          // The approvers of the previous kind no longer apply.
          onChange={() => {
            setValue(`${path}.role`, null)
            setValue(`${path}.users`, [])
          }}
        />
        {kind === 'role' && <SelectField readOnly={readOnly} name={`${path}.role`} label={t('shared.rule.role')} required data={ctx.roleOptions} error={problems.role} />}
        {kind === 'user' && (
          <MultiSelectField readOnly={readOnly} name={`${path}.users`} label={t('shared.rule.user')} required searchable max={maxUsers} data={ctx.userOptions} error={problems.users} />
        )}
        {/* One chosen user approves alone, unless the count still asks for more and must be fixed. */}
        {(kind !== 'user' || users.length > 1 || String(approvals) !== '1') && <IntegerField readOnly={readOnly} name={`${path}.approvals`} label={t('shared.rule.approvals')} required min={1} max={20} error={problems.approvals} />}
        <IntegerField readOnly={readOnly} name={`${path}.dueHours`} label={t('shared.rule.due_hours')} description={t('shared.rule.due_hours_hint')} min={1} max={maxDueHours} error={problems.dueHours} />
        <Anchor render={<button />} type="button" size="sm" onClick={() => setAdvanced((v) => !v)} aria-expanded={advanced}>
          <Group gap="xxs" wrap={false}>
            {advanced ? <IconChevronDown {...icon.text} aria-hidden /> : <IconChevronRight {...icon.text} aria-hidden />}
            {t('shared.rule.advanced')}
          </Group>
        </Anchor>
        <Collapse expanded={advanced}>
          <IntegerField readOnly={readOnly} name={`${path}.rejections`} label={t('shared.rule.rejections')} description={t('shared.rule.rejections_hint')} required min={1} max={20} error={problems.rejections} />
        </Collapse>
        {!readOnly && (
          <Group gap="xs">
            {!first && (
              <Button variant="default" onClick={() => move(path, -1)}>
                {t('shared.rule.move_up')}
              </Button>
            )}
            {!last && (
              <Button variant="default" onClick={() => move(path, 1)}>
                {t('shared.rule.move_down')}
              </Button>
            )}
            <Button variant="subtle" color="danger" onClick={() => remove(path)}>
              {t('shared.rule.remove_step')}
            </Button>
          </Group>
        )}
      </Stack>
    </Panel>
  )
}

function SplitPanel({ path }: { path: string }) {
  const { t } = useTranslation()
  const { ctx, insert, remove } = useFlowEditor()
  const { getValues, setValue } = useFormContext()
  const [ask, dialog] = useConfirm()
  const type: ElementValues['type'] = useWatch({ name: `${path}.type` })
  const one = type === 'one'

  // A parallel split has no otherwise; dropping steps from it asks first.
  async function changeType(v: string) {
    if (v === type) return
    if (v === 'all' && (getValues(`${path}.otherwise`) as ElementValues[]).length > 0) {
      const label = t('shared.rule.to_parallel')
      if (!(await ask({ title: label, message: t('shared.rule.drop_otherwise'), confirmLabel: label, danger: true }))) return
    }
    setValue(`${path}.type`, v, { shouldDirty: true })
    if (v === 'all') setValue(`${path}.otherwise`, [], { shouldDirty: true })
  }

  function addBranch() {
    const branch: BranchValues = { cond: { mode: 'all', rows: [] }, body: [newStep(ctx.rule.module_approvers)] }
    insert(`${path}.branches`, (getValues(`${path}.branches`) as unknown[]).length, branch)
  }

  return (
    <Panel title={t(`shared.rule.split.${type}`)}>
      <Stack gap="sm">
        <IssueAlert name={`${path}.issue`} />
        <SegmentedControl
          fullWidth
          readOnly={ctx.readOnly}
          value={type}
          onChange={(v) => void changeType(v)}
          data={[
            { value: 'one', label: t('shared.rule.split.one') },
            { value: 'all', label: t('shared.rule.split.all') },
          ]}
        />
        <Text size="sm" tone="dimmed">
          {t(one ? 'shared.rule.split_hint.one' : 'shared.rule.split_hint.all')}
        </Text>
        {!ctx.readOnly && (
          <Group gap="xs">
            <Button variant="light" leftSection={<IconPlus {...icon.button} />} onClick={addBranch}>
              {t(one ? 'shared.rule.add_case' : 'shared.rule.add_branch')}
            </Button>
            <Button variant="subtle" color="danger" onClick={() => remove(path)}>
              {t('shared.rule.remove')}
            </Button>
          </Group>
        )}
      </Stack>
      {dialog}
    </Panel>
  )
}

function BranchPanel({ path }: { path: string }) {
  const { t } = useTranslation()
  const { ctx, move, remove } = useFlowEditor()
  const [list] = parentOf(path)
  const split = list.slice(0, -'.branches'.length)
  const type: ElementValues['type'] = useWatch({ name: `${split}.type` })
  const one = type === 'one'
  const { first, last, count, index } = useSiblings(path)
  return (
    <Panel title={t(one ? 'shared.rule.case_n' : 'shared.rule.branch', { n: index + 1 })}>
      <Stack gap="sm">
        <Text size="sm" tone="dimmed">
          {t(one ? 'shared.rule.case_hint' : 'shared.rule.branch_hint')}
        </Text>
        <IssueAlert name={`${path}.issue`} />
        <ConditionEditor path={`${path}.cond`} ctx={ctx} depth={0} />
        {!ctx.readOnly && (
          <Group gap="xs">
            {!first && (
              <Button variant="default" onClick={() => move(path, -1)}>
                {t('shared.rule.move_left')}
              </Button>
            )}
            {!last && (
              <Button variant="default" onClick={() => move(path, 1)}>
                {t('shared.rule.move_right')}
              </Button>
            )}
            {/* A split keeps at least one branch. */}
            {count > 1 && (
              <Button variant="subtle" color="danger" onClick={() => remove(path)}>
                {t(one ? 'shared.rule.remove_case' : 'shared.rule.remove_branch')}
              </Button>
            )}
          </Group>
        )}
      </Stack>
    </Panel>
  )
}

function OtherwisePanel() {
  const { t } = useTranslation()
  return (
    <Panel title={t('shared.rule.otherwise')}>
      <Text size="sm" tone="dimmed">
        {t('shared.rule.otherwise_hint')}
      </Text>
    </Panel>
  )
}

function SettingsPanel() {
  const { t } = useTranslation()
  const { ctx } = useFlowEditor()
  const fallbackProblem = useFieldProblem('fallback')
  const levelsProblem = useFieldProblem('max_levels')
  return (
    <Panel title={t('shared.rule.settings')}>
      <Stack gap="sm">
        <IssueAlert name="settings" />
        <SelectField readOnly={ctx.readOnly} name="fallback" label={t('shared.rule.fallback')} description={t('shared.rule.fallback_hint')} required data={ctx.roleOptions} error={fallbackProblem} />
        <DecimalField readOnly={ctx.readOnly} name="max_levels" label={t('shared.rule.max_levels')} required scale={0} min={1} error={levelsProblem} />
      </Stack>
    </Panel>
  )
}
