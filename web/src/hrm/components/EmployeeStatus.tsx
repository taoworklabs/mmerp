import { useTranslation } from 'react-i18next'
import { StatusBadge } from '@/shared/ui/StatusBadge'

// EmployeeStatus shows the status the API computed; the frontend never derives it.
export function EmployeeStatus({ status }: { status: 'active' | 'terminated' }) {
  const { t } = useTranslation()
  // An unknown value shows nothing rather than a guess.
  if (status === 'terminated') return <StatusBadge tone="neutral">{t('hrm.employee.status.terminated')}</StatusBadge>
  if (status === 'active') return <StatusBadge tone="positive">{t('hrm.employee.status.active')}</StatusBadge>
  return null
}
