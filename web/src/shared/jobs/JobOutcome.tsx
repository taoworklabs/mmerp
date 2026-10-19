import { Button, Group, Loader, Stack, Text } from '@mantine/core'
import { IconDownload } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'
import { ApiError } from '@/shared/api/error'
import { errorText, formatNumber } from '@/shared/i18n'
import { DataTable } from '@/shared/ui/DataTable'
import { Notice } from '@/shared/ui/Notice'
import { icon } from '@/shared/ui/theme'
import type { Job } from './useJobStatus'

// jobError is a failed job's reason as the user reads it.
export function jobError(t: (key: string) => string, job: Job): string {
  return errorText(t as never, new ApiError(0, job.error_code ?? 'internal_error', job.error_params ?? {}))
}

export const fileHref = (id: string) => `/api/files/${id}`

// JobOutcome shows where a job is, then what came of it: the rows imported, the file
// to download, or why it failed with the rows to fix. error is a failed status request.
export function JobOutcome({ job, error }: { job: Job | undefined; error?: unknown }) {
  const { t } = useTranslation()
  if (error)
    return (
      <Notice tone="danger">
        {errorText(t, error)}
      </Notice>
    )
  if (!job || job.state === 'queued' || job.state === 'running' || job.state === 'retrying')
    return (
      <Group gap="sm" aria-live="polite">
        <Loader size="sm" />
        <Text>{t(`shared.jobs.${job?.state ?? 'queued'}`)}</Text>
      </Group>
    )
  if (job.state === 'completed')
    return (
      <Notice tone="success" announce="polite">
        {job.file_id ? (
          <Group justify="space-between">
            <Text>{t('shared.jobs.exported')}</Text>
            <Button component="a" href={fileHref(job.file_id)} leftSection={<IconDownload {...icon.button} />}>
              {t('shared.jobs.download')}
            </Button>
          </Group>
        ) : (
          t('shared.jobs.imported', { rows: formatNumber(job.rows ?? 0) })
        )}
      </Notice>
    )
  return (
    <Stack gap="sm">
      <Notice tone="danger">
        {jobError(t, job)}
      </Notice>
      {job.row_errors.length > 0 && (
        <DataTable
          label={t('shared.jobs.row_errors')}
          rows={job.row_errors}
          rowKey={(e) => `${e.row}:${e.message}`}
          columns={[
            { key: 'row', header: t('shared.jobs.row'), role: 'title', kind: 'number', render: (e) => formatNumber(e.row) },
            { key: 'message', header: t('shared.jobs.reason'), role: 'meta', render: (e) => e.message },
          ]}
        />
      )}
    </Stack>
  )
}
