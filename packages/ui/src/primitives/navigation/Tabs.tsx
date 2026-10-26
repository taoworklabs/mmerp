import { Tabs as BaseTabs } from '@base-ui/react/tabs'
import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import styles from './Tabs.module.css'

export interface TabsProps extends Omit<React.ComponentPropsWithRef<'div'>, 'defaultValue' | 'onChange'> {
  value?: string
  defaultValue?: string
  onValueChange?(value: string): void
}

function TabsRoot({ value, defaultValue, onValueChange, className, ...props }: TabsProps) {
  return (
    <BaseTabs.Root
      value={value}
      defaultValue={defaultValue}
      onValueChange={onValueChange && ((next) => onValueChange(next as string))}
      className={cx(styles.root, className)}
      {...props}
    />
  )
}

export interface TabsListProps extends React.ComponentPropsWithRef<'div'> {
  // More tabs than fit: one row that scrolls sideways instead of wrapping.
  scroll?: boolean
  // Sits on a PageHeader's own rule instead of drawing one of its own.
  onRule?: boolean
  // Few tabs on a narrow screen: they share the width evenly.
  grow?: boolean
}

function List({ scroll, onRule, grow, tabIndex, className, ...props }: TabsListProps) {
  // Automatic activation on arrow focus: a user browsing tabs expects the panel to follow immediately.
  // scroll makes the list itself the scrollable region, so it needs its own tab stop for the a11y
  // rule that every scrollable region must be reachable by keyboard; callers may still override it.
  return (
    <BaseTabs.List
      activateOnFocus
      className={cx(styles.list, className)}
      data-scroll={scroll ? '' : undefined}
      data-on-rule={onRule ? '' : undefined}
      data-grow={grow ? '' : undefined}
      tabIndex={tabIndex ?? (scroll ? 0 : undefined)}
      {...props}
    />
  )
}

export interface TabsTabProps extends React.ComponentPropsWithRef<'button'> {
  value: string
}

function Tab({ value, className, ...props }: TabsTabProps) {
  return <BaseTabs.Tab value={value} className={cx(styles.tab, className)} {...props} />
}

export interface TabsPanelProps extends React.ComponentPropsWithRef<'div'> {
  value: string
}

function Panel({ value, className, ...props }: TabsPanelProps) {
  return <BaseTabs.Panel value={value} className={cx(styles.panel, className)} {...props} />
}

export const Tabs = Object.assign(TabsRoot, { List, Tab, Panel })
