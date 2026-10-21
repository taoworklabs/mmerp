import { Toast as BaseToast } from '@base-ui/react/toast'
import type * as React from 'react'
import type { StyleVars } from '../../internal/style.ts'
import { ActionIcon } from '../input/ActionIcon.tsx'
import { Button } from '../input/Button.tsx'
import styles from './Toast.module.css'
import { CloseIcon } from '../../internal/icons.tsx'

export type ToastColor = 'success' | 'danger' | 'info' | 'warning'

export interface ToastOptions {
  message: React.ReactNode
  color?: ToastColor
  timeout?: number | null
  action?: { label: string; onClick(): void }
}

interface ToastData {
  color: ToastColor
  action?: { label: string; onClick(): void }
}

// Module-level so callers outside React (mutation callbacks, non-component code) can show a toast;
// Toaster subscribes this same manager to a store once mounted.
const manager = BaseToast.createToastManager<ToastData>()

export const toast = {
  show(options: ToastOptions): string {
    const timeout = options.timeout === undefined ? 3000 : options.timeout === null ? 0 : options.timeout
    return manager.add({
      description: options.message,
      timeout,
      data: { color: options.color ?? 'info', action: options.action },
    })
  },
  close(id: string) {
    manager.close(id)
  },
}

export interface ToasterProps {
  closeLabel: string
}

function ToastList({ closeLabel }: { closeLabel: string }) {
  const { toasts, close } = BaseToast.useToastManager<ToastData>()
  return toasts.map((item) => {
    const color = item.data?.color ?? 'info'
    const action = item.data?.action
    const vars: StyleVars = { '--mmerp-toast-bar': `var(--mmerp-color-${color}-6)` }
    return (
      <BaseToast.Root
        key={item.id}
        toast={item}
        role="status"
        aria-modal={undefined}
        className={styles.root}
        style={vars}
      >
        <BaseToast.Description className={styles.description} />
        {action && (
          <Button
            variant="subtle"
            size="xs"
            onClick={() => {
              action.onClick()
              close(item.id)
            }}
          >
            {action.label}
          </Button>
        )}
        <ActionIcon variant="subtle" color="gray" aria-label={closeLabel} onClick={() => close(item.id)}>
          <CloseIcon size={12} />
        </ActionIcon>
      </BaseToast.Root>
    )
  })
}

export function Toaster({ closeLabel }: ToasterProps) {
  return (
    <BaseToast.Provider toastManager={manager}>
      <BaseToast.Portal>
        <BaseToast.Viewport className={styles.viewport}>
          <ToastList closeLabel={closeLabel} />
        </BaseToast.Viewport>
      </BaseToast.Portal>
    </BaseToast.Provider>
  )
}
