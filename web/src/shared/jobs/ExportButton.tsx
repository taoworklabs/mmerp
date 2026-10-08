import { Button } from '@mantine/core'
import { IconDownload, IconFileExport } from '@tabler/icons-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { coreClient, unwrap } from '@/shared/api/client'
import { errorText } from '@/shared/i18n'
import { notifyError } from '@/shared/ui/notify'
import { icon } from '@/shared/ui/theme'
import { fileHref, jobError } from './JobOutcome'
import { finished, useJobStatus } from './useJobStatus'

// ExportButton starts an export, waits for its job, then turns into the download of the file.
// A failure is reported as a notification, since the user may have moved on.
export function ExportButton({ kind, params = {}, label }: { kind: string; params?: Record<string, unknown>; label: string }) {
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
      <Button component="a" href={fileHref(j.file_id)} variant="default" leftSection={<IconDownload {...icon.button} />} onClick={() => setJob(null)}>
        {t('shared.jobs.download')}
      </Button>
    )

  async function start() {
    setStarting(true)
    try {
      setJob((await unwrap(coreClient.POST('/exports/{kind}', { params: { path: { kind } }, body: { params } }))).job_id)
    } catch (err) {
      notifyError(errorText(t, err))
    } finally {
      setStarting(false)
    }
  }

  return (
    <Button variant="default" leftSection={<IconFileExport {...icon.button} />} loading={starting || (job !== null && !finished(j))} onClick={() => void start()}>
      {label}
    </Button>
  )
}
