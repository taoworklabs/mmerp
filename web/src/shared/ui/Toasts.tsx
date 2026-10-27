import { Toaster, useMediaQuery } from '@mmerp/ui'
import { useTranslation } from 'react-i18next'

// Toasts sit bottom right; below 1024px they go to the top, clear of a document's action bar.
export function Toasts() {
  const { t } = useTranslation()
  const wide = useMediaQuery('(min-width: 64em)', true)
  return <Toaster position={wide ? 'bottom-right' : 'top-center'} closeLabel={t('shared.toast.close')} />
}
