// The Accounting API as the accounting area sees it.
import type { components, operations } from './schema.gen'

import { productClient } from './client'

export { unwrap } from './client'
export const api = productClient<'/accounting/'>()
export { ApiError } from './error'

export type Account = components['schemas']['Account']
export type AccountFields = Omit<components['schemas']['AccountFields'], '$schema'>
export type Rule = components['schemas']['Rule']
export type Start = components['schemas']['Start']
export type PendingDoc = components['schemas']['PendingDoc']
export type JournalEntry = components['schemas']['JournalEntry']
export type JournalEntryLine = components['schemas']['JournalEntryLine']
export type JournalEntryListItem = components['schemas']['JournalEntryListItem']
export type JournalEntrySort = NonNullable<NonNullable<operations['list-journal-entries']['parameters']['query']>['sort']>
export type LedgerLine = components['schemas']['LedgerLine']
export type TrialRow = components['schemas']['TrialRow']
export type PartnerBalanceRow = components['schemas']['PartnerBalanceRow']
