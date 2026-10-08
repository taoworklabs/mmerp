import { useQuery, useQueryClient, type QueryClient, type QueryKey } from '@tanstack/react-query'
import { useEffect, useRef } from 'react'
import { coreClient, unwrap } from '@/shared/api/client'
import type { components } from '@/shared/api/schema.gen'
import { pollDelay } from './schedule'

export type Job = components['schemas']['Job']

// Query keys of the user's background jobs; they belong to core.
export const jobKeys = {
  all: () => ['core', 'jobs'] as const,
  list: () => ['core', 'jobs', 'list'] as const,
  detail: (id: number) => ['core', 'jobs', id] as const,
}

export const finished = (job: Job | undefined) => job?.state === 'completed' || job?.state === 'failed'

const fetchJob = (id: number) => unwrap(coreClient.GET('/jobs/{id}', { params: { path: { id } } }))

function refresh(qc: QueryClient, invalidate: QueryKey[]) {
  for (const queryKey of [jobKeys.list(), ...invalidate]) void qc.invalidateQueries({ queryKey })
}

// useJobStatus follows a job until it ends (see pollDelay), then refreshes the job list
// and the given queries, e.g. the record an import wrote to. It stops on a failed request.
export function useJobStatus(id: number | null, invalidate: QueryKey[] = []) {
  const qc = useQueryClient()
  const started = useRef(0)
  const keys = useRef(invalidate)
  keys.current = invalidate
  useEffect(() => {
    started.current = Date.now()
  }, [id])
  const q = useQuery({
    queryKey: jobKeys.detail(id ?? 0),
    queryFn: () => fetchJob(id ?? 0),
    enabled: id !== null,
    refetchInterval: (query) => (query.state.status === 'error' || finished(query.state.data) ? false : pollDelay(Date.now() - started.current)),
  })
  const done = finished(q.data)
  useEffect(() => {
    if (done) refresh(qc, keys.current)
  }, [done, qc])
  return q
}

// followJob keeps following a job no screen shows any more (its dialog was closed early),
// so the queries it changes are still refreshed when it ends.
export async function followJob(qc: QueryClient, id: number, invalidate: QueryKey[]) {
  const started = Date.now()
  for (;;) {
    await new Promise((resolve) => setTimeout(resolve, pollDelay(Date.now() - started)))
    let job: Job
    try {
      job = await qc.fetchQuery({ queryKey: jobKeys.detail(id), queryFn: () => fetchJob(id), staleTime: 0 })
    } catch {
      return
    }
    if (finished(job)) return refresh(qc, invalidate)
  }
}
