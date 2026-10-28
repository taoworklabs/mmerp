import type { RecordTypeEntry } from '@/shared/area'
import { hrmKeys } from './keys'

// The request types both HRM manifests declare; each adds the path of its own page.
export const leaveRecord: Omit<RecordTypeEntry, 'path'> = {
  preview: () => import('./components/LeavePreview'),
  invalidate: (id) => [hrmKeys.leaves.detail(id), hrmKeys.leaves.lists(), hrmKeys.balances.all()],
}

export const overtimeRecord: Omit<RecordTypeEntry, 'path'> = {
  preview: () => import('./components/OvertimePreview'),
  invalidate: (id) => [hrmKeys.overtimes.detail(id), hrmKeys.overtimes.lists()],
}
