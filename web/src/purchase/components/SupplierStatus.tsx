import { useTranslation } from 'react-i18next'
import { StatusBadge } from '@/shared/ui/StatusBadge'

export function SupplierStatus({ active }: { active: boolean }) {
  const { t } = useTranslation()
  return <StatusBadge tone={active ? 'success' : 'neutral'}>{t(active ? 'purchase.supplier.status.active' : 'purchase.supplier.status.inactive')}</StatusBadge>
}
