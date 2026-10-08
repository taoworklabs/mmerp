import { Center } from '@mantine/core'
import { useTranslation } from 'react-i18next'
import { ErrorState } from '@/shared/ui/states'

// The server could not answer /me; there is nothing to render but a retry.
export function BootError() {
  const { t } = useTranslation()
  return (
    <Center mih="100vh" p="md">
      <ErrorState message={t('shared.error.network_error')} onRetry={() => location.reload()} />
    </Center>
  )
}
