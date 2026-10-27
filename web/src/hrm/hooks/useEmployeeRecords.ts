import { useQuery } from '@tanstack/react-query'
import { api, unwrap, type Employee } from '@/shared/api/hrm'
import { hrmKeys } from '../keys'

// An employee's contracts and leave balances, one query each, shared by the page's summary and its tabs.
// Contracts wait for the employee, whose allowed_actions say whether they may be read.
export function useEmployeeContracts(e: Employee | undefined) {
  const id = e?.id ?? 0
  return useQuery({
    queryKey: hrmKeys.contracts.employee(id),
    queryFn: async () => (await unwrap(api.GET('/hrm/employees/{id}/contracts', { params: { path: { id } } }))) ?? [],
    enabled: !!e?.allowed_actions.includes('view_contracts'),
  })
}

export function useLeaveBalances(e: Employee) {
  return useQuery({
    queryKey: hrmKeys.balances.employee(e.id),
    queryFn: async () => (await unwrap(api.GET('/hrm/employees/{id}/leave-balances', { params: { path: { id: e.id } } }))) ?? [],
  })
}
