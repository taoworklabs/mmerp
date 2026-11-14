import { IconBook, IconBook2, IconCalculator, IconCalendarEvent, IconClockHour4, IconGitBranch, IconListNumbers, IconRoute, IconScale, IconUsers } from '@tabler/icons-react'
import type { AreaManifest } from '@/shared/area'
import { accountingKeys, journalEntryDocType } from './keys'

export const accounting: AreaManifest = {
  product: 'accounting',
  label: 'accounting.nav.group',
  basePath: '/accounting',
  routes: () => import('./routes'),
  nav: [
    { label: 'accounting.nav.journal_entries', path: 'journal-entries', icon: IconBook, permission: 'accounting.view' },
    { label: 'accounting.nav.ledger', path: 'ledger', icon: IconBook2, permission: 'accounting.view' },
    { label: 'accounting.nav.trial_balance', path: 'trial-balance', icon: IconScale, permission: 'accounting.view' },
    { label: 'accounting.nav.partner_balances', path: 'partner-balances', icon: IconUsers, permission: 'accounting.view' },
    { label: 'accounting.nav.pending', path: 'pending', icon: IconClockHour4, permission: 'accounting.view' },
    { label: 'accounting.nav.accounts', path: 'accounts', icon: IconListNumbers, permission: 'accounting.manage', group: 'accounting.nav.group_config' },
    { label: 'accounting.nav.rules', path: 'rules', icon: IconRoute, permission: 'accounting.manage', group: 'accounting.nav.group_config' },
    { label: 'accounting.nav.starts', path: 'starts', icon: IconCalendarEvent, permission: 'accounting.manage', group: 'accounting.nav.group_config' },
    {
      label: 'accounting.nav.approval_rules',
      path: 'approval-rules',
      icon: IconGitBranch,
      permission: ['accounting.approval.manage', 'core.approval.manage'],
      group: 'accounting.nav.group_config',
    },
  ],
  collapsibleGroups: ['accounting.nav.group_config'],
  homeIcon: IconCalculator,
  i18n: {
    meta: { vi: () => import('./i18n/vi/meta.json'), en: () => import('./i18n/en/meta.json') },
    main: { vi: () => import('./i18n/vi/main.json'), en: () => import('./i18n/en/main.json') },
  },
  recordTypes: {
    [journalEntryDocType]: {
      path: (id) => `journal-entries/${id}`,
      preview: () => import('./components/JournalEntryPreview'),
      // Posting or cancelling an entry moves the ledger, the trial balance and what is pending.
      invalidate: () => [accountingKeys.books()],
    },
  },
}
