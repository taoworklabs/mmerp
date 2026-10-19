import { Alert, Button, Group, Menu, Stack, Text } from '@mantine/core'
import { IconAlertCircle, IconChevronDown, IconPlus } from '@tabler/icons-react'
import { get, useFieldArray, useFormContext, useFormState } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import type { TypeRule } from '@/shared/api/core'
import { fieldId, type Option } from '@/shared/ui/form'
import { MoreActions } from '@/shared/ui/MoreActions'
import { icon } from '@/shared/ui/theme'
import { newSplit, newStep, type ElementValues } from '../flowForm'
import { SplitCard } from './SplitCard'
import { StepCard } from './StepCard'

export type EditorCtx = { rule: TypeRule; roleOptions: Option[]; userOptions: Option[]; readOnly: boolean }

// The server allows at most this many splits nested in one another.
const maxDepth = 5

// FlowBlock edits one flow (the top-level flow, a branch's body or an otherwise): its elements in
// order, then the Add menu. depth is the number of splits it sits inside.
export function FlowBlock({ path, ctx, depth }: { path: string; ctx: EditorCtx; depth: number }) {
  const { t } = useTranslation()
  const { control } = useFormContext()
  const top = depth === 0
  const arr = useFieldArray({
    control,
    name: path as never,
    // Only the top-level flow must hold something: an empty branch body just goes on.
    rules: top ? { minLength: { value: 1, message: t('shared.rule.flow_required') } } : undefined,
  })
  const { errors } = useFormState({ name: path })
  const rootError: string | undefined = top ? get(errors, `${path}.root`)?.message : undefined
  const fields = arr.fields as unknown as (ElementValues & { id: string })[]
  const add = (e: ElementValues) => arr.append(e as never)
  const m = ctx.rule.module_approvers
  return (
    <Stack gap="sm">
      {rootError && (
        <Alert id={fieldId(`${path}.root`)} tabIndex={-1} color="danger" icon={<IconAlertCircle {...icon.button} />} role="alert">
          {rootError}
        </Alert>
      )}
      {fields.length === 0 && (
        <Text size="sm" c="dimmed">
          {t('shared.rule.empty_block')}
        </Text>
      )}
      {fields.map((f, i) => {
        const p = `${path}.${i}`
        const card = {
          path: p,
          ctx,
          first: i === 0,
          last: i === fields.length - 1,
          onMove: (by: -1 | 1) => arr.move(i, i + by),
          onRemove: () => arr.remove(i),
        }
        return f.type === 'step' ? <StepCard key={f.id} {...card} /> : <SplitCard key={f.id} {...card} depth={depth} />
      })}
      {!ctx.readOnly && (
        <Group>
          <Menu position="bottom-start">
            <Menu.Target>
              <Button variant="light" leftSection={<IconPlus {...icon.button} />} rightSection={<IconChevronDown {...icon.button} />}>
                {t('shared.rule.add')}
              </Button>
            </Menu.Target>
            <Menu.Dropdown>
              <Menu.Item onClick={() => add(newStep(m))}>{t('shared.rule.step')}</Menu.Item>
              {depth < maxDepth && (
                <>
                  <Menu.Item onClick={() => add(newSplit('one', m))}>{t('shared.rule.split.one')}</Menu.Item>
                  <Menu.Item onClick={() => add(newSplit('all', m))}>{t('shared.rule.split.all')}</Menu.Item>
                </>
              )}
            </Menu.Dropdown>
          </Menu>
        </Group>
      )}
    </Stack>
  )
}

// CardMenu is a card's "…": move within its block, or delete it.
// onRemove is left out where the card must stay (a split's only branch).
export function CardMenu({ first, last, onMove, onRemove, removeLabel }: { first: boolean; last: boolean; onMove: (by: -1 | 1) => void; onRemove?: () => void; removeLabel: string }) {
  const { t } = useTranslation()
  if (first && last && !onRemove) return null
  return (
    <MoreActions>
      {!first && <Menu.Item onClick={() => onMove(-1)}>{t('shared.rule.move_up')}</Menu.Item>}
      {!last && <Menu.Item onClick={() => onMove(1)}>{t('shared.rule.move_down')}</Menu.Item>}
      {onRemove && (
        <Menu.Item color="danger" onClick={onRemove}>
          {removeLabel}
        </Menu.Item>
      )}
    </MoreActions>
  )
}

// IssueAlert shows the server's complaint about one element or branch, at the top of its card.
export function IssueAlert({ path }: { path: string }) {
  const { errors } = useFormState({ name: `${path}.issue` })
  const issue: string | undefined = get(errors, `${path}.issue`)?.message
  if (!issue) return null
  return (
    <Alert id={fieldId(`${path}.issue`)} tabIndex={-1} color="danger" icon={<IconAlertCircle {...icon.button} />} role="alert">
      {issue}
    </Alert>
  )
}
