import { Route, Routes } from 'react-router'
import { useCan } from '@/shared/auth/me'
import { ForbiddenPage, NotFoundPage } from '@/shared/ui/states'
import { MailServerPage } from './pages/MailServerPage'
import { OrgUnitsPage } from './pages/OrgUnitsPage'
import { PeriodLocksPage } from './pages/PeriodLocksPage'
import { SettingsPage } from './pages/SettingsPage'
import { UserPage } from './pages/UserPage'
import { UsersPage } from './pages/UsersPage'

export default function CoreRoutes() {
  const canOrg = useCan('core.org.manage')
  const canUsers = useCan('core.user.manage')
  const canPeriods = useCan('core.period.manage')
  const canSettings = useCan('core.setting.manage')
  const canMail = useCan('core.mail.manage')
  return (
    <Routes>
      <Route path="org-units" element={canOrg ? <OrgUnitsPage /> : <ForbiddenPage />} />
      <Route path="users" element={canUsers ? <UsersPage /> : <ForbiddenPage />} />
      <Route path="users/:id" element={canUsers ? <UserPage /> : <ForbiddenPage />} />
      <Route path="period-locks" element={canPeriods ? <PeriodLocksPage /> : <ForbiddenPage />} />
      <Route path="settings" element={canSettings ? <SettingsPage /> : <ForbiddenPage />} />
      <Route path="mail-server" element={canMail ? <MailServerPage /> : <ForbiddenPage />} />
      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  )
}
