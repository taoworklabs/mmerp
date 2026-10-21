import { cloneElement, Fragment, useCallback, useRef, useState } from 'react'
import type { KeyboardEvent, PointerEvent, ReactElement, ReactNode, RefObject } from 'react'
import { useElementSize } from '../../hooks/useElementSize.ts'
import { useMediaQuery } from '../../hooks/useMediaQuery.ts'
import { cx } from '../../internal/cx.ts'
import { ArrowDownIcon, ArrowUpIcon, ArrowsSortIcon } from '../../internal/icons.tsx'
import type { StyleVars } from '../../internal/style.ts'
import { Checkbox } from '../../primitives/input/Checkbox.tsx'
import { UnstyledButton } from '../../primitives/input/UnstyledButton.tsx'
import { Group } from '../../primitives/layout/Group.tsx'
import { Paper } from '../../primitives/layout/Paper.tsx'
import { ScrollArea } from '../../primitives/layout/ScrollArea.tsx'
import { Stack } from '../../primitives/layout/Stack.tsx'
import { VisuallyHidden } from '../../primitives/layout/VisuallyHidden.tsx'
import { Table } from '../../primitives/data-display/Table.tsx'
import { Text } from '../../primitives/typography/Text.tsx'
import { breakpoint } from '../../tokens/tokens.ts'
import { MoreActions } from '../list/MoreActions.tsx'
import { bandStart, bandWidth, clampWidth, fits, grouped, look, maxWidth, minWidth, type Column } from './columns.ts'
import styles from './DataTable.module.css'

export type { Column, ColumnKind, ColumnRole } from './columns.ts'

// Selection of rows for a bulk action: the keys of the selected rows on this page.
export type Selection = { selected: ReadonlySet<string | number>; onChange: (next: Set<string | number>) => void }

export interface DataTableLabels {
  actions: string // aria-label of the menu column
  selectAll: string
  selectRow: string
  resize(column: string): string
}

export type DataTableProps<T> = {
  columns: Column<T>[]
  rows: T[]
  rowKey: (row: T) => string | number
  // The first column's link (and the card title's); the whole row clicks it. The router link the
  // caller renders handles navigation itself: the kit never calls one.
  renderLink?: (row: T, children: ReactNode) => ReactElement
  // Clicking a row outside its link runs this instead: a list's side panel, or the edit modal of a
  // settings row that has no page (its first cell becomes the keyboard's button).
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
  // The card list at every width, for a table in a page's narrow side column.
  cards?: boolean
  labels: DataTableLabels
}

// DataTable is the only table areas use: compact, sortable on the server, a card list below 1024px.
export function DataTable<T>({
  columns,
  rows,
  rowKey,
  renderLink,
  onRowClick,
  selection,
  onResize,
  menu,
  sort,
  onSort,
  label,
  group,
  footer,
  maxHeight,
  cards,
  labels,
}: DataTableProps<T>) {
  const wide = useMediaQuery(`(min-width: ${breakpoint.md})`, true)
  const [scrolled, setScrolled] = useState(false)
  // The sticky header's and footer's heights (wrapped titles and bands included) keep the vertical
  // scrollbar between them.
  const thead = useElementSize<HTMLTableSectionElement>()
  const tfoot = useElementSize<HTMLTableSectionElement>()
  const onViewport = useCallback((el: HTMLDivElement | null) => {
    if (!el) return
    const onScroll = () => setScrolled(el.scrollLeft > 0)
    el.addEventListener('scroll', onScroll)
    return () => el.removeEventListener('scroll', onScroll)
  }, [])
  const banded = columns.some((c) => c.band)
  const keys = rows.map(rowKey)
  const all = !!selection && keys.length > 0 && keys.every((k) => selection.selected.has(k))
  const some = !!selection && keys.some((k) => selection.selected.has(k))

  if (!wide || cards) {
    return <Cards columns={columns} rows={rows} rowKey={rowKey} renderLink={renderLink} menu={menu} label={label} group={group} footer={footer} labels={labels} />
  }

  const head = (rowSpan: number) => ({ columns, sort, onSort, onResize, labels, rowSpan })
  const vars: StyleVars = { '--mmerp-head-height': `${thead.height}px`, '--mmerp-foot-height': `${tfoot.height}px` }

  return (
    <div className={cx(styles.root, selection && styles.selectable)} style={vars} data-scrolled={scrolled ? '' : undefined}>
      {/* Without maxHeight the table takes the room its parent gives it, e.g. a list page's frame. */}
      <ScrollArea viewportRef={onViewport} style={{ maxHeight: maxHeight ?? '100%' }}>
        <Table highlightOnHover stickyHeader scrolled={scrolled} aria-label={label}>
          <Table.Thead ref={thead.ref}>
            {banded && (
              <Table.Tr>
                {selection && <SelectAll rowSpan={2} all={all} some={some} onToggle={() => toggleAll(selection, keys, all)} labels={labels} />}
                {columns.map((c, i) =>
                  !c.band ? (
                    <Header key={c.key} column={c} index={i} {...head(2)} />
                  ) : bandStart(columns, i) ? (
                    <Table.Th key={`band:${c.key}`} colSpan={bandWidth(columns, i)} className={cx(styles.bandLabel, styles.bandStart)}>
                      {c.band}
                    </Table.Th>
                  ) : null,
                )}
                {menu && (
                  <Table.Th rowSpan={2} style={{ width: 48 }}>
                    {/* aria-label alone fails axe's empty-table-header rule; real (hidden) text passes it. */}
                    <VisuallyHidden>{labels.actions}</VisuallyHidden>
                  </Table.Th>
                )}
              </Table.Tr>
            )}
            <Table.Tr>
              {selection && !banded && <SelectAll all={all} some={some} onToggle={() => toggleAll(selection, keys, all)} labels={labels} />}
              {columns.map((c, i) => (!banded || c.band) && <Header key={c.key} column={c} index={i} {...head(1)} />)}
              {menu && !banded && (
                <Table.Th style={{ width: 48 }}>
                  <VisuallyHidden>{labels.actions}</VisuallyHidden>
                </Table.Th>
              )}
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {grouped(rows, group?.of).map((g) => (
              <Fragment key={`group:${g.name}`}>
                {group && (
                  <Table.Tr className={styles.groupRow}>
                    <Table.Td colSpan={columns.length + (menu ? 1 : 0) + (selection ? 1 : 0)}>
                      <span className={styles.groupName}>{g.name}</span>
                    </Table.Td>
                  </Table.Tr>
                )}
                {g.rows.map((row) => (
                  <Table.Tr
                    key={rowKey(row)}
                    className={renderLink || onRowClick ? styles.row : undefined}
                    aria-selected={selection ? selection.selected.has(rowKey(row)) : undefined}
                    // The link (or button) in the first cell handles keyboard, middle click and Ctrl+click itself.
                    onClick={(e) => {
                      if ((e.target as HTMLElement).closest('a,button,input,[role=menu]')) return
                      if (onRowClick) onRowClick(row)
                      else e.currentTarget.querySelector('a')?.click()
                    }}
                  >
                    {selection && (
                      <Table.Td className={styles.select}>
                        <Checkbox label={<VisuallyHidden>{labels.selectRow}</VisuallyHidden>} checked={selection.selected.has(rowKey(row))} onChange={() => toggle(selection, rowKey(row))} />
                      </Table.Td>
                    )}
                    <Cells columns={columns} row={row} renderLink={renderLink} onOpen={onRowClick && !renderLink ? () => onRowClick(row) : undefined} />
                    {menu && (
                      <Table.Td>
                        <MoreActions label={labels.actions}>{menu(row)}</MoreActions>
                      </Table.Td>
                    )}
                  </Table.Tr>
                ))}
                {group?.total && (
                  <Table.Tr className={styles.totalRow}>
                    {selection && <Table.Td />}
                    <Cells columns={columns} row={group.total(g.rows, g.name)} total />
                    {menu && <Table.Td />}
                  </Table.Tr>
                )}
              </Fragment>
            ))}
          </Table.Tbody>
          {footer !== undefined && (
            <Table.Tfoot ref={tfoot.ref}>
              <Table.Tr className={styles.totalRow}>
                {selection && <Table.Td />}
                <Cells columns={columns} row={footer} total />
                {menu && <Table.Td />}
              </Table.Tr>
            </Table.Tfoot>
          )}
        </Table>
      </ScrollArea>
    </div>
  )
}

type HeaderProps<T> = {
  column: Column<T>
  index: number
  columns: Column<T>[]
  sort?: string
  onSort?: (s: string) => void
  onResize?: (key: string, width: number) => void
  labels: DataTableLabels
  rowSpan?: number
}

function Header<T>({ column: c, index, columns, sort, onSort, onResize, labels, rowSpan }: HeaderProps<T>) {
  const ref = useRef<HTMLTableCellElement>(null)
  return (
    <Table.Th
      ref={ref}
      look={look(c).right ? 'numeric' : undefined}
      sticky={index === 0}
      fit={fits(columns, index)}
      rowSpan={rowSpan}
      style={{ width: c.width }}
      className={cx(c.strong && styles.strong, bandStart(columns, index) && styles.bandStart)}
      aria-sort={sortState(sort, c.key)}
    >
      {c.sortable && onSort ? <SortButton column={c} sort={sort} onSort={onSort} /> : c.header}
      {onResize && <ResizeHandle column={c} cell={ref} onResize={onResize} labels={labels} />}
    </Table.Th>
  )
}

// ResizeHandle is a column's right edge: drag it, or focus it and use the arrow keys (16px a step).
// The width follows the pointer live and is saved once, when the drag ends.
function ResizeHandle<T>({ column, cell, onResize, labels }: { column: Column<T>; cell: RefObject<HTMLTableCellElement | null>; onResize: (key: string, width: number) => void; labels: DataTableLabels }) {
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
      className={styles.resize}
      role="separator"
      aria-orientation="vertical"
      aria-label={labels.resize(column.header)}
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

function SelectAll({ all, some, onToggle, rowSpan, labels }: { all: boolean; some: boolean; onToggle: () => void; rowSpan?: number; labels: DataTableLabels }) {
  return (
    <Table.Th rowSpan={rowSpan} className={styles.select}>
      {/* Checkbox's aria-label prop lands on its outer wrapper, not the checkbox control itself
          (which names itself from the label's content); a visually hidden label fixes that. */}
      <Checkbox label={<VisuallyHidden>{labels.selectAll}</VisuallyHidden>} checked={all} indeterminate={some && !all} onChange={onToggle} />
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

// A zero figure shows "–", and so does an empty cell of a row (a total row leaves it blank), so the
// eye stops on cells that hold something.
function Cells<T>({ columns, row, renderLink, onOpen, total }: { columns: Column<T>[]; row: T; renderLink?: (row: T, children: ReactNode) => ReactElement; onOpen?: () => void; total?: boolean }) {
  return columns.map((c, i) => {
    const value = c.render(row)
    const blank = value === '' || value === null || value === undefined
    const zero = c.kind === 'number' && (value === 0 || value === '0')
    const shown = zero || (blank && !total) ? <span className={styles.secondary}>–</span> : value
    let content: ReactNode = shown
    if (i === 0 && renderLink) {
      const link = renderLink(row, shown) as ReactElement<{ className?: string }>
      content = cloneElement(link, { className: cx(styles.idLink, link.props.className) })
    } else if (i === 0 && onOpen) {
      content = (
        <UnstyledButton onClick={onOpen} className={styles.idLink}>
          {shown}
        </UnstyledButton>
      )
    }
    return (
      <Table.Td
        key={c.key}
        look={look(c).right ? 'numeric' : i > 0 && look(c).secondary ? 'secondary' : undefined}
        sticky={i === 0}
        fit={fits(columns, i)}
        className={cx(c.strong && styles.strong, bandStart(columns, i) && styles.bandStart)}
      >
        {content}
      </Table.Td>
    )
  })
}

function sortState(sort: string | undefined, key: string) {
  if (sort === key) return 'ascending' as const
  if (sort === `-${key}`) return 'descending' as const
  return undefined
}

function SortButton<T>({ column, sort, onSort }: { column: Column<T>; sort?: string; onSort: (s: string) => void }) {
  const state = sortState(sort, column.key)
  const Icon = state === 'ascending' ? ArrowUpIcon : state === 'descending' ? ArrowDownIcon : ArrowsSortIcon
  return (
    <UnstyledButton onClick={() => onSort(state === 'ascending' ? `-${column.key}` : column.key)} className={styles.sortButton} data-sorted={state ? '' : undefined}>
      {column.header}
      <Icon />
    </UnstyledButton>
  )
}

type CardsProps<T> = Pick<DataTableProps<T>, 'columns' | 'rows' | 'rowKey' | 'renderLink' | 'menu' | 'label' | 'group' | 'footer' | 'labels'>

function Cards<T>({ columns, rows, rowKey, renderLink, menu, label, group, footer, labels }: CardsProps<T>) {
  return (
    <Stack gap="xs" role="list" aria-label={label}>
      {grouped(rows, group?.of).map((g) => (
        <Fragment key={`group:${g.name}`}>
          {group && (
            <Text weight={600} size="sm" render={<div role="listitem" />}>
              {g.name}
            </Text>
          )}
          {g.rows.map((row) => (
            <Card key={rowKey(row)} columns={columns} row={row} renderLink={renderLink} menu={menu} labels={labels} />
          ))}
          {group?.total && <Card columns={columns} row={group.total(g.rows, g.name)} labels={labels} total />}
        </Fragment>
      ))}
      {footer !== undefined && <Card columns={columns} row={footer} labels={labels} total />}
    </Stack>
  )
}

// Card is one row as a card; a total card has no link and no menu.
function Card<T>({
  columns,
  row,
  renderLink,
  menu,
  labels,
  total,
}: {
  columns: Column<T>[]
  row: T
  renderLink?: (row: T, children: ReactNode) => ReactElement
  menu?: (row: T) => ReactNode
  labels: DataTableLabels
  total?: boolean
}) {
  const title = columns.find((c) => c.role === 'title') ?? columns[0]
  const status = columns.filter((c) => c.role === 'status')
  const meta = columns.filter((c) => c.role === 'meta')
  const titleContent = title?.render(row)
  return (
    <Paper withBorder padding="sm" role="listitem" className={cx(styles.card, total && styles.totalCard)}>
      <Group justify="space-between" wrap={false} align="start">
        {!total && renderLink ? (
          <CardTitle link={renderLink(row, titleContent) as ReactElement<{ className?: string }>} />
        ) : (
          <Text weight={600}>{titleContent}</Text>
        )}
        {status.map((c) => (
          <Text key={c.key} size="sm">
            {c.render(row)}
          </Text>
        ))}
        {menu && (
          <div className={styles.cardAction}>
            <MoreActions label={labels.actions}>{menu(row)}</MoreActions>
          </div>
        )}
      </Group>
      {meta.map((c) => (
        <Text key={c.key} size="sm" tone={total ? undefined : 'dimmed'}>
          {total ? `${c.header}: ` : ''}
          {c.render(row)}
        </Text>
      ))}
    </Paper>
  )
}

function CardTitle({ link }: { link: ReactElement<{ className?: string }> }) {
  return cloneElement(link, { className: cx(styles.cardLink, link.props.className) })
}
