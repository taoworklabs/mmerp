import { Center } from '@mmerp/ui'
import { useTranslation } from 'react-i18next'
import { ErrorState } from '@/shared/ui/states'
import classes from './BootError.module.css'

// The server could not answer /me; there is nothing to render but a retry.
export function BootError() {
  const { t } = useTranslation()
  return (
    <Center className={classes.root}>
      <ErrorState message={t('shared.error.network_error')} onRetry={() => location.reload()} />
    </Center>
  )
}
