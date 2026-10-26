import { toast } from '@mmerp/ui'

// notifySuccess is the toast after a save: bottom right, gone after 3 seconds.
export function notifySuccess(message: string) {
  toast.show({ color: 'success', message })
}

// notifyError reports a background job that failed while the user was elsewhere; it stays until closed.
export function notifyError(message: string) {
  toast.show({ color: 'danger', message, timeout: null })
}

// notifyUndo confirms a reversible action that already took effect, with a way back, for 5 seconds.
// Only for actions that write no ledger, approval or notification; lifecycle actions ask first instead.
export function notifyUndo(message: string, undoLabel: string, onUndo: () => void) {
  toast.show({ color: 'success', timeout: 5000, message, action: { label: undoLabel, onClick: onUndo } })
}
