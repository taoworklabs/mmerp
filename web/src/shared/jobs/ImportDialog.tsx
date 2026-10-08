import { Anchor, Button, Group, Stack, Text } from '@mantine/core'
import { IconDownload } from '@tabler/icons-react'
import { useQueryClient, type QueryKey } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { coreClient, unwrap } from '@/shared/api/client'
import { FileField, Form, FormActions } from '@/shared/ui/form'
import { FormModal } from '@/shared/ui/page'
import { icon } from '@/shared/ui/theme'
import { JobOutcome } from './JobOutcome'
import { finished, followJob, useJobStatus } from './useJobStatus'

type Props = {
  kind: string // the import, e.g. hrm.timesheet
  params?: Record<string, unknown>
  title: string
  description: string
  // Queries to refresh once the import has written.
  invalidate: QueryKey[]
  onClose: () => void
}

// ImportDialog uploads an Excel file, then follows the import job to its end. The import
// writes all rows or none; a failed one lists the rows to fix.
export function ImportDialog({ kind, params = {}, title, description, invalidate, onClose }: Props) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [job, setJob] = useState<number | null>(null)
  const status = useJobStatus(job, invalidate)
  // Closed while the import still runs: follow it from outside, so its result still shows.
  const close = () => {
    if (job !== null && !finished(status.data) && !status.isError) void followJob(qc, job, invalidate)
    onClose()
  }
  const form = useForm<{ file: File | null }>({ mode: 'onBlur', defaultValues: { file: null } })
  const json = JSON.stringify(params)

  async function start(v: { file: File | null }) {
    const out = await unwrap(
      coreClient.POST('/imports/{kind}', {
        params: { path: { kind } },
        body: { file: '', params: json },
        bodySerializer: () => {
          const data = new FormData()
          data.append('file', v.file as File)
          data.append('params', json)
          return data
        },
      }),
    )
    setJob(out.job_id)
  }

  return (
    <FormModal opened title={title} onClose={close}>
      {job === null ? (
        <Form form={form} onSubmit={start}>
          <Stack gap="sm">
            <Text size="sm">{description}</Text>
            <Anchor href={`/api/imports/${kind}/template?params=${encodeURIComponent(json)}`} size="sm" w="fit-content">
              <Group gap="xxs" component="span">
                <IconDownload {...icon.text} aria-hidden />
                {t('shared.jobs.template')}
              </Group>
            </Anchor>
            <FileField name="file" label={t('shared.jobs.file')} description={t('shared.jobs.file_hint')} accept=".xlsx" required />
          </Stack>
          <FormActions submitLabel={t('shared.jobs.start_import')} onCancel={onClose} cancelLabel={t('shared.jobs.close')} />
        </Form>
      ) : (
        <Stack gap="md">
          <JobOutcome job={status.data} error={status.error} />
          <Group justify="flex-end">
            {status.data?.state === 'failed' && (
              <Button variant="default" onClick={() => setJob(null)}>
                {t('shared.jobs.retry')}
              </Button>
            )}
            <Button onClick={close} variant={finished(status.data) ? 'filled' : 'default'}>
              {t('shared.jobs.close')}
            </Button>
          </Group>
        </Stack>
      )}
    </FormModal>
  )
}
