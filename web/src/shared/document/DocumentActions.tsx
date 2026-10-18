import { Button, Group } from '@mantine/core'
import { IconArrowBackUp, IconPrinter, IconSend, IconTrash, IconX } from '@tabler/icons-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router'
import { api, unwrap } from '@/shared/api/core'
import { errorText } from '@/shared/i18n'
import { ExportButton } from '@/shared/jobs'
import { useConfirm } from '@/shared/ui/confirm'
import { useUnsaved } from '@/shared/ui/form'
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
  // The list the page leaves for once its draft is deleted.
  listPath: string
  // Where the page shows a failed action: a banner at its top.
  onError: (message: string | null) => void
  // The print button, shown when print is allowed: its label, and the template's export
  // kind when it is not printing.<docType>.
  printLabel?: string
  printKind?: string
}

// DocumentActions draws the lifecycle buttons from allowed_actions and runs them: Submit is the
// primary button, Delete and Cancel quiet red text that asks before acting. Print starts the
// print of the document's template.
export function DocumentActions({ docType, id, version, number, actions, listPath, onError, printLabel, printKind }: Props) {
  const { t } = useTranslation()
  const { run, dialog } = useDocumentMutation(docType, id, version)
  const [ask, confirm] = useConfirm()
  const [busy, setBusy] = useState(false)
  const unsaved = useUnsaved()
  const navigate = useNavigate()

  async function act(action: string, call: (version: number) => Promise<unknown>) {
    // Moving on would reload the form at the new version and drop what is not saved yet.
    if (unsaved && (action === 'submit' || action === 'withdraw')) return onError(t('shared.document.save_first'))
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

  // Leaves the page before the refresh, so nothing reloads the deleted draft.
  const remove = async (v: number) => {
    await unwrap(api.DELETE('/documents/{type}/{id}', { params: { path: { type: docType, id }, query: { version: v } } }))
    navigate(listPath, { replace: true })
  }

  const destructive = async (action: 'cancel' | 'delete', call: (v: number) => Promise<unknown>) => {
    const label = t(`shared.document.action.${action}`)
    if (await ask({ title: label, message: t(`shared.document.confirm.${action}`, { number }), confirmLabel: label, danger: true })) await act(action, call)
  }

  // Withdrawing undoes the approvals given so far, so it asks first, though nothing is lost for good.
  const withdraw = async () => {
    const label = t('shared.document.action.withdraw')
    if (await ask({ title: label, message: t('shared.document.confirm.withdraw', { number }), confirmLabel: label })) await act('withdraw', transition('draft'))
  }

  const has = (a: string) => actions.includes(a)
  return (
    <Group gap="xs" justify="flex-end">
      {has('print') && (
        <ExportButton kind={printKind ?? `printing.${docType}`} params={{ id }} label={printLabel ?? t('shared.document.action.print')} icon={IconPrinter} />
      )}
      {has('delete') && (
        <Button variant="subtle" color="danger" leftSection={<IconTrash {...icon.button} />} disabled={busy} onClick={() => void destructive('delete', remove)}>
          {t('shared.document.action.delete')}
        </Button>
      )}
      {has('cancel') && (
        <Button variant="subtle" color="danger" leftSection={<IconX {...icon.button} />} disabled={busy} onClick={() => void destructive('cancel', transition('cancelled'))}>
          {t('shared.document.action.cancel')}
        </Button>
      )}
      {has('withdraw') && (
        <Button variant="default" leftSection={<IconArrowBackUp {...icon.button} />} disabled={busy} onClick={() => void withdraw()}>
          {t('shared.document.action.withdraw')}
        </Button>
      )}
      {has('submit') && (
        // Sending on is the next step of a draft, so it is the page's primary button.
        <Button leftSection={<IconSend {...icon.button} />} disabled={busy} onClick={() => void act('submit', transition('posted'))}>
          {t('shared.document.action.submit')}
        </Button>
      )}
      {confirm}
      {dialog}
    </Group>
  )
}
