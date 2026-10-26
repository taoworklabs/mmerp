import { Drawer as BaseDrawer } from '@base-ui/react/drawer'
import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import { CloseIcon } from '../../internal/icons.tsx'
import { ActionIcon } from '../input/ActionIcon.tsx'
import styles from './Drawer.module.css'

export type DrawerPosition = 'left' | 'right' | 'bottom'
type DrawerFocusProp = React.ComponentProps<typeof BaseDrawer.Popup>['finalFocus']

export interface DrawerProps extends Omit<React.ComponentPropsWithRef<'div'>, 'children' | 'title'> {
  opened: boolean
  onClose(): void
  title: React.ReactNode
  position?: DrawerPosition
  // Width at the left or right, height at the bottom.
  size?: number | string
  closeLabel: string
  // Non-modal: no backdrop, the page behind stays usable and an outside click leaves it open.
  modal?: boolean
  // Where focus goes on close; default back to what opened it.
  finalFocus?: DrawerFocusProp
  children: React.ReactNode
}

export function Drawer({
  opened,
  onClose,
  title,
  position = 'right',
  size,
  closeLabel,
  modal = true,
  finalFocus,
  className,
  style,
  children,
  ...props
}: DrawerProps) {
  const length = typeof size === 'number' ? `${size}px` : size
  return (
    <BaseDrawer.Root
      open={opened}
      onOpenChange={(next) => {
        if (!next) onClose()
      }}
      swipeDirection={position === 'bottom' ? 'down' : position}
      modal={modal}
      disablePointerDismissal={!modal}
    >
      <BaseDrawer.Portal>
        {modal && <BaseDrawer.Backdrop className={styles.backdrop} />}
        <BaseDrawer.Viewport className={styles.viewport} data-position={position}>
          <BaseDrawer.Popup
            className={cx(styles.popup, className)}
            finalFocus={finalFocus}
            style={{ ...style, [position === 'bottom' ? 'height' : 'width']: length }}
            data-position={position}
            {...props}
          >
            <div className={styles.header}>
              <BaseDrawer.Title className={styles.title}>{title}</BaseDrawer.Title>
              <ActionIcon variant="subtle" color="gray" aria-label={closeLabel} onClick={onClose}>
                <CloseIcon size={12} />
              </ActionIcon>
            </div>
            <div className={styles.body}>{children}</div>
          </BaseDrawer.Popup>
        </BaseDrawer.Viewport>
      </BaseDrawer.Portal>
    </BaseDrawer.Root>
  )
}
