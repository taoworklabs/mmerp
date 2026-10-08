import { useQuery } from '@tanstack/react-query'
import { api, unwrap } from '@/shared/api/hrm'
import { hrmKeys } from '../keys'

export function useContractTypes() {
  return useQuery({ queryKey: hrmKeys.contractTypes(), queryFn: async () => (await unwrap(api.GET('/hrm/contract-types'))) ?? [] })
}

// useContract reads a contract; with the salary permission the read is audited, so it is not refetched on focus.
export function useContract(id: number, enabled = true) {
  return useQuery({
    queryKey: hrmKeys.contracts.detail(id),
    queryFn: () => unwrap(api.GET('/hrm/contracts/{id}', { params: { path: { id } } })),
    enabled,
    refetchOnWindowFocus: false,
  })
}
