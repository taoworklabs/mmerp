import { useQuery } from '@tanstack/react-query'
import { api, unwrap } from '@/shared/api/hrm'
import { hrmKeys } from '../keys'

// useEmployeeActions: what the user may do before picking an employee (create, view_sensitive),
// as the API decides; the frontend never derives it.
export function useEmployeeActions() {
  return useQuery({
    queryKey: hrmKeys.employees.actions(),
    queryFn: async () => (await unwrap(api.GET('/hrm/employees/actions'))).allowed_actions,
  })
}
