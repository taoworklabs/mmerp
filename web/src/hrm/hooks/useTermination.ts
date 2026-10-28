import { useQuery } from '@tanstack/react-query'
import { api, unwrap } from '@/shared/api/hrm'
import { hrmKeys } from '../keys'

// useTermination reads a termination; with the salary permission the allowance read is
// audited, so it is not refetched on focus.
export function useTermination(id: number) {
  return useQuery({
    queryKey: hrmKeys.terminations.detail(id),
    queryFn: () => unwrap(api.GET('/hrm/terminations/{id}', { params: { path: { id } } })),
    refetchOnWindowFocus: false,
  })
}
