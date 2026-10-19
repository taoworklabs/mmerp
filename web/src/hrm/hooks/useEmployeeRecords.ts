import { useQuery } from '@tanstack/react-query'
import { api, unwrap, type ContractListItem, type Employee } from '@/shared/api/hrm'
import { hrmKeys } from '../keys'

// An employee's contracts and leave balances, one query each, shared by the page's summary and its tabs.
export function useEmployeeContracts(e: Employee) {
  return useQuery({
    queryKey: hrmKeys.contracts.employee(e.id),
    queryFn: async () => (await unwrap(api.GET('/hrm/employees/{id}/contracts', { params: { path: { id: e.id } } }))) ?? [],
    enabled: e.allowed_actions.includes('view_contracts'),
  })
}

export function useLeaveBalances(e: Employee) {
  return useQuery({
    queryKey: hrmKeys.balances.employee(e.id),
    queryFn: async () => (await unwrap(api.GET('/hrm/employees/{id}/leave-balances', { params: { path: { id: e.id } } }))) ?? [],
  })
}

// currentContract is the original contract in force on day (YYYY-MM-DD); appendices only amend it.
export function currentContract(list: ContractListItem[] | undefined, day: string) {
  return list?.find((c) => !c.parent_id && c.status === 'posted' && c.start_date <= day && (!c.end_date || c.end_date >= day))
}
