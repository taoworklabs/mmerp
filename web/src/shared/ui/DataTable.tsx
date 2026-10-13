import { Checkbox, Group, Paper, ScrollArea, Stack, Table, Text, UnstyledButton } from '@mantine/core'
import { useMediaQuery } from '@mantine/hooks'
import { IconArrowDown, IconArrowUp, IconArrowsSort } from '@tabler/icons-react'
import { Fragment, useRef, type KeyboardEvent, type PointerEvent, type ReactNode, type RefObject } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, useNavigate } from 'react-router'
import classes from './DataTable.module.css'
import { MoreActions } from './MoreActions'
import { icon } from './theme'

// Role decides where a column goes on the card list below 1024px.
export type ColumnRole = 'title' | 'status' | 'meta' | 'hidden'

export type Column<T> = {
  key: string // also the sort key sent to the API
  header: string
  render: (row: T) => ReactNode
  role: ColumnRole
  sortable?: boolean
  numeric?: boolean
  // Columns sharing a band sit under one heading, the band set off by a rule on its left.
  band?: string
  // A result column (net pay, cost): bold.
  strong?: boolean
  // Width in pixels, from the user's list settings; the table sizes the column otherwise.
  width?: number
}

// Selection of rows for a bulk action: the keys of the selected rows on this page.
export type Selection = { selected: ReadonlySet<string | number>; onChange: (next: Set<string | number>) => void }

type Props<T> = {
  columns: Column<T>[]
  rows: T[]
  rowKey: (row: T) => string | number
  // The first column links here and the whole row is clickable; without it rows are plain.
  href?: (row: T) => string
  // Clicking a row outside its link runs this instead of following href: a list's side panel, or
  // the edit modal of a settings row that has no page (its first cell becomes the keyboard's button).
  onRowClick?: (row: T) => void
  // A checkbox column for bulk actions; wide screens only.
  selection?: Selection
  // Lets the user drag a column's right edge (or use arrow keys on it) to set its width.
  onResize?: (key: string, width: number) => void
  // Secondary actions of a row, as menu items of the "…" menu at its end; null for none.
  menu?: (row: T) => ReactNode
  // "key" ascending, "-key" descending.
  sort?: string
  onSort?: (sort: string) => void
  label: string // accessible name of the table
  // Consecutive rows of a group follow a row naming it; total, when given, sums each group in a row after it.
  group?: { of: (row: T) => string; total?: (rows: T[], name: string) => T }
  // A summary row after every row, e.g. the grand total.
  footer?: T
  // Caps the table's height (e.g. "70dvh") so the header and footer stay in view inside it.
  maxHeight?: string
}

// grouped splits rows into runs of the same group, in order; one run without a name when not grouped.
function grouped<T>(rows: T[], of?: (row: T) => string): { name: string; rows: T[] }[] {
  const out: { name: string; rows: T[] }[] = []
  for (const row of rows) {
    const name = of ? of(row) : ''
    const last = out[out.length - 1]
    if (last && last.name === name) last.rows.push(row)
    else out.push({ name, rows: [row] })
  }
  return out
}

// DataTable is the only table areas use: compact, sortable on the server, a card list below 1024px.
export function DataTable<T>({ columns, rows, rowKey, href, onRowClick, selection, onResize, menu, sort, onSort, label, group, footer, maxHeight }: Props<T>) {
  const { t } = useTranslation()
  const wide = useMediaQuery('(min-width: 64em)', true)
  const navigate = useNavigate()
  const banded = columns.some((c) => c.band)
  const keys = rows.map(rowKey)
  const all = selection && keys.length > 0 && keys.every((k) => selection.selected.has(k))
  const some = selection && keys.some((k) => selection.selected.has(k))
  const head = (n: number) => ({ columns, sort, onSort, onResize, rowSpan: n })
  if (!wide) return <Cards columns={columns} rows={rows} rowKey={rowKey} href={href} menu={menu} label={label} group={group} footer={footer} />
  return (
    <div className={selection ? classes.selectable : undefined}>
      <ScrollArea.Autosize type="auto" mah={maxHeight}>
        <Table highlightOnHover stickyHeader verticalSpacing={0} className={classes.compact} aria-label={label}>
          <Table.Thead>
            {banded && (
              <Table.Tr>
                {selection && <SelectAll rowSpan={2} all={!!all} some={!!some} onToggle={() => toggleAll(selection, keys, !!all)} />}
                {columns.map((c, i) =>
                  !c.band ? (
                    <Header key={c.key} column={c} index={i} {...head(2)} />
                  ) : bandStart(columns, i) ? (
                    <Table.Th key={`band:${c.key}`} colSpan={bandWidth(columns, i)} ta="center" className={classes.bandStart}>
                      {c.band}
                    </Table.Th>
                  ) : null,
                )}
                {menu && <Table.Th w={48} rowSpan={2} aria-label={t('shared.table.actions')} />}
              </Table.Tr>
            )}
            <Table.Tr>
              {selection && !banded && <SelectAll all={!!all} some={!!some} onToggle={() => toggleAll(selection, keys, !!all)} />}
              {columns.map((c, i) => (!banded || c.band) && <Header key={c.key} column={c} index={i} {...head(1)} />)}
              {menu && !banded && <Table.Th w={48} aria-label={t('shared.table.actions')} />}
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {grouped(rows, group?.of).map((g) => (
              <Fragment key={`group:${g.name}`}>
                {group && (
                  <Table.Tr className={classes.groupRow}>
                    <Table.Td colSpan={columns.length + (menu ? 1 : 0) + (selection ? 1 : 0)}>
                      <span className={classes.groupName}>{g.name}</span>
                    </Table.Td>
                  </Table.Tr>
                )}
                {g.rows.map((row) => (
                  <Table.Tr
                    key={rowKey(row)}
                    className={href || onRowClick ? classes.row : undefined}
                    aria-selected={selection ? selection.selected.has(rowKey(row)) : undefined}
                    // The link (or button) in the first cell handles keyboard, middle click and Ctrl+click itself.
                    onClick={(e) => {
                      if ((e.target as HTMLElement).closest('a,button,input,[role=menu]')) return
                      if (onRowClick) onRowClick(row)
                      else if (href) navigate(href(row))
                    }}
                  >
                    {selection && (
                      <Table.Td className={classes.select}>
                        <Checkbox
                          aria-label={t('shared.table.select_row')}
                          checked={selection.selected.has(rowKey(row))}
                          onChange={() => toggle(selection, rowKey(row))}
                        />
                      </Table.Td>
                    )}
                    <Cells columns={columns} row={row} href={href} onOpen={onRowClick && !href ? () => onRowClick(row) : undefined} />
                    {menu && (
                      <Table.Td>
                        <RowMenu>{menu(row)}</RowMenu>
                      </Table.Td>
                    )}
                  </Table.Tr>
                ))}
                {group?.total && (
                  <Table.Tr className={classes.totalRow}>
                    {selection && <Table.Td />}
                    <Cells columns={columns} row={group.total(g.rows, g.name)} />
                    {menu && <Table.Td />}
                  </Table.Tr>
                )}
              </Fragment>
            ))}
          </Table.Tbody>
          {footer !== undefined && (
            <Table.Tfoot>
              <Table.Tr className={classes.totalRow}>
                {selection && <Table.Td />}
                <Cells columns={columns} row={footer} />
                {menu && <Table.Td />}
              </Table.Tr>
            </Table.Tfoot>
          )}
        </Table>
      </ScrollArea.Autosize>
    </div>
  )
}

// bandStart: the column opens a band (or leaves one), so it carries the rule on its left.
function bandStart<T>(columns: Column<T>[], i: number) {
  return i > 0 && columns[i - 1]?.band !== columns[i]?.band
}

function bandWidth<T>(columns: Column<T>[], i: number) {
  let n = 1
  while (columns[i + n]?.band === columns[i]?.band) n++
  return n
}

function cellClass<T>(columns: Column<T>[], i: number) {
  const c = columns[i]
  return [i === 0 && classes.sticky, c?.numeric && classes.numeric, c?.strong && classes.strong, bandStart(columns, i) && classes.bandStart].filter(Boolean).join(' ') || undefined
}

type HeaderProps<T> = { column: Column<T>; index: number; columns: Column<T>[]; sort?: string; onSort?: (s: string) => void; onResize?: (key: string, width: number) => void; rowSpan?: number }

function Header<T>({ column: c, index, columns, sort, onSort, onResize, rowSpan }: HeaderProps<T>) {
  const ref = useRef<HTMLTableCellElement>(null)
  return (
    <Table.Th ref={ref} w={c.width} ta={c.numeric ? 'right' : undefined} rowSpan={rowSpan} className={cellClass(columns, index)} aria-sort={sortState(sort, c.key)}>
      {c.sortable && onSort ? <SortButton column={c} sort={sort} onSort={onSort} /> : c.header}
      {onResize && <ResizeHandle column={c} cell={ref} onResize={onResize} />}
    </Table.Th>
  )
}

const minWidth = 48
const maxWidth = 800
const clampWidth = (w: number) => Math.round(Math.min(maxWidth, Math.max(minWidth, w)))

// ResizeHandle is a column's right edge: drag it, or focus it and use the arrow keys (16px a step).
// The width follows the pointer live and is saved once, when the drag ends.
function ResizeHandle<T>({ column, cell, onResize }: { column: Column<T>; cell: RefObject<HTMLTableCellElement | null>; onResize: (key: string, width: number) => void }) {
  const { t } = useTranslation()
  const current = () => cell.current?.getBoundingClientRect().width ?? column.width ?? 120
  function onPointerDown(e: PointerEvent<HTMLSpanElement>) {
    e.preventDefault()
    e.stopPropagation()
    const th = cell.current
    if (!th) return
    const startX = e.clientX
    const start = current()
    const move = (ev: globalThis.PointerEvent) => (th.style.width = `${clampWidth(start + ev.clientX - startX)}px`)
    const up = (ev: globalThis.PointerEvent) => {
      window.removeEventListener('pointermove', move)
      window.removeEventListener('pointerup', up)
      onResize(column.key, clampWidth(start + ev.clientX - startX))
    }
    window.addEventListener('pointermove', move)
    window.addEventListener('pointerup', up)
  }
  function onKeyDown(e: KeyboardEvent<HTMLSpanElement>) {
    const step = e.key === 'ArrowRight' ? 16 : e.key === 'ArrowLeft' ? -16 : 0
    if (!step) return
    e.preventDefault()
    onResize(column.key, clampWidth(current() + step))
  }
  return (
    <span
      className={classes.resize}
      role="separator"
      aria-orientation="vertical"
      aria-label={t('shared.table.resize', { column: column.header })}
      aria-valuenow={Math.round(current())}
      aria-valuemin={minWidth}
      aria-valuemax={maxWidth}
      tabIndex={0}
      onPointerDown={onPointerDown}
      onKeyDown={onKeyDown}
      onClick={(e) => e.stopPropagation()}
    />
  )
}

function SelectAll({ all, some, onToggle, rowSpan }: { all: boolean; some: boolean; onToggle: () => void; rowSpan?: number }) {
  const { t } = useTranslation()
  return (
    <Table.Th rowSpan={rowSpan} className={classes.select}>
      <Checkbox aria-label={t('shared.table.select_all')} checked={all} indeterminate={some && !all} onChange={onToggle} />
    </Table.Th>
  )
}

function toggle(selection: Selection, key: string | number) {
  const next = new Set(selection.selected)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  selection.onChange(next)
}

// toggleAll selects every row of the page, or clears them when all already are.
function toggleAll(selection: Selection, keys: (string | number)[], all: boolean) {
  const next = new Set(selection.selected)
  for (const k of keys) {
    if (all) next.delete(k)
    else next.add(k)
  }
  selection.onChange(next)
}

function Cells<T>({ columns, row, href, onOpen }: { columns: Column<T>[]; row: T; href?: (row: T) => string; onOpen?: () => void }) {
  return columns.map((c, i) => (
    <Table.Td key={c.key} ta={c.numeric ? 'right' : undefined} className={cellClass(columns, i)}>
      {i === 0 && href ? (
        <Link to={href(row)} className={classes.idLink}>
          {c.render(row)}
        </Link>
      ) : i === 0 && onOpen ? (
        <UnstyledButton onClick={onOpen} className={classes.idLink}>
          {c.render(row)}
        </UnstyledButton>
      ) : (
        c.render(row)
      )}
    </Table.Td>
  ))
}

function sortState(sort: string | undefined, key: string) {
  if (sort === key) return 'ascending'
  if (sort === `-${key}`) return 'descending'
  return undefined
}

function SortButton<T>({ column, sort, onSort }: { column: Column<T>; sort?: string; onSort: (s: string) => void }) {
  const state = sortState(sort, column.key)
  const Icon = state === 'ascending' ? IconArrowUp : state === 'descending' ? IconArrowDown : IconArrowsSort
  return (
    <UnstyledButton onClick={() => onSort(state === 'ascending' ? `-${column.key}` : column.key)} fw={500} fz="xs" c={state ? 'var(--mantine-color-text)' : undefined}>
      <Group gap="xxs" wrap="nowrap">
        {column.header}
        <Icon {...icon.text} />
      </Group>
    </UnstyledButton>
  )
}

// RowMenu shows nothing for a row without secondary actions.
function RowMenu({ children }: { children: ReactNode }) {
  return <MoreActions>{children}</MoreActions>
}

function Cards<T>({ columns, rows, rowKey, href, menu, label, group, footer }: Omit<Props<T>, 'sort' | 'onSort'>) {
  return (
    <Stack gap="xs" role="list" aria-label={label}>
      {grouped(rows, group?.of).map((g) => (
        <Fragment key={`group:${g.name}`}>
          {group && (
            <Text fw={600} size="sm" role="listitem">
              {g.name}
            </Text>
          )}
          {g.rows.map((row) => (
            <Card key={rowKey(row)} columns={columns} row={row} href={href} menu={menu} />
          ))}
          {group?.total && <Card columns={columns} row={group.total(g.rows, g.name)} total />}
        </Fragment>
      ))}
      {footer !== undefined && <Card columns={columns} row={footer} total />}
    </Stack>
  )
}

// Card is one row as a card; a total card has no link and no menu.
function Card<T>({ columns, row, href, menu, total }: { columns: Column<T>[]; row: T; href?: (row: T) => string; menu?: (row: T) => ReactNode; total?: boolean }) {
  const title = columns.find((c) => c.role === 'title') ?? columns[0]
  const status = columns.filter((c) => c.role === 'status')
  const meta = columns.filter((c) => c.role === 'meta')
  return (
    <Paper withBorder p="sm" role="listitem" className={total ? classes.totalCard : classes.card}>
      <Group justify="space-between" wrap="nowrap" align="flex-start">
        {href ? (
          <Text fw={600} component={Link} to={href(row)} className={classes.cardLink}>
            {title?.render(row)}
          </Text>
        ) : (
          <Text fw={600}>{title?.render(row)}</Text>
        )}
        {status.map((c) => (
          <Text key={c.key} size="sm">
            {c.render(row)}
          </Text>
        ))}
        {menu && (
          <div className={classes.cardAction}>
            <RowMenu>{menu(row)}</RowMenu>
          </div>
        )}
      </Group>
      {meta.map((c) => (
        <Text key={c.key} size="sm" c={total ? undefined : 'dimmed'}>
          {total ? `${c.header}: ` : ''}
          {c.render(row)}
        </Text>
      ))}
    </Paper>
  )
}
