import { ActionIcon, Button, FieldList, Group, Paper, ScrollArea, SimpleGrid, Stack, Text, Tooltip, useMediaQuery } from '@mmerp/ui'
import { Table } from '@mantine/core'
import { IconPlus, IconTrash } from '@tabler/icons-react'
import type { KeyboardEvent, ReactNode } from 'react'
import { useFieldArray, useFormContext } from 'react-hook-form'
import { icon } from '../theme'
import classes from './LinesTable.module.css'

// A column of a document's lines. cell renders the field of line `index`; bare is true in the
// table, where the header names the field, and false on the stacked layout below 1024px.
// line is the line as the form started it (its id and initial values). value is the column's
// text on read-only lines. A column without cell shows only on read-only lines (a stored
// amount), one without value only while editing.
export type LineColumn<L extends object = Record<string, unknown>> = {
  key: string
  header: string
  numeric?: boolean
  required?: boolean
  // Smallest width in pixels; the column without one takes what is left (200px at least).
  // Too narrow a frame scrolls sideways rather than squeezing the inputs.
  width?: number
  cell?: (index: number, bare: boolean, line: Record<string, unknown>) => ReactNode
  value?: (line: L, index: number) => ReactNode
}

type Common<L extends object> = {
  // Accessible name of the table.
  label: string
  columns: LineColumn<L>[]
  // Title of line n on the stacked layout.
  lineLabel: (n: number) => string
  // Under the table, right-aligned: usually LinesTotals.
  footer?: ReactNode
}

type EditProps<L extends object> = Common<L> & {
  readOnly?: false
  // The form's array field holding the lines, e.g. "lines".
  name: string
  newLine: () => Record<string, unknown>
  addLabel: string
  // The remove button's name for line n.
  removeLabel: (n: number) => string
  // Lines that cannot be removed below this count (a document needs one).
  minLines?: number
}

// Read-only: the saved lines as text, with the same columns; no form needed.
type ReadOnlyProps<L extends object> = Common<L> & { readOnly: true; lines: L[] }

// LinesTable edits a document's lines: one row per line, typed cells, Tab or Enter to the next
// cell, a line added at the end, a grey remove button per line. Below 1024px each line is a card.
// readOnly shows saved lines in the same columns, as text.
export function LinesTable<L extends object = Record<string, unknown>>(props: EditProps<L> | ReadOnlyProps<L>) {
  return props.readOnly ? <ReadOnlyLines {...props} /> : <EditLines {...props} />
}

function EditLines<L extends object>({ name, label, columns: all, newLine, addLabel, lineLabel, removeLabel, minLines = 0, footer }: EditProps<L>) {
  const wide = useMediaQuery('(min-width: 64em)', true)
  const { control } = useFormContext()
  const lines = useFieldArray({ control, name })
  const columns = all.filter((c) => c.cell)
  const removable = lines.fields.length > minLines
  const remove = (i: number) => (
    <Tooltip label={removeLabel(i + 1)}>
      <ActionIcon variant="subtle" color="gray" aria-label={removeLabel(i + 1)} onClick={() => lines.remove(i)}>
        <IconTrash {...icon.button} />
      </ActionIcon>
    </Tooltip>
  )
  const add = (
    <Group className={classes.add}>
      <Button variant="light" leftSection={<IconPlus {...icon.button} />} onClick={() => lines.append(newLine())}>
        {addLabel}
      </Button>
    </Group>
  )

  if (!wide)
    return (
      <Stack gap="sm">
        {lines.fields.map((line, i) => (
          <Paper key={line.id} withBorder padding="sm">
            <Stack gap="sm">
              <Group justify="space-between" wrap={false}>
                <Text weight={600} size="sm">
                  {lineLabel(i + 1)}
                </Text>
                {removable && remove(i)}
              </Group>
              <SimpleGrid cols={1} spacing="sm">
                {columns.map((c) => (
                  <div key={c.key}>{c.cell?.(i, false, line)}</div>
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
        <ScrollArea>
          <Table aria-label={label} className={classes.table} onKeyDown={nextCellOnEnter} verticalSpacing="xxs">
            <Head columns={columns} />
            <Table.Tbody>
              {lines.fields.map((line, i) => (
                <Table.Tr key={line.id}>
                  <Table.Td className={classes.n}>{i + 1}</Table.Td>
                  {columns.map((c) => (
                    <Table.Td key={c.key}>{c.cell?.(i, true, line)}</Table.Td>
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

function ReadOnlyLines<L extends object>({ label, columns: all, lines, lineLabel, footer }: ReadOnlyProps<L>) {
  const wide = useMediaQuery('(min-width: 64em)', true)
  const columns = all.filter((c) => c.value)
  return (
    <Stack gap="sm">
      {wide ? (
        <Paper withBorder className={classes.frame}>
          <ScrollArea>
            <Table aria-label={label} className={classes.table} data-readonly>
              <Head columns={columns} readOnly />
              <Table.Tbody>
                {lines.map((line, i) => (
                  <Table.Tr key={i}>
                    <Table.Td className={classes.n}>{i + 1}</Table.Td>
                    {columns.map((c) => (
                      <Table.Td key={c.key} ta={c.numeric ? 'right' : undefined}>
                        {c.value?.(line, i)}
                      </Table.Td>
                    ))}
                  </Table.Tr>
                ))}
              </Table.Tbody>
            </Table>
          </ScrollArea>
        </Paper>
      ) : (
        lines.map((line, i) => (
          <Paper key={i} withBorder padding="sm">
            <Stack gap="xs">
              <Text weight={600} size="sm">
                {lineLabel(i + 1)}
              </Text>
              <FieldList rows={columns.map((c) => [c.header, c.value?.(line, i)])} />
            </Stack>
          </Paper>
        ))
      )}
      {footer}
    </Stack>
  )
}

// Head is the header row; read-only lines take only their text's width and mark nothing required.
function Head<L extends object>({ columns, readOnly }: { columns: LineColumn<L>[]; readOnly?: boolean }) {
  return (
    <Table.Thead>
      <Table.Tr>
        <Table.Th className={classes.n}>#</Table.Th>
        {columns.map((c) => (
          <Table.Th key={c.key} miw={readOnly ? undefined : (c.width ?? 200)} ta={c.numeric ? 'right' : undefined}>
            {c.header}
            {c.required && !readOnly && (
              <Text render={<span />} tone="danger" aria-hidden>
                {' *'}
              </Text>
            )}
          </Table.Th>
        ))}
        {!readOnly && <Table.Th className={classes.remove} aria-hidden />}
      </Table.Tr>
    </Table.Thead>
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
    <Stack gap="xxs" align="end">
      {note && (
        <Text size="sm" tone="dimmed">
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
