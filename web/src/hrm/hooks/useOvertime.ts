import { useQuery } from '@tanstack/react-query'
import { api, unwrap } from '@/shared/api/hrm'
import { hrmKeys } from '../keys'

// useOvertimeActions: what the user may do before picking an overtime request, and their own employee record.
export function useOvertimeActions() {
  return useQuery({ queryKey: hrmKeys.overtimes.actions(), queryFn: () => unwrap(api.GET('/hrm/overtimes/actions')) })
}

export function useOvertime(id: number) {
  return useQuery({ queryKey: hrmKeys.overtimes.detail(id), queryFn: () => unwrap(api.GET('/hrm/overtimes/{id}', { params: { path: { id } } })) })
}
