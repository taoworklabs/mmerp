import { ActionIcon, Button, Group, Paper, ScrollArea, SimpleGrid, Stack, Table, Text, Tooltip } from '@mantine/core'
import { useMediaQuery } from '@mantine/hooks'
import { IconPlus, IconTrash } from '@tabler/icons-react'
import type { KeyboardEvent, ReactNode } from 'react'
import { useFieldArray, useFormContext } from 'react-hook-form'
import { icon } from '../theme'
import classes from './LinesTable.module.css'

// A column of a document's lines. cell renders the field of line `index`; bare is true in the
// table, where the header names the field, and false on the stacked layout below 1024px.
// line is the line as the form started it (its id and initial values).
export type LineColumn = {
  key: string
  header: string
  numeric?: boolean
  required?: boolean
  // Smallest width in pixels; the column without one takes what is left (200px at least).
  // Too narrow a frame scrolls sideways rather than squeezing the inputs.
  width?: number
  cell: (index: number, bare: boolean, line: Record<string, unknown>) => ReactNode
}

type Props = {
  // The form's array field holding the lines, e.g. "lines".
  name: string
  // Accessible name of the table.
  label: string
  columns: LineColumn[]
  newLine: () => Record<string, unknown>
  addLabel: string
  // Title of line n on the stacked layout, and the remove button's name.
  lineLabel: (n: number) => string
  removeLabel: (n: number) => string
  // Lines that cannot be removed below this count (a document needs one).
  minLines?: number
  // Under the table, right-aligned: usually LinesTotals.
  footer?: ReactNode
}

// LinesTable edits a document's lines: one row per line, typed cells, Tab or Enter to the next
// cell, a line added at the end, a grey remove button per line. Below 1024px each line is a card.
export function LinesTable({ name, label, columns, newLine, addLabel, lineLabel, removeLabel, minLines = 0, footer }: Props) {
  const wide = useMediaQuery('(min-width: 64em)', true)
  const { control } = useFormContext()
  const lines = useFieldArray({ control, name })
  const removable = lines.fields.length > minLines
  const remove = (i: number) => (
    <Tooltip label={removeLabel(i + 1)}>
      <ActionIcon variant="subtle" color="gray" size="input-sm" aria-label={removeLabel(i + 1)} onClick={() => lines.remove(i)}>
        <IconTrash {...icon.button} />
      </ActionIcon>
    </Tooltip>
  )
  const add = (
    <Group>
      <Button variant="light" leftSection={<IconPlus {...icon.button} />} onClick={() => lines.append(newLine())}>
        {addLabel}
      </Button>
    </Group>
  )

  if (!wide)
    return (
      <Stack gap="sm">
        {lines.fields.map((line, i) => (
          <Paper key={line.id} withBorder p="sm">
            <Stack gap="sm">
              <Group justify="space-between" wrap="nowrap">
                <Text fw={600} size="sm">
                  {lineLabel(i + 1)}
                </Text>
                {removable && remove(i)}
              </Group>
              <SimpleGrid cols={1} spacing="sm">
                {columns.map((c) => (
                  <div key={c.key}>{c.cell(i, false, line)}</div>
                ))}
              </SimpleGrid>
            </Stack>
          </Paper>
        ))}
        {add}
        {footer}
      </Stack>
    )

  return (
    <Stack gap="sm">
      <Paper withBorder className={classes.frame}>
        <ScrollArea type="auto">
          <Table aria-label={label} className={classes.table} onKeyDown={nextCellOnEnter} verticalSpacing="xxs">
            <Table.Thead>
              <Table.Tr>
                <Table.Th className={classes.n}>#</Table.Th>
                {columns.map((c) => (
                  <Table.Th key={c.key} miw={c.width ?? 200} ta={c.numeric ? 'right' : undefined}>
                    {c.header}
                    {c.required && (
                      <Text component="span" c="var(--mantine-color-error)" aria-hidden>
                        {' *'}
                      </Text>
                    )}
                  </Table.Th>
                ))}
                <Table.Th className={classes.remove} aria-hidden />
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {lines.fields.map((line, i) => (
                <Table.Tr key={line.id}>
                  <Table.Td className={classes.n}>{i + 1}</Table.Td>
                  {columns.map((c) => (
                    <Table.Td key={c.key}>{c.cell(i, true, line)}</Table.Td>
                  ))}
                  <Table.Td className={classes.remove}>{removable && remove(i)}</Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        </ScrollArea>
      </Paper>
      {add}
      {footer}
    </Stack>
  )
}

// nextCellOnEnter moves Enter to the next input of the table, as in a spreadsheet, instead of
// submitting the form; an open dropdown keeps Enter to pick its option.
function nextCellOnEnter(e: KeyboardEvent<HTMLTableElement>) {
  const target = e.target as HTMLElement
  if (e.key !== 'Enter' || target.tagName !== 'INPUT' || target.getAttribute('aria-expanded') === 'true') return
  e.preventDefault()
  const inputs = [...e.currentTarget.querySelectorAll<HTMLInputElement>('input:not([type=hidden]):not([disabled]):not([readonly])')]
  inputs[inputs.indexOf(target as HTMLInputElement) + 1]?.focus()
}

// LinesTotals sums a document under its lines, right-aligned; the last row is the total, in bold.
// note says where the figures come from (the server computes them when the draft is saved).
export function LinesTotals({ rows, note }: { rows: [string, string][]; note?: string }) {
  return (
    <Stack gap="xxs" align="flex-end">
      {note && (
        <Text size="sm" c="dimmed">
          {note}
        </Text>
      )}
      <dl className={classes.totals}>
        {rows.map(([label, value], i) => (
          <div key={label} className={i === rows.length - 1 ? classes.grand : undefined}>
            <dt>{label}</dt>
            <dd>{value}</dd>
          </div>
        ))}
      </dl>
    </Stack>
  )
}
