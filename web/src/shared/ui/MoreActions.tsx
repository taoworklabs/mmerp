import { MoreActions as KitMoreActions } from '@mmerp/ui'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

// MoreActions holds the actions that do not earn a button, as MenuAction children: at the end of a
// table row, or after a page's buttons once there are more than three.
export function MoreActions({ children, place = 'row' }: { children: ReactNode; place?: 'page' | 'row' }) {
  const { t } = useTranslation()
  return (
    <KitMoreActions label={t('shared.table.actions')} place={place}>
      {children}
    </KitMoreActions>
  )
}
