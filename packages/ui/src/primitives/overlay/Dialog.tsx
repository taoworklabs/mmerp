import { Dialog as BaseDialog } from '@base-ui/react/dialog'
import type * as React from 'react'
import { cx } from '../../internal/cx.ts'
import { CloseIcon } from '../../internal/icons.tsx'
import { ActionIcon } from '../input/ActionIcon.tsx'
import styles from './Dialog.module.css'

export type DialogSize = 'sm' | 'md' | 'lg' | 'xl'

const sizeWidths: Record<DialogSize, number> = { sm: 380, md: 440, lg: 620, xl: 780 }

export interface DialogProps extends Omit<React.ComponentPropsWithRef<'div'>, 'children' | 'title'> {
  opened: boolean
  onClose(): void
  title: React.ReactNode
  size?: DialogSize | number
  closeLabel: string
  withCloseButton?: boolean
  children: React.ReactNode
}

function DialogRoot({
  opened,
  onClose,
  title,
  size = 'md',
  closeLabel,
  withCloseButton = true,
  className,
  style,
  children,
  ...props
}: DialogProps) {
  const width = typeof size === 'number' ? size : sizeWidths[size]
  return (
    <BaseDialog.Root
      open={opened}
      onOpenChange={(next) => {
        if (!next) onClose()
      }}
    >
      <BaseDialog.Portal>
        <BaseDialog.Backdrop className={styles.backdrop} />
        <BaseDialog.Popup className={cx(styles.popup, className)} style={{ ...style, width }} {...props}>
          <div className={styles.header}>
            <BaseDialog.Title className={styles.title}>{title}</BaseDialog.Title>
            {withCloseButton && (
              <ActionIcon variant="subtle" color="gray" aria-label={closeLabel} onClick={onClose}>
                <CloseIcon size={12} />
              </ActionIcon>
            )}
          </div>
          <div className={styles.body}>{children}</div>
        </BaseDialog.Popup>
      </BaseDialog.Portal>
    </BaseDialog.Root>
  )
}

// The dialog's buttons: the way out, then the primary, together at the right edge. Placed inside the
// body (within a form, if any) so a submit button still submits it.
function DialogActions({ className, ...props }: React.ComponentPropsWithRef<'div'>) {
  return <div className={cx(styles.actions, className)} {...props} />
}

export const Dialog = Object.assign(DialogRoot, { Actions: DialogActions })
