import { ActionIcon, Button, Group, Paper, SegmentedControl, SimpleGrid, Stack, Text, Tooltip } from '@mantine/core'
import { IconPlus, IconX } from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { useController, useFieldArray, useFormContext, useWatch } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { api, unwrap } from '@/shared/api/core'
import { DecimalField, OrgUnitField, RemoteSelectField, SelectField } from '@/shared/ui/form'
import { icon } from '@/shared/ui/theme'
import type { CondRow } from '../flowForm'
import { documentKeys } from '../keys'
import type { EditorCtx } from './FlowBlock'

// Groups nest at most this deep, so a condition stays readable.
const maxGroupDepth = 3

const newLeaf = (): CondRow => ({ group: null, field: null, op: null, value: '' })

// ConditionEditor edits a condition group: whether all or any of its rows must hold, and the rows,
// each a leaf (field, comparison, value) or a nested group. A group without rows always holds.
export function ConditionEditor({ path, ctx, depth, onEmpty }: { path: string; ctx: EditorCtx; depth: number; onEmpty?: () => void }) {
  const { t } = useTranslation()
  const { control } = useFormContext()
  const rows = useFieldArray({ control, name: `${path}.rows` as never })
  const mode = useController({ control, name: `${path}.mode` })
  const fields = rows.fields as unknown as (CondRow & { id: string })[]
  const { readOnly } = ctx
  return (
    <Stack gap="xs">
      {depth === 0 && (
        <Text size="sm" fw={600}>
          {t('shared.rule.condition')}
        </Text>
      )}
      {fields.length === 0 && (
        <Text size="sm" c="dimmed">
          {t('shared.rule.always')}
        </Text>
      )}
      {/* With one row there is nothing to combine. */}
      {fields.length > 1 && (
        <SegmentedControl
          w="fit-content"
          readOnly={readOnly}
          value={mode.field.value}
          onChange={mode.field.onChange}
          data={[
            { value: 'all', label: t('shared.rule.cond_all') },
            { value: 'any', label: t('shared.rule.cond_any') },
          ]}
        />
      )}
      {fields.map((r, i) => {
        const rp = `${path}.rows.${i}`
        const removeRow = () => {
          rows.remove(i)
          // Depth 0 (a branch's own condition) has no parent row to drop; an empty group there just always runs.
          if (depth > 0 && fields.length === 1) onEmpty?.()
        }
        const remove = !readOnly && (
          <Tooltip label={t(r.group ? 'shared.rule.remove_group' : 'shared.rule.remove_cond')}>
            <ActionIcon variant="subtle" color="gray" size="input-sm" aria-label={t(r.group ? 'shared.rule.remove_group' : 'shared.rule.remove_cond')} onClick={removeRow}>
              <IconX {...icon.button} />
            </ActionIcon>
          </Tooltip>
        )
        if (r.group)
          return (
            <Paper key={r.id} withBorder p="sm">
              <Group align="flex-start" wrap="nowrap" gap="xs">
                <Stack flex={1} miw={0}>
                  {/* An empty nested group has no valid server form; dropping its last row drops the group itself. */}
                  <ConditionEditor path={`${rp}.group`} ctx={ctx} depth={depth + 1} onEmpty={() => rows.remove(i)} />
                </Stack>
                {remove}
              </Group>
            </Paper>
          )
        return (
          <Group key={r.id} align="flex-end" wrap="nowrap" gap="xs">
            <Leaf path={rp} ctx={ctx} />
            {remove}
          </Group>
        )
      })}
      {!readOnly && (
        <Group gap="xs">
          <Button variant="subtle" leftSection={<IconPlus {...icon.button} />} onClick={() => rows.append(newLeaf() as never)}>
            {t('shared.rule.add_cond')}
          </Button>
          {depth < maxGroupDepth && (
            <Button
              variant="subtle"
              leftSection={<IconPlus {...icon.button} />}
              onClick={() => rows.append({ ...newLeaf(), group: { mode: 'all', rows: [newLeaf()] } } as never)}
            >
              {t('shared.rule.add_group')}
            </Button>
          )}
        </Group>
      )}
    </Stack>
  )
}

// Leaf is one test on an approval field; picking another field clears the comparison and value.
function Leaf({ path, ctx }: { path: string; ctx: EditorCtx }) {
  const { t } = useTranslation()
  const { setValue } = useFormContext()
  const key: string | null = useWatch({ name: `${path}.field` })
  const { readOnly, rule } = ctx
  const field = rule.fields.find((f) => f.key === key)
  const value = { readOnly, name: `${path}.value`, label: t('shared.rule.value'), required: true }
  return (
    <SimpleGrid flex={1} miw={0} cols={{ base: 1, sm: 3 }} spacing="md">
      <SelectField
        readOnly={readOnly}
        name={`${path}.field`}
        label={t('shared.rule.field')}
        required
        data={rule.fields.map((f) => ({ value: f.key, label: t(f.label, { defaultValue: f.key }) }))}
        onChange={(k) => {
          const ops = rule.fields.find((f) => f.key === k)?.ops ?? []
          setValue(`${path}.op`, ops.length === 1 ? ops[0] : null)
          setValue(`${path}.value`, '')
        }}
      />
      {field && (
        <>
          <SelectField readOnly={readOnly} name={`${path}.op`} label={t('shared.rule.op')} required data={field.ops.map((o) => ({ value: o, label: t(`shared.rule.op.${o}`) }))} />
          {field.kind === 'choice' && <SelectField {...value} data={field.options ?? []} />}
          {field.kind === 'number' && <DecimalField {...value} scale={2} />}
          {field.kind === 'org_unit' && <OrgUnitField {...value} />}
          {field.searchable && <SearchedValue key={field.key} {...value} docType={rule.doc_type} field={field.key} />}
        </>
      )}
    </SimpleGrid>
  )
}

// SearchedValue picks one value of a field whose options are searched on the server; the saved
// value's label is fetched once, so the condition reads in full before any search.
function SearchedValue({ docType, field, ...props }: { docType: string; field: string; name: string; label: string; readOnly: boolean; required: boolean }) {
  const { getValues } = useFormContext()
  const [saved] = useState(() => Number(getValues(props.name)) || 0)
  const params = { path: { doc_type: docType, key: field } }
  const options = async (query: { q?: string; ids?: number[] }) =>
    (await unwrap(api.GET('/approval-rules/{doc_type}/fields/{key}/options', { params: { ...params, query } }))) ?? []
  const current = useQuery({
    queryKey: documentKeys.approvalFieldOptions(docType, field, { ids: [saved] }),
    queryFn: () => options({ ids: [saved] }),
    enabled: saved > 0,
  })
  return (
    <RemoteSelectField
      {...props}
      current={current.data?.[0]}
      queryKey={(q) => documentKeys.approvalFieldOptions(docType, field, { q })}
      search={(q) => options({ q: q || undefined })}
    />
  )
}
