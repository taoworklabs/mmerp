import { Box, Button, Group, Stack } from '@mantine/core'
import { IconPlus } from '@tabler/icons-react'
import { get, useFieldArray, useFormContext, useWatch } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { useConfirm } from '@/shared/ui/confirm'
import { fieldId, SelectField } from '@/shared/ui/form'
import { Panel } from '@/shared/ui/page'
import { icon } from '@/shared/ui/theme'
import { newStep, type BranchValues, type ElementValues } from '../flowForm'
import { ConditionEditor } from './ConditionEditor'
import { CardMenu, FlowBlock, IssueAlert, type EditorCtx } from './FlowBlock'

type Props = { path: string; ctx: EditorCtx; depth: number; onMove: (by: -1 | 1) => void; onRemove: () => void; first: boolean; last: boolean }

// SplitCard edits a choose-one or parallel split: its branches, each a condition over a flow, and
// for choose-one the otherwise flow that runs when no branch holds.
export function SplitCard({ path, ctx, depth, onMove, onRemove, first, last }: Props) {
  const { t } = useTranslation()
  const { control, formState, getValues, resetField, setValue } = useFormContext()
  const [ask, dialog] = useConfirm()
  const type: ElementValues['type'] = useWatch({ name: `${path}.type` })
  const branches = useFieldArray({ control, name: `${path}.branches` as never })
  const { readOnly } = ctx

  // Back to choose-one after a cancelled switch, clean again when that is what was saved.
  function restoreOne() {
    const name = `${path}.type`
    if (get(formState.defaultValues, name) === 'one') resetField(name)
    else setValue(name, 'one')
  }

  // A parallel split has no otherwise; dropping steps from it asks first.
  async function changeType(v: string | null) {
    if (v !== 'all') return
    const otherwise = getValues(`${path}.otherwise`) as ElementValues[]
    if (otherwise.length > 0) {
      const label = t('shared.rule.to_parallel')
      if (!(await ask({ title: label, message: t('shared.rule.drop_otherwise'), confirmLabel: label, danger: true }))) return restoreOne()
    }
    setValue(`${path}.otherwise`, [], { shouldDirty: true })
  }

  const newBranch = (): BranchValues => ({ cond: { mode: 'all', rows: [] }, body: [newStep(ctx.rule.module_approvers)] })
  return (
    <Box id={fieldId(path)}>
      <Panel
        title={t(`shared.rule.split.${type}`)}
        actions={!readOnly && <CardMenu first={first} last={last} onMove={onMove} onRemove={onRemove} removeLabel={t('shared.rule.remove_split')} />}
      >
        <Stack gap="sm">
          <IssueAlert path={path} />
          <SelectField
            readOnly={readOnly}
            name={`${path}.type`}
            label={t('shared.rule.split_type')}
            required
            short
            data={(['one', 'all'] as const).map((v) => ({ value: v, label: t(`shared.rule.split.${v}`) }))}
            onChange={(v) => void changeType(v)}
          />
          {branches.fields.map((b, i) => {
            const bp = `${path}.branches.${i}`
            return (
              <Box key={b.id} id={fieldId(bp)}>
                <Panel
                  title={t('shared.rule.branch', { n: i + 1 })}
                  actions={
                    !readOnly && (
                      <CardMenu
                        first={i === 0}
                        last={i === branches.fields.length - 1}
                        onMove={(by) => branches.move(i, i + by)}
                        // A split keeps at least one branch.
                        onRemove={branches.fields.length > 1 ? () => branches.remove(i) : undefined}
                        removeLabel={t('shared.rule.remove_branch')}
                      />
                    )
                  }
                >
                  <Stack gap="md">
                    <IssueAlert path={bp} />
                    <ConditionEditor path={`${bp}.cond`} ctx={ctx} depth={0} />
                    <FlowBlock path={`${bp}.body`} ctx={ctx} depth={depth + 1} />
                  </Stack>
                </Panel>
              </Box>
            )
          })}
          {!readOnly && (
            <Group>
              <Button variant="light" leftSection={<IconPlus {...icon.button} />} onClick={() => branches.append(newBranch() as never)}>
                {t('shared.rule.add_branch')}
              </Button>
            </Group>
          )}
          {type === 'one' && (
            <Panel title={t('shared.rule.otherwise')}>
              <FlowBlock path={`${path}.otherwise`} ctx={ctx} depth={depth + 1} />
            </Panel>
          )}
        </Stack>
      </Panel>
      {dialog}
    </Box>
  )
}
