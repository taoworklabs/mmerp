import { useQuery } from '@tanstack/react-query'
import { api, unwrap } from '@/shared/api/core'

// Query keys of the shared master data, under each one's namespace.
export const masterKeys = {
  partners: {
    all: () => ['partner'] as const,
    picker: (q: string) => ['partner', 'picker', q] as const,
    detail: (id: number) => ['partner', 'detail', id] as const,
  },
}

// usePartner: one partner with what the user may do to it; off until there is an id.
export function usePartner(id: number | undefined) {
  return useQuery({
    queryKey: masterKeys.partners.detail(id ?? 0),
    queryFn: () => unwrap(api.GET('/partners/{id}', { params: { path: { id: id! } } })),
    enabled: !!id,
  })
}
