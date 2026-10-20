import { Popover as BasePopover } from '@base-ui/react/popover'
import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import styles from './Popover.module.css'

export type PopoverSide = 'top' | 'bottom' | 'left' | 'right'
export type PopoverAlign = 'start' | 'center' | 'end'

export interface PopoverPopupProps extends Omit<React.ComponentPropsWithRef<'div'>, 'children'> {
  side?: PopoverSide
  align?: PopoverAlign
  width?: number | 'target'
  children: React.ReactNode
}

function PopoverPopup({ side = 'bottom', align = 'center', width, className, style, children, ...props }: PopoverPopupProps) {
  return (
    <BasePopover.Portal>
      <BasePopover.Positioner side={side} align={align} sideOffset={4}>
        <BasePopover.Popup
          className={cx(styles.popup, className)}
          style={{ ...style, width: width === 'target' ? 'var(--anchor-width)' : width }}
          {...props}
        >
          {children}
        </BasePopover.Popup>
      </BasePopover.Positioner>
    </BasePopover.Portal>
  )
}

// Compound export, same shape as Base UI's own namespace: Popup wraps Portal + Positioner + Popup
// so callers only ever position against side/align/width, never the floating-ui plumbing.
export const Popover = {
  Root: BasePopover.Root,
  Trigger: BasePopover.Trigger,
  Popup: PopoverPopup,
}
