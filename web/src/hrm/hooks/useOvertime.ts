import { useQuery } from '@tanstack/react-query'
import { api, unwrap } from '@/shared/api/hrm'
import { hrmKeys } from '../keys'

const overtimeActions = () => ({ queryKey: hrmKeys.overtimes.actions(), queryFn: () => unwrap(api.GET('/hrm/overtimes/actions')) })

// useOvertimeActions: what the user may do before picking an overtime request.
export function useOvertimeActions() {
  return useQuery({ ...overtimeActions(), select: (d) => d.allowed_actions })
}

// useOvertimeSelf: the user's own employee record, whom a new request is for by default; null without one.
export function useOvertimeSelf() {
  return useQuery({ ...overtimeActions(), select: (d) => (d.self_employee_id ? { id: d.self_employee_id, name: d.self_employee_name ?? '' } : null) })
}

export function useOvertime(id: number) {
  return useQuery({ queryKey: hrmKeys.overtimes.detail(id), queryFn: () => unwrap(api.GET('/hrm/overtimes/{id}', { params: { path: { id } } })) })
}
