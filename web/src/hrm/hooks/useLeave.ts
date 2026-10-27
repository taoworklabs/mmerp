import { useQuery } from '@tanstack/react-query'
import { api, unwrap, type Leave } from '@/shared/api/hrm'
import { hrmKeys } from '../keys'

const leaveActions = () => ({ queryKey: hrmKeys.leaves.actions(), queryFn: () => unwrap(api.GET('/hrm/leaves/actions')) })

// useLeaveActions: what the user may do before picking a leave request.
export function useLeaveActions() {
  return useQuery({ ...leaveActions(), select: (d) => d.allowed_actions })
}

// useLeaveSelf: the user's own employee record, whom a new request is for by default; null without one.
export function useLeaveSelf() {
  return useQuery({ ...leaveActions(), select: (d) => (d.self_employee_id ? { id: d.self_employee_id, name: d.self_employee_name ?? '' } : null) })
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

// useOverlappingLeaves: the other requests of the same department over the same days, waiting or
// approved, as far as the user may see them. Idle until the department is known.
export function useOverlappingLeaves(leave: Leave | undefined, orgUnitId: number | undefined) {
  return useQuery({
    queryKey: hrmKeys.leaves.overlapping(leave?.id ?? 0),
    queryFn: async () => {
      const { items } = await unwrap(
        api.GET('/hrm/leaves', { params: { query: { org_unit_id: orgUnitId, status: ['pending_approval', 'posted'], from: leave!.start_date, to: leave!.end_date, sort: 'start_date', page: 1, page_size: 50 } } }),
      )
      return items.filter((x) => x.id !== leave!.id)
    },
    enabled: !!leave && !!orgUnitId,
  })
}
