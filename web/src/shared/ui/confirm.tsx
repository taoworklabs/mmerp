import { Button, Group, Modal, Stack, Text } from '@mantine/core'
import { useCallback, useRef, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

type Options = { title: string; message: string; confirmLabel: string; danger?: boolean }

// useConfirm asks before a destructive or irreversible step. The confirm button repeats
// the verb ("Thu hồi"), never "OK". Returns the ask function and the dialog to render.
export function useConfirm(): [(o: Options) => Promise<boolean>, ReactNode] {
  const { t } = useTranslation()
  const [opts, setOpts] = useState<Options | null>(null)
  const resolve = useRef<(ok: boolean) => void>(() => {})
  const ask = useCallback((o: Options) => {
    setOpts(o)
    return new Promise<boolean>((r) => (resolve.current = r))
  }, [])
  const close = (ok: boolean) => {
    setOpts(null)
    resolve.current(ok)
  }
  const dialog = opts && (
    <Modal opened onClose={() => close(false)} title={opts.title}>
      <Stack gap="md">
        <Text>{opts.message}</Text>
        <Group justify="flex-end">
          <Button variant="default" onClick={() => close(false)}>
            {t('shared.confirm.back')}
          </Button>
          <Button color={opts.danger ? 'danger' : undefined} onClick={() => close(true)} data-autofocus>
            {opts.confirmLabel}
          </Button>
        </Group>
      </Stack>
    </Modal>
  )
  return [ask, dialog]
}
