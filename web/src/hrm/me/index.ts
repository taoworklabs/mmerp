import { IconCalendarEvent, IconCalendarTime, IconClockHour4, IconReceipt, IconSitemap, IconUser, IconUserCircle } from '@tabler/icons-react'
import type { AreaManifest } from '@/shared/area'
import { useHasReports, useLeaveSelf } from '../hooks/useLeave'
import { leaveDocType, overtimeDocType } from '../keys'
import { leaveRecord, overtimeRecord } from '../records'

// The personal space: the user's own days, payslips, requests and profile, and a direct manager's
// reports' requests. It belongs to HRM and shares its translations; only people with an employee
// record see it, and it is the only HRM home of those without an HRM role.
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
    { label: 'hrm.me.nav.team_leaves', path: 'team/leaves', icon: IconCalendarEvent, group: 'hrm.me.nav.group_team', useShown: useHasReports },
    { label: 'hrm.me.nav.team_overtimes', path: 'team/overtimes', icon: IconClockHour4, group: 'hrm.me.nav.group_team', useShown: useHasReports },
  ],
  homeIcon: IconUserCircle,
  useEntitled: () => {
    const self = useLeaveSelf()
    return self.isPending ? undefined : self.data != null
  },
  // For someone who may not enter HRM, a request opens here; the HRM manifest comes first and wins otherwise.
  recordTypes: {
    [leaveDocType]: { ...leaveRecord, path: (id) => `requests/leave/${id}` },
    [overtimeDocType]: { ...overtimeRecord, path: (id) => `requests/overtime/${id}` },
  },
  i18n: {
    meta: { vi: () => import('../i18n/vi/meta.json'), en: () => import('../i18n/en/meta.json') },
    main: { vi: () => import('../i18n/vi/main.json'), en: () => import('../i18n/en/main.json') },
  },
}
