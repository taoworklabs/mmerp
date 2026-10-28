import { Button } from '@mmerp/ui'
import { IconDownload, IconFileExport, type Icon } from '@tabler/icons-react'
import type { QueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { coreClient, unwrap } from '@/shared/api/client'
import { errorText } from '@/shared/i18n'
import { notifyError, notifySuccess } from '@/shared/ui/notify'
import { icon } from '@/shared/ui/theme'
import { fileHref, jobError } from './JobOutcome'
import { finished, followJob, jobKeys, useJobStatus } from './useJobStatus'

const startExport = async (kind: string, params: Record<string, unknown>) =>
  (await unwrap(coreClient.POST('/exports/{kind}', { params: { path: { kind } }, body: { params } }))).job_id

// exportInBackground starts an export from a menu entry, which has no room to wait for it: it
// follows the job and tells the user how it ended; the file is in their background jobs.
export async function exportInBackground(qc: QueryClient, t: (key: string) => string, kind: string, params: Record<string, unknown>) {
  try {
    const id = await startExport(kind, params)
    notifySuccess(t('shared.jobs.export_started'))
    const job = await followJob(qc, id, () => qc.invalidateQueries({ queryKey: jobKeys.list() }))
    if (job?.state === 'failed') notifyError(jobError(t, job))
    else if (job) notifySuccess(t('shared.jobs.exported'))
  } catch (err) {
    notifyError(errorText(t as never, err))
  }
}

// ExportButton starts an export, waits for its job, then turns into the download of the file.
// A failure is reported as a notification, since the user may have moved on. A print is an
// export too, shown with its own icon.
export function ExportButton({
  kind,
  params = {},
  label,
  icon: StartIcon = IconFileExport,
}: {
  kind: string
  params?: Record<string, unknown>
  label: string
  icon?: Icon
}) {
  const { t } = useTranslation()
  const [job, setJob] = useState<number | null>(null)
  const [starting, setStarting] = useState(false)
  const status = useJobStatus(job)
  const j = status.data

  useEffect(() => {
    if (j?.state === 'failed') notifyError(jobError(t, j))
    else if (status.isError) notifyError(errorText(t, status.error))
    else return
    setJob(null)
  }, [j, status.isError, status.error, t])

  if (job !== null && j?.state === 'completed' && j.file_id)
    return (
      <Button render={<a href={fileHref(j.file_id)} />} variant="default" leftSection={<IconDownload {...icon.button} />} onClick={() => setJob(null)}>
        {t('shared.jobs.download')}
      </Button>
    )

  async function start() {
    setStarting(true)
    try {
      setJob(await startExport(kind, params))
    } catch (err) {
      notifyError(errorText(t, err))
    } finally {
      setStarting(false)
    }
  }

  return (
    <Button variant="default" leftSection={<StartIcon {...icon.button} />} loading={starting || (job !== null && !finished(j))} onClick={() => void start()}>
      {label}
    </Button>
  )
}
