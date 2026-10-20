import { Box, SimpleGrid, Stack } from '@mantine/core'
import { useEffect } from 'react'
import { get, useFormContext, useWatch } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { fieldId, IntegerField, MultiSelectField, SelectField } from '@/shared/ui/form'
import { Panel } from '@/shared/ui/page'
import { CardMenu, IssueAlert, type EditorCtx } from './FlowBlock'

// The server caps both counts, and a step's list of users, at 20.
const maxCount = 20

type Props = { path: string; ctx: EditorCtx; onMove: (by: -1 | 1) => void; onRemove: () => void; first: boolean; last: boolean }

// maxDueHours is the server's limit on a step's due time, 30 days.
const maxDueHours = 720

// StepCard edits one approval step: who approves, how many approvals or rejections decide it,
// and an optional due time after which approvers and the submitter are told.
export function StepCard({ path, ctx, onMove, onRemove, first, last }: Props) {
  const { t } = useTranslation()
  const { getFieldState, setValue, trigger } = useFormContext()
  const kind: string | null = useWatch({ name: `${path}.kind` })
  const users: string[] | undefined = useWatch({ name: `${path}.users` })
  // The approvals check depends on the users; recheck it once it has failed.
  useEffect(() => {
    if (getFieldState(`${path}.approvals`).error) void trigger(`${path}.approvals`)
  }, [users, path, getFieldState, trigger])
  const { readOnly, rule } = ctx
  const kinds = [...(rule.module_approvers ? ['module'] : []), 'role', 'user']
  const range = { min: { value: 1, message: t('shared.rule.count_range') }, max: { value: maxCount, message: t('shared.rule.count_range') } }
  const title = t('shared.rule.step')
  return (
    <Box id={fieldId(path)}>
      <Panel title={title} actions={!readOnly && <CardMenu first={first} last={last} onMove={onMove} onRemove={onRemove} removeLabel={t('shared.rule.remove_step')} />}>
        <Stack gap="sm">
          <IssueAlert path={path} />
          <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="md">
            <SelectField
              readOnly={readOnly}
              name={`${path}.kind`}
              label={t('shared.rule.approver')}
              required
              data={kinds.map((k) => ({ value: k, label: t(`shared.rule.kind.${k}`) }))}
              // The approvers of the previous kind no longer apply.
              onChange={() => {
                setValue(`${path}.role`, null)
                setValue(`${path}.users`, [])
              }}
            />
            {kind === 'role' && <SelectField readOnly={readOnly} name={`${path}.role`} label={t('shared.rule.role')} required data={ctx.roleOptions} />}
            {kind === 'user' && (
              <MultiSelectField readOnly={readOnly} name={`${path}.users`} label={t('shared.rule.user')} required searchable max={maxCount} data={ctx.userOptions} />
            )}
          </SimpleGrid>
          <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="md">
            <IntegerField
              readOnly={readOnly}
              name={`${path}.approvals`}
              label={t('shared.rule.approvals')}
              required
              min={1}
              max={maxCount}
              rules={{
                ...range,
                validate: (v, values) => {
                  const n = (get(values, `${path}.users`) as string[] | undefined)?.length ?? 0
                  return get(values, `${path}.kind`) !== 'user' || n === 0 || Number(v) <= n || t('shared.rule.reason.approvals_exceed_users')
                },
              }}
            />
            <IntegerField readOnly={readOnly} name={`${path}.rejections`} label={t('shared.rule.rejections')} required min={1} max={maxCount} rules={range} />
            <IntegerField
              readOnly={readOnly}
              name={`${path}.dueHours`}
              label={t('shared.rule.due_hours')}
              description={t('shared.rule.due_hours_hint')}
              min={1}
              max={maxDueHours}
              rules={{
                min: { value: 1, message: t('shared.rule.due_range') },
                max: { value: maxDueHours, message: t('shared.rule.due_range') },
              }}
            />
          </SimpleGrid>
        </Stack>
      </Panel>
    </Box>
  )
}
