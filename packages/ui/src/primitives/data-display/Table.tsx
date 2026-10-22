import type * as React from 'react'
import type { ReactNode } from 'react'
import { cx } from '../../internal/cx.ts'
import { ArrowDownIcon, ArrowsSortIcon, ArrowUpIcon } from '../../internal/icons.tsx'
import { UnstyledButton } from '../input/UnstyledButton.tsx'
import styles from './Table.module.css'

export interface TableProps extends React.ComponentPropsWithRef<'table'> {
  highlightOnHover?: boolean
  stickyHeader?: boolean
  // Set by the caller (DataTable) from the scroll position, so the sticky column's edge shows only
  // once something is hidden behind it.
  scrolled?: boolean
}

function TableRoot({ highlightOnHover, stickyHeader, scrolled, className, ...props }: TableProps) {
  return (
    <table
      className={cx(styles.root, className)}
      data-highlight-on-hover={highlightOnHover ? '' : undefined}
      data-sticky-header={stickyHeader ? '' : undefined}
      data-scrolled={scrolled ? '' : undefined}
      {...props}
    />
  )
}

function Thead({ className, ...props }: React.ComponentPropsWithRef<'thead'>) {
  return <thead className={cx(styles.thead, className)} {...props} />
}

function Tbody({ className, ...props }: React.ComponentPropsWithRef<'tbody'>) {
  return <tbody className={cx(styles.tbody, className)} {...props} />
}

function Tfoot({ className, ...props }: React.ComponentPropsWithRef<'tfoot'>) {
  return <tfoot className={cx(styles.tfoot, className)} {...props} />
}

function Tr({ className, ...props }: React.ComponentPropsWithRef<'tr'>) {
  return <tr className={cx(styles.tr, className)} {...props} />
}

// A cell's look and whether it stays on one line, pinned to the viewport's left edge, or wraps.
export type CellLook = 'secondary' | 'numeric' | 'strong'

export interface ThProps extends React.ComponentPropsWithRef<'th'> {
  look?: CellLook
  sticky?: boolean
  fit?: boolean
}

function Th({ look, sticky, fit, className, ...props }: ThProps) {
  return (
    <th
      className={cx(styles.th, className)}
      data-look={look}
      data-sticky={sticky ? '' : undefined}
      data-fit={fit ? '' : undefined}
      {...props}
    />
  )
}

export interface TdProps extends React.ComponentPropsWithRef<'td'> {
  look?: CellLook
  sticky?: boolean
  fit?: boolean
}

function Td({ look, sticky, fit, className, ...props }: TdProps) {
  return (
    <td
      className={cx(styles.td, className)}
      data-look={look}
      data-sticky={sticky ? '' : undefined}
      data-fit={fit ? '' : undefined}
      {...props}
    />
  )
}

export interface SortHeaderProps extends Omit<ThProps, 'children'> {
  label: ReactNode
  state: 'ascending' | 'descending' | undefined
  onSort: () => void
}

// A sortable column title: a button (so Enter/Space toggle it) holding the label and an arrow that
// shows the current direction, or both directions when the column is not the one sorted.
function SortHeader({ label, state, onSort, className, ...props }: SortHeaderProps) {
  const Icon = state === 'ascending' ? ArrowUpIcon : state === 'descending' ? ArrowDownIcon : ArrowsSortIcon
  return (
    <Th aria-sort={state} className={className} {...props}>
      <UnstyledButton onClick={onSort} className={styles.sortButton} data-sorted={state ? '' : undefined}>
        {label}
        <Icon />
      </UnstyledButton>
    </Th>
  )
}

export const Table = Object.assign(TableRoot, { Thead, Tbody, Tfoot, Tr, Th, Td, SortHeader })
