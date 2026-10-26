// Planted: shared code calling Mantine notifications instead of the kit toast behind shared/ui/notify.
import { notifications } from '@mantine/notifications'

export const fixture = (message: string) => notifications.show({ message })
