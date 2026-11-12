// The record type of a manual journal entry, as registered by the backend.
export const journalEntryDocType = 'accounting.journal_entry'

// Query keys of the accounting area; every parameter that changes a result is part of its key.
export type JournalEntryQuery = { q: string; status: string; legal_entity: string; sort: string; page: number; pageSize: number }
export type RangeQuery = { legal_entity_id: number; from: string; to: string }

export const accountingKeys = {
  accounts: () => ['accounting', 'accounts'] as const,
  rules: () => ['accounting', 'rules'] as const,
  starts: () => ['accounting', 'starts'] as const,
  // Everything an entry or a posted document moves: entries, ledger, trial balance, pending.
  books: () => ['accounting', 'books'] as const,
  entries: {
    all: () => ['accounting', 'books', 'entries'] as const,
    actions: () => ['accounting', 'books', 'entries', 'actions'] as const,
    list: (q: JournalEntryQuery) => ['accounting', 'books', 'entries', 'list', q] as const,
    detail: (id: number) => ['accounting', 'books', 'entries', 'detail', id] as const,
  },
  ledger: (q: RangeQuery & { account_id: number }) => ['accounting', 'books', 'ledger', q] as const,
  trialBalance: (q: RangeQuery) => ['accounting', 'books', 'trial-balance', q] as const,
  partnerBalances: (q: RangeQuery & { account_id?: number }) => ['accounting', 'books', 'partner-balances', q] as const,
  partnerLedger: (q: RangeQuery & { partner_id: number; account_id?: number }) => ['accounting', 'books', 'partner-ledger', q] as const,
  partners: (q: string) => ['accounting', 'partners', q] as const,
  pending: (legalEntity: number) => ['accounting', 'books', 'pending', legalEntity] as const,
}
