import { useTranslation } from 'react-i18next'
import { StatusBadge } from '@/shared/ui/StatusBadge'

export function CustomerStatus({ active }: { active: boolean }) {
  const { t } = useTranslation()
  return <StatusBadge tone={active ? 'positive' : 'neutral'}>{t(active ? 'sales.customer.status.active' : 'sales.customer.status.inactive')}</StatusBadge>
}
