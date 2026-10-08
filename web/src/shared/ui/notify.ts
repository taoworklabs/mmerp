import { notifications } from '@mantine/notifications'

// notifySuccess is the toast after a save: bottom right, gone after 3 seconds.
export function notifySuccess(message: string) {
  notifications.show({ color: 'success', message })
}

// notifyError reports a background job that failed while the user was elsewhere; it stays until closed.
export function notifyError(message: string) {
  notifications.show({ color: 'danger', message, autoClose: false })
}
