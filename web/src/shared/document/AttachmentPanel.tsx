import { ActionIcon, Anchor, Button, FileButton, Group, Paper, Stack, Text, ThemeIcon } from '@mantine/core'
import {
  IconFile,
  IconFileTypeDocx,
  IconFileTypePdf,
  IconFileTypeXls,
  IconPaperclip,
  IconPhoto,
  IconTrash,
  type Icon as TablerIcon,
} from '@tabler/icons-react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState, type DragEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { api, unwrap, type Attachment } from '@/shared/api/core'
import { useMe } from '@/shared/auth/me'
import { errorText, formatDateTime, formatFileSize } from '@/shared/i18n'
import { useConfirm } from '@/shared/ui/confirm'
import { notifySuccess } from '@/shared/ui/notify'
import type { DocumentSection } from '@/shared/ui/page'
import { ContentSkeleton, ErrorState } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import classes from './AttachmentPanel.module.css'
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

// Each kind of file has its own icon and accent colour, so a list of them reads at a glance.
const photo = { Icon: IconPhoto, color: 'teal' }
const fileLooks: Record<string, { Icon: TablerIcon; color: string }> = {
  pdf: { Icon: IconFileTypePdf, color: 'red' },
  png: photo,
  jpg: photo,
  jpeg: photo,
  docx: { Icon: IconFileTypeDocx, color: 'blue' },
  xlsx: { Icon: IconFileTypeXls, color: 'green' },
}
const fileLook = (name: string) => fileLooks[name.toLowerCase().split('.').pop() ?? ''] ?? { Icon: IconFile, color: 'gray' }

// AttachmentPanel lists a record's files to download and, as allowed_actions say, adds and removes
// them: through the button above the list, or by dropping files anywhere on the panel.
function AttachmentPanel({ docType, id }: { docType: string; id: number }) {
  const { t } = useTranslation()
  const me = useMe()
  const qc = useQueryClient()
  const q = useAttachments(docType, id)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [dragging, setDragging] = useState(false)
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
  const canAttach = q.data.allowed_actions.includes('attach')
  const empty = q.data.items.length === 0

  // One file at a time, so each upload succeeds or fails on its own.
  async function addAll(files: File[]) {
    for (const f of files) await add(f)
  }

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

  const drop = canAttach
    ? {
        onDragOver: (e: DragEvent) => {
          if (!e.dataTransfer.types.includes('Files')) return
          e.preventDefault()
          setDragging(true)
        },
        // Leaving for a child of the panel is not leaving the panel.
        onDragLeave: (e: DragEvent) => {
          if (!e.currentTarget.contains(e.relatedTarget as Node | null)) setDragging(false)
        },
        onDrop: (e: DragEvent) => {
          e.preventDefault()
          setDragging(false)
          void addAll([...e.dataTransfer.files])
        },
      }
    : {}

  return (
    <Stack gap="sm" className={classes.panel} {...drop}>
      {confirm}
      {canAttach && (
        <Group justify="flex-end">
          <FileButton multiple onChange={(files) => void addAll(files)} accept={accept.join(',')}>
            {(props) => (
              <Button {...props} variant="default" loading={busy} leftSection={<IconPaperclip {...icon.button} />}>
                {t('shared.attachments.add')}
              </Button>
            )}
          </FileButton>
        </Group>
      )}
      {error && (
        <Text size="xs" c="danger" role="alert">
          {error}
        </Text>
      )}
      {/* The files sit in a frame, like a table inside a page; it lights up while files are dragged over the panel. */}
      <Paper withBorder className={classes.list} data-active={dragging || undefined}>
        {empty ? (
          <Stack align="center" gap="xs" p="lg">
            <ThemeIcon variant="light" color={dragging ? 'primary' : 'gray'} size={40} radius="xl" className={classes.bigIcon} aria-hidden>
              <IconPaperclip {...icon.button} />
            </ThemeIcon>
            <Text size="sm" c="dimmed">
              {t(dragging ? 'shared.attachments.drop' : 'shared.attachments.none')}
            </Text>
          </Stack>
        ) : (
          q.data.items.map((a) => {
            const { Icon, color } = fileLook(a.name)
            return (
              <Group key={a.id} gap="sm" wrap="nowrap" align="center" p="sm" className={classes.row}>
                <ThemeIcon variant="light" color={color} size={36} radius="md" aria-hidden>
                  <Icon {...icon.button} />
                </ThemeIcon>
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
            )
          })
        )}
      </Paper>
    </Stack>
  )
}
