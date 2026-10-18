import { useQuery, useQueryClient, type QueryClient, type QueryKey } from '@tanstack/react-query'
import { useCallback, useEffect, useRef } from 'react'
import { useInvalidateRecord } from '@/shared/area'
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

// What a job writes: a record, refreshed as its area declares, or other queries.
export type JobWrites = { docType: string; id: number } | QueryKey[]

// useJobRefresh is what to run once a job has ended: refresh the job list and what it wrote.
export function useJobRefresh(writes: JobWrites = []) {
  const qc = useQueryClient()
  const invalidateRecord = useInvalidateRecord()
  const target = useRef(writes)
  target.current = writes
  return useCallback(async () => {
    const w = target.current
    await Promise.all([
      qc.invalidateQueries({ queryKey: jobKeys.list() }),
      Array.isArray(w) ? Promise.all(w.map((queryKey) => qc.invalidateQueries({ queryKey }))) : invalidateRecord(w.docType, w.id),
    ])
  }, [qc, invalidateRecord])
}

// useJobStatus follows a job until it ends (see pollDelay), then refreshes the job list
// and what the job wrote, e.g. the record an import wrote to. It stops on a failed request.
export function useJobStatus(id: number | null, writes?: JobWrites) {
  const refresh = useJobRefresh(writes)
  const started = useRef(0)
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
    if (done) void refresh()
  }, [done, refresh])
  return q
}

// followJob keeps following a job no screen shows any more (its dialog was closed early),
// so what it writes is still refreshed when it ends (refresh from useJobRefresh).
export async function followJob(qc: QueryClient, id: number, refresh: () => Promise<void>) {
  const started = Date.now()
  for (;;) {
    await new Promise((resolve) => setTimeout(resolve, pollDelay(Date.now() - started)))
    let job: Job
    try {
      job = await qc.fetchQuery({ queryKey: jobKeys.detail(id), queryFn: () => fetchJob(id), staleTime: 0 })
    } catch {
      return
    }
    if (finished(job)) return refresh()
  }
}
