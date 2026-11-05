import { useQuery, type UseQueryResult } from '@tanstack/react-query'
import { useSearchParams } from 'react-router'
import { api, unwrap, type Account } from '@/shared/api/accounting'
import { useMe } from '@/shared/auth/me'
import { today as todayIn } from '@/shared/i18n'
import { useLegalEntities, type Option } from '@/shared/ui/form'
import { accountingKeys } from '../keys'

// useAccounts loads the whole chart of accounts, ordered by code.
export function useAccounts() {
  return useQuery({
    queryKey: accountingKeys.accounts(),
    queryFn: async () => (await unwrap(api.GET('/accounting/accounts'))).items,
  })
}

export const accountLabel = (a: Pick<Account, 'code' | 'name'>) => `${a.code} · ${a.name}`

// accountOptions lists accounts as "code · name" options, those passing keep.
export const accountOptions = (accounts: Account[] | undefined, keep: (a: Account) => boolean = () => true): Option[] =>
  (accounts ?? []).filter(keep).map((a) => ({ value: String(a.id), label: accountLabel(a) }))

// useJournalEntryActions: what the user may do before picking a journal entry (create).
export function useJournalEntryActions() {
  return useQuery({
    queryKey: accountingKeys.entries.actions(),
    queryFn: async () => (await unwrap(api.GET('/accounting/journal-entries/actions'))).allowed_actions,
  })
}

export function useJournalEntry(id: number) {
  return useQuery({
    queryKey: accountingKeys.entries.detail(id),
    queryFn: () => unwrap(api.GET('/accounting/journal-entries/{id}', { params: { path: { id } } })),
  })
}

// asList feeds rows loaded whole, without paging, to a ListPage.
export const asList = <T>(q: UseQueryResult<unknown>, items: T[] | undefined) => ({
  isPending: q.isPending,
  isError: q.isError,
  error: q.error,
  refetch: q.refetch,
  data: items && { items, total: items.length },
})

// useReportParams reads a report's legal entity, account and range from the URL; the range
// defaults to the current month up to today, the legal entity to the first the user may view.
export function useReportParams() {
  const today = todayIn(useMe().timezone)
  const [search, setSearch] = useSearchParams()
  const entities = useLegalEntities({ product: 'accounting', permission: 'accounting.view' }, search.get('legal_entity_id') ?? undefined)
  const p = {
    legal_entity_id: entities.chosen ?? 0,
    account_id: Number(search.get('account_id')) || 0,
    from: search.get('from') || `${today.slice(0, 7)}-01`,
    to: search.get('to') || today,
  }
  // The chips edit the URL, which stays the source, so Back reaches them.
  const put = (key: keyof typeof p) => (v: string | null) => {
    const next = Object.entries({ ...p, [key]: v ?? '' }).filter((e) => !!e[1])
    setSearch(new URLSearchParams(next.map(([k, x]) => [k, String(x)])))
  }
  return { p, put, entities }
}
