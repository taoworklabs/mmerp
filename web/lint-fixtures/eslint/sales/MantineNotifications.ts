// Planted: an area calling Mantine notifications instead of the shared/ui helpers.
import { notifications } from '@mantine/notifications'

export const fixture = (message: string) => notifications.show({ message })
