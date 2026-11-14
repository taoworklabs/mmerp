import type { ReactNode } from 'react'
import { Navigate, Route, Routes } from 'react-router'
import { useCan } from '@/shared/auth/me'
import { ApprovalRules } from '@/shared/document'
import { ForbiddenPage, NotFoundPage } from '@/shared/ui/states'
import { AccountsPage } from './pages/AccountsPage'
import { JournalEntriesPage } from './pages/JournalEntriesPage'
import { JournalEntryPage } from './pages/JournalEntryPage'
import { LedgerPage } from './pages/LedgerPage'
import { NewJournalEntryPage } from './pages/NewJournalEntryPage'
import { PartnerBalancesPage } from './pages/PartnerBalancesPage'
import { PartnerLedgerPage } from './pages/PartnerLedgerPage'
import { PendingPage } from './pages/PendingPage'
import { RulesPage } from './pages/RulesPage'
import { StartsPage } from './pages/StartsPage'
import { TrialBalancePage } from './pages/TrialBalancePage'

export default function AccountingRoutes() {
  const canView = useCan('accounting.view')
  const canManage = useCan('accounting.manage')
  const canRules = useCan(['accounting.approval.manage', 'core.approval.manage'])
  const view = (page: ReactNode) => (canView ? page : <ForbiddenPage />)
  const manage = (page: ReactNode) => (canManage ? page : <ForbiddenPage />)
  return (
    <Routes>
      <Route index element={<Navigate to={canView ? 'journal-entries' : 'approval-rules'} replace />} />
      <Route path="journal-entries" element={view(<JournalEntriesPage />)} />
      {/* Without create in the type's actions the new page itself answers forbidden. */}
      <Route path="journal-entries/new" element={view(<NewJournalEntryPage />)} />
      {/* The API decides per record; one outside the user's reach answers 404. */}
      <Route path="journal-entries/:id" element={view(<JournalEntryPage />)} />
      <Route path="ledger" element={view(<LedgerPage />)} />
      <Route path="trial-balance" element={view(<TrialBalancePage />)} />
      <Route path="partner-balances" element={view(<PartnerBalancesPage />)} />
      <Route path="partner-ledger" element={view(<PartnerLedgerPage />)} />
      <Route path="pending" element={view(<PendingPage />)} />
      <Route path="accounts" element={manage(<AccountsPage />)} />
      <Route path="rules" element={manage(<RulesPage />)} />
      <Route path="starts" element={manage(<StartsPage />)} />
      <Route path="approval-rules/*" element={canRules ? <ApprovalRules product="accounting" basePath="/accounting/approval-rules" /> : <ForbiddenPage />} />
      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  )
}
