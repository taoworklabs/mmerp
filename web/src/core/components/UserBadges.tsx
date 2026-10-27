import { Group } from '@mmerp/ui'
import { useTranslation } from 'react-i18next'
import type { User } from '@/shared/api/core'
import { StatusBadge } from '@/shared/ui/StatusBadge'

// UserBadges marks the root account and a locked one.
export function UserBadges({ user }: { user: Pick<User, 'root' | 'locked'> }) {
  const { t } = useTranslation()
  if (!user.root && !user.locked) return null
  return (
    <Group gap="xs">
      {user.root && <StatusBadge tone="info">{t('core.user.root')}</StatusBadge>}
      {user.locked && <StatusBadge tone="negative">{t('core.user.locked')}</StatusBadge>}
    </Group>
  )
}
