import { useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { errorText } from '@/shared/i18n'
import { useConfirm, type ConfirmOptions } from './confirm'
import { notifySuccess } from './notify'

// useAction runs one of a region's actions (delete a row, revoke, send a test): asks first when
// given `confirm`, keeps the failure as `error` for the region to show next to its cause, and
// toasts `success`. `fn` does the call and whatever must follow it (refetch). run resolves to
// whether the action went through. Render `dialog` once in the region.
export function useAction(): {
  run: (fn: () => Promise<unknown>, opts?: { confirm?: ConfirmOptions; success?: string }) => Promise<boolean>
  error: string | null
  dialog: ReactNode
} {
  const { t } = useTranslation()
  const [error, setError] = useState<string | null>(null)
  const [ask, dialog] = useConfirm()
  async function run(fn: () => Promise<unknown>, opts: { confirm?: ConfirmOptions; success?: string } = {}) {
    if (opts.confirm && !(await ask(opts.confirm))) return false
    setError(null)
    try {
      await fn()
    } catch (err) {
      setError(errorText(t, err))
      return false
    }
    if (opts.success) notifySuccess(opts.success)
    return true
  }
  return { run, error, dialog }
}
