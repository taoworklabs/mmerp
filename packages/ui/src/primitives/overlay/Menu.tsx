import { Menu as BaseMenu } from '@base-ui/react/menu'
import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import styles from './Menu.module.css'

export type MenuSide = 'top' | 'bottom' | 'left' | 'right'
export type MenuAlign = 'start' | 'center' | 'end'
type MenuFocusProp = React.ComponentProps<typeof BaseMenu.Popup>['finalFocus']

export interface MenuPopupProps extends Omit<React.ComponentPropsWithRef<'div'>, 'children'> {
  side?: MenuSide
  align?: MenuAlign
  // Where focus goes on close; default back to the trigger.
  finalFocus?: MenuFocusProp
  children: React.ReactNode
}

function MenuPopup({ side = 'bottom', align = 'start', finalFocus, className, children, ...props }: MenuPopupProps) {
  return (
    <BaseMenu.Portal>
      <BaseMenu.Positioner side={side} align={align} sideOffset={4} className={styles.positioner}>
        <BaseMenu.Popup finalFocus={finalFocus} className={cx(styles.popup, className)} {...props}>
          {children}
        </BaseMenu.Popup>
      </BaseMenu.Positioner>
    </BaseMenu.Portal>
  )
}

export interface MenuItemProps extends Omit<React.ComponentPropsWithRef<'div'>, 'color' | 'onClick'> {
  leftSection?: React.ReactNode
  // At the far end, e.g. the check beside the language in use.
  rightSection?: React.ReactNode
  color?: 'danger'
  disabled?: boolean
  onClick?(): void
  children: React.ReactNode
}

function MenuItem({ leftSection, rightSection, color, disabled, onClick, className, children, ...props }: MenuItemProps) {
  return (
    <BaseMenu.Item
      disabled={disabled}
      data-color={color}
      className={cx(styles.item, className)}
      onClick={onClick ? () => onClick() : undefined}
      {...props}
    >
      {leftSection && <span className={styles.section}>{leftSection}</span>}
      {children}
      {rightSection && <span className={cx(styles.section, styles.end)}>{rightSection}</span>}
    </BaseMenu.Item>
  )
}

function MenuSeparator({ className, ...props }: React.ComponentPropsWithRef<'div'>) {
  return <BaseMenu.Separator className={cx(styles.separator, className)} {...props} />
}

function MenuLabel({ className, ...props }: React.ComponentPropsWithRef<'div'>) {
  return <div className={cx(styles.label, className)} {...props} />
}

export const Menu = {
  Root: BaseMenu.Root,
  Trigger: BaseMenu.Trigger,
  Popup: MenuPopup,
  Item: MenuItem,
  Separator: MenuSeparator,
  Label: MenuLabel,
}
