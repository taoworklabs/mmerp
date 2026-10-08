import { useQuery } from '@tanstack/react-query'
import { api, unwrap } from '@/shared/api/hrm'
import { hrmKeys } from '../keys'

// usePayrollActions: what the user may do before picking a payroll (create).
export function usePayrollActions() {
  return useQuery({
    queryKey: hrmKeys.payrolls.actions(),
    queryFn: async () => (await unwrap(api.GET('/hrm/payrolls/actions'))).allowed_actions,
  })
}

// usePayroll reads a payroll; amounts of a person are never kept once the screen is left.
export function usePayroll(id: number) {
  return useQuery({ queryKey: hrmKeys.payrolls.detail(id), queryFn: () => unwrap(api.GET('/hrm/payrolls/{id}', { params: { path: { id } } })), gcTime: 0 })
}
