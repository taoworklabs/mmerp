import { Box, Button, Combobox, Group, Paper, Stack, Text, Textarea, ThemeIcon, Tooltip, useCombobox } from '@mantine/core'
import { IconMessages } from '@tabler/icons-react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api, unwrap } from '@/shared/api/core'
import { useMe } from '@/shared/auth/me'
import { errorText, formatAgo, formatDateTime } from '@/shared/i18n'
import { PersonAvatar } from '@/shared/ui/avatar'
import type { DocumentSection } from '@/shared/ui/page'
import { ContentSkeleton, ErrorState } from '@/shared/ui/states'
import { icon } from '@/shared/ui/theme'
import classes from './DiscussionPanel.module.css'
import { documentKeys } from './keys'
import { mentionAt, splitMentions } from './mentions'
import { sectionTitle } from './sectionTitle'

function useDiscussion(docType: string, id: number) {
  return useQuery({
    queryKey: documentKeys.discussion(docType, id),
    queryFn: () => unwrap(api.GET('/records/{type}/{id}/comments', { params: { path: { type: docType, id } } })),
  })
}

// useDiscussionSection is the discussion of a record as a DocumentPage section, its count in the title.
export function useDiscussionSection(docType: string, id: number): DocumentSection {
  const { t } = useTranslation()
  const count = useDiscussion(docType, id).data?.items.length
  return {
    key: 'discussion',
    title: sectionTitle(t, 'shared.discussion.title', count),
    content: <DiscussionPanel docType={docType} id={id} />,
  }
}

// DiscussionPanel is the comments on a record as a conversation, oldest first, each under its
// author's avatar and name, and a box to add one when allowed_actions say so.
function DiscussionPanel({ docType, id }: { docType: string; id: number }) {
  const { t } = useTranslation()
  const me = useMe()
  const qc = useQueryClient()
  const q = useDiscussion(docType, id)
  const [body, setBody] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  if (q.isPending) return <ContentSkeleton />
  if (q.isError) return <ErrorState message={errorText(t, q.error)} onRetry={() => void q.refetch()} />

  async function send() {
    if (!body.trim()) return
    setError(null)
    setBusy(true)
    try {
      await unwrap(api.POST('/records/{type}/{id}/comments', { params: { path: { type: docType, id } }, body: { body } }))
      setBody('')
      await qc.invalidateQueries({ queryKey: documentKeys.document(docType, id) })
    } catch (err) {
      setError(errorText(t, err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Stack gap="md" className={classes.panel}>
      {q.data.items.length === 0 ? (
        <Paper withBorder>
          <Stack align="center" gap="xs" p="lg">
            <ThemeIcon variant="light" color="gray" size={40} radius="xl" className={classes.bigIcon} aria-hidden>
              <IconMessages {...icon.button} />
            </ThemeIcon>
            <Text size="sm" c="dimmed">
              {t('shared.discussion.none')}
            </Text>
          </Stack>
        </Paper>
      ) : (
        <Stack component="ol" gap="md" className={classes.thread}>
          {q.data.items.map((c) => (
            <Group key={c.id} component="li" align="flex-start" gap="sm" wrap="nowrap">
              <PersonAvatar name={c.author_name} size="md" />
              <Stack gap="xxs" miw={0} flex={1}>
                <Group gap="xs" wrap="nowrap">
                  <Text size="sm" fw={600} truncate>
                    {c.author_name}
                  </Text>
                  {/* How long ago reads at a glance; the exact time is a hover away. */}
                  <Tooltip label={formatDateTime(c.created_at, me.timezone)}>
                    <Text component="time" dateTime={c.created_at} size="xs" c="dimmed" className={classes.time}>
                      {formatAgo(c.created_at, me.timezone)}
                    </Text>
                  </Tooltip>
                </Group>
                <Box className={classes.bubble}>
                  <Text size="sm" className={classes.body}>
                    {splitMentions(c.body).map((part, i) =>
                      part.mention ? (
                        <Text key={i} span fw={600} c="primary">
                          {part.text}
                        </Text>
                      ) : (
                        part.text
                      ),
                    )}
                  </Text>
                </Box>
              </Stack>
            </Group>
          ))}
        </Stack>
      )}
      {q.data.allowed_actions.includes('comment') && (
        // The box sits where the next comment will, under the writer's own avatar.
        <Group align="flex-start" gap="sm" wrap="nowrap">
          <PersonAvatar name={me.name} size="md" />
          <Stack gap="xs" miw={0} flex={1}>
            <MentionTextarea docType={docType} id={id} maxLength={q.data.max_length} value={body} error={error} onChange={setBody} />
            <Group justify="flex-end">
              <Button variant="default" loading={busy} disabled={!body.trim()} onClick={() => void send()}>
                {t('shared.discussion.send')}
              </Button>
            </Group>
          </Stack>
        </Group>
      )}
    </Stack>
  )
}

// MentionTextarea is the comment box; typing @ offers the users who may view the record,
// and picking one writes @login.
function MentionTextarea({
  docType,
  id,
  maxLength,
  value,
  error,
  onChange,
}: {
  docType: string
  id: number
  maxLength: number
  value: string
  error: string | null
  onChange: (value: string) => void
}) {
  const { t } = useTranslation()
  const combobox = useCombobox({ onDropdownClose: () => combobox.resetSelectedOption() })
  const ref = useRef<HTMLTextAreaElement>(null)
  const [token, setToken] = useState<{ start: number; query: string } | null>(null)
  const users = useQuery({
    queryKey: documentKeys.mentionable(docType, id),
    queryFn: () => unwrap(api.GET('/records/{type}/{id}/mentionable', { params: { path: { type: docType, id } } })),
    enabled: token !== null,
  })
  const query = token?.query.toLowerCase() ?? ''
  const options = (users.data ?? []).filter((u) => u.login.toLowerCase().includes(query) || u.name.toLowerCase().includes(query)).slice(0, 8)
  const open = token !== null && options.length > 0

  // Not on combobox: it is a new object every render, and re-selecting the first option would undo the arrow keys.
  useEffect(() => {
    if (open) {
      combobox.openDropdown()
      combobox.selectFirstOption()
    } else combobox.closeDropdown()
  }, [open, query])

  const follow = (text: string, caret: number) => {
    onChange(text)
    setToken(mentionAt(text, caret))
  }
  const pick = (login: string) => {
    if (!token) return
    const caret = token.start + 1 + token.query.length
    const text = `${value.slice(0, token.start)}@${login} ${value.slice(caret)}`
    const at = token.start + login.length + 2
    onChange(text)
    setToken(null)
    requestAnimationFrame(() => ref.current?.setSelectionRange(at, at))
  }

  return (
    <Combobox store={combobox} onOptionSubmit={pick} position="bottom-start">
      <Combobox.Target>
        <Textarea
          ref={ref}
          aria-label={t('shared.discussion.input')}
          placeholder={t('shared.discussion.placeholder')}
          description={t('shared.discussion.mention_hint')}
          autosize
          minRows={2}
          maxRows={8}
          maxLength={maxLength}
          value={value}
          error={error}
          onChange={(e) => follow(e.currentTarget.value, e.currentTarget.selectionStart)}
          onClick={(e) => setToken(mentionAt(value, e.currentTarget.selectionStart))}
          onBlur={() => setToken(null)}
        />
      </Combobox.Target>
      <Combobox.Dropdown>
        <Combobox.Options aria-label={t('shared.discussion.mention')}>
          {options.map((u) => (
            <Combobox.Option value={u.login} key={u.login}>
              <Text size="sm">
                {u.name}{' '}
                <Text span size="xs" c="dimmed">
                  @{u.login}
                </Text>
              </Text>
            </Combobox.Option>
          ))}
        </Combobox.Options>
      </Combobox.Dropdown>
    </Combobox>
  )
}
