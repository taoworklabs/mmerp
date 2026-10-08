import { Button, Group } from '@mantine/core'
import { IconArrowBackUp, IconSend, IconTrash, IconX } from '@tabler/icons-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api, unwrap } from '@/shared/api/core'
import { errorText } from '@/shared/i18n'
import { useConfirm } from '@/shared/ui/confirm'
import { notifySuccess } from '@/shared/ui/notify'
import { icon } from '@/shared/ui/theme'
import { useDocumentMutation } from './useDocumentMutation'

type Props = {
  docType: string
  id: number
  version: number
  number: string
  // The document's allowed_actions; only these buttons show.
  actions: string[]
  // Deleting a draft goes through the area's own route.
  onDelete?: (version: number) => Promise<unknown>
  // Where the page shows a failed action: a banner at its top.
  onError: (message: string | null) => void
}

// DocumentActions draws the lifecycle buttons from allowed_actions and runs them.
export function DocumentActions({ docType, id, version, number, actions, onDelete, onError }: Props) {
  const { t } = useTranslation()
  const { run, dialog } = useDocumentMutation(docType, id, version)
  const [ask, confirm] = useConfirm()
  const [busy, setBusy] = useState(false)

  async function act(action: string, call: (version: number) => Promise<unknown>) {
    onError(null)
    setBusy(true)
    try {
      await run(call)
      notifySuccess(t(`shared.document.done.${action}`, { number }))
    } catch (err) {
      onError(errorText(t, err))
    } finally {
      setBusy(false)
    }
  }

  const transition = (to: 'posted' | 'draft' | 'cancelled') => (v: number) =>
    unwrap(api.POST('/documents/{type}/{id}/transitions', { params: { path: { type: docType, id } }, body: { to, version: v } }))

  const destructive = async (action: 'cancel' | 'delete', call: (v: number) => Promise<unknown>) => {
    const label = t(`shared.document.action.${action}`)
    if (await ask({ title: label, message: t(`shared.document.confirm.${action}`, { number }), confirmLabel: label, danger: true })) await act(action, call)
  }

  const has = (a: string) => actions.includes(a)
  return (
    <Group gap="xs" justify="flex-end">
      {has('delete') && onDelete && (
        <Button variant="outline" color="danger" leftSection={<IconTrash {...icon.button} />} disabled={busy} onClick={() => void destructive('delete', onDelete)}>
          {t('shared.document.action.delete')}
        </Button>
      )}
      {has('cancel') && (
        <Button variant="outline" color="danger" leftSection={<IconX {...icon.button} />} disabled={busy} onClick={() => void destructive('cancel', transition('cancelled'))}>
          {t('shared.document.action.cancel')}
        </Button>
      )}
      {has('withdraw') && (
        <Button variant="default" leftSection={<IconArrowBackUp {...icon.button} />} disabled={busy} onClick={() => void act('withdraw', transition('draft'))}>
          {t('shared.document.action.withdraw')}
        </Button>
      )}
      {has('submit') && (
        <Button variant="default" leftSection={<IconSend {...icon.button} />} disabled={busy} onClick={() => void act('submit', transition('posted'))}>
          {t('shared.document.action.submit')}
        </Button>
      )}
      {confirm}
      {dialog}
    </Group>
  )
}
