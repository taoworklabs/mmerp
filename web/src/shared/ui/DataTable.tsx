import { ActionIcon, Group, Menu, Paper, ScrollArea, Stack, Table, Text, Tooltip, UnstyledButton } from '@mantine/core'
import { useMediaQuery } from '@mantine/hooks'
import { IconArrowDown, IconArrowUp, IconArrowsSort, IconDots } from '@tabler/icons-react'
import { Fragment, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, useNavigate } from 'react-router'
import classes from './DataTable.module.css'
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
}

type Props<T> = {
  columns: Column<T>[]
  rows: T[]
  rowKey: (row: T) => string | number
  // The first column links here and the whole row is clickable; without it rows are plain.
  href?: (row: T) => string
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
  // Caps the frame's height (e.g. "70dvh") so the header and footer stay in view inside it.
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
export function DataTable<T>({ columns, rows, rowKey, href, menu, sort, onSort, label, group, footer, maxHeight }: Props<T>) {
  const { t } = useTranslation()
  const wide = useMediaQuery('(min-width: 64em)', true)
  const navigate = useNavigate()
  const banded = columns.some((c) => c.band)
  if (!wide) return <Cards columns={columns} rows={rows} rowKey={rowKey} href={href} menu={menu} label={label} group={group} footer={footer} />
  return (
    <Paper withBorder className={classes.frame}>
      <ScrollArea.Autosize type="auto" mah={maxHeight}>
        <Table highlightOnHover stickyHeader verticalSpacing={0} className={classes.compact} aria-label={label}>
          <Table.Thead>
            {banded && (
              <Table.Tr>
                {columns.map((c, i) =>
                  !c.band ? (
                    <Header key={c.key} column={c} index={i} columns={columns} sort={sort} onSort={onSort} rowSpan={2} />
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
              {columns.map((c, i) => (!banded || c.band) && <Header key={c.key} column={c} index={i} columns={columns} sort={sort} onSort={onSort} />)}
              {menu && !banded && <Table.Th w={48} aria-label={t('shared.table.actions')} />}
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {grouped(rows, group?.of).map((g) => (
              <Fragment key={`group:${g.name}`}>
                {group && (
                  <Table.Tr className={classes.groupRow}>
                    <Table.Td colSpan={columns.length + (menu ? 1 : 0)}>
                      <span className={classes.groupName}>{g.name}</span>
                    </Table.Td>
                  </Table.Tr>
                )}
                {g.rows.map((row) => (
                  <Table.Tr
                    key={rowKey(row)}
                    className={href ? classes.row : undefined}
                    // The link in the first cell handles keyboard, middle click and Ctrl+click itself.
                    onClick={(e) => href && !(e.target as HTMLElement).closest('a,button,[role=menu]') && navigate(href(row))}
                  >
                    <Cells columns={columns} row={row} href={href} />
                    {menu && (
                      <Table.Td>
                        <RowMenu>{menu(row)}</RowMenu>
                      </Table.Td>
                    )}
                  </Table.Tr>
                ))}
                {group?.total && (
                  <Table.Tr className={classes.totalRow}>
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
                <Cells columns={columns} row={footer} />
                {menu && <Table.Td />}
              </Table.Tr>
            </Table.Tfoot>
          )}
        </Table>
      </ScrollArea.Autosize>
    </Paper>
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

function Header<T>({ column: c, index, columns, sort, onSort, rowSpan }: { column: Column<T>; index: number; columns: Column<T>[]; sort?: string; onSort?: (s: string) => void; rowSpan?: number }) {
  return (
    <Table.Th ta={c.numeric ? 'right' : undefined} rowSpan={rowSpan} className={cellClass(columns, index)} aria-sort={sortState(sort, c.key)}>
      {c.sortable && onSort ? <SortButton column={c} sort={sort} onSort={onSort} /> : c.header}
    </Table.Th>
  )
}

function Cells<T>({ columns, row, href }: { columns: Column<T>[]; row: T; href?: (row: T) => string }) {
  return columns.map((c, i) => (
    <Table.Td key={c.key} ta={c.numeric ? 'right' : undefined} className={cellClass(columns, i)}>
      {i === 0 && href ? (
        <Link to={href(row)} className={classes.idLink}>
          {c.render(row)}
        </Link>
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
  const { t } = useTranslation()
  if (!children) return null
  return (
    <Menu position="bottom-end">
      <Menu.Target>
        <Tooltip label={t('shared.table.actions')}>
          <ActionIcon variant="subtle" color="gray" size="lg" aria-label={t('shared.table.actions')}>
            <IconDots {...icon.button} />
          </ActionIcon>
        </Tooltip>
      </Menu.Target>
      <Menu.Dropdown>{children}</Menu.Dropdown>
    </Menu>
  )
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
