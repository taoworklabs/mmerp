import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import styles from './Table.module.css'

export interface TableProps extends React.ComponentPropsWithRef<'table'> {
  highlightOnHover?: boolean
  stickyHeader?: boolean
}

function TableRoot({ highlightOnHover, stickyHeader, className, ...props }: TableProps) {
  return (
    <table
      className={cx(styles.root, className)}
      data-highlight-on-hover={highlightOnHover ? '' : undefined}
      data-sticky-header={stickyHeader ? '' : undefined}
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

function Th({ className, ...props }: React.ComponentPropsWithRef<'th'>) {
  return <th className={cx(styles.th, className)} {...props} />
}

function Td({ className, ...props }: React.ComponentPropsWithRef<'td'>) {
  return <td className={cx(styles.td, className)} {...props} />
}

export const Table = Object.assign(TableRoot, { Thead, Tbody, Tfoot, Tr, Th, Td })
