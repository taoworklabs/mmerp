import { Popover as BasePopover } from '@base-ui/react/popover'
import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import styles from './Popover.module.css'

export type PopoverSide = 'top' | 'bottom' | 'left' | 'right'
export type PopoverAlign = 'start' | 'center' | 'end'
// Same shape Base UI's own Positioner accepts: an element, a ref to one, a virtual element, or a
// function returning one — lets a controlled Popover (no Trigger) anchor to an arbitrary node.
export type PopoverAnchor = React.ComponentProps<typeof BasePopover.Positioner>['anchor']
type PopoverPopupFocusProp = React.ComponentProps<typeof BasePopover.Popup>['initialFocus']

export interface PopoverPopupProps extends Omit<React.ComponentPropsWithRef<'div'>, 'children'> {
  side?: PopoverSide
  align?: PopoverAlign
  width?: number | 'target'
  anchor?: PopoverAnchor
  // Anchoring to an external control (e.g. a text input) needs the popup to not steal its focus.
  initialFocus?: PopoverPopupFocusProp
  finalFocus?: PopoverPopupFocusProp
  children: React.ReactNode
}

function PopoverPopup({
  side = 'bottom',
  align = 'center',
  width,
  anchor,
  initialFocus,
  finalFocus,
  className,
  style,
  children,
  ...props
}: PopoverPopupProps) {
  return (
    <BasePopover.Portal>
      <BasePopover.Positioner side={side} align={align} sideOffset={4} anchor={anchor} className={styles.positioner}>
        <BasePopover.Popup
          initialFocus={initialFocus}
          finalFocus={finalFocus}
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
