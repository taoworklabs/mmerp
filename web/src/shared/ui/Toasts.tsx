import { useMediaQuery } from '@mantine/hooks'
import { Notifications } from '@mantine/notifications'

// Toasts sit bottom right; below 1024px they go to the top, clear of a document's action bar.
export function Toasts() {
  const wide = useMediaQuery('(min-width: 64em)', true)
  return <Notifications position={wide ? 'bottom-right' : 'top-center'} autoClose={3000} />
}
