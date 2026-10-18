import { Anchor, Button, Group, Stack, Text } from '@mantine/core'
import { IconDownload } from '@tabler/icons-react'
import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { coreClient, unwrap } from '@/shared/api/client'
import { FileField, Form, FormActions } from '@/shared/ui/form'
import { FormModal } from '@/shared/ui/page'
import { ModalFooter } from '@/shared/ui/ModalFooter'
import { icon } from '@/shared/ui/theme'
import { JobOutcome } from './JobOutcome'
import { finished, followJob, useJobRefresh, useJobStatus, type JobWrites } from './useJobStatus'

type Props = {
  kind: string // the import, e.g. hrm.timesheet
  params?: Record<string, unknown>
  title: string
  description: string
  // What the import writes, refreshed once it has: a record, or other queries.
  writes: JobWrites
  onClose: () => void
}

// ImportDialog uploads an Excel file, then follows the import job to its end. The import
// writes all rows or none; a failed one lists the rows to fix.
export function ImportDialog({ kind, params = {}, title, description, writes, onClose }: Props) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [job, setJob] = useState<number | null>(null)
  const refresh = useJobRefresh(writes)
  const status = useJobStatus(job, writes)
  // Closed while the import still runs: follow it from outside, so its result still shows.
  const close = () => {
    if (job !== null && !finished(status.data) && !status.isError) void followJob(qc, job, refresh)
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
    <FormModal opened wide title={title} onClose={close}>
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
          <ModalFooter>
            {status.data?.state === 'failed' && (
              <Button variant="default" onClick={() => setJob(null)}>
                {t('shared.jobs.retry')}
              </Button>
            )}
            <Button onClick={close} variant={finished(status.data) ? 'filled' : 'default'}>
              {t('shared.jobs.close')}
            </Button>
          </ModalFooter>
        </Stack>
      )}
    </FormModal>
  )
}
