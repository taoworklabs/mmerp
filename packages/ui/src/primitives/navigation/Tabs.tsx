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

export type TabsListProps = React.ComponentPropsWithRef<'div'>

function List({ className, ...props }: TabsListProps) {
  // Automatic activation on arrow focus: a user browsing tabs expects the panel to follow immediately.
  return <BaseTabs.List activateOnFocus className={cx(styles.list, className)} {...props} />
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
