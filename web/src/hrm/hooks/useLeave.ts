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

export function useLeave(id: number) {
  return useQuery({ queryKey: hrmKeys.leaves.detail(id), queryFn: () => unwrap(api.GET('/hrm/leaves/{id}', { params: { path: { id } } })) })
}
