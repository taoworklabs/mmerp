import { Button, Dialog, Stack, Text } from '@mmerp/ui'
import { useCallback, useRef, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { ModalFooter } from './ModalFooter'

export type ConfirmOptions = { title: string; message: string; confirmLabel: string; danger?: boolean }

// useConfirm asks before a destructive or irreversible step. The confirm button repeats
// the verb ("Thu hồi"), never "OK". Returns the ask function and the dialog to render.
export function useConfirm(): [(o: ConfirmOptions) => Promise<boolean>, ReactNode] {
  const { t } = useTranslation()
  const [opts, setOpts] = useState<ConfirmOptions | null>(null)
  const resolve = useRef<(ok: boolean) => void>(() => {})
  const confirmRef = useRef<HTMLButtonElement>(null)
  const ask = useCallback((o: ConfirmOptions) => {
    setOpts(o)
    return new Promise<boolean>((r) => (resolve.current = r))
  }, [])
  const close = (ok: boolean) => {
    setOpts(null)
    resolve.current(ok)
  }
  const dialog = opts && (
    <Dialog opened onClose={() => close(false)} title={opts.title} closeLabel={t('shared.modal.close')} initialFocus={confirmRef}>
      <Stack gap="md">
        <Text>{opts.message}</Text>
        <ModalFooter>
          <Button variant="subtle" color="gray" onClick={() => close(false)}>
            {t('shared.confirm.back')}
          </Button>
          <Button color={opts.danger ? 'danger' : undefined} onClick={() => close(true)} ref={confirmRef}>
            {opts.confirmLabel}
          </Button>
        </ModalFooter>
      </Stack>
    </Dialog>
  )
  return [ask, dialog]
}
