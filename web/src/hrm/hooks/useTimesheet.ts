import { useQuery } from '@tanstack/react-query'
import { api, unwrap } from '@/shared/api/hrm'
import { hrmKeys } from '../keys'

// useTimesheetActions: what the user may do before picking a timesheet (create).
export function useTimesheetActions() {
  return useQuery({
    queryKey: hrmKeys.timesheets.actions(),
    queryFn: async () => (await unwrap(api.GET('/hrm/timesheets/actions'))).allowed_actions,
  })
}

export function useTimesheet(id: number) {
  return useQuery({ queryKey: hrmKeys.timesheets.detail(id), queryFn: () => unwrap(api.GET('/hrm/timesheets/{id}', { params: { path: { id } } })) })
}
