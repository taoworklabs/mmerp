import { Button, Group } from '@mantine/core'
import { notifications } from '@mantine/notifications'

// notifySuccess is the toast after a save: bottom right, gone after 3 seconds.
export function notifySuccess(message: string) {
  notifications.show({ color: 'success', message })
}

// notifyError reports a background job that failed while the user was elsewhere; it stays until closed.
export function notifyError(message: string) {
  notifications.show({ color: 'danger', message, autoClose: false })
}

// notifyUndo confirms a reversible action that already took effect, with a way back, for 5 seconds.
// Only for actions that write no ledger, approval or notification; lifecycle actions ask first instead.
export function notifyUndo(message: string, undoLabel: string, onUndo: () => void) {
  const id = notifications.show({
    color: 'success',
    autoClose: 5000,
    message: (
      <UndoMessage
        message={message}
        undoLabel={undoLabel}
        onUndo={() => {
          notifications.hide(id)
          onUndo()
        }}
      />
    ),
  })
}

// UndoMessage is the body of an undo toast, exported so the catalogue can show it at rest.
export function UndoMessage({ message, undoLabel, onUndo }: { message: string; undoLabel: string; onUndo: () => void }) {
  return (
    <Group justify="space-between" wrap="nowrap" gap="sm">
      <span>{message}</span>
      <Button variant="subtle" onClick={onUndo}>
        {undoLabel}
      </Button>
    </Group>
  )
}
