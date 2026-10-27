import { IconCalendarEvent, IconCalendarTime, IconReceipt, IconSitemap, IconUser, IconUserCircle } from '@tabler/icons-react'
import type { AreaManifest } from '@/shared/area'
import { useLeaveSelf } from '../hooks/useLeave'

// The personal space: the user's own days, payslips, requests and profile. It belongs to HRM
// and shares its translations; only people with an employee record see it.
export const hrmMe: AreaManifest = {
  product: 'hrm',
  label: 'hrm.me.nav.group',
  basePath: '/me',
  routes: () => import('./routes'),
  nav: [
    { label: 'hrm.me.nav.days', path: 'days', icon: IconCalendarTime },
    { label: 'hrm.me.nav.payslips', path: 'payslips', icon: IconReceipt },
    { label: 'hrm.me.nav.requests', path: 'requests', icon: IconCalendarEvent },
    { label: 'hrm.me.nav.profile', path: 'profile', icon: IconUser },
    { label: 'hrm.me.nav.org', path: 'org', icon: IconSitemap },
  ],
  homeIcon: IconUserCircle,
  useEntitled: () => {
    const self = useLeaveSelf()
    return self.isPending ? undefined : self.data != null
  },
  i18n: {
    meta: { vi: () => import('../i18n/vi/meta.json'), en: () => import('../i18n/en/meta.json') },
    main: { vi: () => import('../i18n/vi/main.json'), en: () => import('../i18n/en/main.json') },
  },
}
