import { ActionIcon, Anchor, Button, FileButton, Group, Stack, Text } from '@mantine/core'
import { IconFile, IconPaperclip, IconTrash } from '@tabler/icons-react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type Attachment } from '@/shared/api/core'
import { useMe } from '@/shared/auth/me'
import { errorText, formatDateTime, formatFileSize } from '@/shared/i18n'
import { useConfirm } from '@/shared/ui/confirm'
import { notifySuccess } from '@/shared/ui/notify'
import type { DocumentSection } from '@/shared/ui/page'
import { ContentSkeleton, ErrorState } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import { documentKeys } from './keys'
import { sectionTitle } from './sectionTitle'

function useAttachments(docType: string, id: number) {
  return useQuery({
    queryKey: documentKeys.attachments(docType, id),
    queryFn: () => unwrap(api.GET('/records/{type}/{id}/attachments', { params: { path: { type: docType, id } } })),
  })
}

// useAttachmentSection is the attachments of a record as a DocumentPage section, its count in the title.
export function useAttachmentSection(docType: string, id: number): DocumentSection {
  const { t } = useTranslation()
  const count = useAttachments(docType, id).data?.items.length
  return {
    key: 'attachments',
    title: sectionTitle(t, 'shared.attachments.title', count),
    content: <AttachmentPanel docType={docType} id={id} />,
  }
}

// AttachmentPanel lists a record's files to download and, as allowed_actions say, adds and removes them.
function AttachmentPanel({ docType, id }: { docType: string; id: number }) {
  const { t } = useTranslation()
  const me = useMe()
  const qc = useQueryClient()
  const q = useAttachments(docType, id)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [ask, confirm] = useConfirm()
  if (q.isPending) return <ContentSkeleton />
  if (q.isError) return <ErrorState message={errorText(t, q.error)} onRetry={() => void q.refetch()} />
  if (q.data.hidden)
    return (
      <Text size="sm" c="dimmed">
        {t('shared.attachments.forbidden')}
      </Text>
    )
  // The server tells a file's type from its content; checking the name first only spares
  // a long upload that would be refused.
  const { max_mb: maxMB, accept } = q.data

  async function add(file: File | null) {
    if (!file) return
    setError(null)
    if (!accept.some((ext) => file.name.toLowerCase().endsWith(ext))) return setError(t('shared.error.file_type_not_allowed'))
    if (file.size > maxMB << 20) return setError(t('shared.error.file_too_large', { max_mb: maxMB }))
    setBusy(true)
    try {
      await unwrap(
        api.POST('/records/{type}/{id}/attachments', {
          params: { path: { type: docType, id } },
          body: { file: '' },
          bodySerializer: () => {
            const data = new FormData()
            data.append('file', file)
            return data
          },
        }),
      )
      await qc.invalidateQueries({ queryKey: documentKeys.document(docType, id) })
      notifySuccess(t('shared.attachments.added', { name: file.name }))
    } catch (err) {
      setError(errorText(t, err))
    } finally {
      setBusy(false)
    }
  }

  async function remove(a: Attachment) {
    const yes = await ask({
      title: t('shared.attachments.remove_title'),
      message: t('shared.attachments.remove_confirm', { name: a.name }),
      confirmLabel: t('shared.attachments.remove'),
      danger: true,
    })
    if (!yes) return
    setError(null)
    try {
      await unwrap(api.DELETE('/attachments/{id}', { params: { path: { id: a.id } } }))
      await qc.invalidateQueries({ queryKey: documentKeys.document(docType, id) })
      notifySuccess(t('shared.attachments.removed', { name: a.name }))
    } catch (err) {
      setError(errorText(t, err))
    }
  }

  return (
    <Stack gap="sm">
      {confirm}
      {q.data.items.length === 0 ? (
        <Text size="sm" c="dimmed">
          {t('shared.attachments.none')}
        </Text>
      ) : (
        <Stack gap="xs">
          {q.data.items.map((a) => (
            <Group key={a.id} gap="xs" wrap="nowrap" align="center">
              <IconFile {...icon.text} aria-hidden />
              <Stack gap={0} miw={0} flex={1}>
                <Anchor href={`/api/attachments/${a.id}`} size="sm" truncate="end">
                  {a.name}
                </Anchor>
                <Text size="xs" c="dimmed">
                  {[formatFileSize(a.size), a.uploaded_by_name, formatDateTime(a.uploaded_at, me.timezone)].join(' · ')}
                </Text>
              </Stack>
              {a.allowed_actions.includes('delete') && (
                <ActionIcon variant="subtle" color="gray" size="lg" aria-label={t('shared.attachments.remove_named', { name: a.name })} onClick={() => void remove(a)}>
                  <IconTrash {...icon.text} />
                </ActionIcon>
              )}
            </Group>
          ))}
        </Stack>
      )}
      {error && (
        <Text size="xs" c="danger" role="alert">
          {error}
        </Text>
      )}
      {q.data.allowed_actions.includes('attach') && (
        <Group justify="flex-end">
          <FileButton onChange={(f) => void add(f)} accept={accept.join(',')}>
            {(props) => (
              <Button {...props} size="xs" variant="default" loading={busy} leftSection={<IconPaperclip {...icon.text} />}>
                {t('shared.attachments.add')}
              </Button>
            )}
          </FileButton>
        </Group>
      )}
    </Stack>
  )
}
