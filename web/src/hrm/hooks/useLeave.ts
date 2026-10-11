import { useQuery } from '@tanstack/react-query'
import { api, unwrap } from '@/shared/api/hrm'
import { hrmKeys } from '../keys'

// useLeaveActions: what the user may do before picking a leave request, and their own employee record.
export function useLeaveActions() {
  return useQuery({ queryKey: hrmKeys.leaves.actions(), queryFn: () => unwrap(api.GET('/hrm/leaves/actions')) })
}

export function useLeaveTypes() {
  return useQuery({ queryKey: hrmKeys.leaveTypes(), queryFn: async () => (await unwrap(api.GET('/hrm/leave-types'))) ?? [] })
}

// useLeaveWorkingDays: how many days the chosen range can hold, the ceiling for the day
// count. Idle until both dates are set, since that is when the ceiling is knowable.
export function useLeaveWorkingDays(start: string | null, end: string | null, employee?: number) {
  return useQuery({
    queryKey: hrmKeys.leaves.workingDays(start, end, employee),
    queryFn: () => unwrap(api.GET('/hrm/leaves/working-days', { params: { query: { start: start!, end: end!, employee_id: employee } } })),
    enabled: !!start && !!end && start <= end,
    retry: false,
  })
}

export function useLeave(id: number) {
  return useQuery({ queryKey: hrmKeys.leaves.detail(id), queryFn: () => unwrap(api.GET('/hrm/leaves/{id}', { params: { path: { id } } })) })
}
