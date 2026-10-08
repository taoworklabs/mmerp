import { IconSettings, IconAdjustments, IconLock, IconMail, IconSitemap, IconUserShield } from '@tabler/icons-react'
import { lazy } from 'react'
import type { AreaManifest } from '@/shared/area'

// The home page and login live at the root; the admin screens under /admin.
export const core: AreaManifest = {
  product: 'core',
  label: 'core.nav.group',
  basePath: '/admin',
  routes: () => import('./routes'),
  nav: [
    { label: 'core.nav.org_units', path: 'org-units', icon: IconSitemap, permission: 'core.org.manage' },
    { label: 'core.nav.users', path: 'users', icon: IconUserShield, permission: 'core.user.manage' },
    { label: 'core.nav.period_locks', path: 'period-locks', icon: IconLock, permission: 'core.period.manage' },
    { label: 'core.nav.settings', path: 'settings', icon: IconAdjustments, permission: 'core.setting.manage' },
    { label: 'core.nav.mail_server', path: 'mail-server', icon: IconMail, permission: 'core.mail.manage' },
  ],
  homeIcon: IconSettings,
  i18n: {
    // Holds the login screen too, so it is loaded before /me.
    meta: { vi: () => import('./i18n/vi/meta.json'), en: () => import('./i18n/en/meta.json') },
    main: { vi: () => import('./i18n/vi/main.json'), en: () => import('./i18n/en/main.json') },
  },
}

export const HomePage = lazy(() => import('./pages/HomePage'))
// At the root, not under /admin: every user has an inbox. Its strings are in meta.
export const InboxPage = lazy(() => import('./pages/InboxPage'))
// At the root as well: every user follows their own background jobs. Its strings are in meta.
export const JobsPage = lazy(() => import('./pages/JobsPage'))
// At the root as well: every user has notifications. Its strings are in meta.
export const NotificationsPage = lazy(() => import('./pages/NotificationsPage'))
export { notificationKeys } from './notificationKeys'
// Eager: shown before /me, when nothing else of the app is loaded.
export { LoginPage } from './pages/LoginPage'
export { nextFromLocation } from './login'
