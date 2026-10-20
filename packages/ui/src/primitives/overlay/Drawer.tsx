import { Drawer as BaseDrawer } from '@base-ui/react/drawer'
import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import { ActionIcon } from '../input/ActionIcon.tsx'
import styles from './Drawer.module.css'

export type DrawerPosition = 'left' | 'right'

export interface DrawerProps extends Omit<React.ComponentPropsWithRef<'div'>, 'children' | 'title'> {
  opened: boolean
  onClose(): void
  title: React.ReactNode
  position?: DrawerPosition
  size?: number | string
  closeLabel: string
  children: React.ReactNode
}

function CloseIcon() {
  return (
    <svg width="12" height="12" viewBox="0 0 10 10" aria-hidden="true">
      <path d="M1 1 9 9M9 1 1 9" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" />
    </svg>
  )
}

export function Drawer({
  opened,
  onClose,
  title,
  position = 'right',
  size,
  closeLabel,
  className,
  style,
  children,
  ...props
}: DrawerProps) {
  const width = typeof size === 'number' ? `${size}px` : size
  return (
    <BaseDrawer.Root
      open={opened}
      onOpenChange={(next) => {
        if (!next) onClose()
      }}
      swipeDirection={position}
    >
      <BaseDrawer.Portal>
        <BaseDrawer.Backdrop className={styles.backdrop} />
        <BaseDrawer.Viewport className={styles.viewport} data-position={position}>
          <BaseDrawer.Popup
            className={cx(styles.popup, className)}
            style={{ ...style, width }}
            data-position={position}
            {...props}
          >
            <div className={styles.header}>
              <BaseDrawer.Title className={styles.title}>{title}</BaseDrawer.Title>
              <ActionIcon variant="subtle" color="gray" aria-label={closeLabel} onClick={onClose}>
                <CloseIcon />
              </ActionIcon>
            </div>
            <div className={styles.body}>{children}</div>
          </BaseDrawer.Popup>
        </BaseDrawer.Viewport>
      </BaseDrawer.Portal>
    </BaseDrawer.Root>
  )
}
