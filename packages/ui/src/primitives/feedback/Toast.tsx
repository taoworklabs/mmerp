import { Toast as BaseToast } from '@base-ui/react/toast'
import type * as React from 'react'
import type { StyleVars } from '../../internal/style.ts'
import type { Tone } from '../../internal/tone.ts'
import { ActionIcon } from '../input/ActionIcon.tsx'
import { Button } from '../input/Button.tsx'
import styles from './Toast.module.css'
import { CloseIcon } from '../../internal/icons.tsx'

type ToastTone = Extract<Tone, 'success' | 'danger' | 'info' | 'warning'>

export interface ToastOptions {
  message: React.ReactNode
  tone?: ToastTone
  timeout?: number | null
  action?: { label: string; onClick(): void }
}

interface ToastData {
  tone: ToastTone
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
      data: { tone: options.tone ?? 'info', action: options.action },
    })
  },
  close(id: string) {
    manager.close(id)
  },
}

export interface ToasterProps {
  closeLabel: string
  // Where the stack sits; top-center keeps it clear of a narrow screen's bottom action bar.
  position?: 'bottom-right' | 'top-center'
}

function ToastList({ closeLabel }: { closeLabel: string }) {
  const { toasts, close } = BaseToast.useToastManager<ToastData>()
  return toasts.map((item) => {
    const tone = item.data?.tone ?? 'info'
    const action = item.data?.action
    const vars: StyleVars = { '--mmerp-toast-bar': `var(--mmerp-color-${tone}-6)` }
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
        <ActionIcon variant="subtle" tone="neutral" aria-label={closeLabel} onClick={() => close(item.id)}>
          <CloseIcon size={12} />
        </ActionIcon>
      </BaseToast.Root>
    )
  })
}

export function Toaster({ closeLabel, position = 'bottom-right' }: ToasterProps) {
  return (
    <BaseToast.Provider toastManager={manager}>
      <BaseToast.Portal>
        <BaseToast.Viewport className={styles.viewport} data-position={position}>
          <ToastList closeLabel={closeLabel} />
        </BaseToast.Viewport>
      </BaseToast.Portal>
    </BaseToast.Provider>
  )
}
